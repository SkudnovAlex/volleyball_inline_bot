package bot

import (
	"errors"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/game"
	"volleyball_inline_bot/storage"
)

const testChatID = -1001

// testBoardID это доска, которая уже есть в чате и закреплена.
const testBoardID = 900

var maria = &tgbotapi.User{ID: 1, FirstName: "Мария", LastName: "Иванова"}

// press собирает нажатие кнопки в сообщении игры.
func press(user *tgbotapi.User, data string, chatID int64) *tgbotapi.CallbackQuery {
	return &tgbotapi.CallbackQuery{
		ID:      "query",
		From:    user,
		Data:    data,
		Message: &tgbotapi.Message{MessageID: 50, Chat: &tgbotapi.Chat{ID: chatID}},
	}
}

// openGame возвращает данные с одной игрой 17.11, где записались players человек.
func openGame(players int) storage.Data {
	match := storage.Game{Date: "2026-11-17", MessageID: 50}
	for index := range players {
		match.Players = append(match.Players, storage.Player{UserID: int64(100 + index), Name: "Игрок", Status: storage.StatusGoing})
	}
	return storage.Data{BoardMessageID: testBoardID, Games: []storage.Game{match}}
}

// callbacks возвращает ответы на нажатия из запросов к Telegram.
func callbacks(requests []tgbotapi.Chattable) []tgbotapi.CallbackConfig {
	var result []tgbotapi.CallbackConfig
	for _, request := range requests {
		if callback, ok := request.(tgbotapi.CallbackConfig); ok {
			result = append(result, callback)
		}
	}
	return result
}

// countPins считает запросы на закрепление.
func countPins(requests []tgbotapi.Chattable) int {
	count := 0
	for _, request := range requests {
		if _, ok := request.(tgbotapi.PinChatMessageConfig); ok {
			count++
		}
	}
	return count
}

// edits возвращает редактирования сообщений из запросов к Telegram.
func edits(requests []tgbotapi.Chattable) []tgbotapi.EditMessageTextConfig {
	var result []tgbotapi.EditMessageTextConfig
	for _, request := range requests {
		if edit, ok := request.(tgbotapi.EditMessageTextConfig); ok {
			result = append(result, edit)
		}
	}
	return result
}

func TestHandleCallback_Join(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: openGame(3)}
	newService(sender, repository, tuesday).HandleCallback(press(maria, "join:2026-11-17", testChatID))

	players := repository.data.Games[0].Players
	if repository.saves != 1 || len(players) != 4 || players[3].Name != "Мария Иванова" {
		t.Fatalf("Мария должна быть записана и сохранена: saves=%d players=%+v", repository.saves, players)
	}

	allEdits := edits(sender.requested)
	if len(allEdits) != 2 {
		t.Fatalf("ожидалось редактирование игры и доски, получено %d", len(allEdits))
	}
	gameEdit, boardEdit := allEdits[0], allEdits[1]
	if gameEdit.MessageID != 50 || gameEdit.ParseMode != tgbotapi.ModeHTML || !strings.Contains(gameEdit.Text, "4/12") || gameEdit.ReplyMarkup == nil {
		t.Errorf("неверное редактирование игры: %+v", gameEdit)
	}
	if boardEdit.MessageID != testBoardID || !strings.Contains(boardEdit.Text, "Вторник 17.11</a> · 4/12") {
		t.Errorf("неверное редактирование доски: %+v", boardEdit)
	}
	if pins := countPins(sender.requested); pins != 0 {
		t.Errorf("доска уже в закрепе, повторно закреплять не нужно: %d", pins)
	}

	answers := callbacks(sender.requested)
	if len(answers) != 1 || answers[0].ShowAlert || answers[0].Text != "" {
		t.Errorf("ожидался пустой ответ без окна: %+v", answers)
	}
	if len(sender.sent) != 0 {
		t.Errorf("объявлений быть не должно: %+v", sender.sent)
	}
}

