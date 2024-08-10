package message

import (
	"context"
	"fmt"

	"github.com/eikarna/weago/enums"
	"github.com/eikarna/weago/functions"
	"github.com/goccy/go-json"
	"go.mau.fi/whatsmeow"
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
			Text: proto.String(msg),
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

func SendImage(target types.JID, targetQuoted *waE2E.Message, buffer []byte, mimetype string, teks ...interface{}) (*wa.SendResponse, error) {
	msg := fmt.Sprintf(teks[0].(string), teks[1:]...)
	// Upload first
	resp, err := enums.Client.Upload(context.Background(), buffer, whatsmeow.MediaImage)
	if err != nil {
		return nil, err
	}
	buildMsg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Mimetype:      proto.String(mimetype),
			URL:           &resp.URL,
			DirectPath:    &resp.DirectPath,
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    &resp.FileLength,
			Caption:       proto.String(msg),
			ContextInfo: &waE2E.ContextInfo{
				QuotedMessage: targetQuoted,
			},
		},
	}
	respClient, err := enums.Client.SendMessage(context.Background(), target, buildMsg)
	if err != nil {
		return nil, err
	}
	return &respClient, nil
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
	} else if *v.Message.Conversation != "" {
		return 0
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
		// Get Group Info
		groupInfo, err := enums.Client.GetGroupInfo(targetJid)
		if err != nil {
			fmt.Errorf(err.Error())
		}
		if enums.GroupChat[targetJid.String()] == nil {
			enums.GroupChat[targetJid.String()] = &enums.ChatSettings{
				JID:            targetJid,
				OwnerJID:       groupInfo.OwnerJID,
				JIDString:      targetJid.String(),
				OwnerJIDString: groupInfo.OwnerJID.String(),
				Name:           groupInfo.GroupName.Name,
				UseAI:          false,
				IsPremium:      false,
				Limit:          10,
			}
		}
	} else {
		targetJid, err = types.ParseJID(functions.RemoveColonDigits(v.Info.Sender.String()))
		if err != nil {
			fmt.Errorf(err.Error())
		}
	}
	if strings.Split(targetJid.String(), "@")[1] == "s.whatsapp.net" || strings.Split(targetJid.String(), "@")[1] == "g.us" {
		// Get PushName
		var contactName types.ContactInfo
		var pName string
		if strings.Split(targetJid.String(), "@")[1] == "s.whatsapp.net" {
			contactName, err = enums.Client.Store.Contacts.GetContact(targetJid)
			if err != nil {
				fmt.Errorf(err.Error())
			}
			pName = contactName.PushName
		} else if strings.Split(targetJid.String(), "@")[1] == "g.us" {
			target, err := types.ParseJID(functions.RemoveColonDigits(v.Info.Sender.String()))
			if err != nil {
				fmt.Errorf(err.Error())
			}
			contactName, err = enums.Client.Store.Contacts.GetContact(target)
			if err != nil {
				fmt.Errorf(err.Error())
			}
			pName = contactName.PushName
		}
		// Load History if nil
		if enums.ChatCache[targetJid.String()] == nil {
			var tempCache map[string]interface{}
			tempCache, err = enums.GetChatData(targetJid.String())
			if err != nil && strings.HasPrefix(err.Error(), "failed to retrieve chat data") {
				SendQuoted(targetJid, msg, "*OH NO!* Something happened to your chat history! Contact Developer if this problem persists :(\n\n> %s", err.Error())
				return
			}
			if tempCache != nil {
				enums.ChatCache[targetJid.String()] = tempCache
			}
		}
		switch mtype {
		// Text Message
		case enums.Text:
			teks := ""
			if msg.ExtendedTextMessage != nil {
				teks = *msg.ExtendedTextMessage.Text
			} else if *msg.Conversation != "" {
				teks = *msg.Conversation
			}
			command := strings.ToLower(strings.Split(teks, " ")[0])
			args := strings.Split(teks, " ")[1:]
			switch command {
			case "ping":
				SendQuoted(targetJid, msg, "pong! from *weago* btw..")
			case "use-ai":
				enums.GroupChat[targetJid.String()].UseAI = true
				SendQuoted(targetJid, msg, "> *Aika* has been enabled for this chat!")
			case "disable-ai":
				enums.GroupChat[targetJid.String()].UseAI = false
				SendQuoted(targetJid, msg, "> *Aika* has been disabled for this chat!")
			case "db-info":
				if args[0] != "" {
					maps, err := enums.GetChatData(args[0])
					if err != nil {
						SendQuoted(targetJid, msg, err.Error())
						fmt.Errorf(err.Error())
						return
					}
					SendQuoted(targetJid, msg, "%#v", maps)
				}
			default:
				if enums.GroupChat[targetJid.String()] != nil && enums.GroupChat[targetJid.String()].UseAI {
					if len(teks) >= 9000 {
						SendQuoted(targetJid, msg, "Maaf ya! Aika gabisa nerima pesan yang panjang 😭")
						return
					}
					fmt.Printf("Display Name: %s\n\n", pName)
					enums.AddMessage(targetJid, "user", pName+": "+teks, nil, false)
					// Prepare json data body
					jsonData, err := json.Marshal(enums.ChatCache[targetJid.String()])
					if err != nil {
						fmt.Println(err.Error())
						return
					}

					responseText, err := functions.Post("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key="+enums.BotInfo.ApiKey, jsonData)
					if err != nil {
						fmt.Println(err.Error())
						SendQuoted(targetJid, msg, err.Error())
						return
					}
					if strings.ToLower(strings.TrimSpace(responseText)) != "disable_response" {
						SendQuoted(targetJid, msg, responseText)

						// Capture assistant message
						enums.AddMessage(targetJid, "model", responseText, nil, false)

						// Print Total TextMemory
						fmt.Printf("Text Memory:\n%v\n\n", enums.ChatCache[targetJid.String()])

						// After processing the message
						err = enums.SaveChatData(targetJid.String(), enums.ChatCache[targetJid.String()])
						if err != nil {
							fmt.Println("Error saving chat data:", err)
						}
					}
				}
			}
			// }
		// Image Message
		case enums.Image:
			teks := ""
			if msg.ImageMessage.Caption != nil {
				teks = *msg.ImageMessage.Caption
			}
			switch teks {
			case "uploadsw":
				target, err := types.ParseJID("status@broadcast")
				if err != nil {
					fmt.Errorf(err.Error())
					return
				}
				bufferData, err := enums.Client.DownloadAny(msg)
				if err != nil {
					fmt.Errorf(err.Error())
				}
				SendImage(target, nil, bufferData, "", "image/png", "Test Image")
			default:
				if enums.GroupChat[targetJid.String()] != nil && enums.GroupChat[targetJid.String()].UseAI {
					if len(teks) >= 9000 {
						SendQuoted(targetJid, msg, "Maaf ya! Aika gabisa nerima pesan yang panjang 😭")
						return
					}
					// Capture Image Buffer
					bufferData, err := enums.Client.DownloadAny(msg)
					if err != nil {
						fmt.Errorf(err.Error())
					}

					// Capture user message
					enums.AddMessage(targetJid, "user", pName+": "+teks, bufferData, true)
					// Prepare json data body
					jsonData, err := json.Marshal(enums.ChatCache[targetJid.String()])
					if err != nil {
						fmt.Println(err.Error())
						return
					}

					responseText, err := functions.Post("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key="+enums.BotInfo.ApiKey, jsonData)
					if err != nil {
						fmt.Println(err.Error())
						SendQuoted(targetJid, msg, err.Error())
						return
					}
					SendQuoted(targetJid, msg, clearText)

					// Capture assistant message
					enums.AddMessage(targetJid, "model", responseText, nil, false)

					// Print Total TextMemory
					fmt.Printf("(IMAGE) Text Memory:\n%v\n\n", enums.ChatCache[targetJid.String()])

					// After processing the message
					err = enums.SaveChatData(targetJid.String(), enums.ChatCache[targetJid.String()])
					if err != nil {
						fmt.Println("Error saving chat data:", err)
					}

				}
			}
		default:
			SendConversation(*enums.BotInfo.NumberJid, "Got Type Message: %d (%s),\n\nMessage Struct: %v", mtype, v.Info.MediaType, v)
		}
	}
}
