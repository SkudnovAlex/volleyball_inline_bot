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

// Шаги оформления оплаты в Draft.Step.
const (
	StepCame    = "came"    // ждём, сколько было людей
	StepAbsent  = "absent"  // ждём отметки неявки
	StepTotal   = "total"   // ждём сумму и реквизиты
	StepConfirm = "confirm" // ждём «Отправить в группу» или «Заново»
)

// Payment описывает оплату зала после игры. Поля кроме Draft заполняются,
// когда плательщик отправил оплату в группу (RenterID не 0).
type Payment struct {
	RenterID    int64    `json:"renter_id"`
	RenterName  string   `json:"renter_name"`
	Requisites  string   `json:"requisites"`
	Came        int      `json:"came"`   // сколько было, со слов плательщика
	Absent      []string `json:"absent"` // строки списка: "<user_id>" или "<user_id>#<n>" для n-го друга
	Total       int      `json:"total"`
	PerPerson   int      `json:"per_person"`
	MessageID   int      `json:"message_id"` // сообщение оплаты в группе
	PaidUserIDs []int64  `json:"paid_user_ids"`
	Draft       *Draft   `json:"draft,omitempty"`
}

// Draft описывает оформление оплаты в личке, ещё не отправленное в группу.
type Draft struct {
	UserID     int64    `json:"user_id"`
	UserName   string   `json:"user_name"`
	Step       string   `json:"step"`
	Came       int      `json:"came"`
	Absent     []string `json:"absent"`
	Requisites string   `json:"requisites"`
	Total      int      `json:"total"`
	StartedAt  int64    `json:"started_at"` // unix-время
}

// Store читает и пишет Data в JSON-файл. Конкурентный доступ не защищён:
// вызывающий держит общий мьютекс на всё чтение, изменение и запись.
type Store struct {
	path string
}
