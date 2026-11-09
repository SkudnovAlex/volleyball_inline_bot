package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/config"
	"volleyball_inline_bot/scheduler"
)

// updatesTimeout задаёт длительность long polling запроса getUpdates в секундах.
const updatesTimeout = 60

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

	planner := scheduler.New(settings.Location)
	planner.Start()
	log.Printf("Планировщик запущен (создание: %q, очистка: %q, зона: %s, удаление через %d дн.)",
		settings.Schedule, settings.CleanupSchedule, settings.Location, settings.DeleteAfterDays)

	updatesConfig := tgbotapi.NewUpdate(0)
	updatesConfig.Timeout = updatesTimeout
	// Нажатия кнопок и сообщения в личке с ботом (оформление оплаты).
	updatesConfig.AllowedUpdates = []string{"message", "callback_query"}
	go handleUpdates(botAPI.GetUpdatesChan(updatesConfig))

	scheduler.WaitForShutdown(planner)
	botAPI.StopReceivingUpdates()
}

// handleUpdates читает апдейты до закрытия канала.
// Обработчики нажатий и сообщений появятся в следующих задачах.
func handleUpdates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		log.Printf("Апдейт %d пропущен: обработчиков пока нет", update.UpdateID)
	}
}
