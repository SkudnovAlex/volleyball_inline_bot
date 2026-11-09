package game

import (
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/storage"
)

// buttons возвращает callback_data всех кнопок построчно.
func buttons(keyboard *tgbotapi.InlineKeyboardMarkup) [][]string {
	var rows [][]string
	for _, buttonsRow := range keyboard.InlineKeyboard {
		var row []string
		for _, button := range buttonsRow {
			row = append(row, *button.CallbackData)
		}
		rows = append(rows, row)
	}
	return rows
}

func equalRows(got, want [][]string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if strings.Join(got[index], ",") != strings.Join(want[index], ",") {
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
	game := &storage.Game{Date: "2026-11-17", Players: []storage.Player{
		{UserID: 1, Name: "Мария", Status: storage.StatusGoing},
		{UserID: 2, Name: "Иван", Status: storage.StatusGoing, Friends: 1},
		{UserID: 3, Name: "Павел", Status: storage.StatusDeclined},
		{UserID: 4, Name: "Анна", Status: storage.StatusDeclined, Friends: 2},
		{UserID: 5, Name: "Олег", Status: storage.StatusNone, Friends: 1},
	}}

	text, keyboard := RenderGame(game)

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
	if keyboard == nil || !equalRows(buttons(keyboard), wantKb) {
		t.Fatalf("кнопки = %v, ожидалось %v", keyboard, wantKb)
	}
}

func TestRenderGame_Full(t *testing.T) {
	game := gameWith(Limit)
	game.Date = "2026-11-17"

	text, keyboard := RenderGame(game)

	if !strings.Contains(text, "Состав собран · 12/12") {
		t.Errorf("нет строки о наборе:\n%s", text)
	}
	if strings.Contains(text, "свободно") {
		t.Errorf("при полном составе не нужна строка о свободных местах:\n%s", text)
	}
	wantKb := [][]string{{"decline:2026-11-17", "minus:2026-11-17"}}
	if keyboard == nil || !equalRows(buttons(keyboard), wantKb) {
		t.Fatalf("кнопки = %v, ожидалось %v", buttons(keyboard), wantKb)
	}
}

func TestRenderGame_Finished(t *testing.T) {
	game := &storage.Game{Date: "2026-11-17", Finished: true, Players: []storage.Player{
		{UserID: 1, Name: "Мария", Status: storage.StatusGoing},
		{UserID: 3, Name: "Павел", Status: storage.StatusDeclined},
	}}

	text, keyboard := RenderGame(game)

	want := "<b>Вторник 17.11 · игра прошла</b>\nБыло человек: 1\n1. Мария"
	if text != want {
		t.Fatalf("текст:\n%s\n\nожидалось:\n%s", text, want)
	}
	wantKeyboard := [][]string{{"hall:2026-11-17"}}
	if keyboard == nil || !equalRows(buttons(keyboard), wantKeyboard) {
		t.Fatalf("у прошедшей игры одна кнопка оплаты: %v", keyboard)
	}
}

func TestRenderGame_FinishedPaidMarksAbsent(t *testing.T) {
	game := &storage.Game{Date: "2026-11-17", Finished: true,
		Players: []storage.Player{
			{UserID: 1, Name: "Мария", Status: storage.StatusGoing},
			{UserID: 2, Name: "Сергей", Status: storage.StatusGoing, Friends: 1},
		},
		Payment: &storage.Payment{RenterID: 1, Came: 9, Absent: []string{"2"}},
	}

	text, _ := RenderGame(game)

	want := strings.Join([]string{
		"<b>Вторник 17.11 · игра прошла</b>",
		"Было человек: 9",
		"1. Мария",
		"2. Сергей · не пришёл",
		"3. друг (Сергей)",
	}, "\n")
	if text != want {
		t.Fatalf("текст:\n%s\n\nожидалось:\n%s", text, want)
	}
}

func TestRenderGame_DraftDoesNotMarkAbsent(t *testing.T) {
	game := &storage.Game{Date: "2026-11-17", Finished: true,
		Players: []storage.Player{{UserID: 2, Name: "Сергей", Status: storage.StatusGoing}},
		Payment: &storage.Payment{Draft: &storage.Draft{UserID: 1, Absent: []string{"2"}}},
	}

	text, _ := RenderGame(game)

	if strings.Contains(text, "не пришёл") || !strings.Contains(text, "Было человек: 1") {
		t.Fatalf("пока оплата не отправлена, черновик в группе не виден:\n%s", text)
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
	game := &storage.Game{Date: "2026-11-17", Players: []storage.Player{
		{UserID: 1, Name: "<b>Хакер</b> & Ко", Status: storage.StatusGoing, Friends: 1},
	}}

	text, _ := RenderGame(game)

	if strings.Contains(text, "<b>Хакер</b>") {
		t.Fatalf("имя не экранировано:\n%s", text)
	}
	if !strings.Contains(text, "1. &lt;b&gt;Хакер&lt;/b&gt; &amp; Ко") || !strings.Contains(text, "друг (&lt;b&gt;Хакер") {
		t.Fatalf("ожидалось экранированное имя:\n%s", text)
	}
}
