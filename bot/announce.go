package bot

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/game"
	"volleyball_inline_bot/storage"
)

// announceCooldown задаёт, сколько времени после объявления бот
// редактирует его вместо отправки нового.
const announceCooldown = 10 * time.Minute

// announce сообщает ответом на сообщение игры, что состав собран
// или освободились места, и записывает объявление в match.Announce.
func (s *Service) announce(match *storage.Game, event game.Event) {
	full := event == game.EventFilled
	text := announceText(match, full)
	now := s.now()

	previous := match.Announce
	if previous != nil && now.Sub(time.Unix(previous.SentAt, 0)) < announceCooldown {
		edit := tgbotapi.NewEditMessageText(s.chatID, previous.MessageID, text)
		edit.ParseMode = tgbotapi.ModeHTML
		_, err := s.sender.Request(edit)
		if err == nil || isNotModified(err) {
			previous.Full = full
			return
		}
		// Объявление могли удалить вручную: отправляем новое.
		log.Printf("Не удалось обновить объявление по игре %s: %v", match.Date, err)
	}

	message := tgbotapi.NewMessage(s.chatID, text)
	message.ParseMode = tgbotapi.ModeHTML
	message.ReplyToMessageID = match.MessageID
	sentMessage, err := s.sender.Send(message)
	if err != nil {
		log.Printf("Не удалось отправить объявление по игре %s: %v", match.Date, err)
		return
	}
	match.Announce = &storage.Announce{MessageID: sentMessage.MessageID, Full: full, SentAt: now.Unix()}
}

func announceText(match *storage.Game, full bool) string {
	day := game.DayTitle(match.Date)
	if full {
		return fmt.Sprintf("<b>%s</b>: состав собран, %d/%d. Запись закрыта", day, game.Count(match), game.Limit)
	}
	return fmt.Sprintf("<b>%s</b>: освободилось мест: %d. Запись снова открыта", day, game.Free(match))
}
