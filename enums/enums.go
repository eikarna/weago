package enums

import (
	"fmt"
	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/goccy/go-json"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"io/ioutil"
	"sync"
)

var Client *whatsmeow.Client

var EventHandlerID uint32 = 0

type MType int

const (
	Text MType = iota
	Image
	Audio
	Video
	Document
	Location
	Contact
	Sticker
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Conversation struct {
	Messages []Message `json:"messages"`
}

type BotInformation struct {
	NumberJid    *types.JID
	NumberString string `json:"bot_number"`
	DBPath       string `json:"db_path"`
}

var BotInfo BotInformation
var LLM = make(map[string]string)
var Once sync.Once

var ChatCache = make(map[string]*Conversation, 1)

func AddMessage(jid types.JID, role string, content string) {
	if _, exists := ChatCache[jid.String()]; !exists {
		ChatCache[jid.String()] = &Conversation{}
		ChatCache[jid.String()].Messages = append(ChatCache[jid.String()].Messages, Message{Role: "system", Content: "You are a friendly and empathetic assistant who is very creative and rational. You understand human emotions and moods well, and you always strive to connect closely with users. You must communicate in Indonesian language."})
	}
	ChatCache[jid.String()].Messages = append(ChatCache[jid.String()].Messages, Message{Role: role, Content: content})
}

func GetAllKeyString(mapTarget map[string]string) []string {
	strs := make([]string, 0, len(mapTarget))
	for str := range LLM {
		strs = append(strs, str)
	}
	return strs
}

func GetValueString(target string, mapTarget map[string]string) string {
	jaroW := metrics.NewJaroWinkler()
	jaroW.CaseSensitive = false
	for key := range mapTarget {
		sim := strutil.Similarity(key, target, jaroW)
		if sim > 0.90 {
			fmt.Printf("(%s) Similarity: %.2f\n", target, sim)
			fmt.Printf("MapTarget (%s:%s): %#v\n", key, mapTarget[key], mapTarget)
			return mapTarget[key]
		}
	}
	return ""
}

func LoadConfig() error {
	file, err := ioutil.ReadFile("./config.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(file, &BotInfo)
	if err != nil {
		return err
	}
	if BotInfo.NumberJid == nil {
		parsedJid, err := types.ParseJID(BotInfo.NumberString + "@s.whatsapp.net")
		if err != nil {
			return err
		}
		BotInfo.NumberJid = &parsedJid
	}
	return nil
}
