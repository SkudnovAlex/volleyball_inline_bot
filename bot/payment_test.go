package bot

import (
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/storage"
)

var ivan = &tgbotapi.User{ID: 7, FirstName: "Иван"}

// finishedGame возвращает данные с прошедшей игрой 09.11:
// Мария, Сергей с другом, Иван. В списке 4 строки.
func finishedGame() storage.Data {
	return storage.Data{
		BoardMessageID: testBoardID,
		Requisites:     map[int64]string{},
		Games: []storage.Game{{
			Date: "2026-11-09", MessageID: 50, Finished: true,
			Players: []storage.Player{
				{UserID: 1, Name: "Мария", Status: storage.StatusGoing},
				{UserID: 2, Name: "Сергей", Status: storage.StatusGoing, Friends: 1},
				{UserID: 7, Name: "Иван", Status: storage.StatusGoing},
			},
		}},
	}
}

// pressPrivate собирает нажатие кнопки в личке пользователя.
func pressPrivate(user *tgbotapi.User, data string) *tgbotapi.CallbackQuery {
	return &tgbotapi.CallbackQuery{
		ID: "query", From: user, Data: data,
		Message: &tgbotapi.Message{MessageID: 300, Chat: &tgbotapi.Chat{ID: user.ID, Type: "private"}},
	}
}

// write собирает текстовое сообщение пользователя в личке.
func write(user *tgbotapi.User, text string) *tgbotapi.Message {
	message := &tgbotapi.Message{From: user, Text: text, Chat: &tgbotapi.Chat{ID: user.ID, Type: "private"}}
	if strings.HasPrefix(text, "/") {
		command, _, _ := strings.Cut(text, " ")
		message.Entities = []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: len(command)}}
	}
	return message
}

// lastPrivate возвращает текст последнего сообщения, отправленного в личку.
func lastPrivate(t *testing.T, sender *mockSender) tgbotapi.MessageConfig {
	t.Helper()
	if len(sender.sent) == 0 {
		t.Fatal("в личку ничего не отправлено")
	}
	return sender.sent[len(sender.sent)-1].(tgbotapi.MessageConfig)
}

func draftOf(repository *mockRepo) *storage.Draft {
	return repository.data.Games[0].Payment.Draft
}

func TestStartPayment_OpensPrivateChat(t *testing.T) {
	data := finishedGame()
	data.Requisites[ivan.ID] = "Т-Банк +7 900 123-45-67"
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	newService(sender, repository, tuesday).HandleCallback(press(ivan, "hall:2026-11-09", testChatID))

	answers := callbacks(sender.requested)
	if len(answers) != 1 || answers[0].URL != "https://t.me/LiberoCoachBot?start=pay_2026-11-09" {
		t.Fatalf("ответ на нажатие должен открыть личку: %+v", answers)
	}
	draft := draftOf(repository)
	if draft == nil || draft.UserID != ivan.ID || draft.UserName != "Иван" || draft.Step != storage.StepCame ||
		draft.Requisites != "Т-Банк +7 900 123-45-67" || draft.StartedAt != tuesday().Unix() {
		t.Fatalf("черновик с сохранёнными реквизитами: %+v", draft)
	}
}

func TestStartPayment_BeforeGame(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: openGame(3)}
	newService(sender, repository, tuesday).HandleCallback(press(ivan, "hall:2026-11-17", testChatID))

	if answers := callbacks(sender.requested); len(answers) != 1 || !answers[0].ShowAlert || repository.saves != 0 {
		t.Fatalf("до игры оплату не оформить: %+v saves=%d", answers, repository.saves)
	}
}

func TestStartPayment_BusyByOther(t *testing.T) {
	data := finishedGame()
	data.Games[0].Payment = &storage.Payment{Draft: &storage.Draft{
		UserID: 1, UserName: "Мария", StartedAt: tuesday().Add(-10 * time.Minute).Unix(),
	}}
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	newService(sender, repository, tuesday).HandleCallback(press(ivan, "hall:2026-11-09", testChatID))

	answers := callbacks(sender.requested)
	if len(answers) != 1 || !answers[0].ShowAlert || answers[0].Text != "Оплату уже оформляет Мария" {
		t.Fatalf("ожидалось окно о чужом оформлении: %+v", answers)
	}
}

func TestStartPayment_TakesOverStaleDraft(t *testing.T) {
	data := finishedGame()
	data.Games[0].Payment = &storage.Payment{Draft: &storage.Draft{
		UserID: 1, UserName: "Мария", StartedAt: tuesday().Add(-2 * time.Hour).Unix(),
	}}
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	newService(sender, repository, tuesday).HandleCallback(press(ivan, "hall:2026-11-09", testChatID))

	if draft := draftOf(repository); draft.UserID != ivan.ID {
		t.Fatalf("черновик старше часа перехватывается: %+v", draft)
	}
}

