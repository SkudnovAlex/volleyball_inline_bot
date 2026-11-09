package config

import "time"

// Config хранит конфигурацию приложения.
type Config struct {
	BotToken           string
	ChatID             int64
	StorePath          string
	Schedule           string
	CleanupSchedule    string
	BoardCheckSchedule string
	Location           *time.Location
	GameDays           map[time.Weekday]bool
	GameTime           string
	DeleteAfterDays    int
	CreateDaysAhead    int
	PlayersLimit       int
	MaxFriends         int
	AnnounceCooldown   time.Duration
	PaymentDraftLock   time.Duration
	PaymentMaxCame     int
	PaymentMinTotal    int
	PaymentMaxTotal    int
	PaymentRoundTo     int
}