func TestHandleCallback_ActionError(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: openGame(0)}
	service := newService(sender, repository, tuesday)
	service.HandleCallback(press(maria, "join:2026-11-17", testChatID))
	sender.requested = nil

	service.HandleCallback(press(maria, "join:2026-11-17", testChatID))

	if repository.saves != 1 {
		t.Errorf("повторное нажатие не должно сохраняться, saves=%d", repository.saves)
	}
	if len(edits(sender.requested)) != 0 {
		t.Errorf("при ошибке сообщение игры не редактируется")
	}
	answers := callbacks(sender.requested)
	if len(answers) != 1 || !answers[0].ShowAlert || answers[0].Text != game.ErrAlreadyGoing.Error() {
		t.Fatalf("ожидалось окно «%s»: %+v", game.ErrAlreadyGoing, answers)
	}
}

func TestHandleCallback_Closed(t *testing.T) {
	finished := openGame(2)
	finished.Games[0].Finished = true

	cases := []struct {
		name string
		data storage.Data
		key  string
	}{
		{"неизвестная игра", openGame(2), "join:2026-12-01"},
		{"прошедшая игра", finished, "join:2026-11-17"},
		{"неизвестное действие", openGame(2), "dance:2026-11-17"},
	}
	for _, testCase := range cases {
		sender := &mockSender{pinnedMessageID: testBoardID}
		repository := &mockRepo{data: testCase.data}
		newService(sender, repository, tuesday).HandleCallback(press(maria, testCase.key, testChatID))

		answers := callbacks(sender.requested)
		if repository.saves != 0 || len(answers) != 1 || !answers[0].ShowAlert || answers[0].Text != textClosed {
			t.Errorf("%s: ожидалось окно «%s» без сохранения: saves=%d answers=%+v", testCase.name, textClosed, repository.saves, answers)
		}
	}
}

func TestHandleCallback_IgnoresOtherChats(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: openGame(0)}
	newService(sender, repository, tuesday).HandleCallback(press(maria, "join:2026-11-17", -999))

	if repository.saves != 0 || len(edits(sender.requested)) != 0 {
		t.Fatalf("нажатия из другого чата пропускаются")
	}
	if answers := callbacks(sender.requested); len(answers) != 1 || answers[0].ShowAlert {
		t.Fatalf("на нажатие всё равно нужен пустой ответ: %+v", answers)
	}
}

func TestHandleCallback_BadData(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: openGame(0)}
	newService(sender, repository, tuesday).HandleCallback(press(maria, "мусор", testChatID))

	if repository.saves != 0 || len(callbacks(sender.requested)) != 1 {
		t.Fatalf("непонятные данные пропускаются с пустым ответом")
	}
}

func TestHandleCallback_SaveError(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: openGame(0), saveErr: errors.New("диск")}
	newService(sender, repository, tuesday).HandleCallback(press(maria, "join:2026-11-17", testChatID))

	if len(edits(sender.requested)) != 0 {
		t.Errorf("если не сохранилось, сообщение игры не редактируется")
	}
	if answers := callbacks(sender.requested); len(answers) != 1 || !answers[0].ShowAlert {
		t.Errorf("ожидалось окно с просьбой повторить: %+v", answers)
	}
}

func TestHandleCallback_NotModifiedIgnored(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID, requestErr: func(request tgbotapi.Chattable) error {
		if _, ok := request.(tgbotapi.EditMessageTextConfig); ok {
			return &tgbotapi.Error{Code: 400, Message: "Bad Request: message is not modified"}
		}
		return nil
	}}
	repository := &mockRepo{data: openGame(0)}
	newService(sender, repository, tuesday).HandleCallback(press(maria, "join:2026-11-17", testChatID))

	if answers := callbacks(sender.requested); len(answers) != 1 || answers[0].ShowAlert {
		t.Fatalf("ошибка «not modified» не должна мешать ответу: %+v", answers)
	}
}

func TestHandleCallback_AnnounceFilled(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: openGame(game.Limit - 1)}
	newService(sender, repository, tuesday).HandleCallback(press(maria, "join:2026-11-17", testChatID))

	if len(sender.sent) != 1 {
		t.Fatalf("ожидалось одно объявление, отправлено %d", len(sender.sent))
	}
	message := sender.sent[0].(tgbotapi.MessageConfig)
	if message.ReplyToMessageID != 50 || message.Text != "<b>Вторник 17.11</b>: состав собран, 12/12. Запись закрыта" {
		t.Errorf("неверное объявление: %+v", message)
	}
	announce := repository.data.Games[0].Announce
	if announce == nil || announce.MessageID != 101 || !announce.Full || announce.SentAt != tuesday().Unix() {
		t.Errorf("объявление должно сохраниться: %+v", announce)
	}
}

