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
	s.mutex.Lock()
	defer s.mutex.Unlock()

	gameDay := s.now().AddDate(0, 0, createDaysAhead)
	date := gameDay.Format(game.DateLayout)
	day := game.DayTitle(date)

	if !s.gameDays[gameDay.Weekday()] {
		log.Printf("На %s игры нет, пропускаем", day)
		return
	}

	data, err := s.repository.Load()
	if err != nil {
		log.Printf("Ошибка чтения хранилища: %v", err)
		return
	}
	if findGame(data.Games, date) != nil {
		log.Printf("Игра на %s уже создана, пропускаем", day)
		return
	}

	newGame := storage.Game{Date: date}
	text, keyboard := game.RenderGame(&newGame)
	message := tgbotapi.NewMessage(s.chatID, text)
	message.ParseMode = tgbotapi.ModeHTML
	message.ReplyMarkup = keyboard

	sentMessage, err := s.sender.Send(message)
	if err != nil {
		log.Printf("Ошибка отправки игры на %s: %v", day, err)
		return
	}
	newGame.MessageID = sentMessage.MessageID

	data.Games = append(data.Games, newGame)
	if err := s.repository.Save(data); err != nil {
		// Сообщение уже в чате, но бот о нём не знает: его придётся удалить вручную.
		log.Printf("Игра на %s отправлена (сообщение %d), но не сохранена: %v", day, sentMessage.MessageID, err)
		return
	}
	log.Printf("Игра на %s создана, сообщение %d", day, sentMessage.MessageID)
	s.updateBoardAndSave(&data)
}

// findGame возвращает игру по дате или nil.
func findGame(games []storage.Game, date string) *storage.Game {
	for index := range games {
		if games[index].Date == date {
			return &games[index]
		}
	}
	return nil
}
