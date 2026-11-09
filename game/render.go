package game

import (
	"fmt"
	"html"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/storage"
)

// DateLayout задаёт формат storage.Game.Date.
const DateLayout = "2006-01-02"

// Действия кнопок. В callback_data передаётся "<действие>:<дата>".
const (
	ActionJoin    = "join"
	ActionFriend  = "friend"
	ActionDecline = "decline"
	ActionMinus   = "minus"
)

var weekdays = map[time.Weekday]string{
	time.Monday:    "Понедельник",
	time.Tuesday:   "Вторник",
	time.Wednesday: "Среда",
	time.Thursday:  "Четверг",
	time.Friday:    "Пятница",
	time.Saturday:  "Суббота",
	time.Sunday:    "Воскресенье",
}

// Символы полосы заполнения: занятое и свободное место.
const (
	barTaken = "▰"
	barFree  = "▱"
)

// RenderGame возвращает текст сообщения игры в HTML и клавиатуру. У прошедшей игры одна кнопка "Я оплатил зал".
func RenderGame(game *storage.Game) (string, *tgbotapi.InlineKeyboardMarkup) {
	var text strings.Builder
	count := Count(game)

	if game.Finished {
		fmt.Fprintf(&text, "🏐 <b>%s · игра прошла</b>\n", DayTitle(game.Date))
		fmt.Fprintf(&text, "Было человек: %d\n", Attended(game))
	} else {
		fmt.Fprintf(&text, "🏐 <b>%s · %s</b>\n", DayTitle(game.Date), rules.GameTime)
		status := fmt.Sprintf("свободно %d", rules.Limit-count)
		if count >= rules.Limit {
			status = "состав собран"
		}
		fmt.Fprintf(&text, "%s %d/%d · %s\n", progressBar(count), count, rules.Limit, status)
	}

	if lineup := lineupLines(game); len(lineup) > 0 {
		fmt.Fprintf(&text, "\n%s\n", strings.Join(lineup, "\n"))
	}

	var declined []string
	for _, player := range game.Players {
		if player.Status == storage.StatusDeclined {
			declined = append(declined, html.EscapeString(player.Name))
		}
	}
	if len(declined) > 0 && !game.Finished {
		fmt.Fprintf(&text, "\n❌ Не смогут: %s\n", strings.Join(declined, ", "))
	}

	return strings.TrimRight(text.String(), "\n"), buildKeyboard(game, count)
}

// lineupLines возвращает пронумерованный состав. Друг того, кто идёт сам, выводится под ним как "└ друг", иначе с именем
// приведшего.
func lineupLines(game *storage.Game) []string {
	var absent []string
	if IsPaid(game) {
		absent = game.Payment.Absent
	}
	var lines []string
	// Строки одного человека идут подряд: сначала он сам, затем друзья.
	goingOwner := int64(0)
	for index, line := range Lines(game) {
		isFriend := strings.Contains(line.ID, "#")
		label := line.Label
		if !isFriend {
			goingOwner = line.OwnerID
		} else if line.OwnerID == goingOwner {
			label = "  └ друг"
		}
		if IsAbsent(absent, line.ID) {
			label += " · не пришёл"
		}
		lines = append(lines, fmt.Sprintf("%d. %s", index+1, label))
	}
	return lines
}

// progressBar рисует заполнение состава из rules.Limit делений.
func progressBar(count int) string {
	taken := min(count, rules.Limit)
	return strings.Repeat(barTaken, taken) + strings.Repeat(barFree, rules.Limit-taken)
}

// DayTitle превращает "2026-11-18" в "Вторник 18.11".
func DayTitle(date string) string {
	day, err := time.Parse(DateLayout, date)
	if err != nil {
		return date
	}
	return fmt.Sprintf("%s %s", weekdays[day.Weekday()], day.Format("02.01"))
}

func buildKeyboard(game *storage.Game, count int) *tgbotapi.InlineKeyboardMarkup {
	button := func(text, action string) tgbotapi.InlineKeyboardButton {
		return tgbotapi.NewInlineKeyboardButtonData(text, action+":"+game.Date)
	}
	if game.Finished {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(button("💳 Я оплатил зал", ActionHall)),
		)
		return &keyboard
	}
	leave := tgbotapi.NewInlineKeyboardRow(
		button("❌ Не смогу", ActionDecline),
		button("➖ Минус друг", ActionMinus),
	)

	var keyboard tgbotapi.InlineKeyboardMarkup
	if count >= rules.Limit {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(leave)
	} else {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				button("✅ Иду", ActionJoin),
				button("➕ Друг", ActionFriend),
			),
			leave,
		)
	}
	return &keyboard
}
