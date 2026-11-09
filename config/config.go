package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	// Встраиваем базу таймзон, чтобы time.LoadLocation работал
	// независимо от наличия zoneinfo в ОС (минимальные Docker-образы, Windows).
	_ "time/tzdata"

	"github.com/joho/godotenv"
)

// weekdayByName сопоставляет англоязычное название дня недели с time.Weekday.
var weekdayByName = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

// Load читает конфигурацию из .env (если есть) и переменных окружения.
func Load() (*Config, error) {
	// .env не обязателен: если файла нет, читаем переменные из реального окружения.
	if err := godotenv.Load(); err != nil {
		log.Println(".env не найден, использую переменные окружения")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("не задан TELEGRAM_BOT_TOKEN")
	}

	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	if chatIDStr == "" {
		return nil, fmt.Errorf("не задан TELEGRAM_CHAT_ID")
	}
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("некорректный TELEGRAM_CHAT_ID: %w", err)
	}

	timezoneName := getenvDefault("TIMEZONE", defaultTimezone)
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		return nil, fmt.Errorf("некорректный TIMEZONE %q: %w", timezoneName, err)
	}

	gameDays, err := parseGameDays(getenvDefault("GAME_DAYS", defaultGameDays))
	if err != nil {
		return nil, fmt.Errorf("некорректный GAME_DAYS: %w", err)
	}

	deleteAfterDays, err := strconv.Atoi(getenvDefault("DELETE_AFTER_DAYS", defaultDeleteAfterDays))
	if err != nil || deleteAfterDays < 1 {
		return nil, fmt.Errorf("некорректный DELETE_AFTER_DAYS: нужно целое число дней больше нуля")
	}

	return &Config{
		BotToken:        token,
		ChatID:          chatID,
		StorePath:       getenvDefault("STORE_PATH", defaultStorePath),
		Schedule:        getenvDefault("CRON_SCHEDULE", defaultSchedule),
		CleanupSchedule: getenvDefault("CLEANUP_SCHEDULE", defaultCleanupSchedule),
		Location:        location,
		GameDays:        gameDays,
		DeleteAfterDays: deleteAfterDays,
	}, nil
}

// parseGameDays разбирает список игровых дней вида "Tuesday,Thursday"
// (англ. названия, регистр не важен) в map[time.Weekday]bool.
func parseGameDays(list string) (map[time.Weekday]bool, error) {
	result := make(map[time.Weekday]bool)
	for _, part := range strings.Split(list, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		weekday, ok := weekdayByName[name]
		if !ok {
			return nil, fmt.Errorf("неизвестный день недели %q", part)
		}
		result[weekday] = true
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("список игровых дней пуст")
	}
	return result, nil
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
