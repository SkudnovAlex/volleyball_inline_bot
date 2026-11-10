package config

import "time"

// Config хранит конфигурацию приложения. FinishSchedule вычисляется из GAME_TIME и FINISH_AFTER: ежедневно в момент
// закрытия игры.
type Config struct {
	BotToken           string
	ChatID             int64
	StorePath          string
	Schedule           string
	CleanupSchedule    string
	BoardCheckSchedule string
	FinishSchedule     string
	Location           *time.Location
	GameDays           map[time.Weekday]bool
	GameTime           string
	FinishAfter        time.Duration
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
