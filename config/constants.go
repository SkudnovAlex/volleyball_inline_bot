package config

// Значения по умолчанию для необязательных переменных окружения. Описание каждой переменной в .env.example.
const (
	defaultStorePath          = "data.json"
	defaultSchedule           = "0 20 * * *"
	defaultCleanupSchedule    = "0 6 * * *"
	defaultBoardCheckSchedule = "0 * * * *"
	defaultTimezone           = "Europe/Moscow"
	defaultGameDays           = "Tuesday,Thursday,Friday,Sunday"
	defaultGameTime           = "19:00"
	defaultDeleteAfterDays    = "14"
	defaultCreateDaysAhead    = "7"
	defaultPlayersLimit       = "12"
	defaultMaxFriends         = "4"
	defaultAnnounceCooldown   = "10m"
	defaultPaymentDraftLock   = "1h"
	defaultPaymentMaxCame     = "30"
	defaultPaymentMinTotal    = "1000"
	defaultPaymentMaxTotal    = "10000"
	defaultPaymentRoundTo     = "10"
)
