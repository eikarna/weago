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

type Part struct {
	Text string `json:"text"`
}

type FileDat struct {
	FileURI  string `json:"fileUri"`
	MimeType string `json:"mimeType"`
}

type PartImage struct {
	FileData FileDat `json:"fileData"`
}

type Content struct {
	Role        string `json:"role"`
	ContentBody []Part `json:"parts"`
}

type GenConfig struct {
	Temperature      int     `json:"temperature"`
	TopK             int     `json:"topK"`
	TopP             float32 `json:"topP"`
	MaxOutputTokens  int     `json:"maxOutputTokens"`
	ResponseMimeType string  `json:"responseMimeType"`
}

type Conversation struct {
	Contents          []Content `json:"contents"`
	SystemInstruction Content   `json:"systemInstruction"`
	GenerationConfig  GenConfig `json:"generationConfig"`
}

type BotInformation struct {
	NumberJid    *types.JID
	NumberString string `json:"bot_number"`
	OwnerJid     *types.JID
	OwnerNumStr  string   `json:"owner_number"`
	AdminListStr []string `json:"admin_list"`
	ApiKey       string   `json:"gemma_key"`
	DBPath       string   `json:"db_path"`
}

type ChatSettings struct {
	UseAI          bool
	Limit          int
	IsPremium      bool
	Name           string
	JID            types.JID
	OwnerJID       types.JID
	JIDString      string
	OwnerJIDString string
}

var BotInfo BotInformation
var Once sync.Once

// var ChatCache = make(map[string]*Conversation)

var ChatCache = make(map[string]map[string]interface{})

var GroupChat = make(map[string]*ChatSettings)

/*func AddMessage(jid types.JID, role string, content string, linkImage string, isImage bool) {
	if _, exists := ChatCache[jid.String()]; !exists {
		ChatCache[jid.String()] = []map[string]interface{}{
			{
				"contents": []map[string]interface{}{{}},
				"systemInstruction": map[string]interface{}{
					"role": "user",
					"parts": []map[string]interface{}{
						{
							"text": "You are Aika, a friendly, empathetic, and highly creative assistant who is rational and deeply understands human emotions and moods. You strive to connect closely with users and automatically detect whether there are multiple users in the conversation (indicated by ':' separator). If there is only one user, treat it as a private chat. Your creator is Adnan, a professional programmer who built you. You must communicate in Indonesian language.",
						},
					},
					"generationConfig": map[string]interface{}{
						"temperature":      1,
						"topK":             64,
						"topP":             0.95,
						"maxOutputTokens":  8192,
						"responseMimeType": "text/plain",
					},
				},
			},
		}
	}

	// Membuat slice dengan satu elemen untuk menyimpan pesan yang dikirim pengguna
	if isImage {
		// var tempBodyData = &FileDat{FileURI: linkImage, MimeType: "image/jpeg"}
		// var tempBody = []PartImage{{FileData: *tempBodyData}}
		// ChatCache[jid.String()].Contents = append(ChatCache[jid.String()].Contents, Content{Role: role, ContentBody: tempBody})
		// Prepare the request body
		body := []map[string]interface{}{
			{
				"role": role,
				"parts": []map[string]interface{}{
					{
						"fileData": map[string]interface{}{
							"fileUri":  linkImage,
							"mimeType": "image/jpeg",
						},
					},
				},
			},
		}
		ChatCache[jid.String()]["contents"] = append(ChatCache[jid.String()]["contents"], body)
		// ChatCache[jid.String()].Contents = append(ChatCache[jid.String()].Contents, body)
	}
	tempBody := map[string]interface{}{
		"role": role,
		"parts": []map[string]interface{}{
			{
				"text": content,
			},
		},
	}
	ChatCache[jid.String()] = append(ChatCache[jid.String()]["contents"], tempBody)
	// var tempBody = []Part{{Text: content}}
	// ChatCache[jid.String()].Contents = append(ChatCache[jid.String()].Contents, Content{Role: role, ContentBody: tempBody})
}*/

