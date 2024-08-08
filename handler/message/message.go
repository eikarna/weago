package message

import (
	"context"
	"fmt"

	"github.com/eikarna/weago/enums"
	"github.com/eikarna/weago/functions"
	"github.com/goccy/go-json"
	wa "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
	"strings"
)

func SendText(target types.JID, teks ...interface{}) (*wa.SendResponse, error) {
	msg := fmt.Sprintf(teks[0].(string), teks[1:]...)
	buildMsg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: &msg,
		},
	}
	resp, err := enums.Client.SendMessage(context.Background(), target, buildMsg)
	if err != nil {
		return nil, err
	}
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
	return &resp, nil
}

func SendQuoted(target types.JID, targetQuoted *waE2E.Message, teks ...interface{}) (*wa.SendResponse, error) {
	msg := fmt.Sprintf(teks[0].(string), teks[1:]...)
	buildMsg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: &msg,
			ContextInfo: &waE2E.ContextInfo{
				QuotedMessage: targetQuoted,
			},
		},
	}
	resp, err := enums.Client.SendMessage(context.Background(), target, buildMsg)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func CheckType(v *events.Message) enums.MType {
	if v.Message.GetExtendedTextMessage() != nil {
		return 0
	} else if v.Message.GetCommentMessage() != nil {
		return 0
	} else if *v.Message.Conversation != "" {
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
	fmt.Printf("V.Message: %s\n\n", msg.String())
	mtype := CheckType(v)
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
	if strings.Split(targetJid.String(), "@")[1] == "s.whatsapp.net" || strings.Split(targetJid.String(), "@")[1] == "g.us" {
		switch mtype {
		case enums.Text:
			teks := ""
			if msg.ExtendedTextMessage != nil {
				teks = *msg.ExtendedTextMessage.Text
			} else if *msg.Conversation != "" {
				teks = *msg.Conversation
			}
			switch teks {
			case "ping":
				SendQuoted(targetJid, msg, "pong! from *weago* btw..")
			default:
				// if targetJid.String() == enums.BotInfo.NumberJid.String() {
				cachedVal := enums.GetValueString(teks, enums.LLM)
				if cachedVal != "" {
					SendQuoted(targetJid, msg, cachedVal)
				} else {
					// Capture user message
					enums.AddMessage(targetJid, "user", teks)
					// Prepare json data body
					jsonData, err := json.Marshal(enums.ChatCache[targetJid.String()])
					if err != nil {
						return
					}

					responseText, err := functions.Post("https://curhat.yuv.workers.dev", jsonData)
					if err != nil {
						SendQuoted(targetJid, msg, err.Error())
						return
					}
					SendQuoted(targetJid, msg, responseText)
					enums.LLM[teks] = responseText

					// Capture assistant message
					enums.AddMessage(targetJid, "assistant", responseText)
				}
			}
			// }
		default:
			SendConversation(*enums.BotInfo.NumberJid, "Got Type Message: %d (%s),\n\nMessage Struct: %v", mtype, v.Info.MediaType, v)
		}
	}
}
