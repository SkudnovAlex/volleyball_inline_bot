package game

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"volleyball_inline_bot/storage"
)

// RenderBoard возвращает текст доски в HTML: ближайшие игры по дате,
// ниже прошедшие, новые сверху. Дата игры это ссылка на её сообщение.
func RenderBoard(games []storage.Game, chatID int64) string {
	var upcoming, finished []storage.Game
	for _, match := range games {
		if match.Finished {
			finished = append(finished, match)
		} else {
			upcoming = append(upcoming, match)
		}
	}
	sort.Slice(upcoming, func(left, right int) bool { return upcoming[left].Date < upcoming[right].Date })
	sort.Slice(finished, func(left, right int) bool { return finished[left].Date > finished[right].Date })

	var text strings.Builder
	text.WriteString("<b>Расписание игр</b>\n")
	if len(upcoming) == 0 {
		text.WriteString("Ближайших игр пока нет\n")
	}
	for _, match := range upcoming {
		count := Count(&match)
		status := fmt.Sprintf("%d/%d", count, Limit)
		if count >= Limit {
			status += " собран"
		}
		fmt.Fprintf(&text, "%s · %s\n", gameLink(match, chatID), status)
	}

	if len(finished) > 0 {
		text.WriteString("\n<b>Прошедшие</b>\n")
		for _, match := range finished {
			fmt.Fprintf(&text, "%s · %s\n", gameLink(match, chatID), finishedStatus(&match))
		}
	}
	return strings.TrimRight(text.String(), "\n")
}

// finishedStatus описывает прошедшую игру на доске: после оплаты доля
// и сколько перевели, до неё сколько было.
func finishedStatus(match *storage.Game) string {
	if !IsPaid(match) {
		return fmt.Sprintf("было %d, зал не оплачен", Attended(match))
	}
	debtors := Debtors(match)
	return fmt.Sprintf("по %d ₽ · перевели %d из %d", match.Payment.PerPerson, PaidCount(match, debtors), len(debtors))
}

// gameLink возвращает дату игры ссылкой на её сообщение. Ссылки вида
// t.me/c/... работают только в супергруппах (ID начинается с -100),
// в обычной группе дата выводится без ссылки.
func gameLink(match storage.Game, chatID int64) string {
	title := DayTitle(match.Date)
	internalID, isSupergroup := strings.CutPrefix(strconv.FormatInt(chatID, 10), "-100")
	if !isSupergroup || match.MessageID == 0 {
		return title
	}
	return fmt.Sprintf(`<a href="https://t.me/c/%s/%d">%s</a>`, internalID, match.MessageID, title)
}
