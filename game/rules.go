package game

import "fmt"

// Rules содержит правила записи и оплаты из конфигурации.
type Rules struct {
	GameTime   string // время игры для сообщения, "ЧЧ:ММ"
	Limit      int    // сколько человек нужно на игру
	MaxFriends int    // сколько друзей может привести один человек
	MaxCame    int    // максимум в ответе "сколько было" при оплате
	MinTotal   int    // минимальная сумма за зал, ₽
	MaxTotal   int    // максимальная сумма за зал, ₽
	RoundTo    int    // шаг округления доли вверх, ₽
}

// DefaultRules это правила по умолчанию, они же действуют в тестах.
var DefaultRules = Rules{
	GameTime:   "19:00",
	Limit:      12,
	MaxFriends: 4,
	MaxCame:    30,
	MinTotal:   1000,
	MaxTotal:   10000,
	RoundTo:    10,
}

// MinCame это минимум в ответе "сколько было", он не настраивается.
const MinCame = 1

// rules это действующие правила. Задаются один раз при старте через Configure, до запуска бота, и дальше не меняются,
// поэтому чтение из разных горутин безопасно.
var rules Rules

// Ошибки, текст которых зависит от правил. Пересоздаются в Configure.
var (
	ErrTooManyFriends error
	ErrBadCame        error
	ErrBadAmount      error
)

func init() {
	Configure(DefaultRules)
}

// Configure задаёт правила. Вызывается из main до запуска бота.
func Configure(configured Rules) {
	rules = configured
	ErrTooManyFriends = fmt.Errorf("Можно привести не больше %d друзей", rules.MaxFriends)
	ErrBadCame = fmt.Errorf("Пришли число от %d до %d", MinCame, rules.MaxCame)
	ErrBadAmount = fmt.Errorf("Не понял сумму. Напиши её в конце строки числом от %d до %d, например: Т-Банк +7 900 "+
		"123-45-67 5400", rules.MinTotal, rules.MaxTotal)
}

// Current возвращает действующие правила.
func Current() Rules {
	return rules
}
