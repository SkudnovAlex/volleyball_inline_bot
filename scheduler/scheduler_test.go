package scheduler

import (
	"testing"
	"time"
)

func TestAdd_InvalidSchedule(t *testing.T) {
	planner := New(time.UTC)
	if err := Add(planner, "not a cron", func() {}); err == nil {
		t.Fatal("ожидалась ошибка для некорректного расписания")
	}
	planner.Stop()
}

func TestAdd_MultipleJobs(t *testing.T) {
	planner := New(time.UTC)
	if err := Add(planner, "0 20 * * *", func() {}); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if err := Add(planner, "0 6 * * *", func() {}); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	planner.Start()
	defer planner.Stop()

	if len(planner.Entries()) != 2 {
		t.Errorf("ожидалось 2 задачи, получено: %d", len(planner.Entries()))
	}
}
