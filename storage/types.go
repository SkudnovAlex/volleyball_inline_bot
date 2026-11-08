package storage

// Статусы игрока в Player.Status.
const (
	StatusNone     = ""         // нажимал только кнопки друзей
	StatusGoing    = "going"    // нажал «Иду»
	StatusDeclined = "declined" // нажал «Не смогу»
)

// Data хранит всё состояние бота.
type Data struct {
	BoardMessageID int              `json:"board_message_id"`
	Games          []Game           `json:"games"`
	Requisites     map[int64]string `json:"requisites"` // user_id и последние реквизиты плательщика зала
}

// Game описывает одну игру и её сообщение в чате.
type Game struct {
	Date      string    `json:"date"` // 2006-01-02, он же ID игры
	MessageID int       `json:"message_id"`
	Finished  bool      `json:"finished"`
	Players   []Player  `json:"players"` // в порядке записи
	Announce  *Announce `json:"announce,omitempty"`
	Payment   *Payment  `json:"payment,omitempty"`
}

// Player описывает человека, который нажимал кнопки игры.
type Player struct {
	UserID  int64  `json:"user_id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Friends int    `json:"friends"`
}

// Announce описывает последнее объявление «состав собран» или «освободилось место».
type Announce struct {
	MessageID int   `json:"message_id"`
	Full      bool  `json:"full"`
	SentAt    int64 `json:"sent_at"` // unix-время
}

// Payment описывает оплату зала после игры.
type Payment struct {
	AwaitingUserID  int64   `json:"awaiting_user_id"`  // кто нажал «Я оплатил зал» и ещё не ответил
	PromptMessageID int     `json:"prompt_message_id"` // сообщение бота с ForceReply
	RenterID        int64   `json:"renter_id"`
	Requisites      string  `json:"requisites"`
	Total           int     `json:"total"`
	PerPerson       int     `json:"per_person"`
	PaidUserIDs     []int64 `json:"paid_user_ids"`
	SetAt           int64   `json:"set_at"` // unix-время указания реквизитов
	Reminded        bool    `json:"reminded"`
}

// Store читает и пишет Data в JSON-файл. Конкурентный доступ не защищён:
// вызывающий держит общий мьютекс на всё чтение, изменение и запись.
type Store struct {
	path string
}
