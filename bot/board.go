package bot

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/game"
	"volleyball_inline_bot/storage"
)

// RefreshBoard отправляет сообщения игр, у которых их ещё нет, обновляет доску и закрепляет её, если её открепили.
// Вызывается при старте бота и по расписанию BOARD_CHECK_SCHEDULE.
func (s *Service) RefreshBoard() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := s.repository.Load()
	if err != nil {
		log.Printf("Ошибка чтения хранилища: %v", err)
		return
	}
	if s.sendMissingGames(&data) {
		if err := s.repository.Save(data); err != nil {
			log.Printf("Сообщения игр отправлены, но не сохранены: %v", err)
			return
		}
	}
	previousID := data.BoardMessageID
	s.updateBoardAndSave(&data)
	// Только что отправленная доска самая новая и уже закреплена.
	if data.BoardMessageID == previousID {
		s.raiseBoard(&data, false)
	}
}

// RaiseBoard поднимает доску наверх закрепа: если закреплено сообщение новее доски, отправляет доску заново.
// Вызывается по расписанию BOARD_RAISE_SCHEDULE.
func (s *Service) RaiseBoard() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := s.repository.Load()
	if err != nil {
		log.Printf("Ошибка чтения хранилища: %v", err)
		return
	}
	if s.raiseBoard(&data, true) {
		if err := s.repository.Save(data); err != nil {
			log.Printf("Новая доска %d отправлена, но не сохранена: %v", data.BoardMessageID, err)
		}
	}
}

// RedrawGames перерисовывает сообщения всех игр и оплат по текущим данным. Вызывается при старте, чтобы после обновления
// бота сообщения в чате получили новое оформление. Если сообщение удалили из чата, бот забывает его номер, и
// RefreshBoard отправляет его заново.
func (s *Service) RedrawGames() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := s.repository.Load()
	if err != nil {
		log.Printf("Ошибка чтения хранилища: %v", err)
		return
	}
	changed := false
	for index := range data.Games {
		match := &data.Games[index]
		if match.MessageID == 0 {
			continue
		}
		if !s.editGame(match) {
			log.Printf("Сообщение игры %s удалено из чата, отправлю заново", match.Date)
			match.MessageID = 0
			// Сообщение оплаты отвечает на сообщение игры, поэтому переотправляется вместе с ней. Старое, если осталось,
			// удаляем.
			if match.Payment != nil && match.Payment.MessageID != 0 {
				s.sender.Request(tgbotapi.NewDeleteMessage(s.chatID, match.Payment.MessageID))
				match.Payment.MessageID = 0
			}
			changed = true
			continue
		}
		if !s.editPaymentMessage(match) {
			log.Printf("Сообщение оплаты %s удалено из чата, отправлю заново", match.Date)
			match.Payment.MessageID = 0
			changed = true
		}
	}
	if !changed {
		return
	}
	if err := s.repository.Save(data); err != nil {
		log.Printf("Не удалось сохранить данные после перерисовки: %v", err)
	}
}

// sendMissingGames отправляет сообщения для игр с message_id 0 и сообщения оплаты для оплаченных игр без него: так игру
// можно добавить или восстановить, вписав её в файл данных вручную. Возвращает true, если что-то отправлено.
func (s *Service) sendMissingGames(data *storage.Data) bool {
	sent := false
	for index := range data.Games {
		match := &data.Games[index]
		if match.MessageID != 0 {
			continue
		}
		text, keyboard := game.RenderGame(match)
		message := tgbotapi.NewMessage(s.chatID, text)
		message.ParseMode = tgbotapi.ModeHTML
		message.ReplyMarkup = keyboard
		sentMessage, err := s.sender.Send(message)
		if err != nil {
			log.Printf("Не удалось отправить сообщение игры %s: %v", match.Date, err)
			continue
		}
		match.MessageID = sentMessage.MessageID
		sent = true
		log.Printf("Отправлено сообщение игры %s: %d", match.Date, sentMessage.MessageID)
	}

	// Оплаченной игре без сообщения оплаты отправляем и его, ответом на игру.
	for index := range data.Games {
		match := &data.Games[index]
		if !game.IsPaid(match) || match.Payment.MessageID != 0 || match.MessageID == 0 {
			continue
		}
		text, keyboard := game.RenderPayment(match)
		message := tgbotapi.NewMessage(s.chatID, text)
		message.ParseMode = tgbotapi.ModeHTML
		message.ReplyToMessageID = match.MessageID
		message.ReplyMarkup = keyboard
		sentMessage, err := s.sender.Send(message)
		if err != nil {
			log.Printf("Не удалось отправить сообщение оплаты %s: %v", match.Date, err)
			continue
		}
		match.Payment.MessageID = sentMessage.MessageID
		sent = true
	}
	return sent
}

