package bot

import (
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"volleyball_inline_bot/storage"
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

func TestRefreshBoard_SendsMissingGames(t *testing.T) {
	data := openGame(2)
	data.Games[0].MessageID = 0
	data.Games = append(data.Games, storage.Game{Date: "2026-11-09", MessageID: 0, Finished: true})
	data.Games = append(data.Games, storage.Game{Date: "2026-11-19", MessageID: 60})
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	newService(sender, repository, tuesday).RefreshBoard()

	if len(sender.sent) != 2 {
		t.Fatalf("отправляются только игры без сообщения, отправлено %d", len(sender.sent))
	}
	games := repository.data.Games
	if games[0].MessageID != 101 || games[1].MessageID != 102 || games[2].MessageID != 60 {
		t.Fatalf("номера новых сообщений сохраняются: %+v", games)
	}
	first := sender.sent[0].(tgbotapi.MessageConfig)
	if first.ParseMode != tgbotapi.ModeHTML || first.ReplyMarkup == nil {
		t.Errorf("сообщение игры с кнопками в HTML: %+v", first)
	}
	if repository.saves != 1 {
		t.Errorf("данные сохраняются один раз, saves=%d", repository.saves)
	}
}

func TestRefreshBoard_SendsMissingPaymentMessage(t *testing.T) {
	data := finishedGame()
	data.Games[0].MessageID = 0
	data.Games[0].Payment = &storage.Payment{RenterID: ivan.ID, RenterName: "Иван", Came: 4, PerPerson: 1000}
	sender := &mockSender{pinnedMessageID: testBoardID}
	repository := &mockRepo{data: data}
	newService(sender, repository, tuesday).RefreshBoard()

	if len(sender.sent) != 2 {
		t.Fatalf("ожидались сообщение игры и сообщение оплаты, отправлено %d", len(sender.sent))
	}
	payment := sender.sent[1].(tgbotapi.MessageConfig)
	if payment.ReplyToMessageID != 101 || repository.data.Games[0].Payment.MessageID != 102 {
		t.Fatalf("сообщение оплаты ответом на новое сообщение игры: %+v", payment)
	}
}
