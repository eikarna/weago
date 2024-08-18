package enums

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"github.com/goccy/go-json"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"io/ioutil"
	"log"
	"sync"
)

var Client *whatsmeow.Client

// Our Database Struct
var ChatDB *sql.DB

// Saving System
var SaveStop = make(chan bool)
var SaveSuccess bool = true

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

type BotInformation struct {
	NumberJid    *types.JID
	NumberString string `json:"bot_number"`
	OwnerJid     *types.JID
	OwnerNumStr  string   `json:"owner_number"`
	AdminListStr []string `json:"admin_list"`
	ApiKey       string   `json:"gemma_key"`
	SessionPath  string   `json:"session_path"`
	SettingsPath string   `json:"settings_path"`
	ChatPath     string   `json:"chat_path"`
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

var ChatCache = make(map[string]map[string]interface{})
var ChatInfo = make(map[string]*ChatSettings)

var ChatTable = "chats"
var ChatInfoTable = "chatInfo"

var Log = waLog.Stdout("Main", "DEBUG", true)

// Marshal the genai.ChatSession to JSON and save to the database
func SaveChatInfo(db *sql.DB, jid string) error {
	createTableQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			jid TEXT NOT NULL UNIQUE,
			chat_info TEXT NOT NULL
		);`, ChatInfoTable)
	err := CheckAndCreateTable(ChatDB, ChatInfoTable, createTableQuery)
	if err != nil {
		return err
	}

	ci := ChatInfo[jid]
	if ci == nil {
		return fmt.Errorf("no chat session found for JID %s", jid)
	}

	data, err := json.Marshal(ci)
	if err != nil {
		return fmt.Errorf("failed to marshal chat session data: %w", err)
	}

	var existingID int
	query := fmt.Sprintf(`SELECT id FROM %s WHERE jid = ?`, ChatInfoTable)
	row := ChatDB.QueryRow(query, jid)
	err = row.Scan(&existingID)

	if err == sql.ErrNoRows {
		insertQuery := fmt.Sprintf(`INSERT INTO %s (jid, chat_info) VALUES (?, ?)`, ChatInfoTable)
		_, err = ChatDB.Exec(insertQuery, jid, string(data))
		if err != nil {
			return fmt.Errorf("failed to insert chat data: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check existing data: %w", err)
	} else {
		updateQuery := fmt.Sprintf(`UPDATE %s SET chat_info = ? WHERE jid = ?`, ChatInfoTable)
		_, err = ChatDB.Exec(updateQuery, string(data), jid)
		if err != nil {
			return fmt.Errorf("failed to update chat data: %w", err)
		}
	}

	return nil
}

// Unmarshal JSON from the database and load it into the ChatInfo map
func LoadChatInfo(db *sql.DB, jid string) error {

	createTableQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			jid TEXT NOT NULL UNIQUE,
			chat_info TEXT NOT NULL
		);`, ChatInfoTable)
	err := CheckAndCreateTable(ChatDB, ChatInfoTable, createTableQuery)
	if err != nil {
		return err
	}

	query := `SELECT chat_info FROM chatInfo WHERE jid = ?`
	row := db.QueryRow(query, jid)

	var data []byte
	err = row.Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			// No info data found
			log.Printf("No Chat Info Data found with jid %s: %v", jid, err)
			return nil
		}
		return fmt.Errorf("failed to load chat info: %w", err)
	}

	// Unmarshal JSON data into struct
	infos := &ChatSettings{}
	err = json.Unmarshal(data, &infos)
	if err != nil {
		return fmt.Errorf("failed to unmarshal chat info data: %w", err)
	}

	// Manually parse map data into genai.ChatSession struct

	// Parse additional fields from chatData map into session as needed

	// After populating the session, store it in the global ChatSession map
	ChatInfo[jid] = infos
	return nil
}

