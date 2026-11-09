package bot

import (
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestRefreshBoard_SendsAndPinsWhenMissing(t *testing.T) {
	sender := &mockSender{}
	repository := &mockRepo{data: openGame(2)}
	repository.data.BoardMessageID = 0
	newService(sender, repository, tuesday).RefreshBoard()

	if len(sender.sent) != 1 {
		t.Fatalf("ожидалась одна новая доска, отправлено %d", len(sender.sent))
	}
	board := sender.sent[0].(tgbotapi.MessageConfig)
	if board.ParseMode != tgbotapi.ModeHTML || !board.DisableWebPagePreview {
		t.Errorf("доска в HTML и без превью ссылок: %+v", board)
	}
	if repository.data.BoardMessageID != 101 || repository.saves != 1 {
		t.Errorf("ID доски должен сохраниться: id=%d saves=%d", repository.data.BoardMessageID, repository.saves)
	}
	var pin *tgbotapi.PinChatMessageConfig
	for _, request := range sender.requested {
		if config, ok := request.(tgbotapi.PinChatMessageConfig); ok {
			pin = &config
		}
	}
	if pin == nil || pin.MessageID != 101 || !pin.DisableNotification {
		t.Fatalf("доска закрепляется без уведомления: %+v", pin)
	}
}

func TestRefreshBoard_EditsExistingPinned(t *testing.T) {
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: openGame(2)}
	newService(sender, repository, tuesday).RefreshBoard()

	if len(sender.sent) != 0 || countPins(sender.requested) != 0 || repository.saves != 0 {
		t.Fatalf("доска есть и закреплена: только редактирование, sent=%d pins=%d saves=%d",
			len(sender.sent), countPins(sender.requested), repository.saves)
	}
	if boardEdits := edits(sender.requested); len(boardEdits) != 1 || boardEdits[0].MessageID != testBoardID {
		t.Fatalf("ожидалось редактирование доски: %+v", boardEdits)
	}
}

func TestRefreshBoard_RepinsWhenUnpinned(t *testing.T) {
	for _, pinned := range []int{0, 555} {
		sender := &mockSender{pinnedMessageID: pinned}
		repository := &mockRepo{data: openGame(2)}
		newService(sender, repository, tuesday).RefreshBoard()

		if countPins(sender.requested) != 1 || len(sender.sent) != 0 {
			t.Errorf("закреп %d: доска закрепляется заново без новой отправки, pins=%d sent=%d",
				pinned, countPins(sender.requested), len(sender.sent))
		}
	}
}

func TestRefreshBoard_SendsNewWhenDeleted(t *testing.T) {
	sender := &mockSender{requestErr: func(request tgbotapi.Chattable) error {
		if edit, ok := request.(tgbotapi.EditMessageTextConfig); ok && edit.MessageID == testBoardID {
			return &tgbotapi.Error{Code: 400, Message: "Bad Request: message to edit not found"}
		}
		return nil
	}}
	repository := &mockRepo{data: openGame(2)}
	newService(sender, repository, tuesday).RefreshBoard()

	if len(sender.sent) != 1 || repository.data.BoardMessageID != 101 || countPins(sender.requested) != 1 {
		t.Fatalf("удалённая доска отправляется заново и закрепляется: sent=%d id=%d pins=%d",
			len(sender.sent), repository.data.BoardMessageID, countPins(sender.requested))
	}
}

func TestRefreshBoard_OtherEditErrorKeepsBoard(t *testing.T) {
	sender := &mockSender{requestErr: func(request tgbotapi.Chattable) error {
		if _, ok := request.(tgbotapi.EditMessageTextConfig); ok {
			return errors.New("сеть")
		}
		return nil
	}}
	repository := &mockRepo{data: openGame(2)}
	newService(sender, repository, tuesday).RefreshBoard()

	if len(sender.sent) != 0 || repository.data.BoardMessageID != testBoardID {
		t.Fatalf("при сетевой ошибке новая доска не отправляется: sent=%d id=%d", len(sender.sent), repository.data.BoardMessageID)
	}
}
