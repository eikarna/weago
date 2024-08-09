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
	"os"
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
		switch mtype {
		// Text Message
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
			case "use-ai":
				enums.GroupChat[targetJid.String()].UseAI = true
				SendQuoted(targetJid, msg, "> *Aika* has been enabled for this chat!")
			case "disable-ai":
				enums.GroupChat[targetJid.String()].UseAI = false
				SendQuoted(targetJid, msg, "> *Aika* has been disabled for this chat!")
			default:
				if enums.GroupChat[targetJid.String()] != nil && enums.GroupChat[targetJid.String()].UseAI {
					// if enums.SimilarLong(teks, "aika", 0.18) {
					// if targetJid.String() == enums.BotInfo.NumberJid.String() {
					fmt.Printf("Display Name: %s\n\n", pName)
					// cachedVal := enums.GetValueString(teks, enums.LLM)
					// if cachedVal != "" {
					// SendQuoted(targetJid, msg, cachedVal)
					// } else {
					// Capture user message
					enums.AddMessage(targetJid, "user", pName+": "+teks, "", false)
					// Prepare json data body
					jsonData, err := json.Marshal(enums.ChatCache[targetJid.String()])
					if err != nil {
						fmt.Println(err.Error())
						return
					}
					// SendQuoted(targetJid, msg, "%#v", enums.ChatCache[targetJid.String()])

					// responseText, err := functions.Post("https://curhat.yuv.workers.dev", jsonData)
					responseText, err := functions.Post("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key="+enums.BotInfo.ApiKey, jsonData)
					if err != nil {
						fmt.Println(err.Error())
						SendQuoted(targetJid, msg, err.Error())
						return
					}
					if strings.ToLower(strings.TrimSpace(responseText)) != "disable_response" {
						SendQuoted(targetJid, msg, responseText)

						// Capture assistant message
						enums.AddMessage(targetJid, "model", responseText, "", false)

						// Print Total TextMemory
						fmt.Printf("Text Memory:\n%v\n\n", enums.ChatCache[targetJid.String()])
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
					// fmt.Printf("Got Image Type: %#v")
					// Capture Image Buffer
					bufferData, err := enums.Client.DownloadAny(msg)
					if err != nil {
						fmt.Errorf(err.Error())
					}

					// Save the buffer to a temp file
					fileExtension := ".jpeg" // Adjust this based on your image type
					tempFilePath, err := functions.SaveBufferToTempFile(bufferData, fileExtension)
					if err != nil {
						fmt.Printf("Error saving buffer to temp file: %v\n", err)
						return
					}
					defer os.Remove(tempFilePath) // Clean up the temp file after use

					// Upload Image
					fileURI, err := functions.UploadImage(tempFilePath, "image/jpeg")
					if err != nil {
						fmt.Printf("Error uploading image: %v\n", err)
						return
					}

					fmt.Printf("File URI: %s\n", fileURI)

					// Capture user message
					enums.AddMessage(targetJid, "user", pName+": "+teks, fileURI, true)
					// Prepare json data body
					jsonData, err := json.Marshal(enums.ChatCache[targetJid.String()])
					if err != nil {
						fmt.Println(err.Error())
						return
					}

					// responseText, err := functions.Post("https://curhat.yuv.workers.dev", jsonData)
					responseText, err := functions.Post("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key="+enums.BotInfo.ApiKey, jsonData)
					if err != nil {
						fmt.Println(err.Error())
						SendQuoted(targetJid, msg, err.Error())
						return
					}
					SendQuoted(targetJid, msg, responseText)

					// Capture assistant message
					enums.AddMessage(targetJid, "model", responseText, "", false)

					// Print Total TextMemory
					fmt.Printf("(IMAGE) Text Memory:\n%v\n\n", enums.ChatCache[targetJid.String()])
				}
			}
		default:
			SendConversation(*enums.BotInfo.NumberJid, "Got Type Message: %d (%s),\n\nMessage Struct: %v", mtype, v.Info.MediaType, v)
		}
	}
}
