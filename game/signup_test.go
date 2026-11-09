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
func gameWith(n int) *storage.Game {
	g := &storage.Game{Date: "2026-11-18"}
	for i := 0; i < n; i++ {
		g.Players = append(g.Players, storage.Player{UserID: int64(100 + i), Name: "Игрок", Status: storage.StatusGoing})
	}
	return g
}

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
}

func TestCount_GoingAndFriends(t *testing.T) {
	g := &storage.Game{Players: []storage.Player{
		{UserID: 1, Status: storage.StatusGoing},
		{UserID: 2, Status: storage.StatusGoing, Friends: 1},
		{UserID: 3, Status: storage.StatusDeclined, Friends: 2},
		{UserID: 4, Status: storage.StatusDeclined},
	}}

	if got := Count(g); got != 5 {
		t.Fatalf("Count = %d, ожидалось 5 (1 + 1+1 + 0+2)", got)
	}
	if got := Free(g); got != Limit-5 {
		t.Fatalf("Free = %d, ожидалось %d", got, Limit-5)
	}
}

func TestJoin(t *testing.T) {
	g := gameWith(0)
	mustOK(t, Join(g, maria))

	if len(g.Players) != 1 || g.Players[0].Status != storage.StatusGoing {
		t.Fatalf("Мария должна быть в списке со статусом going: %+v", g.Players)
	}
	if err := Join(g, maria); !errors.Is(err, ErrAlreadyGoing) {
		t.Fatalf("повторное Иду: ожидалась ErrAlreadyGoing, получено %v", err)
	}
}

func TestJoin_Full(t *testing.T) {
	g := gameWith(Limit)
	if err := Join(g, maria); !errors.Is(err, ErrNoPlaces) {
		t.Fatalf("ожидалась ErrNoPlaces, получено %v", err)
	}
	if len(g.Players) != Limit {
		t.Fatalf("при отказе список не должен меняться")
	}
}

func TestJoin_AfterDeclineKeepsPlace(t *testing.T) {
	g := gameWith(0)
	mustOK(t, Join(g, maria))
	mustOK(t, Join(g, ivan))
	mustOK(t, Decline(g, maria))
	mustOK(t, Join(g, maria))

	if g.Players[0].UserID != maria.ID || g.Players[0].Status != storage.StatusGoing {
		t.Fatalf("Мария должна вернуться на первое место: %+v", g.Players)
	}
}

func TestJoin_UpdatesName(t *testing.T) {
	g := gameWith(0)
	mustOK(t, Decline(g, maria))
	mustOK(t, Join(g, User{ID: maria.ID, Name: "Маша"}))

	if g.Players[0].Name != "Маша" {
		t.Fatalf("имя должно обновиться, получено %q", g.Players[0].Name)
	}
}

func TestDecline_KeepsFriends(t *testing.T) {
	g := gameWith(0)
	mustOK(t, Join(g, ivan))
	mustOK(t, AddFriend(g, ivan))
	mustOK(t, AddFriend(g, ivan))
	mustOK(t, Decline(g, ivan))

	if got := Count(g); got != 2 {
		t.Fatalf("после Не смогу друзья остаются: Count = %d, ожидалось 2", got)
	}
	if err := Decline(g, ivan); !errors.Is(err, ErrAlreadyDecline) {
		t.Fatalf("повторное Не смогу: ожидалась ErrAlreadyDecline, получено %v", err)
	}
}

func TestDecline_NewPlayerListed(t *testing.T) {
	g := gameWith(0)
	mustOK(t, Decline(g, anna))

	if len(g.Players) != 1 || g.Players[0].Status != storage.StatusDeclined {
		t.Fatalf("Анна должна попасть в список не идущих: %+v", g.Players)
	}
}

func TestAddFriend_Limit(t *testing.T) {
	g := gameWith(0)
	for i := 0; i < MaxFriends; i++ {
		mustOK(t, AddFriend(g, ivan))
	}

	if err := AddFriend(g, ivan); !errors.Is(err, ErrTooManyFriends) {
		t.Fatalf("друг сверх MaxFriends: ожидалась ErrTooManyFriends, получено %v", err)
	}
	if g.Players[0].Friends != MaxFriends {
		t.Fatalf("Friends = %d, ожидалось %d", g.Players[0].Friends, MaxFriends)
	}
}

func TestAddFriend_WithoutJoin(t *testing.T) {
	g := gameWith(0)
	mustOK(t, AddFriend(g, ivan))

	if g.Players[0].Status != storage.StatusNone || Count(g) != 1 {
		t.Fatalf("записан только друг Ивана: %+v, Count = %d", g.Players, Count(g))
	}
}

func TestAddFriend_LastPlace(t *testing.T) {
	g := gameWith(Limit - 1)
	mustOK(t, AddFriend(g, ivan))

	if err := AddFriend(g, ivan); !errors.Is(err, ErrNoPlaces) {
		t.Fatalf("сверх лимита: ожидалась ErrNoPlaces, получено %v", err)
	}
	if Count(g) != Limit {
		t.Fatalf("Count = %d, ожидалось %d", Count(g), Limit)
	}
}

func TestRemoveFriend(t *testing.T) {
	g := gameWith(0)
	if err := RemoveFriend(g, ivan); !errors.Is(err, ErrNoFriends) {
		t.Fatalf("без записи: ожидалась ErrNoFriends, получено %v", err)
	}

	mustOK(t, Join(g, ivan))
	if err := RemoveFriend(g, ivan); !errors.Is(err, ErrNoFriends) {
		t.Fatalf("без друзей: ожидалась ErrNoFriends, получено %v", err)
	}

	mustOK(t, AddFriend(g, ivan))
	mustOK(t, AddFriend(g, ivan))
	mustOK(t, RemoveFriend(g, ivan))
	if g.Players[0].Friends != 1 || g.Players[0].Status != storage.StatusGoing {
		t.Fatalf("должен остаться Иван с одним другом: %+v", g.Players)
	}
}

func TestRemoveFriend_DropsEmptyPlayer(t *testing.T) {
	g := gameWith(0)
	mustOK(t, AddFriend(g, ivan))
	mustOK(t, Join(g, maria))
	mustOK(t, RemoveFriend(g, ivan))

	if len(g.Players) != 1 || g.Players[0].UserID != maria.ID {
		t.Fatalf("Иван без статуса и друзей должен уйти из списка: %+v", g.Players)
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
	for _, c := range cases {
		if got := Transition(c.before, c.after); got != c.want {
			t.Errorf("Transition(%d, %d) = %d, ожидалось %d", c.before, c.after, got, c.want)
		}
	}
}

func TestTransition_TwelveElevenTwelve(t *testing.T) {
	g := gameWith(Limit - 1)

	before := Count(g)
	mustOK(t, Join(g, maria))
	if ev := Transition(before, Count(g)); ev != EventFilled {
		t.Fatalf("11 → 12: ожидалось EventFilled, получено %d", ev)
	}

	before = Count(g)
	mustOK(t, Decline(g, maria))
	if ev := Transition(before, Count(g)); ev != EventOpened {
		t.Fatalf("12 → 11: ожидалось EventOpened, получено %d", ev)
	}

	before = Count(g)
	mustOK(t, Join(g, maria))
	if ev := Transition(before, Count(g)); ev != EventFilled {
		t.Fatalf("11 → 12: ожидалось EventFilled, получено %d", ev)
	}
}
