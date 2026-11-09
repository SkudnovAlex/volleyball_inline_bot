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

// RefreshBoard отправляет сообщения игр, у которых их ещё нет, затем
// обновляет доску и проверяет, что она закреплена. Вызывается при старте
// бота и по расписанию раз в час.
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
	s.updateBoardAndSave(&data)
}

// sendMissingGames отправляет сообщения для игр с message_id 0 и сообщения
// оплаты для оплаченных игр без него: так игру можно добавить или
// восстановить, вписав её в файл данных вручную. Возвращает true,
// если что-то отправлено.
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

// updateBoardAndSave обновляет доску по уже сохранённым данным и, если
// пришлось отправить новую доску, сохраняет её ID.
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

// updateBoard редактирует доску или отправляет новую, если её ещё нет
// или её удалили, и закрепляет её. Новый ID записывается в data.
func (s *Service) updateBoard(data *storage.Data) {
	text := game.RenderBoard(data.Games, s.chatID)

	if data.BoardMessageID != 0 {
		edit := tgbotapi.NewEditMessageText(s.chatID, data.BoardMessageID, text)
		edit.ParseMode = tgbotapi.ModeHTML
		edit.DisableWebPagePreview = true
		_, err := s.sender.Request(edit)
		switch {
		case err == nil || isNotModified(err):
			s.ensurePinned(data.BoardMessageID)
			return
		case isMessageNotFound(err):
			log.Printf("Доска %d удалена из чата, отправляю новую", data.BoardMessageID)
		default:
			log.Printf("Не удалось обновить доску %d: %v", data.BoardMessageID, err)
			return
		}
	}

	message := tgbotapi.NewMessage(s.chatID, text)
	message.ParseMode = tgbotapi.ModeHTML
	message.DisableWebPagePreview = true
	sentMessage, err := s.sender.Send(message)
	if err != nil {
		log.Printf("Не удалось отправить доску: %v", err)
		return
	}
	data.BoardMessageID = sentMessage.MessageID
	log.Printf("Отправлена новая доска %d", sentMessage.MessageID)
	s.pin(sentMessage.MessageID)
}

// ensurePinned закрепляет доску, если последним закреплённым сообщением
// чата сейчас не она: Telegram не сообщает боту об откреплении.
func (s *Service) ensurePinned(boardMessageID int) {
	response, err := s.sender.Request(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: s.chatID}})
	if err != nil {
		log.Printf("Не удалось получить закреп чата: %v", err)
		return
	}
	var chat tgbotapi.Chat
	if err := json.Unmarshal(response.Result, &chat); err != nil {
		log.Printf("Не удалось разобрать ответ getChat: %v", err)
		return
	}
	if chat.PinnedMessage != nil && chat.PinnedMessage.MessageID == boardMessageID {
		return
	}
	log.Printf("Доска %d не в закрепе, закрепляю снова", boardMessageID)
	s.pin(boardMessageID)
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