func TestPayment_FullFlowWithAbsent(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: finishedGame()}
	service := newService(sender, repository, tuesday)

	service.HandleCallback(press(ivan, "hall:2026-11-09", testChatID))

	service.HandlePrivateMessage(write(ivan, "/start pay_2026-11-09"))
	if text := lastPrivate(t, sender).Text; !strings.Contains(text, "Сколько человек было на игре? В списке 4.") {
		t.Fatalf("первый вопрос: %q", text)
	}

	service.HandlePrivateMessage(write(ivan, "много"))
	if text := lastPrivate(t, sender).Text; text != "Пришли число от 1 до 40" {
		t.Fatalf("ошибка ввода числа: %q", text)
	}

	service.HandlePrivateMessage(write(ivan, "3"))
	if draft := draftOf(repository); draft.Came != 3 || draft.Step != storage.StepAbsent {
		t.Fatalf("после числа меньше списка шаг неявки: %+v", draft)
	}
	absentMessage := lastPrivate(t, sender)
	keyboard := absentMessage.ReplyMarkup.(*tgbotapi.InlineKeyboardMarkup)
	if len(keyboard.InlineKeyboard) != 3 || keyboard.InlineKeyboard[0][0].Text != "✅ Мария" ||
		*keyboard.InlineKeyboard[0][1].CallbackData != "absent:2026-11-09:2" ||
		*keyboard.InlineKeyboard[1][0].CallbackData != "absent:2026-11-09:2#1" {
		t.Fatalf("кнопки неявки по две в ряд и «Готово»: %+v", keyboard.InlineKeyboard)
	}

	service.HandlePrivateCallback(pressPrivate(ivan, "absent:2026-11-09:2"))
	if draft := draftOf(repository); len(draft.Absent) != 1 || draft.Absent[0] != "2" {
		t.Fatalf("Сергей отмечен как неявка: %+v", draft.Absent)
	}

	service.HandlePrivateCallback(pressPrivate(ivan, "absdone:2026-11-09"))
	if draft := draftOf(repository); draft.Step != storage.StepTotal {
		t.Fatalf("после «Готово» шаг суммы: %+v", draft)
	}
	if text := lastPrivate(t, sender).Text; !strings.Contains(text, textNeedDetails) {
		t.Fatalf("без сохранённых реквизитов просим строку целиком: %q", text)
	}

	service.HandlePrivateMessage(write(ivan, "5400"))
	if text := lastPrivate(t, sender).Text; text != textNeedDetails {
		t.Fatalf("одна сумма без реквизитов не принимается: %q", text)
	}

	service.HandlePrivateMessage(write(ivan, "Т-Банк +7 900 123-45-67 5400"))
	confirm := lastPrivate(t, sender)
	wantConfirm := "<b>Понедельник 09.11</b>\nБыло 3, зал 5400 ₽, по 1800 ₽ с человека\nПеревод на Т-Банк +7 900 123-45-67\nНе пришли: Сергей"
	if confirm.Text != wantConfirm || confirm.ReplyMarkup == nil {
		t.Fatalf("проверка перед отправкой:\n%s\nожидалось:\n%s", confirm.Text, wantConfirm)
	}

	sender.requested = nil
	service.HandlePrivateCallback(pressPrivate(ivan, "post:2026-11-09"))

	payment := repository.data.Games[0].Payment
	if payment.Draft != nil || payment.RenterID != ivan.ID || payment.RenterName != "Иван" || payment.Came != 3 ||
		payment.Total != 5400 || payment.PerPerson != 1800 || payment.Requisites != "Т-Банк +7 900 123-45-67" ||
		len(payment.Absent) != 1 {
		t.Fatalf("оплата записана, черновик убран: %+v", payment)
	}
	if repository.data.Requisites[ivan.ID] != "Т-Банк +7 900 123-45-67" {
		t.Errorf("реквизиты запоминаются: %+v", repository.data.Requisites)
	}
	var gameEdit *tgbotapi.EditMessageTextConfig
	for _, edit := range edits(sender.requested) {
		if edit.MessageID == 50 {
			gameEdit = &edit
		}
	}
	if gameEdit == nil || !strings.Contains(gameEdit.Text, "2. Сергей · не пришёл") || !strings.Contains(gameEdit.Text, "Было человек: 3") {
		t.Fatalf("сообщение игры показывает неявку: %+v", gameEdit)
	}
}

