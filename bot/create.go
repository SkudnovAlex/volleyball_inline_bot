package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/game"
	"volleyball_inline_bot/storage"
)

// createDaysAhead задаёт, за сколько дней до игры создаётся её сообщение.
const createDaysAhead = 7

// CreateGame создаёт сообщение игры, если через createDaysAhead дней
// игровой день и игры на эту дату ещё нет.
func (s *Service) CreateGame() {
	s.mu.Lock()
	defer s.mu.Unlock()

	gameDay := s.now().AddDate(0, 0, createDaysAhead)
	date := gameDay.Format(game.DateLayout)
	day := game.DayTitle(date)

	if !s.gameDays[gameDay.Weekday()] {
		log.Printf("На %s игры нет, пропускаем", day)
		return
	}

	data, err := s.repo.Load()
	if err != nil {
		log.Printf("Ошибка чтения хранилища: %v", err)
		return
	}
	if findGame(data.Games, date) != nil {
		log.Printf("Игра на %s уже создана, пропускаем", day)
		return
	}

	g := storage.Game{Date: date}
	text, kb := game.RenderGame(&g)
	msg := tgbotapi.NewMessage(s.chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = kb

	sent, err := s.sender.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки игры на %s: %v", day, err)
		return
	}
	g.MessageID = sent.MessageID

	data.Games = append(data.Games, g)
	if err := s.repo.Save(data); err != nil {
		// Сообщение уже в чате, но бот о нём не знает: его придётся удалить вручную.
		log.Printf("Игра на %s отправлена (сообщение %d), но не сохранена: %v", day, sent.MessageID, err)
		return
	}
	log.Printf("Игра на %s создана, сообщение %d", day, sent.MessageID)
}

// findGame возвращает игру по дате или nil.
func findGame(games []storage.Game, date string) *storage.Game {
	for i := range games {
		if games[i].Date == date {
			return &games[i]
		}
	}
	return nil
}
