package game

import (
	"errors"

	"volleyball_inline_bot/storage"
)

// Ошибки действий. Текст показывается пользователю во всплывающем окне.
var (
	ErrAlreadyGoing   = errors.New("Ты уже в списке")
	ErrAlreadyDecline = errors.New("Уже отмечено, что ты не сможешь")
	ErrNoPlaces       = errors.New("Мест нет")
	ErrNoFriends      = errors.New("У тебя нет записанных друзей")
)

// User описывает того, кто нажал кнопку.
type User struct {
	ID   int64
	Name string
}

// Event описывает переход счёта через rules.Limit.
type Event int

const (
	EventNone   Event = iota
	EventFilled       // набралось rules.Limit
	EventOpened       // было rules.Limit, стало меньше
)

// Count возвращает число идущих: записавшиеся сами плюс все друзья.
func Count(game *storage.Game) int {
	count := 0
	for _, player := range game.Players {
		if player.Status == storage.StatusGoing {
			count++
		}
		count += player.Friends
	}
	return count
}

// Free возвращает число свободных мест.
func Free(game *storage.Game) int {
	return rules.Limit - Count(game)
}

// Join записывает пользователя на игру. Человек, который раньше нажал "Не смогу" или записывал только друзей, остаётся
// на своём месте в списке.
func Join(game *storage.Game, user User) error {
	player := find(game, user)
	if player != nil && player.Status == storage.StatusGoing {
		return ErrAlreadyGoing
	}
	if Free(game) < 1 {
		return ErrNoPlaces
	}
	if player == nil {
		player = add(game, user)
	}
	player.Status = storage.StatusGoing
	return nil
}

// Decline отмечает, что пользователь не сможет. Его друзья остаются в списке.
func Decline(game *storage.Game, user User) error {
	player := find(game, user)
	if player != nil && player.Status == storage.StatusDeclined {
		return ErrAlreadyDecline
	}
	if player == nil {
		player = add(game, user)
	}
	player.Status = storage.StatusDeclined
	return nil
}

// AddFriend записывает ещё одного друга пользователя.
func AddFriend(game *storage.Game, user User) error {
	player := find(game, user)
	if player != nil && player.Friends >= rules.MaxFriends {
		return ErrTooManyFriends
	}
	if Free(game) < 1 {
		return ErrNoPlaces
	}
	if player == nil {
		player = add(game, user)
	}
	player.Friends++
	return nil
}

// RemoveFriend убирает одного друга пользователя. Если у человека не осталось ни друзей, ни статуса, он убирается из
// списка.
func RemoveFriend(game *storage.Game, user User) error {
	player := find(game, user)
	if player == nil || player.Friends == 0 {
		return ErrNoFriends
	}
	player.Friends--
	if player.Friends == 0 && player.Status == storage.StatusNone {
		remove(game, user.ID)
	}
	return nil
}

// Transition сравнивает счёт до и после действия.
func Transition(before, after int) Event {
	switch {
	case before < rules.Limit && after >= rules.Limit:
		return EventFilled
	case before >= rules.Limit && after < rules.Limit:
		return EventOpened
	default:
		return EventNone
	}
}

// find возвращает игрока по ID и обновляет его имя: его могли сменить в Telegram.
func find(game *storage.Game, user User) *storage.Player {
	for index := range game.Players {
		if game.Players[index].UserID == user.ID {
			game.Players[index].Name = user.Name
			return &game.Players[index]
		}
	}
	return nil
}

func add(game *storage.Game, user User) *storage.Player {
	game.Players = append(game.Players, storage.Player{UserID: user.ID, Name: user.Name})
	return &game.Players[len(game.Players)-1]
}

func remove(game *storage.Game, userID int64) {
	for index, player := range game.Players {
		if player.UserID == userID {
			game.Players = append(game.Players[:index], game.Players[index+1:]...)
			return
		}
	}
}