func TestPayment_CameEqualsListSkipsAbsent(t *testing.T) {
	data := finishedGame()
	data.Requisites[ivan.ID] = "Сбер 8900"
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	service := newService(sender, repository, tuesday)
	service.HandleCallback(press(ivan, "hall:2026-11-09", testChatID))

	service.HandlePrivateMessage(write(ivan, "4"))
	if draft := draftOf(repository); draft.Step != storage.StepTotal {
		t.Fatalf("все пришли: шаг неявки пропускается: %+v", draft)
	}
	if text := lastPrivate(t, sender).Text; !strings.Contains(text, "Перевод на <b>Сбер 8900</b>?") {
		t.Fatalf("сохранённые реквизиты предлагаются: %q", text)
	}

	service.HandlePrivateMessage(write(ivan, "4000"))
	if draft := draftOf(repository); draft.Step != storage.StepConfirm || draft.Requisites != "Сбер 8900" || draft.Total != 4000 {
		t.Fatalf("сумма без реквизитов берёт сохранённые: %+v", draft)
	}
}

func TestPayment_NewcomersNote(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: finishedGame()}
	service := newService(sender, repository, tuesday)
	service.HandleCallback(press(ivan, "hall:2026-11-09", testChatID))

	service.HandlePrivateMessage(write(ivan, "6"))
	if text := lastPrivate(t, sender).Text; !strings.HasPrefix(text, "В списке 4, было 6: новеньких 2, они переведут сами.") {
		t.Fatalf("ожидалась заметка про новеньких: %q", text)
	}
}

func TestPayment_MismatchNote(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: finishedGame()}
	service := newService(sender, repository, tuesday)
	service.HandleCallback(press(ivan, "hall:2026-11-09", testChatID))
	service.HandlePrivateMessage(write(ivan, "3"))

	service.HandlePrivateCallback(pressPrivate(ivan, "absdone:2026-11-09"))

	if text := lastPrivate(t, sender).Text; !strings.HasPrefix(text, "По отметкам выходит 4, а ты указал 3. Делю на 3") {
		t.Fatalf("ожидалось предупреждение о расхождении: %q", text)
	}
}

func TestPayment_Redo(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: finishedGame()}
	service := newService(sender, repository, tuesday)
	service.HandleCallback(press(ivan, "hall:2026-11-09", testChatID))
	service.HandlePrivateMessage(write(ivan, "4"))
	service.HandlePrivateMessage(write(ivan, "Сбер 8900 4000"))

	service.HandlePrivateCallback(pressPrivate(ivan, "redo:2026-11-09"))

	draft := draftOf(repository)
	if draft.Step != storage.StepCame || draft.Came != 0 || draft.Total != 0 || draft.Requisites != "Сбер 8900" {
		t.Fatalf("«Заново» сбрасывает ответы, реквизиты остаются: %+v", draft)
	}
}

func TestPayment_StrangerCannotPressDraftButtons(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: finishedGame()}
	service := newService(sender, repository, tuesday)
	service.HandleCallback(press(ivan, "hall:2026-11-09", testChatID))
	service.HandlePrivateMessage(write(ivan, "4"))
	service.HandlePrivateMessage(write(ivan, "Сбер 8900 4000"))
	sender.requested = nil

	service.HandlePrivateCallback(pressPrivate(maria, "post:2026-11-09"))

	if answers := callbacks(sender.requested); len(answers) != 1 || answers[0].Text != textDraftExpired {
		t.Fatalf("чужой черновик не отправить: %+v", answers)
	}
	if repository.data.Games[0].Payment.RenterID != 0 {
		t.Fatalf("оплата не должна записаться")
	}
}

func TestPrivateMessage_WithoutDraft(t *testing.T) {
	sender := &mockSender{}
	repository := &mockRepo{data: finishedGame()}
	newService(sender, repository, tuesday).HandlePrivateMessage(write(ivan, "/start"))

	if text := lastPrivate(t, sender).Text; text != textNoDraft {
		t.Fatalf("без черновика бот объясняет, с чего начать: %q", text)
	}
}

func TestStartPayment_DropsDraftInOtherGame(t *testing.T) {
	data := finishedGame()
	other := data.Games[0]
	other.Date, other.MessageID = "2026-11-05", 40
	other.Payment = &storage.Payment{Draft: &storage.Draft{UserID: ivan.ID, Step: storage.StepTotal}}
	data.Games = append(data.Games, other)

	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	newService(sender, repository, tuesday).HandleCallback(press(ivan, "hall:2026-11-09", testChatID))

	if repository.data.Games[1].Payment.Draft != nil {
		t.Fatalf("у человека одно оформление, старое сбрасывается")
	}
}

func TestHandleUpdate_RoutesPrivate(t *testing.T) {
	sender := &mockSender{}
	repository := &mockRepo{data: finishedGame()}
	newService(sender, repository, tuesday).HandleUpdate(tgbotapi.Update{Message: write(ivan, "привет")})

	if text := lastPrivate(t, sender).Text; text != textNoDraft {
		t.Fatalf("сообщение в личке уходит в сценарий оплаты: %q", text)
	}
}

