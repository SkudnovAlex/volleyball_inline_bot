package game

import (
	"fmt"
	"html"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/storage"
)

// GameHour задаёт час начала игры.
const GameHour = 19

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

// RenderGame возвращает текст сообщения игры в HTML и клавиатуру.
// Для прошедшей игры клавиатура nil: при редактировании кнопки исчезнут.
func RenderGame(g *storage.Game) (string, *tgbotapi.InlineKeyboardMarkup) {
	var b strings.Builder
	count := Count(g)

	switch {
	case g.Finished:
		fmt.Fprintf(&b, "<b>%s · игра прошла</b>\n", DayTitle(g.Date))
		fmt.Fprintf(&b, "Было человек: %d\n", count)
	case count >= Limit:
		fmt.Fprintf(&b, "<b>%s в %d:00</b>\n", DayTitle(g.Date), GameHour)
		fmt.Fprintf(&b, "Состав собран · %d/%d\n", count, Limit)
	default:
		fmt.Fprintf(&b, "<b>%s в %d:00</b>\n", DayTitle(g.Date), GameHour)
		fmt.Fprintf(&b, "Запись открыта · %d/%d\n", count, Limit)
	}

	for i, line := range lineup(g) {
		fmt.Fprintf(&b, "%d. %s\n", i+1, line)
	}
	if !g.Finished && count < Limit {
		fmt.Fprintf(&b, "свободно: %d\n", Limit-count)
	}

	var declined []string
	for _, p := range g.Players {
		if p.Status == storage.StatusDeclined {
			declined = append(declined, html.EscapeString(p.Name))
		}
	}
	if len(declined) > 0 && !g.Finished {
		fmt.Fprintf(&b, "Не смогут: %s\n", strings.Join(declined, ", "))
	}

	return strings.TrimRight(b.String(), "\n"), keyboard(g, count)
}

// DayTitle превращает "2026-11-18" в "Вторник 18.11".
func DayTitle(date string) string {
	d, err := time.Parse(DateLayout, date)
	if err != nil {
		return date
	}
	return fmt.Sprintf("%s %s", weekdays[d.Weekday()], d.Format("02.01"))
}

// lineup возвращает строки состава: человек, затем его друзья.
func lineup(g *storage.Game) []string {
	var lines []string
	for _, p := range g.Players {
		name := html.EscapeString(p.Name)
		if p.Status == storage.StatusGoing {
			lines = append(lines, name)
		}
		for i := 0; i < p.Friends; i++ {
			lines = append(lines, fmt.Sprintf("друг (%s)", name))
		}
	}
	return lines
}

func keyboard(g *storage.Game, count int) *tgbotapi.InlineKeyboardMarkup {
	if g.Finished {
		return nil
	}
	btn := func(text, action string) tgbotapi.InlineKeyboardButton {
		return tgbotapi.NewInlineKeyboardButtonData(text, action+":"+g.Date)
	}
	leave := tgbotapi.NewInlineKeyboardRow(
		btn("❌ Не смогу", ActionDecline),
		btn("➖ Минус друг", ActionMinus),
	)

	var kb tgbotapi.InlineKeyboardMarkup
	if count >= Limit {
		kb = tgbotapi.NewInlineKeyboardMarkup(leave)
	} else {
		kb = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				btn("✅ Иду", ActionJoin),
				btn("➕ Друг", ActionFriend),
			),
			leave,
		)
	}
	return &kb
}
