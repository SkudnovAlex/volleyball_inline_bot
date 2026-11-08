package config

const (
	defaultStorePath = "data.json"
	// defaultSchedule задаёт расписание создания игры (ежедневно в 20:00).
	defaultSchedule = "0 20 * * *"
	// defaultCleanupSchedule задаёт расписание очистки (ежедневно в 06:00).
	defaultCleanupSchedule = "0 6 * * *"
	defaultTimezone        = "Europe/Moscow"
	// defaultGameDays содержит игровые дни по умолчанию (англ. названия через запятую).
	defaultGameDays = "Tuesday,Thursday,Friday,Sunday"
	// defaultDeleteAfterDays задаёт, через сколько дней после игры удалять её сообщение.
	defaultDeleteAfterDays = "14"
)
