package game

import (
	"os"
	"strings"
	"testing"

	"volleyball_inline_bot/config"
)

func TestMain(m *testing.M) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "test")
	os.Setenv("TELEGRAM_CHAT_ID", "42")
	os.Setenv("GAME_DAYS", "Tuesday,Thursday,Friday,Sunday")
	os.Setenv("GAME_TIME", "")
	settings, err := config.Load()
	if err != nil {
		panic(err)
	}
	DefaultRules.GameTimes = settings.GameTimes
	Configure(DefaultRules)
	os.Exit(m.Run())
}

func TestConfigure(t *testing.T) {
	t.Cleanup(func() { Configure(DefaultRules) })
	t.Setenv("GAME_TIME", "20:30")
	settings, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	Configure(Rules{
		GameTimes: settings.GameTimes,
		Limit:     4, MaxFriends: 1, MaxCame: 10, MinTotal: 500, MaxTotal: 2000, RoundTo: 50,
	})

	game := gameWith(3)
	game.Date = "2026-11-17"
	text, _ := RenderGame(game)
	if !strings.Contains(text, "Вторник 17.11 · 20:30") || !strings.Contains(text, "▰▰▰▱ 3/4 · свободно 1") {
		t.Errorf("время и лимит из правил:\n%s", text)
	}

	user := User{ID: 99, Name: "Новый"}
	if err := AddFriend(game, user); err != nil {
		t.Fatalf("первый друг помещается: %v", err)
	}
	if err := AddFriend(game, user); err == nil || err.Error() != "Можно привести не больше 1 друзей" {
		t.Errorf("лимит друзей из правил: %v", err)
	}
	if _, err := ParseCame("11"); err == nil || err.Error() != "Пришли число от 1 до 10" {
		t.Errorf("максимум пришедших из правил: %v", err)
	}
	if _, _, err := ParseAmount("2500"); err == nil || !strings.Contains(err.Error(), "от 500 до 2000") {
		t.Errorf("границы суммы из правил: %v", err)
	}
	if got := PerPerson(1000, 3); got != 350 {
		t.Errorf("округление до 50: PerPerson = %d", got)
	}
}
