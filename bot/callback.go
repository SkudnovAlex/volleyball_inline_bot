package bot

import (
	"errors"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/game"
	"volleyball_inline_bot/storage"
)

// textClosed показывается при нажатии на прошедшую или неизвестную игру.
const textClosed = "Запись на эту игру закрыта"

// HandleUpdate разбирает апдейт: нажатия в группе, нажатия и сообщения
// в личке с ботом. Остальное пропускается.
func (s *Service) HandleUpdate(update tgbotapi.Update) {
	switch {
	case update.CallbackQuery != nil && isPrivate(update.CallbackQuery.Message):
		s.HandlePrivateCallback(update.CallbackQuery)
	case update.CallbackQuery != nil:
		s.HandleCallback(update.CallbackQuery)
	case update.Message != nil && isPrivate(update.Message) && update.Message.From != nil:
		s.HandlePrivateMessage(update.Message)
	}
}

// isPrivate сообщает, что сообщение из лички с ботом.
func isPrivate(message *tgbotapi.Message) bool {
	return message != nil && message.Chat != nil && message.Chat.IsPrivate()
}

// HandleCallback обрабатывает нажатие кнопки записи в сообщении игры.
func (s *Service) HandleCallback(query *tgbotapi.CallbackQuery) {
	if query.Message == nil || query.Message.Chat == nil || query.Message.Chat.ID != s.chatID {
		s.answer(query.ID, "")
		return
	}
	action, date, found := strings.Cut(query.Data, ":")
	if !found {
		s.answer(query.ID, "")
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := s.repository.Load()
	if err != nil {
		log.Printf("Ошибка чтения хранилища: %v", err)
		s.alert(query.ID, "Не получилось, попробуй ещё раз")
		return
	}
	match := findGame(data.Games, date)
	if match != nil && action == game.ActionHall {
		s.startPayment(query, &data, match)
		return
	}
	if match == nil || match.Finished {
		s.alert(query.ID, textClosed)
		return
	}

	user := game.User{ID: query.From.ID, Name: displayName(query.From)}
	before := game.Count(match)
	if err := applyAction(match, action, user); err != nil {
		s.alert(query.ID, err.Error())
		return
	}
	event := game.Transition(before, game.Count(match))
	if event != game.EventNone {
		s.announce(match, event)
	}

	if err := s.repository.Save(data); err != nil {
		log.Printf("Ошибка сохранения после нажатия %q на %s: %v", action, date, err)
		s.alert(query.ID, "Не получилось, попробуй ещё раз")
		return
	}
	s.editGame(match)
	s.updateBoardAndSave(&data)
	s.answer(query.ID, "")
}

// applyAction применяет действие кнопки к игре.
func applyAction(match *storage.Game, action string, user game.User) error {
	switch action {
	case game.ActionJoin:
		return game.Join(match, user)
	case game.ActionFriend:
		return game.AddFriend(match, user)
	case game.ActionDecline:
		return game.Decline(match, user)
	case game.ActionMinus:
		return game.RemoveFriend(match, user)
	default:
		return errors.New(textClosed)
	}
}

// editGame перерисовывает сообщение игры по текущим данным.
func (s *Service) editGame(match *storage.Game) {
	text, keyboard := game.RenderGame(match)
	edit := tgbotapi.NewEditMessageText(s.chatID, match.MessageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	edit.ReplyMarkup = keyboard
	if _, err := s.sender.Request(edit); err != nil && !isNotModified(err) {
		log.Printf("Не удалось обновить сообщение игры %s: %v", match.Date, err)
	}
}

// answer закрывает «часики» на кнопке, text показывается коротким уведомлением.
func (s *Service) answer(queryID, text string) {
	if _, err := s.sender.Request(tgbotapi.NewCallback(queryID, text)); err != nil {
		log.Printf("Не удалось ответить на нажатие: %v", err)
	}
}

// alert показывает нажавшему всплывающее окно с текстом.
func (s *Service) alert(queryID, text string) {
	if _, err := s.sender.Request(tgbotapi.NewCallbackWithAlert(queryID, text)); err != nil {
		log.Printf("Не удалось показать окно на нажатие: %v", err)
	}
}

// displayName собирает имя пользователя для списка.
func displayName(user *tgbotapi.User) string {
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		return user.UserName
	}
	return name
}

// isNotModified сообщает, что Telegram отказался редактировать сообщение,
// потому что текст и кнопки не изменились.
func isNotModified(err error) bool {
	var telegramError *tgbotapi.Error
	return errors.As(err, &telegramError) && strings.Contains(telegramError.Message, "message is not modified")
}
