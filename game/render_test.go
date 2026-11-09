package game

import (
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/storage"
)

// buttons возвращает callback_data всех кнопок построчно.
func buttons(kb *tgbotapi.InlineKeyboardMarkup) [][]string {
	var rows [][]string
	for _, row := range kb.InlineKeyboard {
		var r []string
		for _, b := range row {
			r = append(r, *b.CallbackData)
		}
		rows = append(rows, r)
	}
	return rows
}

func equalRows(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.Join(a[i], ",") != strings.Join(b[i], ",") {
			return false
		}
	}
	return true
}

func TestDayTitle(t *testing.T) {
	if got := DayTitle("2026-11-18"); got != "Среда 18.11" {
		t.Errorf("DayTitle = %q", got)
	}
	if got := DayTitle("2026-11-17"); got != "Вторник 17.11" {
		t.Errorf("DayTitle = %q", got)
	}
	if got := DayTitle("мусор"); got != "мусор" {
		t.Errorf("некорректная дата должна вернуться как есть, получено %q", got)
	}
}

func TestRenderGame_Open(t *testing.T) {
	g := &storage.Game{Date: "2026-11-17", Players: []storage.Player{
		{UserID: 1, Name: "Мария", Status: storage.StatusGoing},
		{UserID: 2, Name: "Иван", Status: storage.StatusGoing, Friends: 1},
		{UserID: 3, Name: "Павел", Status: storage.StatusDeclined},
		{UserID: 4, Name: "Анна", Status: storage.StatusDeclined, Friends: 2},
		{UserID: 5, Name: "Олег", Status: storage.StatusNone, Friends: 1},
	}}

	text, kb := RenderGame(g)

	want := strings.Join([]string{
		"<b>Вторник 17.11 в 19:00</b>",
		"Запись открыта · 6/12",
		"1. Мария",
		"2. Иван",
		"3. друг (Иван)",
		"4. друг (Анна)",
		"5. друг (Анна)",
		"6. друг (Олег)",
		"свободно: 6",
		"Не смогут: Павел, Анна",
	}, "\n")
	if text != want {
		t.Fatalf("текст:\n%s\n\nожидалось:\n%s", text, want)
	}

	wantKb := [][]string{
		{"join:2026-11-17", "friend:2026-11-17"},
		{"decline:2026-11-17", "minus:2026-11-17"},
	}
	if kb == nil || !equalRows(buttons(kb), wantKb) {
		t.Fatalf("кнопки = %v, ожидалось %v", kb, wantKb)
	}
}

func TestRenderGame_Full(t *testing.T) {
	g := gameWith(Limit)
	g.Date = "2026-11-17"

	text, kb := RenderGame(g)

	if !strings.Contains(text, "Состав собран · 12/12") {
		t.Errorf("нет строки о наборе:\n%s", text)
	}
	if strings.Contains(text, "свободно") {
		t.Errorf("при полном составе не нужна строка о свободных местах:\n%s", text)
	}
	wantKb := [][]string{{"decline:2026-11-17", "minus:2026-11-17"}}
	if kb == nil || !equalRows(buttons(kb), wantKb) {
		t.Fatalf("кнопки = %v, ожидалось %v", buttons(kb), wantKb)
	}
}

func TestRenderGame_Finished(t *testing.T) {
	g := &storage.Game{Date: "2026-11-17", Finished: true, Players: []storage.Player{
		{UserID: 1, Name: "Мария", Status: storage.StatusGoing},
		{UserID: 3, Name: "Павел", Status: storage.StatusDeclined},
	}}

	text, kb := RenderGame(g)

	want := "<b>Вторник 17.11 · игра прошла</b>\nБыло человек: 1\n1. Мария"
	if text != want {
		t.Fatalf("текст:\n%s\n\nожидалось:\n%s", text, want)
	}
	if kb != nil {
		t.Fatalf("у прошедшей игры не должно быть кнопок: %v", buttons(kb))
	}
}

func TestRenderGame_Empty(t *testing.T) {
	text, _ := RenderGame(&storage.Game{Date: "2026-11-17"})

	want := "<b>Вторник 17.11 в 19:00</b>\nЗапись открыта · 0/12\nсвободно: 12"
	if text != want {
		t.Fatalf("текст:\n%s\n\nожидалось:\n%s", text, want)
	}
}

func TestRenderGame_EscapesNames(t *testing.T) {
	g := &storage.Game{Date: "2026-11-17", Players: []storage.Player{
		{UserID: 1, Name: "<b>Хакер</b> & Ко", Status: storage.StatusGoing, Friends: 1},
	}}

	text, _ := RenderGame(g)

	if strings.Contains(text, "<b>Хакер</b>") {
		t.Fatalf("имя не экранировано:\n%s", text)
	}
	if !strings.Contains(text, "1. &lt;b&gt;Хакер&lt;/b&gt; &amp; Ко") || !strings.Contains(text, "друг (&lt;b&gt;Хакер") {
		t.Fatalf("ожидалось экранированное имя:\n%s", text)
	}
}