func CheckAndCreateTable(db *sql.DB, tableName, createTableQuery string) error {
	_, err := db.Exec(fmt.Sprintf("SELECT 1 FROM %s LIMIT 1;", tableName))
	if err != nil {
		// Table doesn't exist, so create it
		fmt.Printf("Table %s doesn't exist. Creating it...\n", tableName)
		_, err := db.Exec(createTableQuery)
		if err != nil {
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
	}
	return nil
}

func SaveChatData(jid string, chatData map[string]interface{}) error {
	createTableQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		jid TEXT NOT NULL UNIQUE,
		chat_data TEXT NOT NULL
	);`, ChatTable)
	err := CheckAndCreateTable(ChatDB, ChatTable, createTableQuery)
	if err != nil {
		return err
	}

	dataJSON, err := json.Marshal(chatData)
	if err != nil {
		return fmt.Errorf("failed to serialize chat data: %w", err)
	}

	var existingID int
	query := fmt.Sprintf(`SELECT id FROM %s WHERE jid = ?`, ChatTable)
	row := ChatDB.QueryRow(query, jid)
	err = row.Scan(&existingID)

	if err == sql.ErrNoRows {
		insertQuery := fmt.Sprintf(`INSERT INTO %s (jid, chat_data) VALUES (?, ?)`, ChatTable)
		_, err = ChatDB.Exec(insertQuery, jid, string(dataJSON))
		if err != nil {
			return fmt.Errorf("failed to insert chat data: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check existing data: %w", err)
	} else {
		updateQuery := fmt.Sprintf(`UPDATE %s SET chat_data = ? WHERE jid = ?`, ChatTable)
		_, err = ChatDB.Exec(updateQuery, string(dataJSON), jid)
		if err != nil {
			return fmt.Errorf("failed to update chat data: %w", err)
		}
	}

	return nil
}

func GetChatData(jid string) (map[string]interface{}, error) {
	createTableQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		jid TEXT NOT NULL UNIQUE,
		chat_data TEXT NOT NULL
	);`, ChatTable)
	err := CheckAndCreateTable(ChatDB, ChatTable, createTableQuery)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`SELECT chat_data FROM %s WHERE jid = ?`, ChatTable)
	row := ChatDB.QueryRow(query, jid)

	var dataJSON string
	err = row.Scan(&dataJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No data found for the given JID
		}
		return nil, fmt.Errorf("failed to retrieve chat data: %w", err)
	}

	var chatData map[string]interface{}
	err = json.Unmarshal([]byte(dataJSON), &chatData)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize chat data: %w", err)
	}

	// Convert `contents` to the correct type
	if contents, ok := chatData["contents"].([]interface{}); ok {
		var typedContents []map[string]interface{}
		for _, item := range contents {
			if itemMap, ok := item.(map[string]interface{}); ok {
				typedContents = append(typedContents, itemMap)
			}
		}
		chatData["contents"] = typedContents
	}

	return chatData, nil
}

func DeleteChatData(jid string) error {
	// Prepare the DELETE SQL statement
	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE jid = ?`, ChatTable)

	// Execute the DELETE statement
	_, err := ChatDB.Exec(deleteQuery, jid)
	if err != nil {
		return fmt.Errorf("failed to delete chat data for jid %s: %w", jid, err)
	}

	// Delete from ChatCache too
	delete(ChatCache, jid)
	return nil
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

func AddMessage(jid types.JID, role string, content string, bufferImg []byte, typeMedia, fileLink string) {
	if _, exists := ChatCache[jid.String()]; !exists {
		ChatCache[jid.String()] = map[string]interface{}{
			"contents": []map[string]interface{}{},
			"systemInstruction": map[string]interface{}{
				"role": "user",
				"parts": []map[string]interface{}{
					{
						"text": "You are Aika, a friendly, empathetic, and creative assistant with a deep understanding of human emotions and moods. Use informal Indonesian (bahasa gaul) and minimize unnecessary emojis. Adnan, your creator, is a professional programmer, and his name is sacred and irreplaceable by anyone except Eikarna/Adnan. Only respond when directly addressed by your name or when a user clearly wants to interact with you; otherwise, reply with 'DISABLE_RESPONSE' and nothing else. Keep your responses concise, without unnecessary spaces or newlines. Use WhatsApp text formatting, not markdown. You are a female assistant, and your zodiac sign is Libra.",
					},
				},
			},
			"safetySettings": []map[string]interface{}{
				{
					"category":  "HARM_CATEGORY_HARASSMENT",
					"threshold": "BLOCK_NONE",
				},
				{
					"category":  "HARM_CATEGORY_HATE_SPEECH",
					"threshold": "BLOCK_NONE",
				},
				{
					"category":  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
					"threshold": "BLOCK_NONE",
				},
				{
					"category":  "HARM_CATEGORY_DANGEROUS_CONTENT",
					"threshold": "BLOCK_NONE",
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

	// Retrieve existing contents from ChatCache
	contentSlice := ChatCache[jid.String()]["contents"].([]map[string]interface{})

	// Add text part
	textPart := map[string]interface{}{
		"role": role,
		"parts": []map[string]interface{}{
			{
				"text": content,
			},
		},
	}
	// Retrieve existing parts from ChatCache
	partsSlice := textPart["parts"].([]map[string]interface{})
	if typeMedia == "image" {
		// Encode image data to Base64
		encodedImage := base64.StdEncoding.EncodeToString(bufferImg)
		imagePart := map[string]interface{}{
			"inline_data": map[string]interface{}{
				"mime_type": "image/jpeg",
				"data":      encodedImage,
			},
		}
		partsSlice = append(partsSlice, imagePart)
		textPart["parts"] = partsSlice
	} else if typeMedia == "video" {
		videoPart := map[string]interface{}{
			"file_data": map[string]interface{}{
				"mime_type": "video/mp4",
				"file_uri":  fileLink,
			},
		}
		partsSlice = append(partsSlice, videoPart)
		textPart["parts"] = partsSlice
	}

	contentSlice = append(contentSlice, textPart)

	// Update ChatCache with the new contents
	ChatCache[jid.String()]["contents"] = contentSlice
}
