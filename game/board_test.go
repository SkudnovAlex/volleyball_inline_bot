package game

import (
	"strings"
	"testing"

	"volleyball_inline_bot/storage"
)

func TestRenderBoard(t *testing.T) {
	full := gameWith(rules.Limit)
	full.Date, full.MessageID = "2026-11-19", 20

	games := []storage.Game{
		{Date: "2026-11-22", MessageID: 30},
		*full,
		{Date: "2026-11-17", MessageID: 10, Players: []storage.Player{{UserID: 1, Status: storage.StatusGoing, Friends: 1}}},
		{Date: "2026-11-12", MessageID: 5, Finished: true, Players: []storage.Player{{UserID: 1, Status: storage.StatusGoing}}},
		{Date: "2026-11-15", MessageID: 7, Finished: true,
			Players: []storage.Player{
				{UserID: 1, Status: storage.StatusGoing},
				{UserID: 2, Status: storage.StatusGoing},
				{UserID: 3, Status: storage.StatusGoing},
			},
			Payment: &storage.Payment{RenterID: 3, Came: 3, PerPerson: 500, PaidUserIDs: []int64{2}},
		},
	}

	got := RenderBoard(games, -1001234567890)

	want := strings.Join([]string{
		"<b>Расписание игр</b>",
		`<a href="https://t.me/c/1234567890/10">Вторник 17.11</a> · 2/12`,
		`<a href="https://t.me/c/1234567890/20">Четверг 19.11</a> · 12/12 собран`,
		`<a href="https://t.me/c/1234567890/30">Воскресенье 22.11</a> · 0/12`,
		"",
		"<b>Прошедшие</b>",
		`<a href="https://t.me/c/1234567890/7">Воскресенье 15.11</a> · по 500 ₽ · перевели 1 из 2`,
		`<a href="https://t.me/c/1234567890/5">Четверг 12.11</a> · было 1, зал не оплачен`,
	}, "\n")
	if got != want {
		t.Fatalf("доска:\n%s\n\nожидалось:\n%s", got, want)
	}
}

func TestRenderBoard_Empty(t *testing.T) {
	want := "<b>Расписание игр</b>\nБлижайших игр пока нет"
	if got := RenderBoard(nil, -1001234567890); got != want {
		t.Fatalf("доска:\n%s\n\nожидалось:\n%s", got, want)
	}
}

func TestRenderBoard_BasicGroupWithoutLinks(t *testing.T) {
	got := RenderBoard([]storage.Game{{Date: "2026-11-17", MessageID: 10}}, -5542187824)

	if strings.Contains(got, "<a ") || !strings.Contains(got, "Вторник 17.11 · 0/12") {
		t.Fatalf("в обычной группе ссылок нет, дата текстом:\n%s", got)
	}
}
