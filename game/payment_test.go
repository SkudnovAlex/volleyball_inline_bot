package game

import (
	"errors"
	"testing"

	"volleyball_inline_bot/storage"
)

func TestLines(t *testing.T) {
	game := &storage.Game{Players: []storage.Player{
		{UserID: 1, Name: "Мария", Status: storage.StatusGoing},
		{UserID: 2, Name: "Анна", Status: storage.StatusDeclined, Friends: 2},
		{UserID: 3, Name: "<Олег>", Status: storage.StatusGoing, Friends: 1},
	}}

	want := []Line{
		{ID: "1", Label: "Мария", OwnerID: 1},
		{ID: "2#1", Label: "друг (Анна)", OwnerID: 2},
		{ID: "2#2", Label: "друг (Анна)", OwnerID: 2},
		{ID: "3", Label: "&lt;Олег&gt;", OwnerID: 3},
		{ID: "3#1", Label: "друг (&lt;Олег&gt;)", OwnerID: 3},
	}
	got := Lines(game)
	if len(got) != len(want) {
		t.Fatalf("Lines = %+v, ожидалось %+v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("строка %d = %+v, ожидалось %+v", index, got[index], want[index])
		}
	}
}

func TestParseCame(t *testing.T) {
	for _, text := range []string{"10", " 1 ", "30"} {
		if _, err := ParseCame(text); err != nil {
			t.Errorf("ParseCame(%q): неожиданная ошибка %v", text, err)
		}
	}
	for _, text := range []string{"0", "31", "десять", "", "10 человек"} {
		if _, err := ParseCame(text); !errors.Is(err, ErrBadCame) {
			t.Errorf("ParseCame(%q): ожидалась ErrBadCame, получено %v", text, err)
		}
	}
}

func TestParseAmount(t *testing.T) {
	cases := []struct {
		text       string
		requisites string
		total      int
	}{
		{"Т-Банк +7 900 123-45-67 5400", "Т-Банк +7 900 123-45-67", 5400},
		{"Сбер по номеру 89001234567 5400₽", "Сбер по номеру 89001234567", 5400},
		{"Сбер 89001234567 5400р", "Сбер 89001234567", 5400},
		{"  5400  ", "", 5400},
		{"Т-Банк\n+7 900 123-45-67\n5400", "Т-Банк\n+7 900 123-45-67", 5400},
	}
	for _, testCase := range cases {
		requisites, total, err := ParseAmount(testCase.text)
		if err != nil || requisites != testCase.requisites || total != testCase.total {
			t.Errorf("ParseAmount(%q) = %q, %d, %v; ожидалось %q, %d",
				testCase.text, requisites, total, err, testCase.requisites, testCase.total)
		}
	}

	for _, text := range []string{"Т-Банк 5 400", "Т-Банк +7 900 123-45-67", "999", "10001", "", "Т-Банк"} {
		if _, _, err := ParseAmount(text); !errors.Is(err, ErrBadAmount) {
			t.Errorf("ParseAmount(%q): ожидалась ErrBadAmount, получено %v", text, err)
		}
	}
}

func TestPerPerson(t *testing.T) {
	cases := []struct{ total, came, want int }{
		{5400, 10, 540},
		{5400, 12, 450},
		{5400, 11, 500}, // 490,9 → 491 → 500
		{5000, 3, 1670}, // 1666,7 → 1667 → 1670
		{1000, 1, 1000},
	}
	for _, testCase := range cases {
		if got := PerPerson(testCase.total, testCase.came); got != testCase.want {
			t.Errorf("PerPerson(%d, %d) = %d, ожидалось %d", testCase.total, testCase.came, got, testCase.want)
		}
	}
}

func TestToggleAbsent(t *testing.T) {
	absent := ToggleAbsent(nil, "1")
	absent = ToggleAbsent(absent, "2#1")
	if !IsAbsent(absent, "1") || !IsAbsent(absent, "2#1") {
		t.Fatalf("обе строки должны быть отмечены: %v", absent)
	}
	absent = ToggleAbsent(absent, "1")
	if IsAbsent(absent, "1") || len(absent) != 1 {
		t.Fatalf("повторное нажатие снимает отметку: %v", absent)
	}
}

func TestAttended(t *testing.T) {
	game := gameWith(5)
	if Attended(game) != 5 {
		t.Errorf("до оплаты считается по списку")
	}
	game.Payment = &storage.Payment{Draft: &storage.Draft{Came: 3}}
	if Attended(game) != 5 {
		t.Errorf("черновик не влияет на «было»")
	}
	game.Payment = &storage.Payment{RenterID: 1, Came: 7}
	if Attended(game) != 7 {
		t.Errorf("после оплаты берётся число плательщика")
	}
}
