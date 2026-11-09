package bot

import (
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/game"
	"volleyball_inline_bot/storage"
)

// Действия кнопок в личке. callback_data: "<действие>:<дата>[:<строка>]".
const (
	actionAbsent     = "absent"
	actionAbsentDone = "absdone"
	actionPost       = "post"
	actionRedo       = "redo"
)

// startPrefix это параметр ссылки t.me/<бот>?start=pay_<дата>.
const startPrefix = "pay_"

const (
	textNoDraft      = "Чтобы оформить оплату зала, нажми «💳 Я оплатил зал» в сообщении прошедшей игры в группе."
	textDraftExpired = "Оформление устарело. Нажми «Я оплатил зал» в группе ещё раз"
	textPressButton  = "Нажми кнопку под сообщением выше."
	textNeedDetails  = "Напиши одной строкой реквизиты и сумму за зал, например: Т-Банк +7 900 123-45-67 5400"
)

// startPayment начинает оформление оплаты: создаёт черновик и открывает нажавшему личку с ботом. Вызывается под mutex.
func (s *Service) startPayment(query *tgbotapi.CallbackQuery, data *storage.Data, match *storage.Game) {
	if !match.Finished {
		s.alert(query.ID, "Оплату можно оформить после игры")
		return
	}
	now := s.now()
	userID := query.From.ID
	if match.Payment != nil && match.Payment.Draft != nil {
		draft := match.Payment.Draft
		if draft.UserID != userID && now.Sub(time.Unix(draft.StartedAt, 0)) < s.draftLock {
			s.alert(query.ID, fmt.Sprintf("Оплату уже оформляет %s", draft.UserName))
			return
		}
	}

	// У человека одно оформление: черновики по другим играм сбрасываются.
	for index := range data.Games {
		other := &data.Games[index]
		if other.Payment != nil && other.Payment.Draft != nil && other.Payment.Draft.UserID == userID {
			other.Payment.Draft = nil
		}
	}
	if match.Payment == nil {
		match.Payment = &storage.Payment{}
	}
	match.Payment.Draft = &storage.Draft{
		UserID:     userID,
		UserName:   displayName(query.From),
		Step:       storage.StepCame,
		Requisites: data.Requisites[userID],
		StartedAt:  now.Unix(),
	}
	if err := s.repository.Save(*data); err != nil {
		log.Printf("Не удалось сохранить черновик оплаты %s: %v", match.Date, err)
		s.alert(query.ID, "Не получилось, попробуй ещё раз")
		return
	}

	// Telegram открывает ссылку t.me/<бот>?start=... прямо из ответа на нажатие.
	answer := tgbotapi.NewCallback(query.ID, "")
	answer.URL = fmt.Sprintf("https://t.me/%s?start=%s%s", s.botUserName, startPrefix, match.Date)
	if _, err := s.sender.Request(answer); err != nil {
		log.Printf("Не удалось открыть личку для оплаты %s: %v", match.Date, err)
	}
}

// HandlePrivateMessage обрабатывает текст в личке: /start из ссылки и ответы на вопросы оформления оплаты.
func (s *Service) HandlePrivateMessage(message *tgbotapi.Message) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := s.repository.Load()
	if err != nil {
		log.Printf("Ошибка чтения хранилища: %v", err)
		return
	}
	userID := message.From.ID
	match := findDraftGame(data.Games, userID)
	if match == nil {
		s.sendPrivate(userID, textNoDraft, nil)
		return
	}
	draft := match.Payment.Draft

	if message.IsCommand() {
		// /start pay_<дата> и любая другая команда повторяют текущий вопрос.
		s.sendStep(userID, match, "")
		return
	}

	switch draft.Step {
	case storage.StepCame:
		came, err := game.ParseCame(message.Text)
		if err != nil {
			s.sendPrivate(userID, err.Error(), nil)
			return
		}
		draft.Came = came
		draft.Absent = nil
		lines := len(game.Lines(match))
		note := ""
		if came < lines {
			draft.Step = storage.StepAbsent
		} else {
			draft.Step = storage.StepTotal
			if came > lines {
				note = fmt.Sprintf("В списке %d, было %d: новеньких %d, они переведут сами.", lines, came, came-lines)
			}
		}
		s.saveAndSendStep(&data, userID, match, note)

	case storage.StepTotal:
		requisites, total, err := game.ParseAmount(message.Text)
		if err != nil {
			s.sendPrivate(userID, err.Error(), nil)
			return
		}
		if requisites == "" {
			requisites = draft.Requisites
		}
		if requisites == "" {
			s.sendPrivate(userID, textNeedDetails, nil)
			return
		}
		draft.Requisites = requisites
		draft.Total = total
		draft.Step = storage.StepConfirm
		s.saveAndSendStep(&data, userID, match, "")

	default:
		s.sendPrivate(userID, textPressButton, nil)
	}
}

