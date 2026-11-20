package game

import (
	"fmt"
	"html"
	"regexp"
	"slices"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/storage"
)

// ActionPaid это кнопка "Перевёл" под сообщением оплаты.
const ActionPaid = "paid"

// Debtor это человек из списка, который должен перевести плательщику за себя и пришедших друзей.
type Debtor struct {
	UserID int64
	Name   string // уже экранировано для HTML
	People int    // за скольких платит
}

// Debtors возвращает должников по оплате в порядке списка: по каждому человеку считаются его пришедшие строки,
// плательщик не включается.
func Debtors(match *storage.Game) []Debtor {
	payment := match.Payment
	var debtors []Debtor
	for _, line := range Lines(match) {
		if line.OwnerID == payment.RenterID || IsAbsent(payment.Absent, line.ID) {
			continue
		}
		index := slices.IndexFunc(debtors, func(debtor Debtor) bool { return debtor.UserID == line.OwnerID })
		if index < 0 {
			debtors = append(debtors, Debtor{UserID: line.OwnerID, Name: playerName(match, line.OwnerID)})
			index = len(debtors) - 1
		}
		debtors[index].People++
	}
	return debtors
}

// Newcomers возвращает, сколько пришло людей не из списка.
func Newcomers(match *storage.Game) int {
	present := 0
	for _, line := range Lines(match) {
		if !IsAbsent(match.Payment.Absent, line.ID) {
			present++
		}
	}
	return max(0, match.Payment.Came-present)
}

// PaidCount возвращает, сколько должников отметили перевод.
func PaidCount(match *storage.Game, debtors []Debtor) int {
	count := 0
	for _, debtor := range debtors {
		if slices.Contains(match.Payment.PaidUserIDs, debtor.UserID) {
			count++
		}
	}
	return count
}

// RenderPayment возвращает текст сообщения оплаты в HTML и кнопку "Перевёл". Должники упоминаются ссылкой tg://user,
// Telegram присылает им уведомление.
func RenderPayment(match *storage.Game) (string, *tgbotapi.InlineKeyboardMarkup) {
	payment := match.Payment
	debtors := Debtors(match)

	var text strings.Builder
	fmt.Fprintf(&text, "<b>%s</b>: зал оплатил %s\n\n", DayTitle(match.Date), html.EscapeString(payment.RenterName))
	fmt.Fprintf(&text, "Было %d, по <b>%d ₽</b> с человека\n", payment.Came, payment.PerPerson)
	fmt.Fprintf(&text, "Перевод на %s\n", RenderRequisites(payment.Requisites))

	fmt.Fprintf(&text, "\n<b>Перевели %d из %d</b>\n", PaidCount(match, debtors), len(debtors))
	for _, debtor := range debtors {
		mark := "⏳"
		if slices.Contains(payment.PaidUserIDs, debtor.UserID) {
			mark = "✅"
		}
		fmt.Fprintf(&text, `%s <a href="tg://user?id=%d">%s</a> %d ₽`, mark, debtor.UserID, debtor.Name,
			debtor.People*payment.PerPerson)
		if debtor.People > 1 {
			fmt.Fprintf(&text, " (за %d)", debtor.People)
		}
		text.WriteString("\n")
	}
	if newcomers := Newcomers(match); newcomers > 0 {
		fmt.Fprintf(&text, "Вне списка: %d × %d ₽, переводят сами\n", newcomers, payment.PerPerson)
	}
	text.WriteString("\nПеревел? Нажми кнопку ниже, чтобы отметиться.")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✅ Перевёл", ActionPaid+":"+match.Date),
	))
	return strings.TrimRight(text.String(), "\n"), &keyboard
}

// phonePattern находит российский номер телефона в реквизитах: +7 или 8 и еще 10 цифр с пробелами, дефисами и скобками
// между ними. Первая группа это символ перед номером, вторая сам номер. Соседние цифры не допускаются, чтобы не
// захватить часть номера карты.
var phonePattern = regexp.MustCompile(`(^|[^\d+])((?:\+7|8)[\s\-()]*\d(?:[\s\-()]*\d){9})(?:$|[^\d])`)

// RenderRequisites возвращает реквизиты в HTML. Телефон приводится к виду +79001234567 и оборачивается в <code>: в
// мобильном Telegram такой текст копируется нажатием. Если телефона нет, в <code> оборачивается вся строка.
func RenderRequisites(requisites string) string {
	match := phonePattern.FindStringSubmatchIndex(requisites)
	if match == nil {
		return "<code>" + html.EscapeString(requisites) + "</code>"
	}
	start, end := match[4], match[5]
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, requisites[start:end])
	phone := "+7" + digits[1:]
	return html.EscapeString(requisites[:start]) + "<code>" + phone + "</code>" + html.EscapeString(requisites[end:])
}

// MarkPaid ставит отметку перевода и возвращает сумму человека. Снять отметку нельзя: повторное нажатие возвращает
// ошибку с текстом для окна.
func MarkPaid(match *storage.Game, userID int64) (amount int, err error) {
	if !IsPaid(match) {
		return 0, fmt.Errorf("Пока никто не оформил оплату зала")
	}
	payment := match.Payment
	if userID == payment.RenterID {
		return 0, fmt.Errorf("Ты платил за зал, тебе переводить не нужно")
	}
	debtors := Debtors(match)
	index := slices.IndexFunc(debtors, func(debtor Debtor) bool { return debtor.UserID == userID })
	if index < 0 {
		return 0, fmt.Errorf("Тебя нет среди должников по этой игре")
	}
	amount = debtors[index].People * payment.PerPerson
	if slices.Contains(payment.PaidUserIDs, userID) {
		return 0, fmt.Errorf("Ты уже отметил перевод %d ₽", amount)
	}
	payment.PaidUserIDs = append(payment.PaidUserIDs, userID)
	return amount, nil
}

// playerName возвращает экранированное имя игрока по ID.
func playerName(match *storage.Game, userID int64) string {
	for _, player := range match.Players {
		if player.UserID == userID {
			return html.EscapeString(player.Name)
		}
	}
	return ""
}
