package enums

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"github.com/goccy/go-json"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"io/ioutil"
	"strings"
	"sync"
)

var Client *whatsmeow.Client

// Our Database Struct
var ChatDB *sql.DB
var SettingDB *sql.DB

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
var GroupChat = make(map[string]*ChatSettings)

var ChatTable = "chats"
var SettingsTable = "settings"

func CheckAndCreateTable(db *sql.DB, tableName, createTableQuery string) error {
	_, err := db.Exec(fmt.Sprintf("SELECT 1 FROM %s LIMIT 1;", tableName))
	if err != nil {
		// Table doesn't exist, so create it
		fmt.Printf("Table %s doesn't exist. Creating it...\n", tableName)
		_, err := db.Exec(createTableQuery)
		if err != nil {
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
		fmt.Printf("Table %s created successfully.\n", tableName)
	} else {
		fmt.Printf("Table %s exists. Using it...\n", tableName)
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

func SaveSettingsData(jid string, settings *ChatSettings) error {
	tableName := getTableNameFromJID(jid)
	// Sanitize table name
	tableName = sanitizeTableName(tableName)

	// Ensure the table for this JID exists
	createTableQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		use_ai BOOLEAN,
		limit_value INTEGER,
		is_premium BOOLEAN,
		name TEXT,
		jid TEXT UNIQUE,
		owner_jid TEXT
	);`, tableName)
	err := CheckAndCreateTable(SettingDB, tableName, createTableQuery)
	if err != nil {
		return err
	}

	// Insert the settings data into the specific table
	query := fmt.Sprintf(`INSERT OR REPLACE INTO %s (use_ai, limit_value, is_premium, name, jid, owner_jid) VALUES (?, ?, ?, ?, ?, ?)`, tableName)
	_, err = SettingDB.Exec(query, settings.UseAI, settings.Limit, settings.IsPremium, settings.Name, settings.JID.String(), settings.OwnerJID.String())
	if err != nil {
		return fmt.Errorf("failed to save settings data: %w", err)
	}

	return nil
}

func GetSettingsData(jid string) (*ChatSettings, error) {
	tableName := getTableNameFromJID(jid)
	tableName = sanitizeTableName(tableName)

	// Query for settings data based on JID
	query := fmt.Sprintf(`SELECT use_ai, limit_value, is_premium, name, jid, owner_jid FROM %s WHERE jid = ?`, tableName)
	row := SettingDB.QueryRow(query, jid)

	// Variables to store the query result
	var useAI bool
	var limit int
	var isPremium bool
	var name, jidStr, ownerJidStr string

	// Scan the row into the variables
	err := row.Scan(&useAI, &limit, &isPremium, &name, &jidStr, &ownerJidStr)
	if err == sql.ErrNoRows {
		// Handle case where no rows are returned
		return nil, fmt.Errorf("no settings data found for jid %s", jid)
	} else if err != nil {
		return nil, fmt.Errorf("failed to retrieve settings data: %w", err)
	}

	// Parse JIDs from strings
	jidObj, err := types.ParseJID(jidStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JID: %w", err)
	}

	ownerJidObj, err := types.ParseJID(ownerJidStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Owner JID: %w", err)
	}

	return &ChatSettings{
		UseAI:          useAI,
		Limit:          limit,
		IsPremium:      isPremium,
		Name:           name,
		JID:            jidObj,
		OwnerJID:       ownerJidObj,
		JIDString:      jidStr,
		OwnerJIDString: ownerJidStr,
	}, nil
}

func sanitizeTableName(name string) string {
	// Replace invalid characters with underscores
	return strings.Map(func(r rune) rune {
		if r == '@' || r == '-' || r == '.' || r == ':' || r == '/' {
			return '_'
		}
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			return r
		}
		return -1
	}, name)
}

func getTableNameFromJID(jid string) string {
	baseName := strings.Split(jid, "@")[0]
	return sanitizeTableName(baseName)
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

func AddMessage(jid types.JID, role string, content string, bufferImg []byte, isImage bool) {
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

	/* Add image part if necessary
	if isImage {
		/*imagePart := map[string]interface{}{
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
		contentSlice = append(contentSlice, imagePart)
	}*/

	// Add text part
	textPart := map[string]interface{}{
		"role": role,
		"parts": []map[string]interface{}{
			{
				"text": content,
			},
		},
	}

	if isImage {
		// Retrieve existing parts from ChatCache
		partsSlice := textPart["parts"].([]map[string]interface{})
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
	}

	contentSlice = append(contentSlice, textPart)

	// Update ChatCache with the new contents
	ChatCache[jid.String()]["contents"] = contentSlice
}
