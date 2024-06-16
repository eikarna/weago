package message

import (
	"context"
	"fmt"

	"github.com/eikarna/weago/enums"
	"github.com/eikarna/weago/functions"
	wa "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func SendText(target types.JID, teks ...interface{}) (*wa.SendResponse, error) {
	msg := fmt.Sprintf(teks[0].(string), teks[1:]...)
	buildMsg := waE2E.Message{}
	buildMsg.ExtendedTextMessage = &waE2E.ExtendedTextMessage{}
	buildMsg.ExtendedTextMessage.Text = &msg
	resp, err := enums.Client.SendMessage(context.Background(), target, &buildMsg)
	if err != nil {
		return nil, err
	}
	fmt.Println("SUCCESSFULLY SENT SENDTEXT!")
	return &resp, nil
}

func SendConversation(target types.JID, teks ...interface{}) (*wa.SendResponse, error) {
	msg := fmt.Sprintf(teks[0].(string), teks[1:]...)
	resp, err := enums.Client.SendMessage(context.Background(), target, &waE2E.Message{
		Conversation: proto.String(msg),
	})
	if err != nil {
		return nil, err
	}
	fmt.Println("SUCCESSFULLY SENT SENDTEXT!")
	return &resp, nil
}

func CheckType(v *events.Message) enums.MType {
	if v.Message.GetExtendedTextMessage() != nil {
		return 0
	} else if v.Message.GetCommentMessage() != nil {
		return 0
	} else if v.Message.GetImageMessage() != nil { // Not TextMessage
		return 1
	} else if v.Message.GetAudioMessage() != nil {
		return 2
	} else if v.Message.GetVideoMessage() != nil {
		return 3
	} else if v.Message.GetDocumentMessage() != nil {
		return 4
	} else if v.Message.GetLocationMessage() != nil {
		return 5
	} else if v.Message.GetContactMessage() != nil {
		return 6
	} else if v.Message.GetStickerMessage() != nil {
		return 7
	}
	return 0
}

func MessageHandler(v *events.Message) {
	msg := v.Message
	fmt.Printf("Msg: #%v\n", msg)
	mtype := CheckType(v)
	fmt.Printf("Successfully checked the Message, got type: %d\n", mtype)
	var targetJid types.JID
	var err error
	// Removing some :xx betweeen phone number and @s.whatsapp.net
	if v.Info.IsGroup {
		targetJid, err = types.ParseJID(functions.RemoveColonDigits(v.Info.Chat.String()))
		if err != nil {
			fmt.Errorf(err.Error())
		}
	} else {
		targetJid, err = types.ParseJID(functions.RemoveColonDigits(v.Info.Sender.String()))
		if err != nil {
			fmt.Errorf(err.Error())
		}
	}
	fmt.Println("Got Target JID:", targetJid)
	switch mtype {
	case enums.Text:
		teks := msg.ExtendedTextMessage.GetText()
		fmt.Printf("Got ExtendedTextMessage from %v: %s\n", targetJid, teks)
		switch teks {
		case "ping":
			SendConversation(targetJid, "pong! from *weago* btw..")
		default:
			SendText(targetJid, "I don't know that command")
		}
	default:
		SendConversation(targetJid, "Got Type Message: %d (%s),\n\nMessage Struct: %v", mtype, v.Info.MediaType, v)
	}
}
