package bot

import (
	"encoding/json"
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
	// requestErr возвращает ошибку для конкретного запроса, nil означает успех.
	requestErr func(request tgbotapi.Chattable) error
	// pinnedMessageID отдаётся в ответе getChat как закреплённое сообщение, 0 означает «закрепа нет».
	pinnedMessageID int
}

func (sender *mockSender) Send(request tgbotapi.Chattable) (tgbotapi.Message, error) {
	sender.sent = append(sender.sent, request)
	if sender.sendErr != nil {
		return tgbotapi.Message{}, sender.sendErr
	}
	sender.nextID++
	return tgbotapi.Message{MessageID: 100 + sender.nextID}, nil
}

func (sender *mockSender) Request(request tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	sender.requested = append(sender.requested, request)
	if sender.requestErr != nil {
		if err := sender.requestErr(request); err != nil {
			return nil, err
		}
	}
	if _, ok := request.(tgbotapi.ChatInfoConfig); ok {
		chat := tgbotapi.Chat{ID: -1001}
		if sender.pinnedMessageID != 0 {
			chat.PinnedMessage = &tgbotapi.Message{MessageID: sender.pinnedMessageID}
		}
		result, _ := json.Marshal(chat)
		return &tgbotapi.APIResponse{Ok: true, Result: result}, nil
	}
	return &tgbotapi.APIResponse{Ok: true}, nil
}

type mockRepo struct {
	data    storage.Data
	loadErr error
	saveErr error
	saves   int
}

func (repository *mockRepo) Load() (storage.Data, error) {
	return repository.data, repository.loadErr
}

func (repository *mockRepo) Save(data storage.Data) error {
	repository.saves++
	if repository.saveErr != nil {
		return repository.saveErr
	}
	repository.data = data
	return nil
}

// testDeleteAfterDays задаёт срок удаления игр в тестах.
const testDeleteAfterDays = 14

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

func newService(sender *mockSender, repository *mockRepo, now func() time.Time) *Service {
	service := New(sender, repository, -1001, time.UTC, testGameDays, testDeleteAfterDays)
	service.now = now
	return service
}

func TestCreateGame_GameDay(t *testing.T) {
	sender := &mockSender{}
	repository := &mockRepo{}
	newService(sender, repository, tuesday).CreateGame()

	// Первое сообщение это игра, второе это доска: в пустом чате её ещё нет.
	if len(sender.sent) != 2 {
		t.Fatalf("ожидались сообщение игры и доска, отправлено %d", len(sender.sent))
	}
	message, ok := sender.sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("отправлено не MessageConfig: %T", sender.sent[0])
	}
	if message.ChatID != -1001 || message.ParseMode != tgbotapi.ModeHTML {
		t.Errorf("ChatID = %d, ParseMode = %q", message.ChatID, message.ParseMode)
	}
	if want := "<b>Вторник 17.11 в 19:00</b>\nЗапись открыта · 0/12\nсвободно: 12"; message.Text != want {
		t.Errorf("текст:\n%s\nожидалось:\n%s", message.Text, want)
	}
	if keyboard, ok := message.ReplyMarkup.(*tgbotapi.InlineKeyboardMarkup); !ok || keyboard == nil || len(keyboard.InlineKeyboard) != 2 {
		t.Errorf("ожидалась клавиатура записи из двух рядов: %#v", message.ReplyMarkup)
	}

	if len(repository.data.Games) != 1 {
		t.Fatalf("ожидалась одна сохранённая игра: %+v", repository.data.Games)
	}
	if saved := repository.data.Games[0]; saved.Date != "2026-11-17" || saved.MessageID != 101 {
		t.Errorf("сохранено %+v, ожидалась дата 2026-11-17 и сообщение 101", saved)
	}
	if repository.data.BoardMessageID != 102 {
		t.Errorf("BoardMessageID = %d, ожидалось 102", repository.data.BoardMessageID)
	}
	// Закрепляется только доска, сообщение игры нет.
	for _, request := range sender.requested {
		if pin, ok := request.(tgbotapi.PinChatMessageConfig); ok && pin.MessageID != 102 {
			t.Errorf("закреплено не то сообщение: %+v", pin)
		}
	}
}

func TestCreateGame_NonGameDay(t *testing.T) {
	sender := &mockSender{}
	repository := &mockRepo{}
	newService(sender, repository, wednesday).CreateGame()

	if len(sender.sent) != 0 || repository.saves != 0 {
		t.Fatalf("в неигровой день ничего не отправляется и не сохраняется: sent=%d saves=%d", len(sender.sent), repository.saves)
	}
}

func TestCreateGame_SkipIfExists(t *testing.T) {
	sender := &mockSender{}
	repository := &mockRepo{data: storage.Data{Games: []storage.Game{{Date: "2026-11-17", MessageID: 55}}}}
	newService(sender, repository, tuesday).CreateGame()

	if len(sender.sent) != 0 || repository.saves != 0 {
		t.Fatalf("игра уже есть, дубль не создаётся: sent=%d saves=%d", len(sender.sent), repository.saves)
	}
}

func TestCreateGame_KeepsOtherGames(t *testing.T) {
	sender := &mockSender{}
	repository := &mockRepo{data: storage.Data{Games: []storage.Game{{Date: "2026-11-15", MessageID: 55}}}}
	newService(sender, repository, tuesday).CreateGame()

	if len(repository.data.Games) != 2 || repository.data.Games[0].MessageID != 55 {
		t.Fatalf("старая игра должна остаться, новая добавиться: %+v", repository.data.Games)
	}
}

func TestCreateGame_SendError(t *testing.T) {
	sender := &mockSender{sendErr: errors.New("сеть")}
	repository := &mockRepo{}
	newService(sender, repository, tuesday).CreateGame()

	if repository.saves != 0 {
		t.Fatalf("при ошибке отправки ничего не сохраняется")
	}
}

func TestCreateGame_LoadError(t *testing.T) {
	sender := &mockSender{}
	repository := &mockRepo{loadErr: errors.New("битый файл")}
	newService(sender, repository, tuesday).CreateGame()

	if len(sender.sent) != 0 || repository.saves != 0 {
		t.Fatalf("при ошибке чтения ничего не отправляется: sent=%d saves=%d", len(sender.sent), repository.saves)
	}
}
