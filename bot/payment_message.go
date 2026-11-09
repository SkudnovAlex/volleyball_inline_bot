package bot

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/game"
	"volleyball_inline_bot/storage"
)

// sendPaymentMessage публикует сообщение оплаты ответом на сообщение игры.
// Прошлое сообщение оплаты этой игры, если оно было, удаляется.
// Вызывается под mutex после сохранения оплаты.
func (s *Service) sendPaymentMessage(data *storage.Data, match *storage.Game) {
	payment := match.Payment
	if payment.MessageID != 0 {
		if _, err := s.sender.Request(tgbotapi.NewDeleteMessage(s.chatID, payment.MessageID)); err != nil {
			log.Printf("Не удалось удалить прошлое сообщение оплаты %d: %v", payment.MessageID, err)
		}
		payment.MessageID = 0
	}

	text, keyboard := game.RenderPayment(match)
	message := tgbotapi.NewMessage(s.chatID, text)
	message.ParseMode = tgbotapi.ModeHTML
	message.ReplyToMessageID = match.MessageID
	message.ReplyMarkup = keyboard
	sentMessage, err := s.sender.Send(message)
	if err != nil {
		log.Printf("Не удалось отправить сообщение оплаты %s: %v", match.Date, err)
	} else {
		payment.MessageID = sentMessage.MessageID
	}
	if err := s.repository.Save(*data); err != nil {
		log.Printf("Сообщение оплаты %s отправлено, но не сохранено: %v", match.Date, err)
	}
}

// markPaid обрабатывает «Перевёл» под сообщением оплаты. Вызывается под mutex.
func (s *Service) markPaid(query *tgbotapi.CallbackQuery, data *storage.Data, match *storage.Game) {
	amount, err := game.MarkPaid(match, query.From.ID)
	if err != nil {
		s.alert(query.ID, err.Error())
		return
	}
	if !s.save(data, query.ID) {
		return
	}
	s.editPaymentMessage(match)
	s.updateBoardAndSave(data)
	s.answer(query.ID, fmt.Sprintf("Отмечено: ты перевёл %d ₽", amount))
}

// editPaymentMessage перерисовывает сообщение оплаты по текущим данным.
// Возвращает false, если сообщения оплаты больше нет в чате.
func (s *Service) editPaymentMessage(match *storage.Game) bool {
	if match.Payment == nil || match.Payment.MessageID == 0 {
		return true
	}
	text, keyboard := game.RenderPayment(match)
	edit := tgbotapi.NewEditMessageText(s.chatID, match.Payment.MessageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	edit.ReplyMarkup = keyboard
	_, err := s.sender.Request(edit)
	if err == nil || isNotModified(err) {
		return true
	}
	if isMessageNotFound(err) {
		return false
	}
	log.Printf("Не удалось обновить сообщение оплаты %s: %v", match.Date, err)
	return true
}
