package bot

import (
	"errors"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/storage"
)

type mockSender struct {
	sent      []tgbotapi.Chattable
	requested []tgbotapi.Chattable
	sendErr   error
	nextID    int
}

func (m *mockSender) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.sent = append(m.sent, c)
	if m.sendErr != nil {
		return tgbotapi.Message{}, m.sendErr
	}
	m.nextID++
	return tgbotapi.Message{MessageID: 100 + m.nextID}, nil
}

func (m *mockSender) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	m.requested = append(m.requested, c)
	return &tgbotapi.APIResponse{Ok: true}, nil
}

type mockRepo struct {
	data    storage.Data
	loadErr error
	saveErr error
	saves   int
}

func (r *mockRepo) Load() (storage.Data, error) {
	return r.data, r.loadErr
}

func (r *mockRepo) Save(data storage.Data) error {
	r.saves++
	if r.saveErr != nil {
		return r.saveErr
	}
	r.data = data
	return nil
}

var testGameDays = map[time.Weekday]bool{
	time.Tuesday:  true,
	time.Thursday: true,
	time.Sunday:   true,
}

// tuesday: через createDaysAhead дней тоже вторник, 17.11.2026.
func tuesday() time.Time {
	return time.Date(2026, time.November, 10, 20, 0, 0, 0, time.UTC)
}

// wednesday: через createDaysAhead дней среда, не игровой день.
func wednesday() time.Time {
	return time.Date(2026, time.November, 11, 20, 0, 0, 0, time.UTC)
}

func newService(sender *mockSender, repo *mockRepo, now func() time.Time) *Service {
	s := New(sender, repo, -1001, time.UTC, testGameDays)
	s.now = now
	return s
}

func TestCreateGame_GameDay(t *testing.T) {
	sender := &mockSender{}
	repo := &mockRepo{}
	newService(sender, repo, tuesday).CreateGame()

	if len(sender.sent) != 1 {
		t.Fatalf("ожидалось одно сообщение, отправлено %d", len(sender.sent))
	}
	msg, ok := sender.sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("отправлено не MessageConfig: %T", sender.sent[0])
	}
	if msg.ChatID != -1001 || msg.ParseMode != tgbotapi.ModeHTML {
		t.Errorf("ChatID = %d, ParseMode = %q", msg.ChatID, msg.ParseMode)
	}
	if want := "<b>Вторник 17.11 в 19:00</b>\nЗапись открыта · 0/12\nсвободно: 12"; msg.Text != want {
		t.Errorf("текст:\n%s\nожидалось:\n%s", msg.Text, want)
	}
	if kb, ok := msg.ReplyMarkup.(*tgbotapi.InlineKeyboardMarkup); !ok || kb == nil || len(kb.InlineKeyboard) != 2 {
		t.Errorf("ожидалась клавиатура записи из двух рядов: %#v", msg.ReplyMarkup)
	}

	if len(repo.data.Games) != 1 {
		t.Fatalf("ожидалась одна сохранённая игра: %+v", repo.data.Games)
	}
	if g := repo.data.Games[0]; g.Date != "2026-11-17" || g.MessageID != 101 {
		t.Errorf("сохранено %+v, ожидалась дата 2026-11-17 и сообщение 101", g)
	}
	if len(sender.requested) != 0 {
		t.Errorf("сообщение игры не закрепляется: %+v", sender.requested)
	}
}

func TestCreateGame_NonGameDay(t *testing.T) {
	sender := &mockSender{}
	repo := &mockRepo{}
	newService(sender, repo, wednesday).CreateGame()

	if len(sender.sent) != 0 || repo.saves != 0 {
		t.Fatalf("в неигровой день ничего не отправляется и не сохраняется: sent=%d saves=%d", len(sender.sent), repo.saves)
	}
}

func TestCreateGame_SkipIfExists(t *testing.T) {
	sender := &mockSender{}
	repo := &mockRepo{data: storage.Data{Games: []storage.Game{{Date: "2026-11-17", MessageID: 55}}}}
	newService(sender, repo, tuesday).CreateGame()

	if len(sender.sent) != 0 || repo.saves != 0 {
		t.Fatalf("игра уже есть, дубль не создаётся: sent=%d saves=%d", len(sender.sent), repo.saves)
	}
}

func TestCreateGame_KeepsOtherGames(t *testing.T) {
	sender := &mockSender{}
	repo := &mockRepo{data: storage.Data{Games: []storage.Game{{Date: "2026-11-15", MessageID: 55}}}}
	newService(sender, repo, tuesday).CreateGame()

	if len(repo.data.Games) != 2 || repo.data.Games[0].MessageID != 55 {
		t.Fatalf("старая игра должна остаться, новая добавиться: %+v", repo.data.Games)
	}
}

func TestCreateGame_SendError(t *testing.T) {
	sender := &mockSender{sendErr: errors.New("сеть")}
	repo := &mockRepo{}
	newService(sender, repo, tuesday).CreateGame()

	if repo.saves != 0 {
		t.Fatalf("при ошибке отправки ничего не сохраняется")
	}
}

func TestCreateGame_LoadError(t *testing.T) {
	sender := &mockSender{}
	repo := &mockRepo{loadErr: errors.New("битый файл")}
	newService(sender, repo, tuesday).CreateGame()

	if len(sender.sent) != 0 || repo.saves != 0 {
		t.Fatalf("при ошибке чтения ничего не отправляется: sent=%d saves=%d", len(sender.sent), repo.saves)
	}
}
