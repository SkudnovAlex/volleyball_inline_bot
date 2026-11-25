package config

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	// Встраиваем базу таймзон, чтобы time.LoadLocation работал независимо от наличия zoneinfo в ОС (минимальные
	// Docker-образы, Windows).
	_ "time/tzdata"

	"github.com/joho/godotenv"
)

// weekdayByName сопоставляет англоязычное название дня недели с time.Weekday.
var weekdayByName = map[string]time.Weekday{
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
	"sunday":    time.Sunday,
}

// Load читает конфигурацию из .env (если есть) и переменных окружения. Незаданные необязательные переменные получают
// значения по умолчанию из constants.go, некорректные значения дают ошибку с именем переменной.
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

	gameDays, gameTimes, err := parseGameDays(getenvDefault("GAME_DAYS", defaultGameDays))
	if err != nil {
		return nil, fmt.Errorf("некорректный GAME_DAYS: %w", err)
	}

	gameTime := getenvDefault("GAME_TIME", defaultGameTime)
	gameClock, err := time.Parse("15:04", gameTime)
	if err != nil {
		return nil, fmt.Errorf("некорректный GAME_TIME %q: нужен формат ЧЧ:ММ, например 19:00", gameTime)
	}

	finishAfter, err := positiveDuration("FINISH_AFTER", defaultFinishAfter)
	if err != nil {
		return nil, err
	}
	// Расписание закрытия срабатывает раз в минуту по часам, поэтому длительность задаётся целыми минутами.
	if finishAfter%time.Minute != 0 {
		return nil, fmt.Errorf("некорректный FINISH_AFTER: нужно целое число минут, например 60m или 1h30m")
	}
	finishClock := gameClock.Add(finishAfter)
	finishSchedules := []string{dailySchedule(finishClock)}
	for day := time.Sunday; day <= time.Saturday; day++ {
		clock, ok := gameTimes[day]
		if !ok {
			clock = gameClock
			gameTimes[day] = clock
		}
		schedule := dailySchedule(clock.Add(finishAfter))
		if !slices.Contains(finishSchedules, schedule) {
			finishSchedules = append(finishSchedules, schedule)
		}
	}

	settings := &Config{
		BotToken:           token,
		ChatID:             chatID,
		StorePath:          getenvDefault("STORE_PATH", defaultStorePath),
		Schedule:           getenvDefault("CRON_SCHEDULE", defaultSchedule),
		CleanupSchedule:    getenvDefault("CLEANUP_SCHEDULE", defaultCleanupSchedule),
		BoardCheckSchedule: getenvDefault("BOARD_CHECK_SCHEDULE", defaultBoardCheckSchedule),
		BoardRaiseSchedule: getenvDefault("BOARD_RAISE_SCHEDULE", defaultBoardRaiseSchedule),
		Location:           location,
		GameDays:           gameDays,
		GameTime:           gameTime,
		GameTimes:          gameTimes,
		FinishAfter:        finishAfter,
		FinishSchedules:    finishSchedules,
	}

	numbers := []struct {
		key      string
		fallback string
		target   *int
	}{
		{"DELETE_AFTER_DAYS", defaultDeleteAfterDays, &settings.DeleteAfterDays},
		{"CREATE_DAYS_AHEAD", defaultCreateDaysAhead, &settings.CreateDaysAhead},
		{"PLAYERS_LIMIT", defaultPlayersLimit, &settings.PlayersLimit},
		{"MAX_FRIENDS", defaultMaxFriends, &settings.MaxFriends},
		{"PAYMENT_MAX_CAME", defaultPaymentMaxCame, &settings.PaymentMaxCame},
		{"PAYMENT_MIN_TOTAL", defaultPaymentMinTotal, &settings.PaymentMinTotal},
		{"PAYMENT_MAX_TOTAL", defaultPaymentMaxTotal, &settings.PaymentMaxTotal},
		{"PAYMENT_ROUND_TO", defaultPaymentRoundTo, &settings.PaymentRoundTo},
	}
	for _, number := range numbers {
		if *number.target, err = positiveInt(number.key, number.fallback); err != nil {
			return nil, err
		}
	}
	if settings.PaymentMinTotal >= settings.PaymentMaxTotal {
		return nil, fmt.Errorf("PAYMENT_MIN_TOTAL (%d) должен быть меньше PAYMENT_MAX_TOTAL (%d)",
			settings.PaymentMinTotal, settings.PaymentMaxTotal)
	}

	if settings.AnnounceCooldown, err = positiveDuration("ANNOUNCE_COOLDOWN", defaultAnnounceCooldown); err != nil {
		return nil, err
	}
	if settings.PaymentDraftLock, err = positiveDuration("PAYMENT_DRAFT_LOCK", defaultPaymentDraftLock); err != nil {
		return nil, err
	}
	return settings, nil
}

func dailySchedule(clock time.Time) string {
	return fmt.Sprintf("%d %d * * *", clock.Minute(), clock.Hour())
}

// parseGameDays разбирает список игровых дней вида "Tuesday,Thursday@19:30".
func parseGameDays(list string) (map[time.Weekday]bool, map[time.Weekday]time.Time, error) {
	result := make(map[time.Weekday]bool)
	gameTimes := make(map[time.Weekday]time.Time)
	for _, part := range strings.Split(list, ",") {
		name, clockText, hasTime := strings.Cut(strings.TrimSpace(part), "@")
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" && !hasTime {
			continue
		}
		weekday, ok := weekdayByName[name]
		if !ok {
			return nil, nil, fmt.Errorf("неизвестный день недели %q", part)
		}
		result[weekday] = true
		if hasTime {
			clock, err := time.Parse("15:04", strings.TrimSpace(clockText))
			if err != nil {
				return nil, nil, fmt.Errorf("некорректное время в %q: нужен формат ЧЧ:ММ", part)
			}
			gameTimes[weekday] = clock
		}
	}
	if len(result) == 0 {
		return nil, nil, fmt.Errorf("список игровых дней пуст")
	}
	return result, gameTimes, nil
}

// positiveInt читает целое число больше нуля.
func positiveInt(key, fallback string) (int, error) {
	value, err := strconv.Atoi(getenvDefault(key, fallback))
	if err != nil || value < 1 {
		return 0, fmt.Errorf("некорректный %s: нужно целое число больше нуля", key)
	}
	return value, nil
}

// positiveDuration читает длительность больше нуля в формате Go: 90s, 10m, 1h.
func positiveDuration(key, fallback string) (time.Duration, error) {
	value, err := time.ParseDuration(getenvDefault(key, fallback))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("некорректный %s: нужна длительность больше нуля, например 10m или 1h", key)
	}
	return value, nil
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
