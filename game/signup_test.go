package game

import (
	"errors"
	"testing"

	"volleyball_inline_bot/storage"
)

var (
	maria = User{ID: 1, Name: "Мария"}
	ivan  = User{ID: 2, Name: "Иван"}
	anna  = User{ID: 3, Name: "Анна"}
)

// gameWith возвращает игру, где уже записались n человек без друзей.
func gameWith(players int) *storage.Game {
	game := &storage.Game{Date: "2026-11-18"}
	for index := range players {
		game.Players = append(game.Players, storage.Player{UserID: int64(100 + index), Name: "Игрок", Status: storage.StatusGoing})
	}
	return game
}

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
}

func TestCount_GoingAndFriends(t *testing.T) {
	game := &storage.Game{Players: []storage.Player{
		{UserID: 1, Status: storage.StatusGoing},
		{UserID: 2, Status: storage.StatusGoing, Friends: 1},
		{UserID: 3, Status: storage.StatusDeclined, Friends: 2},
		{UserID: 4, Status: storage.StatusDeclined},
	}}

	if got := Count(game); got != 5 {
		t.Fatalf("Count = %d, ожидалось 5 (1 + 1+1 + 0+2)", got)
	}
	if got := Free(game); got != rules.Limit-5 {
		t.Fatalf("Free = %d, ожидалось %d", got, rules.Limit-5)
	}
}

func TestJoin(t *testing.T) {
	game := gameWith(0)
	mustOK(t, Join(game, maria))

	if len(game.Players) != 1 || game.Players[0].Status != storage.StatusGoing {
		t.Fatalf("Мария должна быть в списке со статусом going: %+v", game.Players)
	}
	if err := Join(game, maria); !errors.Is(err, ErrAlreadyGoing) {
		t.Fatalf("повторное Иду: ожидалась ErrAlreadyGoing, получено %v", err)
	}
}

func TestJoin_Full(t *testing.T) {
	game := gameWith(rules.Limit)
	if err := Join(game, maria); !errors.Is(err, ErrNoPlaces) {
		t.Fatalf("ожидалась ErrNoPlaces, получено %v", err)
	}
	if len(game.Players) != rules.Limit {
		t.Fatalf("при отказе список не должен меняться")
	}
}

func TestJoin_AfterDeclineKeepsPlace(t *testing.T) {
	game := gameWith(0)
	mustOK(t, Join(game, maria))
	mustOK(t, Join(game, ivan))
	mustOK(t, Decline(game, maria))
	mustOK(t, Join(game, maria))

	if game.Players[0].UserID != maria.ID || game.Players[0].Status != storage.StatusGoing {
		t.Fatalf("Мария должна вернуться на первое место: %+v", game.Players)
	}
}

func TestJoin_UpdatesName(t *testing.T) {
	game := gameWith(0)
	mustOK(t, Decline(game, maria))
	mustOK(t, Join(game, User{ID: maria.ID, Name: "Маша"}))

	if game.Players[0].Name != "Маша" {
		t.Fatalf("имя должно обновиться, получено %q", game.Players[0].Name)
	}
}

func TestDecline_KeepsFriends(t *testing.T) {
	game := gameWith(0)
	mustOK(t, Join(game, ivan))
	mustOK(t, AddFriend(game, ivan))
	mustOK(t, AddFriend(game, ivan))
	mustOK(t, Decline(game, ivan))

	if got := Count(game); got != 2 {
		t.Fatalf("после Не смогу друзья остаются: Count = %d, ожидалось 2", got)
	}
	if err := Decline(game, ivan); !errors.Is(err, ErrAlreadyDecline) {
		t.Fatalf("повторное Не смогу: ожидалась ErrAlreadyDecline, получено %v", err)
	}
}

func TestDecline_NewPlayerListed(t *testing.T) {
	game := gameWith(0)
	mustOK(t, Decline(game, anna))

	if len(game.Players) != 1 || game.Players[0].Status != storage.StatusDeclined {
		t.Fatalf("Анна должна попасть в список не идущих: %+v", game.Players)
	}
}

