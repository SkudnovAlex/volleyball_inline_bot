package game

import (
	"errors"
	"fmt"

	"volleyball_inline_bot/storage"
)

const (
	// Limit задаёт, сколько человек нужно на игру.
	Limit = 12
	// MaxFriends задаёт, сколько друзей может привести один человек.
	MaxFriends = 4
)

// Ошибки действий. Текст показывается пользователю во всплывающем окне.
var (
	ErrAlreadyGoing   = errors.New("Ты уже в списке")
	ErrAlreadyDecline = errors.New("Уже отмечено, что ты не сможешь")
	ErrNoPlaces       = errors.New("Мест нет")
	ErrTooManyFriends = fmt.Errorf("Можно привести не больше %d друзей", MaxFriends)
	ErrNoFriends      = errors.New("У тебя нет записанных друзей")
)

// User описывает того, кто нажал кнопку.
type User struct {
	ID   int64
	Name string
}

// Event описывает переход счёта через Limit.
type Event int

const (
	EventNone   Event = iota
	EventFilled       // набралось Limit
	EventOpened       // было Limit, стало меньше
)

// Count возвращает число идущих: записавшиеся сами плюс все друзья.
func Count(g *storage.Game) int {
	n := 0
	for _, p := range g.Players {
		if p.Status == storage.StatusGoing {
			n++
		}
		n += p.Friends
	}
	return n
}

// Free возвращает число свободных мест.
func Free(g *storage.Game) int {
	return Limit - Count(g)
}

// Join записывает пользователя на игру. Человек, который раньше нажал
// «Не смогу» или записывал только друзей, остаётся на своём месте в списке.
func Join(g *storage.Game, u User) error {
	p := find(g, u)
	if p != nil && p.Status == storage.StatusGoing {
		return ErrAlreadyGoing
	}
	if Free(g) < 1 {
		return ErrNoPlaces
	}
	if p == nil {
		p = add(g, u)
	}
	p.Status = storage.StatusGoing
	return nil
}

// Decline отмечает, что пользователь не сможет. Его друзья остаются в списке.
func Decline(g *storage.Game, u User) error {
	p := find(g, u)
	if p != nil && p.Status == storage.StatusDeclined {
		return ErrAlreadyDecline
	}
	if p == nil {
		p = add(g, u)
	}
	p.Status = storage.StatusDeclined
	return nil
}

// AddFriend записывает ещё одного друга пользователя.
func AddFriend(g *storage.Game, u User) error {
	p := find(g, u)
	if p != nil && p.Friends >= MaxFriends {
		return ErrTooManyFriends
	}
	if Free(g) < 1 {
		return ErrNoPlaces
	}
	if p == nil {
		p = add(g, u)
	}
	p.Friends++
	return nil
}

// RemoveFriend убирает одного друга пользователя. Если у человека
// не осталось ни друзей, ни статуса, он убирается из списка.
func RemoveFriend(g *storage.Game, u User) error {
	p := find(g, u)
	if p == nil || p.Friends == 0 {
		return ErrNoFriends
	}
	p.Friends--
	if p.Friends == 0 && p.Status == storage.StatusNone {
		remove(g, u.ID)
	}
	return nil
}

// Transition сравнивает счёт до и после действия.
func Transition(before, after int) Event {
	switch {
	case before < Limit && after >= Limit:
		return EventFilled
	case before >= Limit && after < Limit:
		return EventOpened
	default:
		return EventNone
	}
}

// find возвращает игрока по ID и обновляет его имя: его могли сменить в Telegram.
func find(g *storage.Game, u User) *storage.Player {
	for i := range g.Players {
		if g.Players[i].UserID == u.ID {
			g.Players[i].Name = u.Name
			return &g.Players[i]
		}
	}
	return nil
}

func add(g *storage.Game, u User) *storage.Player {
	g.Players = append(g.Players, storage.Player{UserID: u.ID, Name: u.Name})
	return &g.Players[len(g.Players)-1]
}

func remove(g *storage.Game, userID int64) {
	for i, p := range g.Players {
		if p.UserID == userID {
			g.Players = append(g.Players[:i], g.Players[i+1:]...)
			return
		}
	}
}
