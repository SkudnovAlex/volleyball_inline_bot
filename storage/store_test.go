package storage

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "data.json"))

	data, err := s.Load()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if data.BoardMessageID != 0 || len(data.Games) != 0 {
		t.Fatalf("ожидались пустые данные, получено: %+v", data)
	}
	if data.Requisites == nil {
		t.Fatal("Requisites должен быть пустой картой, а не nil")
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}

	data, err := NewStore(path).Load()
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if data.Requisites == nil || len(data.Games) != 0 {
		t.Fatalf("ожидались пустые данные, получено: %+v", data)
	}
}

func TestLoad_BrokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(path, []byte("{не json"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := NewStore(path).Load(); err == nil {
		t.Fatal("ожидалась ошибка разбора JSON")
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	s := NewStore(path)

	want := Data{
		BoardMessageID: 10,
		Requisites:     map[int64]string{42: "Т-Банк +7 900 123-45-67"},
		Games: []Game{
			{
				Date:      "2026-11-18",
				MessageID: 100,
				Players: []Player{
					{UserID: 1, Name: "Мария", Status: StatusGoing},
					{UserID: 2, Name: "Иван", Status: StatusDeclined, Friends: 1},
				},
				Announce: &Announce{MessageID: 101, Full: true, SentAt: 1763400000},
			},
			{
				Date:      "2026-11-16",
				MessageID: 90,
				Finished:  true,
				Players:   []Player{{UserID: 42, Name: "Сергей", Status: StatusGoing}},
				Payment: &Payment{
					RenterID:    42,
					Requisites:  "Т-Банк +7 900 123-45-67",
					Total:       5400,
					PerPerson:   450,
					Came:        11,
					Absent:      []string{"7", "8#1"},
					PaidUserIDs: []int64{1},
					Draft:       &Draft{UserID: 5, Step: StepAbsent, Absent: []string{"9"}, StartedAt: 1763300000},
				},
			},
		},
	}

	if err := s.Save(want); err != nil {
		t.Fatalf("ошибка записи: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("временный файл должен быть переименован, stat: %v", err)
	}

	got, err := s.Load()
	if err != nil {
		t.Fatalf("ошибка чтения: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("прочитано не то, что записано:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestSave_Overwrites(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "data.json"))

	if err := s.Save(Data{BoardMessageID: 1, Games: []Game{{Date: "2026-11-18"}}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(Data{BoardMessageID: 2}); err != nil {
		t.Fatal(err)
	}

	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.BoardMessageID != 2 || len(got.Games) != 0 {
		t.Fatalf("файл должен быть перезаписан целиком, получено: %+v", got)
	}
}
