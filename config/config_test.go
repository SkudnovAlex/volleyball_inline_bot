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

	cfg, err := Load()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if cfg.BotToken != "token123" {
		t.Errorf("BotToken = %q", cfg.BotToken)
	}
	if cfg.ChatID != -1001234567890 {
		t.Errorf("ChatID = %d", cfg.ChatID)
	}
	if cfg.StorePath != defaultStorePath {
		t.Errorf("StorePath = %q, ожидался дефолт %q", cfg.StorePath, defaultStorePath)
	}
	if cfg.Schedule != defaultSchedule {
		t.Errorf("Schedule = %q, ожидался дефолт %q", cfg.Schedule, defaultSchedule)
	}
	if cfg.Location == nil || cfg.Location.String() != defaultTimezone {
		t.Errorf("Location = %v, ожидался дефолт %q", cfg.Location, defaultTimezone)
	}
	if cfg.DeleteAfterDays != 14 {
		t.Errorf("DeleteAfterDays = %d, ожидался дефолт 14", cfg.DeleteAfterDays)
	}
	// По умолчанию игровые: вторник, четверг, пятница, воскресенье.
	wantDays := map[time.Weekday]bool{time.Tuesday: true, time.Thursday: true, time.Friday: true, time.Sunday: true}
	if len(cfg.GameDays) != len(wantDays) {
		t.Fatalf("GameDays = %v, ожидалось %v", cfg.GameDays, wantDays)
	}
	for d := range wantDays {
		if !cfg.GameDays[d] {
			t.Errorf("в GameDays нет ожидаемого дня %s", d)
		}
	}
}

func TestLoad_GameDaysOverride(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "t")
	t.Setenv("TELEGRAM_CHAT_ID", "42")
	t.Setenv("GAME_DAYS", "monday, FRIDAY")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(cfg.GameDays) != 2 || !cfg.GameDays[time.Monday] || !cfg.GameDays[time.Friday] {
		t.Fatalf("GameDays = %v, ожидались Monday и Friday", cfg.GameDays)
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

	cfg, err := Load()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if cfg.StorePath != "/tmp/custom.json" {
		t.Errorf("StorePath = %q", cfg.StorePath)
	}
	if cfg.Schedule != "0 20 * * *" {
		t.Errorf("Schedule = %q", cfg.Schedule)
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

	cfg, err := Load()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if cfg.DeleteAfterDays != 3 {
		t.Errorf("DeleteAfterDays = %d, ожидалось 3", cfg.DeleteAfterDays)
	}
}

func TestLoad_BadDeleteAfterDays(t *testing.T) {
	for _, v := range []string{"abc", "0", "-5"} {
		t.Setenv("TELEGRAM_BOT_TOKEN", "t")
		t.Setenv("TELEGRAM_CHAT_ID", "42")
		t.Setenv("DELETE_AFTER_DAYS", v)

		if _, err := Load(); err == nil {
			t.Errorf("DELETE_AFTER_DAYS=%q: ожидалась ошибка", v)
		}
	}
}