// HandlePrivateCallback обрабатывает кнопки в личке: отметки неявки, "Готово", "Отправить в группу" и "Заново".
func (s *Service) HandlePrivateCallback(query *tgbotapi.CallbackQuery) {
	parts := strings.SplitN(query.Data, ":", 3)
	if len(parts) < 2 {
		s.answer(query.ID, "")
		return
	}
	action, date := parts[0], parts[1]

	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := s.repository.Load()
	if err != nil {
		log.Printf("Ошибка чтения хранилища: %v", err)
		s.alert(query.ID, "Не получилось, попробуй ещё раз")
		return
	}
	userID := query.From.ID
	match := findGame(data.Games, date)
	if match == nil || match.Payment == nil || match.Payment.Draft == nil || match.Payment.Draft.UserID != userID {
		s.alert(query.ID, textDraftExpired)
		return
	}
	draft := match.Payment.Draft
	chatID, messageID := query.Message.Chat.ID, query.Message.MessageID

	switch {
	case action == actionAbsent && draft.Step == storage.StepAbsent && len(parts) == 3:
		draft.Absent = game.ToggleAbsent(draft.Absent, parts[2])
		if !s.save(&data, query.ID) {
			return
		}
		keyboard := absentKeyboard(match)
		if _, err := s.sender.Request(tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, keyboard)); err != nil && !isNotModified(err) {
			log.Printf("Не удалось обновить отметки неявки: %v", err)
		}

	case action == actionAbsentDone && draft.Step == storage.StepAbsent:
		draft.Step = storage.StepTotal
		if !s.save(&data, query.ID) {
			return
		}
		s.closePrompt(chatID, messageID, absentSummary(match, draft.Absent))
		s.sendStep(userID, match, mismatchNote(match, draft))

	case action == actionRedo && draft.Step == storage.StepConfirm:
		draft.Step = storage.StepCame
		draft.Came, draft.Total, draft.Absent = 0, 0, nil
		if !s.save(&data, query.ID) {
			return
		}
		s.closePrompt(chatID, messageID, "Начинаем заново.")
		s.sendStep(userID, match, "")

	case action == actionPost && draft.Step == storage.StepConfirm:
		publish(&data, match)
		if !s.save(&data, query.ID) {
			return
		}
		s.closePrompt(chatID, messageID, confirmText(match, match.Payment.Requisites, match.Payment.Came,
			match.Payment.Total, match.Payment.Absent))
		s.editGame(match)
		s.sendPaymentMessage(&data, match)
		s.sendPrivate(userID, "Готово, сообщение в группе.", nil)
		s.updateBoardAndSave(&data)
	}
	s.answer(query.ID, "")
}

// publish переносит черновик в оплату и запоминает реквизиты плательщика.
func publish(data *storage.Data, match *storage.Game) {
	payment := match.Payment
	draft := payment.Draft
	payment.RenterID = draft.UserID
	payment.RenterName = draft.UserName
	payment.Requisites = draft.Requisites
	payment.Came = draft.Came
	payment.Absent = draft.Absent
	payment.Total = draft.Total
	payment.PerPerson = game.PerPerson(draft.Total, draft.Came)
	payment.PaidUserIDs = nil
	payment.Draft = nil
	data.Requisites[draft.UserID] = draft.Requisites
}

// save сохраняет данные, при ошибке показывает нажавшему окно.
func (s *Service) save(data *storage.Data, queryID string) bool {
	if err := s.repository.Save(*data); err != nil {
		log.Printf("Не удалось сохранить оформление оплаты: %v", err)
		s.alert(queryID, "Не получилось, попробуй ещё раз")
		return false
	}
	return true
}

// saveAndSendStep сохраняет данные и задаёт вопрос текущего шага.
func (s *Service) saveAndSendStep(data *storage.Data, userID int64, match *storage.Game, note string) {
	if err := s.repository.Save(*data); err != nil {
		log.Printf("Не удалось сохранить оформление оплаты: %v", err)
		s.sendPrivate(userID, "Не получилось сохранить, пришли ещё раз.", nil)
		return
	}
	s.sendStep(userID, match, note)
}

