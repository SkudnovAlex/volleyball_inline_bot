package bot

import (
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/storage"
)

// deletedIDs возвращает номера сообщений из запросов на удаление.
func deletedIDs(requests []tgbotapi.Chattable) []int {
	var result []int
	for _, request := range requests {
		if config, ok := request.(tgbotapi.DeleteMessageConfig); ok {
			result = append(result, config.MessageID)
		}
	}
	return result
}

// Сегодня в тестах 10.11.2026 (tuesday), срок удаления 14 дней: удаляются игры с датой 27.10 и раньше.
func TestCleanupGames(t *testing.T) {
	data := storage.Data{BoardMessageID: testBoardID, Games: []storage.Game{
		{Date: "2026-10-27", MessageID: 1, Finished: true, Announce: &storage.Announce{MessageID: 2}},
		{Date: "2026-10-28", MessageID: 3, Finished: true},
		{Date: "2026-11-09", MessageID: 4},
		{Date: "2026-11-10", MessageID: 5},
		{Date: "2026-11-12", MessageID: 6},
	}}
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	newService(sender, repository, tuesday).CleanupGames()

	if got := deletedIDs(sender.requested); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("удаляются сообщение игры 27.10 и её объявление, удалено %v", got)
	}

	games := repository.data.Games
	if len(games) != 4 || games[0].Date != "2026-10-28" {
		t.Fatalf("игра 27.10 должна уйти из данных: %+v", games)
	}
	if !games[1].Finished {
		t.Errorf("вчерашняя игра 09.11 должна стать прошедшей: %+v", games[1])
	}
	if games[2].Finished || games[3].Finished {
		t.Errorf("сегодняшняя и будущая игры остаются открытыми: %+v", games[2:])
	}

	var gameEdit, boardEdit *tgbotapi.EditMessageTextConfig
	for _, edit := range edits(sender.requested) {
		switch edit.MessageID {
		case 4:
			gameEdit = &edit
		case testBoardID:
			boardEdit = &edit
		default:
			t.Errorf("лишнее редактирование сообщения %d", edit.MessageID)
		}
	}
	if gameEdit == nil || gameEdit.ReplyMarkup == nil || len(gameEdit.ReplyMarkup.InlineKeyboard) != 1 ||
		*gameEdit.ReplyMarkup.InlineKeyboard[0][0].CallbackData != "hall:2026-11-09" {
		t.Errorf("у прошедшей игры вместо кнопок записи кнопка оплаты: %+v", gameEdit)
	}
	if boardEdit == nil {
		t.Errorf("после очистки обновляется доска")
	}
}

func TestCleanupGames_NothingToDo(t *testing.T) {
	data := storage.Data{BoardMessageID: testBoardID, Games: []storage.Game{
		{Date: "2026-11-01", MessageID: 3, Finished: true},
		{Date: "2026-11-12", MessageID: 6},
	}}
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	newService(sender, repository, tuesday).CleanupGames()

	if repository.saves != 0 || len(sender.requested) != 0 {
		t.Fatalf("без изменений нет ни записи, ни запросов: saves=%d requests=%d", repository.saves, len(sender.requested))
	}
}

func TestCleanupGames_NoRightsKeepsGame(t *testing.T) {
	for _, message := range []string{
		"Bad Request: not enough rights to delete a message",
		"Bad Request: message can't be deleted",
	} {
		sender := &mockSender{pinnedMessageID: testBoardID, requestErr: func(request tgbotapi.Chattable) error {
			if _, ok := request.(tgbotapi.DeleteMessageConfig); ok {
				return &tgbotapi.Error{Code: 400, Message: message}
			}
			return nil
		}}
		repository := &mockRepo{data: storage.Data{BoardMessageID: testBoardID, Games: []storage.Game{
			{Date: "2026-10-01", MessageID: 1, Finished: true},
		}}}
		newService(sender, repository, tuesday).CleanupGames()

		if len(repository.data.Games) != 1 {
			t.Errorf("%q: игра остаётся до следующей очистки: %+v", message, repository.data.Games)
		}
	}
}

func TestCleanupGames_OtherDeleteErrorRemovesGame(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID, requestErr: func(request tgbotapi.Chattable) error {
		if _, ok := request.(tgbotapi.DeleteMessageConfig); ok {
			return errors.New("message to delete not found")
		}
		return nil
	}}
	repository := &mockRepo{data: storage.Data{BoardMessageID: testBoardID, Games: []storage.Game{
		{Date: "2026-10-01", MessageID: 1, Finished: true},
	}}}
	newService(sender, repository, tuesday).CleanupGames()

	if len(repository.data.Games) != 0 {
		t.Fatalf("если сообщения уже нет, игра убирается из данных: %+v", repository.data.Games)
	}
}

func TestCleanupGames_OverdueUnfinishedDeleted(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: storage.Data{BoardMessageID: testBoardID, Games: []storage.Game{
		{Date: "2026-10-01", MessageID: 1},
	}}}
	newService(sender, repository, tuesday).CleanupGames()

	if len(repository.data.Games) != 0 || len(edits(sender.requested)) != 1 {
		t.Fatalf("старая игра удаляется сразу, без перехода в «прошла»: games=%+v edits=%d",
			repository.data.Games, len(edits(sender.requested)))
	}
}

func TestCleanupGames_DeletesPaymentMessage(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: storage.Data{BoardMessageID: testBoardID, Games: []storage.Game{
		{Date: "2026-10-01", MessageID: 1, Finished: true, Payment: &storage.Payment{RenterID: 5, MessageID: 3}},
	}}}
	newService(sender, repository, tuesday).CleanupGames()

	if got := deletedIDs(sender.requested); len(got) != 2 || got[1] != 3 {
		t.Fatalf("вместе с игрой удаляется сообщение оплаты: %v", got)
	}
}
