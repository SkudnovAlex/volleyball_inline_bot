package config

import "time"

// Config хранит конфигурацию приложения.
type Config struct {
	BotToken        string
	ChatID          int64
	StorePath       string
	Schedule        string
	CleanupSchedule string
	Location        *time.Location
	GameDays        map[time.Weekday]bool
	DeleteAfterDays int
}
