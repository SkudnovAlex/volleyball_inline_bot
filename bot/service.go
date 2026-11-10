package bot

import (
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/storage"
)

// Sender абстрагирует методы Telegram, нужные сервису. Ему удовлетворяет *tgbotapi.BotAPI.
type Sender interface {
	Send(request tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(request tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

// Repository абстрагирует хранилище. Ему удовлетворяет *storage.Store.
type Repository interface {
	Load() (storage.Data, error)
	Save(data storage.Data) error
}

// Settings содержит настройки сервиса из конфигурации.
type Settings struct {
	ChatID   int64
	Location *time.Location
	GameDays map[time.Weekday]bool
	// DeleteAfterDays задаёт, через сколько дней после игры удалить её сообщение.
	DeleteAfterDays int
	// FinishAfter задаёт, через сколько после начала игры закрывается запись и появляется кнопка оплаты зала.
	FinishAfter time.Duration
	// CreateDaysAhead задаёт, за сколько дней до игры создаётся её сообщение.
	CreateDaysAhead int
	// AnnounceCooldown задаёт, сколько времени после объявления бот редактирует его вместо отправки нового.
	AnnounceCooldown time.Duration
	// DraftLock задаёт, сколько оформление оплаты одним человеком не даёт начать его другому.
	DraftLock time.Duration
	// BotUserName нужен для ссылки t.me/<бот>?start=... при переходе в личку.
	BotUserName string
}

// Service содержит работу бота с чатом. Все публичные методы держат mutex на всё чтение, изменение и запись данных: cron
// и нажатия кнопок работают в разных горутинах.
type Service struct {
	sender           Sender
	repository       Repository
	chatID           int64
	gameDays         map[time.Weekday]bool
	deleteAfterDays  int
	finishAfter      time.Duration
	createDaysAhead  int
	announceCooldown time.Duration
	draftLock        time.Duration
	botUserName      string
	now              func() time.Time
	mutex            sync.Mutex
}

func New(sender Sender, repository Repository, settings Settings) *Service {
	return &Service{
		sender:           sender,
		repository:       repository,
		chatID:           settings.ChatID,
		gameDays:         settings.GameDays,
		deleteAfterDays:  settings.DeleteAfterDays,
		finishAfter:      settings.FinishAfter,
		createDaysAhead:  settings.CreateDaysAhead,
		announceCooldown: settings.AnnounceCooldown,
		draftLock:        settings.DraftLock,
		botUserName:      settings.BotUserName,
		now:              func() time.Time { return time.Now().In(settings.Location) },
	}
}