// sendStep отправляет вопрос текущего шага. note добавляется перед вопросом.
func (s *Service) sendStep(userID int64, match *storage.Game, note string) {
	draft := match.Payment.Draft
	day := game.DayTitle(match.Date)
	var text string
	var keyboard *tgbotapi.InlineKeyboardMarkup

	switch draft.Step {
	case storage.StepCame:
		text = fmt.Sprintf("<b>%s</b>. Сколько человек было на игре? В списке %d.", day, len(game.Lines(match)))
	case storage.StepAbsent:
		text = fmt.Sprintf("<b>%s</b>. В списке %d, было %d. Отметь, кого не было, и нажми «Готово».",
			day, len(game.Lines(match)), draft.Came)
		absent := absentKeyboard(match)
		keyboard = &absent
	case storage.StepTotal:
		if draft.Requisites != "" {
			text = fmt.Sprintf("Перевод на <b>%s</b>? Пришли сумму за зал или новую строку: реквизиты и сумма.",
				html.EscapeString(draft.Requisites))
		} else {
			text = textNeedDetails
		}
	case storage.StepConfirm:
		text = confirmText(match, draft.Requisites, draft.Came, draft.Total, draft.Absent)
		keyboard = confirmKeyboard(match.Date)
	}
	if note != "" {
		text = note + "\n\n" + text
	}
	s.sendPrivate(userID, text, keyboard)
}

// sendPrivate отправляет сообщение в личку. Чат лички совпадает с ID пользователя.
func (s *Service) sendPrivate(userID int64, text string, keyboard *tgbotapi.InlineKeyboardMarkup) {
	message := tgbotapi.NewMessage(userID, text)
	message.ParseMode = tgbotapi.ModeHTML
	if keyboard != nil {
		message.ReplyMarkup = keyboard
	}
	if _, err := s.sender.Send(message); err != nil {
		log.Printf("Не удалось написать в личку %d: %v", userID, err)
	}
}

// closePrompt заменяет текст сообщения с кнопками и убирает кнопки, чтобы по старому вопросу нельзя было нажать ещё раз.
func (s *Service) closePrompt(chatID int64, messageID int, text string) {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	if _, err := s.sender.Request(edit); err != nil && !isNotModified(err) {
		log.Printf("Не удалось закрыть вопрос в личке: %v", err)
	}
}

// absentKeyboard строит кнопки отметки неявки по две в ряд и "Готово".
func absentKeyboard(match *storage.Game) tgbotapi.InlineKeyboardMarkup {
	draft := match.Payment.Draft
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for _, line := range game.Lines(match) {
		mark := "✅ "
		if game.IsAbsent(draft.Absent, line.ID) {
			mark = "❌ "
		}
		data := fmt.Sprintf("%s:%s:%s", actionAbsent, match.Date, line.ID)
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(mark+html.UnescapeString(line.Label), data))
		if len(row) == 2 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	done := fmt.Sprintf("Готово · не было: %d", len(draft.Absent))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(done, actionAbsentDone+":"+match.Date),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func confirmKeyboard(date string) *tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✅ Отправить в группу", actionPost+":"+date),
		tgbotapi.NewInlineKeyboardButtonData("🔄 Заново", actionRedo+":"+date),
	))
	return &keyboard
}

// confirmText описывает оплату перед отправкой и после неё.
func confirmText(match *storage.Game, requisites string, came, total int, absent []string) string {
	text := fmt.Sprintf("<b>%s</b>\nБыло %d, зал %d ₽, по %d ₽ с человека\nПеревод на %s",
		game.DayTitle(match.Date), came, total, game.PerPerson(total, came), html.EscapeString(requisites))
	if names := absentNames(match, absent); names != "" {
		text += "\nНе пришли: " + names
	}
	return text
}

// absentSummary заменяет вопрос про неявку после нажатия "Готово".
func absentSummary(match *storage.Game, absent []string) string {
	names := absentNames(match, absent)
	if names == "" {
		return "Неявок нет."
	}
	return "Не пришли: " + names
}

func absentNames(match *storage.Game, absent []string) string {
	var names []string
	for _, line := range game.Lines(match) {
		if game.IsAbsent(absent, line.ID) {
			names = append(names, line.Label)
		}
	}
	return strings.Join(names, ", ")
}

// mismatchNote предупреждает, если отметки неявки не сходятся с числом пришедших.
func mismatchNote(match *storage.Game, draft *storage.Draft) string {
	present := len(game.Lines(match)) - len(draft.Absent)
	if present == draft.Came {
		return ""
	}
	return fmt.Sprintf("По отметкам выходит %d, а ты указал %d. Делю на %d: разница это новенькие или ошибка в списке.",
		present, draft.Came, draft.Came)
}

// findDraftGame возвращает игру, оплату которой сейчас оформляет пользователь.
func findDraftGame(games []storage.Game, userID int64) *storage.Game {
	for index := range games {
		payment := games[index].Payment
		if payment != nil && payment.Draft != nil && payment.Draft.UserID == userID {
			return &games[index]
		}
	}
	return nil
}