func TestAddFriend_Limit(t *testing.T) {
	game := gameWith(0)
	for range rules.MaxFriends {
		mustOK(t, AddFriend(game, ivan))
	}

	if err := AddFriend(game, ivan); !errors.Is(err, ErrTooManyFriends) {
		t.Fatalf("друг сверх rules.MaxFriends: ожидалась ErrTooManyFriends, получено %v", err)
	}
	if game.Players[0].Friends != rules.MaxFriends {
		t.Fatalf("Friends = %d, ожидалось %d", game.Players[0].Friends, rules.MaxFriends)
	}
}

func TestAddFriend_WithoutJoin(t *testing.T) {
	game := gameWith(0)
	mustOK(t, AddFriend(game, ivan))

	if game.Players[0].Status != storage.StatusNone || Count(game) != 1 {
		t.Fatalf("записан только друг Ивана: %+v, Count = %d", game.Players, Count(game))
	}
}

func TestAddFriend_LastPlace(t *testing.T) {
	game := gameWith(rules.Limit - 1)
	mustOK(t, AddFriend(game, ivan))

	if err := AddFriend(game, ivan); !errors.Is(err, ErrNoPlaces) {
		t.Fatalf("сверх лимита: ожидалась ErrNoPlaces, получено %v", err)
	}
	if Count(game) != rules.Limit {
		t.Fatalf("Count = %d, ожидалось %d", Count(game), rules.Limit)
	}
}

func TestRemoveFriend(t *testing.T) {
	game := gameWith(0)
	if err := RemoveFriend(game, ivan); !errors.Is(err, ErrNoFriends) {
		t.Fatalf("без записи: ожидалась ErrNoFriends, получено %v", err)
	}

	mustOK(t, Join(game, ivan))
	if err := RemoveFriend(game, ivan); !errors.Is(err, ErrNoFriends) {
		t.Fatalf("без друзей: ожидалась ErrNoFriends, получено %v", err)
	}

	mustOK(t, AddFriend(game, ivan))
	mustOK(t, AddFriend(game, ivan))
	mustOK(t, RemoveFriend(game, ivan))
	if game.Players[0].Friends != 1 || game.Players[0].Status != storage.StatusGoing {
		t.Fatalf("должен остаться Иван с одним другом: %+v", game.Players)
	}
}

func TestRemoveFriend_DropsEmptyPlayer(t *testing.T) {
	game := gameWith(0)
	mustOK(t, AddFriend(game, ivan))
	mustOK(t, Join(game, maria))
	mustOK(t, RemoveFriend(game, ivan))

	if len(game.Players) != 1 || game.Players[0].UserID != maria.ID {
		t.Fatalf("Иван без статуса и друзей должен уйти из списка: %+v", game.Players)
	}
}

func TestTransition(t *testing.T) {
	cases := []struct {
		before, after int
		want          Event
	}{
		{11, 12, EventFilled},
		{10, 12, EventFilled},
		{12, 11, EventOpened},
		{12, 9, EventOpened},
		{11, 10, EventNone},
		{12, 12, EventNone},
		{5, 6, EventNone},
	}
	for _, testCase := range cases {
		if got := Transition(testCase.before, testCase.after); got != testCase.want {
			t.Errorf("Transition(%d, %d) = %d, ожидалось %d", testCase.before, testCase.after, got, testCase.want)
		}
	}
}

func TestTransition_TwelveElevenTwelve(t *testing.T) {
	game := gameWith(rules.Limit - 1)

	before := Count(game)
	mustOK(t, Join(game, maria))
	if event := Transition(before, Count(game)); event != EventFilled {
		t.Fatalf("11 → 12: ожидалось EventFilled, получено %d", event)
	}

	before = Count(game)
	mustOK(t, Decline(game, maria))
	if event := Transition(before, Count(game)); event != EventOpened {
		t.Fatalf("12 → 11: ожидалось EventOpened, получено %d", event)
	}

	before = Count(game)
	mustOK(t, Join(game, maria))
	if event := Transition(before, Count(game)); event != EventFilled {
		t.Fatalf("11 → 12: ожидалось EventFilled, получено %d", event)
	}
}
