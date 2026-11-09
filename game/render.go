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
// У прошедшей игры одна кнопка «Я оплатил зал».
func RenderGame(game *storage.Game) (string, *tgbotapi.InlineKeyboardMarkup) {
	var text strings.Builder
	count := Count(game)

	switch {
	case game.Finished:
		fmt.Fprintf(&text, "<b>%s · игра прошла</b>\n", DayTitle(game.Date))
		fmt.Fprintf(&text, "Было человек: %d\n", Attended(game))
	case count >= Limit:
		fmt.Fprintf(&text, "<b>%s в %d:00</b>\n", DayTitle(game.Date), GameHour)
		fmt.Fprintf(&text, "Состав собран · %d/%d\n", count, Limit)
	default:
		fmt.Fprintf(&text, "<b>%s в %d:00</b>\n", DayTitle(game.Date), GameHour)
		fmt.Fprintf(&text, "Запись открыта · %d/%d\n", count, Limit)
	}

	var absent []string
	if IsPaid(game) {
		absent = game.Payment.Absent
	}
	for index, line := range Lines(game) {
		mark := ""
		if IsAbsent(absent, line.ID) {
			mark = " · не пришёл"
		}
		fmt.Fprintf(&text, "%d. %s%s\n", index+1, line.Label, mark)
	}
	if !game.Finished && count < Limit {
		fmt.Fprintf(&text, "свободно: %d\n", Limit-count)
	}

	var declined []string
	for _, player := range game.Players {
		if player.Status == storage.StatusDeclined {
			declined = append(declined, html.EscapeString(player.Name))
		}
	}
	if len(declined) > 0 && !game.Finished {
		fmt.Fprintf(&text, "Не смогут: %s\n", strings.Join(declined, ", "))
	}

	return strings.TrimRight(text.String(), "\n"), buildKeyboard(game, count)
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
	if count >= Limit {
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
