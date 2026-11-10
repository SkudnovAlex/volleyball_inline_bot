package bot

import (
	"testing"
	"time"

	"volleyball_inline_bot/storage"
)

// finishService создаёт сервис с закрытием через час после начала. Время игры в тестах 19:00 (game.DefaultRules).
func finishService(sender *mockSender, repository *mockRepo, now time.Time) *Service {
	service := newService(sender, repository, func() time.Time { return now })
	service.finishAfter = time.Hour
	return service
}

func TestFinishGames(t *testing.T) {
	data := storage.Data{BoardMessageID: testBoardID, Games: []storage.Game{
		{Date: "2026-11-08", MessageID: 3},
		{Date: "2026-11-10", MessageID: 4},
		{Date: "2026-11-12", MessageID: 5},
	}}
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	finishService(sender, repository, time.Date(2026, time.November, 10, 20, 0, 0, 0, time.UTC)).FinishGames()

	games := repository.data.Games
	if !games[0].Finished || !games[1].Finished {
		t.Errorf("прошлая игра и сегодняшняя спустя час после начала закрываются: %+v", games[:2])
	}
	if games[2].Finished {
		t.Errorf("будущая игра остаётся открытой: %+v", games[2])
	}

	var gameEdited, boardEdited bool
	for _, edit := range edits(sender.requested) {
		switch edit.MessageID {
		case 4:
			gameEdited = true
			if edit.ReplyMarkup == nil || len(edit.ReplyMarkup.InlineKeyboard) != 1 ||
				*edit.ReplyMarkup.InlineKeyboard[0][0].CallbackData != "hall:2026-11-10" {
				t.Errorf("у закрытой игры вместо кнопок записи кнопка оплаты: %+v", edit.ReplyMarkup)
			}
		case testBoardID:
			boardEdited = true
		}
	}
	if !gameEdited || !boardEdited {
		t.Errorf("перерисовываются сообщение игры и доска: game=%v board=%v", gameEdited, boardEdited)
	}
}

func TestFinishGames_BeforeFinishTime(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: storage.Data{BoardMessageID: testBoardID, Games: []storage.Game{
		{Date: "2026-11-10", MessageID: 4},
	}}}
	finishService(sender, repository, time.Date(2026, time.November, 10, 19, 59, 0, 0, time.UTC)).FinishGames()

	if repository.data.Games[0].Finished || repository.saves != 0 || len(sender.requested) != 0 {
		t.Fatalf("до часа после начала игра открыта и ничего не меняется: %+v saves=%d",
			repository.data.Games, repository.saves)
	}
}
