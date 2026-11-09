package config

import (
	"testing"
	"time"
)

func TestLoad_OK_WithDefaults(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "token123")
	t.Setenv("TELEGRAM_CHAT_ID", "-1001234567890")
	t.Setenv("STORE_PATH", "")
	t.Setenv("CRON_SCHEDULE", "")
	t.Setenv("TIMEZONE", "")
	t.Setenv("GAME_DAYS", "")
	t.Setenv("DELETE_AFTER_DAYS", "")

	settings, err := Load()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if settings.BotToken != "token123" {
		t.Errorf("BotToken = %q", settings.BotToken)
	}
	if settings.ChatID != -1001234567890 {
		t.Errorf("ChatID = %d", settings.ChatID)
	}
	if settings.StorePath != defaultStorePath {
		t.Errorf("StorePath = %q, ожидался дефолт %q", settings.StorePath, defaultStorePath)
	}
	if settings.Schedule != defaultSchedule {
		t.Errorf("Schedule = %q, ожидался дефолт %q", settings.Schedule, defaultSchedule)
	}
	if settings.Location == nil || settings.Location.String() != defaultTimezone {
		t.Errorf("Location = %v, ожидался дефолт %q", settings.Location, defaultTimezone)
	}
	if settings.DeleteAfterDays != 14 {
		t.Errorf("DeleteAfterDays = %d, ожидался дефолт 14", settings.DeleteAfterDays)
	}
	// По умолчанию игровые: вторник, четверг, пятница, воскресенье.
	wantDays := map[time.Weekday]bool{time.Tuesday: true, time.Thursday: true, time.Friday: true, time.Sunday: true}
	if len(settings.GameDays) != len(wantDays) {
		t.Fatalf("GameDays = %v, ожидалось %v", settings.GameDays, wantDays)
	}
	for day := range wantDays {
		if !settings.GameDays[day] {
			t.Errorf("в GameDays нет ожидаемого дня %s", day)
		}
	}
}

func TestLoad_GameDaysOverride(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "t")
	t.Setenv("TELEGRAM_CHAT_ID", "42")
	t.Setenv("GAME_DAYS", "monday, FRIDAY")

	settings, err := Load()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(settings.GameDays) != 2 || !settings.GameDays[time.Monday] || !settings.GameDays[time.Friday] {
		t.Fatalf("GameDays = %v, ожидались Monday и Friday", settings.GameDays)
	}
}

func TestLoad_BadGameDays(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "t")
	t.Setenv("TELEGRAM_CHAT_ID", "42")
	t.Setenv("GAME_DAYS", "Funday")

	if _, err := Load(); err == nil {
		t.Fatal("ожидалась ошибка при неизвестном дне недели")
	}
}

func TestLoad_BadTimezone(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "t")
	t.Setenv("TELEGRAM_CHAT_ID", "42")
	t.Setenv("TIMEZONE", "Nowhere/Nope")

	if _, err := Load(); err == nil {
		t.Fatal("ожидалась ошибка при некорректной таймзоне")
	}
}

func TestLoad_OverridesFromEnv(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "t")
	t.Setenv("TELEGRAM_CHAT_ID", "42")
	t.Setenv("STORE_PATH", "/tmp/custom.json")
	t.Setenv("CRON_SCHEDULE", "0 20 * * *")

	settings, err := Load()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if settings.StorePath != "/tmp/custom.json" {
		t.Errorf("StorePath = %q", settings.StorePath)
	}
	if settings.Schedule != "0 20 * * *" {
		t.Errorf("Schedule = %q", settings.Schedule)
	}
}

func TestLoad_MissingToken(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "42")

	if _, err := Load(); err == nil {
		t.Fatal("ожидалась ошибка при отсутствии токена")
	}
}

func TestLoad_MissingChatID(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "t")
	t.Setenv("TELEGRAM_CHAT_ID", "")

	if _, err := Load(); err == nil {
		t.Fatal("ожидалась ошибка при отсутствии chat id")
	}
}

func TestLoad_BadChatID(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "t")
	t.Setenv("TELEGRAM_CHAT_ID", "not-a-number")

	if _, err := Load(); err == nil {
		t.Fatal("ожидалась ошибка при некорректном chat id")
	}
}

func TestLoad_DeleteAfterDaysOverride(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "t")
	t.Setenv("TELEGRAM_CHAT_ID", "42")
	t.Setenv("DELETE_AFTER_DAYS", "3")

	settings, err := Load()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if settings.DeleteAfterDays != 3 {
		t.Errorf("DeleteAfterDays = %d, ожидалось 3", settings.DeleteAfterDays)
	}
}

func TestLoad_BadDeleteAfterDays(t *testing.T) {
	for _, value := range []string{"abc", "0", "-5"} {
		t.Setenv("TELEGRAM_BOT_TOKEN", "t")
		t.Setenv("TELEGRAM_CHAT_ID", "42")
		t.Setenv("DELETE_AFTER_DAYS", value)

		if _, err := Load(); err == nil {
			t.Errorf("DELETE_AFTER_DAYS=%q: ожидалась ошибка", value)
		}
	}
}
