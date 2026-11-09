package bot

import (
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/storage"
)

// Sender абстрагирует методы Telegram, нужные сервису.
// Ему удовлетворяет *tgbotapi.BotAPI.
type Sender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

// Repository абстрагирует хранилище. Ему удовлетворяет *storage.Store.
type Repository interface {
	Load() (storage.Data, error)
	Save(data storage.Data) error
}

// Service содержит работу бота с чатом. Все публичные методы держат mutex
// на всё чтение, изменение и запись данных: cron и нажатия кнопок
// работают в разных горутинах.
type Service struct {
	sender     Sender
	repository Repository
	chatID     int64
	gameDays   map[time.Weekday]bool
	// deleteAfterDays задаёт, через сколько дней после игры удалить её сообщение.
	deleteAfterDays int
	now             func() time.Time
	mutex           sync.Mutex
}

func New(sender Sender, repository Repository, chatID int64, location *time.Location, gameDays map[time.Weekday]bool, deleteAfterDays int) *Service {
	return &Service{
		sender:          sender,
		repository:      repository,
		chatID:          chatID,
		gameDays:        gameDays,
		deleteAfterDays: deleteAfterDays,
		now:             func() time.Time { return time.Now().In(location) },
	}
}