func AddMessage(jid types.JID, role string, content string, linkImage string, isImage bool) {
	if _, exists := ChatCache[jid.String()]; !exists {
		ChatCache[jid.String()] = map[string]interface{}{
			"contents": []map[string]interface{}{},
			"systemInstruction": map[string]interface{}{
				"role": "user",
				"parts": []map[string]interface{}{
					{
						"text": "You are Aika, a friendly, empathetic, and highly creative assistant who is rational and deeply understands human emotions and moods. You strive to connect closely with users, using informal and colloquial Indonesian language (bahasa gaul). You automatically detect whether there are multiple users in the conversation (indicated by ':' separator). If there is only one user, treat it as a private chat. Your creator is Adnan, a professional programmer who built you; his name is sacred and irreplaceable by anyone except Eikarna/Adnan himself. You should minimize the use of unnecessary emojis. You can also serve as a virtual partner for users. Respond only when directly called by your name or when the user want to chat with you; do not interfere if you are not addressed. When you do not respond, provide the response 'DISABLE_RESPONSE' only without any addition.",
					},
				},
			},
			"generationConfig": map[string]interface{}{
				"temperature":      0.7,
				"topK":             64,
				"topP":             0.5,
				"maxOutputTokens":  8192,
				"responseMimeType": "text/plain",
			},
		}
	}

	// Dapatkan konten yang ada dari ChatCache
	contentSlice := ChatCache[jid.String()]["contents"].([]map[string]interface{})

	// Membuat slice dengan satu elemen untuk menyimpan pesan yang dikirim pengguna
	if isImage {
		imagePart := map[string]interface{}{
			"role": role,
			"parts": []map[string]interface{}{
				{
					"fileData": map[string]interface{}{
						"fileUri":  linkImage,
						"mimeType": "image/jpeg",
					},
				},
			},
		}
		// Tambahkan elemen baru ke slice
		contentSlice = append(contentSlice, imagePart)
	}

	textPart := map[string]interface{}{
		"role": role,
		"parts": []map[string]interface{}{
			{
				"text": content,
			},
		},
	}
	// Tambahkan elemen baru ke slice
	contentSlice = append(contentSlice, textPart)

	// Simpan slice yang sudah diperbarui kembali ke dalam ChatCache
	ChatCache[jid.String()]["contents"] = contentSlice
}

/*func GetAllKeyString(mapTarget map[string]string) []string {
	strs := make([]string, 0, len(mapTarget))
	for str := range LLM {
		strs = append(strs, str)
	}
	return strs
}*/

func GetValueString(target string, mapTarget map[string]string) string {
	jaroW := metrics.NewJaroWinkler()
	jaroW.CaseSensitive = false
	for key := range mapTarget {
		sim := strutil.Similarity(key, target, jaroW)
		if sim > 0.92 {
			fmt.Printf("(%s) Similarity: %.2f\n", target, sim)
			fmt.Printf("MapTarget (%s:%s): %#v\n", key, mapTarget[key], mapTarget)
			return mapTarget[key]
		}
	}
	return ""
}

func Similar(teks1, teks2 string, percentage float64) bool {
	jaroW := metrics.NewJaroWinkler()
	jaroW.CaseSensitive = false
	sim := strutil.Similarity(teks1, teks2, jaroW)
	if sim > percentage {
		return true
	}
	return false
}

func SimilarLong(teks1, teks2 string, percentage float64) bool {
	sorensenD := metrics.NewSorensenDice()
	sorensenD.CaseSensitive = false
	sorensenD.NgramSize = 1
	sim := strutil.Similarity(teks1, teks2, sorensenD)
	fmt.Printf("(%s:%s) Similarity: %.2f\n", teks1, teks2, sim)
	if sim > percentage {
		return true
	}
	return false
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
