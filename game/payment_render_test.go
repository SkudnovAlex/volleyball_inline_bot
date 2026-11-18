package game

import (
	"strings"
	"testing"

	"volleyball_inline_bot/storage"
)

// paidGame: Иван с другом оплатил 5400 ₽ на 9 человек. Сергей не пришёл, но его друг был, Дмитрий привёл двоих, Мария
// уже перевела, двое новеньких.
func paidGame() *storage.Game {
	return &storage.Game{Date: "2026-11-09", MessageID: 50, Finished: true,
		Players: []storage.Player{
			{UserID: 1, Name: "Мария", Status: storage.StatusGoing},
			{UserID: 2, Name: "Сергей", Status: storage.StatusGoing, Friends: 1},
			{UserID: 3, Name: "Дмитрий", Status: storage.StatusGoing, Friends: 2},
			{UserID: 4, Name: "Анна", Status: storage.StatusGoing},
			{UserID: 7, Name: "Иван", Status: storage.StatusGoing, Friends: 1},
		},
		Payment: &storage.Payment{
			RenterID: 7, RenterName: "Иван", Requisites: "Т-Банк +7 900 123-45-67",
			Came: 9, Absent: []string{"2", "4"}, Total: 5400, PerPerson: 540,
			PaidUserIDs: []int64{1},
		},
	}
}

func TestDebtors(t *testing.T) {
	want := []Debtor{
		{UserID: 1, Name: "Мария", People: 1},
		{UserID: 2, Name: "Сергей", People: 1},
		{UserID: 3, Name: "Дмитрий", People: 3},
	}
	got := Debtors(paidGame())
	if len(got) != len(want) {
		t.Fatalf("Debtors = %+v, ожидалось %+v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("должник %d = %+v, ожидалось %+v", index, got[index], want[index])
		}
	}
}

func TestRenderPayment(t *testing.T) {
	text, keyboard := RenderPayment(paidGame())

	want := strings.Join([]string{
		"<b>Понедельник 09.11</b>: зал оплатил Иван",
		"",
		"Было 9, по <b>540 ₽</b> с человека",
		"Перевод на Т-Банк <code>+79001234567</code>",
		"",
		"<b>Перевели 1 из 3</b>",
		`✅ <a href="tg://user?id=1">Мария</a> 540 ₽`,
		`⏳ <a href="tg://user?id=2">Сергей</a> 540 ₽`,
		`⏳ <a href="tg://user?id=3">Дмитрий</a> 1620 ₽ (за 3)`,
		"Новенькие: 2 × 540 ₽, переводят сами",
		"",
		"Перевел? Нажми кнопку ниже, чтобы отметиться.",
	}, "\n")
	if text != want {
		t.Fatalf("текст:\n%s\n\nожидалось:\n%s", text, want)
	}
	if keyboard == nil || !equalRows(buttons(keyboard), [][]string{{"paid:2026-11-09"}}) {
		t.Fatalf("ожидалась одна кнопка «Перевёл»: %v", keyboard)
	}
}

func TestRenderPayment_NoNewcomers(t *testing.T) {
	match := paidGame()
	match.Payment.Came = 7

	text, _ := RenderPayment(match)

	if strings.Contains(text, "Новенькие") {
		t.Fatalf("новеньких нет, строки быть не должно:\n%s", text)
	}
}

func TestRenderRequisites(t *testing.T) {
	for _, testCase := range []struct {
		requisites string
		want       string
	}{
		{"Т-Банк +7 900 123-45-67", "Т-Банк <code>+79001234567</code>"},
		{"Сбер 8 (900) 123 45 67 Иван", "Сбер <code>+79001234567</code> Иван"},
		{"+79001234567", "<code>+79001234567</code>"},
		{"89001234567, Альфа", "<code>+79001234567</code>, Альфа"},
		{"Карта 2200 1234 5678 9012", "<code>Карта 2200 1234 5678 9012</code>"},
		{"Сбер 8900", "<code>Сбер 8900</code>"},
		{"Т&Б 8-900-123-45-67", "Т&amp;Б <code>+79001234567</code>"},
	} {
		if got := RenderRequisites(testCase.requisites); got != testCase.want {
			t.Errorf("%q: получено %q, ожидалось %q", testCase.requisites, got, testCase.want)
		}
	}
}

func TestMarkPaid(t *testing.T) {
	match := paidGame()

	amount, err := MarkPaid(match, 3)
	if err != nil || amount != 1620 {
		t.Fatalf("Дмитрий переводит за троих: amount=%d err=%v", amount, err)
	}
	if _, err := MarkPaid(match, 3); err == nil || err.Error() != "Ты уже отметил перевод 1620 ₽" {
		t.Fatalf("повторное нажатие отметку не снимает: %v", err)
	}
	if paid := match.Payment.PaidUserIDs; len(paid) != 2 || paid[1] != 3 {
		t.Fatalf("отметки Марии и Дмитрия остаются: %v", paid)
	}

	for _, testCase := range []struct {
		userID int64
		want   string
	}{
		{7, "Ты платил за зал, тебе переводить не нужно"},
		{4, "Тебя нет среди должников по этой игре"},
		{99, "Тебя нет среди должников по этой игре"},
	} {
		if _, err := MarkPaid(match, testCase.userID); err == nil || err.Error() != testCase.want {
			t.Errorf("пользователь %d: ожидалось «%s», получено %v", testCase.userID, testCase.want, err)
		}
	}

	unpaid := paidGame()
	unpaid.Payment = &storage.Payment{Draft: &storage.Draft{UserID: 7}}
	if _, err := MarkPaid(unpaid, 1); err == nil {
		t.Errorf("до оформления оплаты отметить перевод нельзя")
	}
}
