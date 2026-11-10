package config

// Значения по умолчанию для необязательных переменных окружения. Описание каждой переменной в .env.example.
const (
	defaultStorePath          = "data.json"
	defaultSchedule           = "0 20 * * *"
	defaultCleanupSchedule    = "10 0 * * *"
	defaultBoardCheckSchedule = "0 * * * *"
	defaultTimezone           = "Europe/Moscow"
	defaultGameDays           = "Tuesday,Thursday,Friday,Sunday"
	defaultGameTime           = "19:00"
	defaultFinishAfter        = "1h"
	defaultDeleteAfterDays    = "7"
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