// updateBoardAndSave обновляет доску по уже сохранённым данным и, если пришлось отправить новую доску, сохраняет её ID.
func (s *Service) updateBoardAndSave(data *storage.Data) {
	previousID := data.BoardMessageID
	s.updateBoard(data)
	if data.BoardMessageID == previousID {
		return
	}
	if err := s.repository.Save(*data); err != nil {
		log.Printf("Новая доска %d отправлена, но не сохранена: %v", data.BoardMessageID, err)
	}
}

// updateBoard редактирует доску или отправляет новую, если её ещё нет или её удалили. Новый ID записывается в data.
func (s *Service) updateBoard(data *storage.Data) {
	text := game.RenderBoard(data.Games, s.chatID)

	if data.BoardMessageID != 0 {
		edit := tgbotapi.NewEditMessageText(s.chatID, data.BoardMessageID, text)
		edit.ParseMode = tgbotapi.ModeHTML
		edit.DisableWebPagePreview = true
		_, err := s.sender.Request(edit)
		switch {
		case err == nil || isNotModified(err):
			return
		case isMessageNotFound(err):
			log.Printf("Доска %d удалена из чата, отправляю новую", data.BoardMessageID)
		default:
			log.Printf("Не удалось обновить доску %d: %v", data.BoardMessageID, err)
			return
		}
	}
	s.sendBoard(data, text)
}

// sendBoard отправляет доску без звука и закрепляет её. Новый ID записывается в data.
func (s *Service) sendBoard(data *storage.Data, text string) {
	message := tgbotapi.NewMessage(s.chatID, text)
	message.ParseMode = tgbotapi.ModeHTML
	message.DisableWebPagePreview = true
	message.DisableNotification = true
	sentMessage, err := s.sender.Send(message)
	if err != nil {
		log.Printf("Не удалось отправить доску: %v", err)
		return
	}
	data.BoardMessageID = sentMessage.MessageID
	log.Printf("Отправлена новая доска %d", sentMessage.MessageID)
	s.pin(sentMessage.MessageID)
}

// raiseBoard делает доску последним закрепом, чтобы она показывалась в плашке сверху. Telegram показывает в плашке
// закреп с самым новым сообщением, а не последний закрепленный, поэтому если закреплено сообщение новее доски, доска
// отправляется заново, но только при resend: иначе новое сообщение в чате появлялось бы на каждую проверку. Если
// последний закреп старше доски или закрепов нет, доску открепили, и она закрепляется снова. Если закреп новее доски,
// узнать, закреплена ли доска, нельзя: это выяснится при переотправке. Возвращает true, если отправлена новая доска.
func (s *Service) raiseBoard(data *storage.Data, resend bool) bool {
	if data.BoardMessageID == 0 {
		return false
	}
	response, err := s.sender.Request(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: s.chatID}})
	if err != nil {
		log.Printf("Не удалось получить закреп чата: %v", err)
		return false
	}
	var chat tgbotapi.Chat
	if err := json.Unmarshal(response.Result, &chat); err != nil {
		log.Printf("Не удалось разобрать ответ getChat: %v", err)
		return false
	}

	oldID := data.BoardMessageID
	switch pinned := chat.PinnedMessage; {
	case pinned != nil && pinned.MessageID == oldID:
		return false
	case pinned == nil || pinned.MessageID < oldID:
		log.Printf("Доска %d не в закрепе, закрепляю снова", oldID)
		s.pin(oldID)
		return false
	case !resend:
		return false
	default:
		log.Printf("Закреплено сообщение %d новее доски %d, отправляю доску заново", pinned.MessageID, oldID)
	}

	unpin := tgbotapi.UnpinChatMessageConfig{ChatID: s.chatID, MessageID: oldID}
	if _, err := s.sender.Request(unpin); err != nil {
		log.Printf("Не удалось открепить доску %d: %v", oldID, err)
	}
	s.sendBoard(data, game.RenderBoard(data.Games, s.chatID))
	if data.BoardMessageID == oldID {
		// Новая доска не отправилась: возвращаем в закреп старую.
		s.pin(oldID)
		return false
	}
	if _, err := s.sender.Request(tgbotapi.NewDeleteMessage(s.chatID, oldID)); err != nil {
		log.Printf("Не удалось удалить старую доску %d, удалите её вручную: %v", oldID, err)
	}
	return true
}

// pin закрепляет сообщение без уведомления участникам.
func (s *Service) pin(messageID int) {
	pin := tgbotapi.PinChatMessageConfig{ChatID: s.chatID, MessageID: messageID, DisableNotification: true}
	if _, err := s.sender.Request(pin); err != nil {
		log.Printf("Не удалось закрепить доску %d: %v", messageID, err)
	}
}

// isMessageNotFound сообщает, что редактируемого сообщения больше нет в чате.
func isMessageNotFound(err error) bool {
	var telegramError *tgbotapi.Error
	return errors.As(err, &telegramError) && strings.Contains(telegramError.Message, "message to edit not found")
}
