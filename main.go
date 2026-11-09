package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/bot"
	"volleyball_inline_bot/config"
	"volleyball_inline_bot/game"
	"volleyball_inline_bot/scheduler"
	"volleyball_inline_bot/storage"
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

	game.Configure(game.Rules{
		GameTime:   settings.GameTime,
		Limit:      settings.PlayersLimit,
		MaxFriends: settings.MaxFriends,
		MaxCame:    settings.PaymentMaxCame,
		MinTotal:   settings.PaymentMinTotal,
		MaxTotal:   settings.PaymentMaxTotal,
		RoundTo:    settings.PaymentRoundTo,
	})
	store := storage.NewStore(settings.StorePath)
	service := bot.New(botAPI, store, bot.Settings{
		ChatID:           settings.ChatID,
		Location:         settings.Location,
		GameDays:         settings.GameDays,
		DeleteAfterDays:  settings.DeleteAfterDays,
		CreateDaysAhead:  settings.CreateDaysAhead,
		AnnounceCooldown: settings.AnnounceCooldown,
		DraftLock:        settings.PaymentDraftLock,
		BotUserName:      botAPI.Self.UserName,
	})

	// Расписания проверяются до первых действий в чате: при ошибке бот завершается, ничего не отправив.
	planner := scheduler.New(settings.Location)
	jobs := []struct {
		key      string
		schedule string
		job      func()
	}{
		{"CRON_SCHEDULE", settings.Schedule, service.CreateGame},
		{"CLEANUP_SCHEDULE", settings.CleanupSchedule, service.CleanupGames},
		{"BOARD_CHECK_SCHEDULE", settings.BoardCheckSchedule, service.RefreshBoard},
	}
	for _, job := range jobs {
		if err := scheduler.Add(planner, job.schedule, job.job); err != nil {
			log.Fatalf("Ошибка в %s %q: %v", job.key, job.schedule, err)
		}
	}

	// При старте сразу выполняем задачи расписания: бот мог быть выключен в момент их запуска. Затем перерисовываем
	// сообщения игр (оформление могло измениться с обновлением бота) и доску.
	service.CleanupGames()
	service.CreateGame()
	service.RedrawGames()
	service.RefreshBoard()

	planner.Start()
	log.Printf("Планировщик запущен (создание: %q, очистка: %q, доска: %q, зона: %s)",
		settings.Schedule, settings.CleanupSchedule, settings.BoardCheckSchedule, settings.Location)

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