// confirmedDraft возвращает прошедшую игру, где Иван дошёл до проверки:
// было 4, 4000 ₽, реквизиты «Сбер 8900».
func confirmedDraft() storage.Data {
	data := finishedGame()
	data.Games[0].Payment = &storage.Payment{Draft: &storage.Draft{
		UserID: ivan.ID, UserName: "Иван", Step: storage.StepConfirm,
		Came: 4, Requisites: "Сбер 8900", Total: 4000, StartedAt: tuesday().Unix(),
	}}
	return data
}

func TestPost_SendsPaymentMessage(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: confirmedDraft()}
	newService(sender, repository, tuesday).HandlePrivateCallback(pressPrivate(ivan, "post:2026-11-09"))

	var paymentMessage *tgbotapi.MessageConfig
	for _, request := range sender.sent {
		if message, ok := request.(tgbotapi.MessageConfig); ok && message.ChatID == testChatID {
			paymentMessage = &message
		}
	}
	if paymentMessage == nil || paymentMessage.ReplyToMessageID != 50 || paymentMessage.ReplyMarkup == nil ||
		!strings.Contains(paymentMessage.Text, `<a href="tg://user?id=1">Мария</a> 1000 ₽`) {
		t.Fatalf("сообщение оплаты ответом на игру с упоминаниями: %+v", paymentMessage)
	}
	if id := repository.data.Games[0].Payment.MessageID; id == 0 {
		t.Fatalf("номер сообщения оплаты сохраняется")
	}
	if text := lastPrivate(t, sender).Text; text != "Готово, сообщение в группе." {
		t.Errorf("ответ в личке: %q", text)
	}
}

func TestPost_ReplacesPreviousPaymentMessage(t *testing.T) {
	data := confirmedDraft()
	data.Games[0].Payment.MessageID = 400
	data.Games[0].Payment.PaidUserIDs = []int64{1}
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	newService(sender, repository, tuesday).HandlePrivateCallback(pressPrivate(ivan, "post:2026-11-09"))

	if deleted := deletedIDs(sender.requested); len(deleted) != 1 || deleted[0] != 400 {
		t.Fatalf("прошлое сообщение оплаты удаляется: %v", deleted)
	}
	payment := repository.data.Games[0].Payment
	if payment.MessageID == 400 || payment.MessageID == 0 || len(payment.PaidUserIDs) != 0 {
		t.Fatalf("новое сообщение, отметки сброшены: %+v", payment)
	}
}

func TestPaid_MarksAndEdits(t *testing.T) {
	data := finishedGame()
	data.Games[0].Payment = &storage.Payment{
		RenterID: ivan.ID, RenterName: "Иван", Requisites: "Сбер 8900",
		Came: 4, Total: 4000, PerPerson: 1000, MessageID: 400,
	}
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	service := newService(sender, repository, tuesday)
	sergey := &tgbotapi.User{ID: 2, FirstName: "Сергей"}

	service.HandleCallback(press(sergey, "paid:2026-11-09", testChatID))

	if paid := repository.data.Games[0].Payment.PaidUserIDs; len(paid) != 1 || paid[0] != 2 {
		t.Fatalf("Сергей отмечен: %v", paid)
	}
	answers := callbacks(sender.requested)
	if len(answers) != 1 || answers[0].ShowAlert || answers[0].Text != "Отмечено: ты перевёл 2000 ₽" {
		t.Fatalf("короткое уведомление с суммой за двоих: %+v", answers)
	}
	var paymentEdit *tgbotapi.EditMessageTextConfig
	for _, edit := range edits(sender.requested) {
		if edit.MessageID == 400 {
			paymentEdit = &edit
		}
	}
	if paymentEdit == nil || !strings.Contains(paymentEdit.Text, "перевели 1 из 2") {
		t.Fatalf("сообщение оплаты перерисовано: %+v", paymentEdit)
	}

	sender.requested = nil
	service.HandleCallback(press(sergey, "paid:2026-11-09", testChatID))
	if answers := callbacks(sender.requested); len(answers) != 1 || !answers[0].ShowAlert ||
		answers[0].Text != "Ты уже отметил перевод 2000 ₽" {
		t.Fatalf("повторное нажатие отметку не снимает: %+v", answers)
	}
	if paid := repository.data.Games[0].Payment.PaidUserIDs; len(paid) != 1 {
		t.Fatalf("отметка Сергея остаётся: %v", paid)
	}

	sender.requested = nil
	service.HandleCallback(press(ivan, "paid:2026-11-09", testChatID))
	if answers := callbacks(sender.requested); len(answers) != 1 || !answers[0].ShowAlert {
		t.Fatalf("плательщику окно «Ты платил за зал»: %+v", answers)
	}
}
