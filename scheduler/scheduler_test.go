package scheduler

import (
	"testing"
	"time"
)

func TestAdd_InvalidSchedule(t *testing.T) {
	c := New(time.UTC)
	if err := Add(c, "not a cron", func() {}); err == nil {
		t.Fatal("ожидалась ошибка для некорректного расписания")
	}
	c.Stop()
}

func TestAdd_MultipleJobs(t *testing.T) {
	c := New(time.UTC)
	if err := Add(c, "0 20 * * *", func() {}); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if err := Add(c, "0 6 * * *", func() {}); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	c.Start()
	defer c.Stop()

	if len(c.Entries()) != 2 {
		t.Errorf("ожидалось 2 задачи, получено: %d", len(c.Entries()))
	}
}