func TestHandleCallback_AnnounceOpenedEditsRecent(t *testing.T) {
	data := openGame(game.Limit)
	data.Games[0].Players[0] = storage.Player{UserID: maria.ID, Name: "Мария", Status: storage.StatusGoing}
	sentAt := tuesday().Add(-5 * time.Minute).Unix()
	data.Games[0].Announce = &storage.Announce{MessageID: 77, Full: true, SentAt: sentAt}

	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	newService(sender, repository, tuesday).HandleCallback(press(maria, "decline:2026-11-17", testChatID))

	if len(sender.sent) != 0 {
		t.Fatalf("объявление моложе 10 минут редактируется, а не отправляется заново: %+v", sender.sent)
	}
	var announceEdit *tgbotapi.EditMessageTextConfig
	for _, edit := range edits(sender.requested) {
		if edit.MessageID == 77 {
			announceEdit = &edit
		}
	}
	if announceEdit == nil || announceEdit.Text != "<b>Вторник 17.11</b>: освободилось мест: 1. Запись снова открыта" {
		t.Fatalf("ожидалось редактирование объявления 77: %+v", edits(sender.requested))
	}
	announce := repository.data.Games[0].Announce
	if announce.MessageID != 77 || announce.Full || announce.SentAt != sentAt {
		t.Errorf("объявление то же, Full=false, время прежнее: %+v", announce)
	}
}

func TestHandleCallback_AnnounceOpenedSendsAfterCooldown(t *testing.T) {
	data := openGame(game.Limit)
	data.Games[0].Players[0] = storage.Player{UserID: maria.ID, Name: "Мария", Status: storage.StatusGoing}
	data.Games[0].Announce = &storage.Announce{MessageID: 77, Full: true, SentAt: tuesday().Add(-time.Hour).Unix()}

	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	newService(sender, repository, tuesday).HandleCallback(press(maria, "decline:2026-11-17", testChatID))

	if len(sender.sent) != 1 {
		t.Fatalf("старое объявление: ожидалось новое сообщение, отправлено %d", len(sender.sent))
	}
	if announce := repository.data.Games[0].Announce; announce.MessageID != 101 || announce.Full {
		t.Errorf("должно сохраниться новое объявление: %+v", announce)
	}
}

func TestHandleCallback_AnnounceEditFailsSendsNew(t *testing.T) {
	data := openGame(game.Limit)
	data.Games[0].Players[0] = storage.Player{UserID: maria.ID, Name: "Мария", Status: storage.StatusGoing}
	data.Games[0].Announce = &storage.Announce{MessageID: 77, Full: true, SentAt: tuesday().Add(-time.Minute).Unix()}

	sender := &mockSender{pinnedMessageID: testBoardID, requestErr: func(request tgbotapi.Chattable) error {
		if edit, ok := request.(tgbotapi.EditMessageTextConfig); ok && edit.MessageID == 77 {
			return &tgbotapi.Error{Code: 400, Message: "Bad Request: message to edit not found"}
		}
		return nil
	}}
	repository := &mockRepo{data: data}
	newService(sender, repository, tuesday).HandleCallback(press(maria, "decline:2026-11-17", testChatID))

	if len(sender.sent) != 1 || repository.data.Games[0].Announce.MessageID != 101 {
		t.Fatalf("если объявление удалили, отправляется новое: sent=%d announce=%+v", len(sender.sent), repository.data.Games[0].Announce)
	}
}

func TestDisplayName(t *testing.T) {
	cases := []struct {
		user *tgbotapi.User
		want string
	}{
		{&tgbotapi.User{FirstName: "Мария", LastName: "Иванова"}, "Мария Иванова"},
		{&tgbotapi.User{FirstName: "Мария"}, "Мария"},
		{&tgbotapi.User{UserName: "masha"}, "masha"},
	}
	for _, testCase := range cases {
		if got := displayName(testCase.user); got != testCase.want {
			t.Errorf("displayName(%+v) = %q, ожидалось %q", testCase.user, got, testCase.want)
		}
	}
}
