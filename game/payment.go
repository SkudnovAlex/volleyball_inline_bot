package game

import (
	"errors"
	"fmt"
	"html"
	"slices"
	"strconv"
	"strings"

	"volleyball_inline_bot/storage"
)

// Границы ввода при оформлении оплаты.
const (
	MinCame  = 1
	MaxCame  = 40
	MinTotal = 1000
	MaxTotal = 50000
	// roundTo задаёт шаг округления доли вверх, в рублях.
	roundTo = 10
)

// ActionHall это кнопка «Я оплатил зал» в сообщении прошедшей игры.
const ActionHall = "hall"

// Ошибки ввода в личке. Текст отправляется пользователю.
var (
	ErrBadCame   = fmt.Errorf("Пришли число от %d до %d", MinCame, MaxCame)
	ErrBadAmount = errors.New("Не понял сумму. Напиши её в конце строки числом от 1000 до 50 000, например: Т-Банк +7 900 123-45-67 5400")
)

// Line это строка состава: сам человек или один из его друзей.
type Line struct {
	ID      string // "<user_id>" или "<user_id>#<n>" для n-го друга
	Label   string // текст для показа, уже экранирован для HTML
	OwnerID int64  // кто записался или привёл друга
}

// Lines возвращает состав игры: человек, затем его друзья.
func Lines(match *storage.Game) []Line {
	var lines []Line
	for _, player := range match.Players {
		userID := strconv.FormatInt(player.UserID, 10)
		name := html.EscapeString(player.Name)
		if player.Status == storage.StatusGoing {
			lines = append(lines, Line{ID: userID, Label: name, OwnerID: player.UserID})
		}
		for friend := 1; friend <= player.Friends; friend++ {
			lines = append(lines, Line{
				ID:      fmt.Sprintf("%s#%d", userID, friend),
				Label:   fmt.Sprintf("друг (%s)", name),
				OwnerID: player.UserID,
			})
		}
	}
	return lines
}

// IsPaid сообщает, что плательщик уже отправил оплату в группу.
func IsPaid(match *storage.Game) bool {
	return match.Payment != nil && match.Payment.RenterID != 0
}

// Attended возвращает, сколько человек было на игре: после оплаты со слов
// плательщика, до неё по списку.
func Attended(match *storage.Game) int {
	if IsPaid(match) {
		return match.Payment.Came
	}
	return Count(match)
}

// ParseCame разбирает ответ на вопрос «сколько человек было».
func ParseCame(text string) (int, error) {
	came, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || came < MinCame || came > MaxCame {
		return 0, ErrBadCame
	}
	return came, nil
}

// ParseAmount разбирает строку «<реквизиты> <сумма>». Сумма это последнее
// слово строки, допустим суффикс «₽» или «р». Реквизиты могут быть пустыми,
// если прислана одна сумма.
func ParseAmount(text string) (requisites string, total int, err error) {
	text = strings.TrimSpace(text)
	cut := strings.LastIndexAny(text, " \t\n")
	last := text[cut+1:]
	last = strings.TrimSuffix(strings.TrimSuffix(last, "₽"), "р")

	total, err = strconv.Atoi(last)
	if err != nil || total < MinTotal || total > MaxTotal {
		return "", 0, ErrBadAmount
	}
	if cut < 0 {
		return "", total, nil
	}
	return strings.TrimSpace(text[:cut]), total, nil
}

// PerPerson делит сумму на пришедших с округлением вверх до roundTo рублей.
func PerPerson(total, came int) int {
	perPerson := (total + came - 1) / came
	return (perPerson + roundTo - 1) / roundTo * roundTo
}

// IsAbsent сообщает, отмечена ли строка состава как неявка.
func IsAbsent(absent []string, lineID string) bool {
	return slices.Contains(absent, lineID)
}

// ToggleAbsent добавляет строку в отметки неявки или убирает её.
func ToggleAbsent(absent []string, lineID string) []string {
	if index := slices.Index(absent, lineID); index >= 0 {
		return slices.Delete(absent, index, index+1)
	}
	return append(absent, lineID)
}
