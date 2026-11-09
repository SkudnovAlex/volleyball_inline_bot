package game

import (
	"strings"
	"testing"
)

func TestConfigure(t *testing.T) {
	t.Cleanup(func() { Configure(DefaultRules) })
	Configure(Rules{GameTime: "20:30", Limit: 4, MaxFriends: 1, MaxCame: 10, MinTotal: 500, MaxTotal: 2000, RoundTo: 50})

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
