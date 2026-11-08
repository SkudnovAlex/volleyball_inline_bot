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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка конфигурации: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}
	log.Printf("Бот @%s запущен", bot.Self.UserName)

	c := scheduler.New(cfg.Location)
	c.Start()
	log.Printf("Планировщик запущен (создание: %q, очистка: %q, зона: %s, удаление через %d дн.)",
		cfg.Schedule, cfg.CleanupSchedule, cfg.Location, cfg.DeleteAfterDays)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = updatesTimeout
	// Нажатия кнопок и ответы на сообщения бота (реквизиты оплаты).
	u.AllowedUpdates = []string{"message", "callback_query"}
	go handleUpdates(bot.GetUpdatesChan(u))

	scheduler.WaitForShutdown(c)
	bot.StopReceivingUpdates()
}

// handleUpdates читает апдейты до закрытия канала.
// Обработчики нажатий и сообщений появятся в следующих задачах.
func handleUpdates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		log.Printf("Апдейт %d пропущен: обработчиков пока нет", update.UpdateID)
	}
}
