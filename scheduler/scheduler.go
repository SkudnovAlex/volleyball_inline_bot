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
func New(location *time.Location) *cron.Cron {
	return cron.New(cron.WithLocation(location))
}

// Add регистрирует задачу с заданным расписанием.
func Add(planner *cron.Cron, schedule string, job func()) error {
	_, err := planner.AddFunc(schedule, job)
	return err
}

// WaitForShutdown блокируется до SIGINT/SIGTERM и корректно останавливает cron.
func WaitForShutdown(planner *cron.Cron) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	<-signals
	log.Println("Получен сигнал завершения. Бот останавливается...")

	stopped := planner.Stop()
	<-stopped.Done()

	log.Println("Бот остановлен")
}
