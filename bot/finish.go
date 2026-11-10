package bot

import (
	"log"
	"time"

	"volleyball_inline_bot/game"
)

// FinishGames закрывает игры, с начала которых прошло finishAfter: вместо кнопок записи в сообщении появляется кнопка
// оплаты зала, на доске игра переходит в "Прошедшие".
func (s *Service) FinishGames() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := s.repository.Load()
	if err != nil {
		log.Printf("Ошибка чтения хранилища: %v", err)
		return
	}

	now := s.now()
	changed := false
	for index := range data.Games {
		match := &data.Games[index]
		if match.Finished {
			continue
		}
		start, err := time.ParseInLocation(game.DateLayout+" 15:04", match.Date+" "+game.Current().GameTime,
			now.Location())
		if err != nil {
			log.Printf("Не удалось определить начало игры %s: %v", match.Date, err)
			continue
		}
		if now.Before(start.Add(s.finishAfter)) {
			continue
		}
		match.Finished = true
		s.editGame(match)
		changed = true
		log.Printf("Игра %s прошла, запись закрыта", match.Date)
	}
	if !changed {
		return
	}

	if err := s.repository.Save(data); err != nil {
		log.Printf("Не удалось сохранить закрытие игр: %v", err)
		return
	}
	s.updateBoardAndSave(&data)
}
