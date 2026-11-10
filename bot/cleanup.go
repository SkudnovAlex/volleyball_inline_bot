package bot

import (
	"errors"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/game"
	"volleyball_inline_bot/storage"
)

// CleanupGames удаляет из чата и из данных игры, с даты которых прошло deleteAfterDays дней. Закрытие прошедших игр
// выполняет FinishGames.
func (s *Service) CleanupGames() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := s.repository.Load()
	if err != nil {
		log.Printf("Ошибка чтения хранилища: %v", err)
		return
	}

	// Игра удаляется, когда с её даты прошло deleteAfterDays дней.
	deleteUpTo := s.now().AddDate(0, 0, -s.deleteAfterDays).Format(game.DateLayout)

	remaining := make([]storage.Game, 0, len(data.Games))
	changed := false
	for _, match := range data.Games {
		if match.Date <= deleteUpTo {
			changed = true
			if s.deleteGame(match) {
				continue
			}
		}
		remaining = append(remaining, match)
	}
	if !changed {
		return
	}

	data.Games = remaining
	if err := s.repository.Save(data); err != nil {
		log.Printf("Не удалось сохранить данные после очистки: %v", err)
		return
	}
	s.updateBoardAndSave(&data)
}

// deleteGame удаляет из чата сообщение игры, объявление и сообщение оплаты. Возвращает true, если игру можно убрать из
// данных. При нехватке прав запись остаётся до следующей очистки, при прочих ошибках (например, сообщение уже удалили
// вручную) убирается.
func (s *Service) deleteGame(match storage.Game) bool {
	messageIDs := []int{match.MessageID}
	if match.Announce != nil {
		messageIDs = append(messageIDs, match.Announce.MessageID)
	}
	if match.Payment != nil && match.Payment.MessageID != 0 {
		messageIDs = append(messageIDs, match.Payment.MessageID)
	}

	for _, messageID := range messageIDs {
		_, err := s.sender.Request(tgbotapi.NewDeleteMessage(s.chatID, messageID))
		if err == nil {
			continue
		}
		if isInsufficientRights(err) {
			log.Printf("Нет прав удалить сообщение %d игры %s, повторю при следующей очистке", messageID, match.Date)
			return false
		}
		log.Printf("Сообщение %d игры %s не удалено, запись убирается: %v", messageID, match.Date, err)
	}
	log.Printf("Игра %s удалена", match.Date)
	return true
}

// isInsufficientRights сообщает, что у бота нет прав на действие в чате. Без права удалять сообщения Telegram отвечает
// "message can't be deleted" на сообщения старше 48 часов, поэтому эта ошибка тоже считается нехваткой прав.
func isInsufficientRights(err error) bool {
	var telegramError *tgbotapi.Error
	if !errors.As(err, &telegramError) {
		return false
	}
	return strings.Contains(telegramError.Message, "not enough rights") ||
		strings.Contains(telegramError.Message, "message can't be deleted")
}
