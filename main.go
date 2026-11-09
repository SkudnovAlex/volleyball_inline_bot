package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/bot"
	"volleyball_inline_bot/config"
	"volleyball_inline_bot/scheduler"
	"volleyball_inline_bot/storage"
)

// updatesTimeout задаёт длительность long polling запроса getUpdates в секундах.
const updatesTimeout = 60

// boardCheckSchedule задаёт проверку закрепа доски (каждый час).
const boardCheckSchedule = "0 * * * *"

func main() {
	settings, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка конфигурации: %v", err)
	}

	botAPI, err := tgbotapi.NewBotAPI(settings.BotToken)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}
	log.Printf("Бот @%s запущен", botAPI.Self.UserName)

	store := storage.NewStore(settings.StorePath)
	service := bot.New(botAPI, store, settings.ChatID, settings.Location, settings.GameDays)
	// При старте сразу создаём игру, если её время уже пришло: бот мог быть
	// выключен в момент запуска по расписанию. Затем доска, если её ещё нет.
	service.CreateGame()
	service.RefreshBoard()

	planner := scheduler.New(settings.Location)
	if err := scheduler.Add(planner, settings.Schedule, service.CreateGame); err != nil {
		log.Fatalf("Ошибка в CRON_SCHEDULE %q: %v", settings.Schedule, err)
	}
	if err := scheduler.Add(planner, boardCheckSchedule, service.RefreshBoard); err != nil {
		log.Fatalf("Ошибка в расписании проверки доски: %v", err)
	}
	planner.Start()
	log.Printf("Планировщик запущен (создание: %q, очистка: %q, зона: %s, удаление через %d дн.)",
		settings.Schedule, settings.CleanupSchedule, settings.Location, settings.DeleteAfterDays)

	updatesConfig := tgbotapi.NewUpdate(0)
	updatesConfig.Timeout = updatesTimeout
	// Нажатия кнопок и сообщения в личке с ботом (оформление оплаты).
	updatesConfig.AllowedUpdates = []string{"message", "callback_query"}
	updates := botAPI.GetUpdatesChan(updatesConfig)
	go func() {
		for update := range updates {
			service.HandleUpdate(update)
		}
	}()

	scheduler.WaitForShutdown(planner)
	botAPI.StopReceivingUpdates()
}
