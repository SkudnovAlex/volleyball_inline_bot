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
func RenderGame(game *storage.Game) (string, *tgbotapi.InlineKeyboardMarkup) {
	var text strings.Builder
	count := Count(game)

	switch {
	case game.Finished:
		fmt.Fprintf(&text, "<b>%s · игра прошла</b>\n", DayTitle(game.Date))
		fmt.Fprintf(&text, "Было человек: %d\n", count)
	case count >= Limit:
		fmt.Fprintf(&text, "<b>%s в %d:00</b>\n", DayTitle(game.Date), GameHour)
		fmt.Fprintf(&text, "Состав собран · %d/%d\n", count, Limit)
	default:
		fmt.Fprintf(&text, "<b>%s в %d:00</b>\n", DayTitle(game.Date), GameHour)
		fmt.Fprintf(&text, "Запись открыта · %d/%d\n", count, Limit)
	}

	for index, line := range lineup(game) {
		fmt.Fprintf(&text, "%d. %s\n", index+1, line)
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

// lineup возвращает строки состава: человек, затем его друзья.
func lineup(game *storage.Game) []string {
	var lines []string
	for _, player := range game.Players {
		name := html.EscapeString(player.Name)
		if player.Status == storage.StatusGoing {
			lines = append(lines, name)
		}
		for range player.Friends {
			lines = append(lines, fmt.Sprintf("друг (%s)", name))
		}
	}
	return lines
}

func buildKeyboard(game *storage.Game, count int) *tgbotapi.InlineKeyboardMarkup {
	if game.Finished {
		return nil
	}
	button := func(text, action string) tgbotapi.InlineKeyboardButton {
		return tgbotapi.NewInlineKeyboardButtonData(text, action+":"+game.Date)
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
