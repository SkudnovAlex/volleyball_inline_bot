package scheduler

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
)

// New создаёт планировщик, привязанный к указанной таймзоне.
func New(loc *time.Location) *cron.Cron {
	return cron.New(cron.WithLocation(loc))
}

// Add регистрирует задачу с заданным расписанием.
func Add(c *cron.Cron, schedule string, job func()) error {
	_, err := c.AddFunc(schedule, job)
	return err
}

// WaitForShutdown блокируется до SIGINT/SIGTERM и корректно останавливает cron.
func WaitForShutdown(c *cron.Cron) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Получен сигнал завершения. Бот останавливается...")

	ctx := c.Stop()
	<-ctx.Done()

	log.Println("Бот остановлен")
}
