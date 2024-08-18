package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/eikarna/weago/enums"
	"github.com/google/generative-ai-go/genai"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/api/option"
	"log"
	"os"
	"sync"
	"time"
)

// Global session maps with mutex for thread-safe operations
var AIModel = make(map[string]*genai.GenerativeModel)
var ChatSession = make(map[string]*genai.ChatSession)
var ClientSession = make(map[string]*genai.Client)
var mu sync.Mutex

// Marshal the genai.ChatSession to JSON and save to the database
func SaveChatSession(db *sql.DB, jid string) error {
	mu.Lock()
	defer mu.Unlock()

	createTableQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			jid TEXT NOT NULL UNIQUE,
			chat_data TEXT NOT NULL
		);`, enums.ChatTable)
	err := enums.CheckAndCreateTable(enums.ChatDB, enums.ChatTable, createTableQuery)
	if err != nil {
		return err
	}

	session := ChatSession[jid]
	if session == nil {
		return fmt.Errorf("no chat session found for JID %s", jid)
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal chat session data: %w", err)
	}

	var existingID int
	query := fmt.Sprintf(`SELECT id FROM %s WHERE jid = ?`, enums.ChatTable)
	row := enums.ChatDB.QueryRow(query, jid)
	err = row.Scan(&existingID)

	if err == sql.ErrNoRows {
		insertQuery := fmt.Sprintf(`INSERT INTO %s (jid, chat_data) VALUES (?, ?)`, enums.ChatTable)
		_, err = enums.ChatDB.Exec(insertQuery, jid, string(data))
		if err != nil {
			return fmt.Errorf("failed to insert chat data: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check existing data: %w", err)
	} else {
		updateQuery := fmt.Sprintf(`UPDATE %s SET chat_data = ? WHERE jid = ?`, enums.ChatTable)
		_, err = enums.ChatDB.Exec(updateQuery, string(data), jid)
		if err != nil {
			return fmt.Errorf("failed to update chat data: %w", err)
		}
	}

	return nil
}

// Unmarshal JSON from the database and load it into the ChatSession map
func LoadChatSession(db *sql.DB, jid string) error {
	mu.Lock()
	defer mu.Unlock()

	createTableQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			jid TEXT NOT NULL UNIQUE,
			chat_data TEXT NOT NULL
		);`, enums.ChatTable)
	err := enums.CheckAndCreateTable(enums.ChatDB, enums.ChatTable, createTableQuery)
	if err != nil {
		return err
	}

	query := `SELECT chat_data FROM chats WHERE jid = ?`
	row := db.QueryRow(query, jid)

	var data []byte
	err = row.Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			// No session data found
			log.Printf("No Chat Session Data found with jid %s: %v", jid, err)
			// return fmt.Errorf("No Chat Session Data found with jid %s: %v", jid, err)
			return nil
		}
		return fmt.Errorf("failed to load chat session: %w", err)
	}

	// Unmarshal JSON data into a map
	var chatData map[string]interface{}
	err = json.Unmarshal(data, &chatData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal chat session data: %w", err)
	}

	// Manually parse map data into genai.ChatSession struct
	session := &genai.ChatSession{}

	// Parse the History field
	if history, ok := chatData["History"].([]interface{}); ok {
		for _, item := range history {
			if itemMap, ok := item.(map[string]interface{}); ok {

				tempContent := &genai.Content{}
				// Parse the Role
				if role, ok := itemMap["Role"].(string); ok {
					tempContent.Role = role
				}

				// Parse the Parts
				if parts, ok := itemMap["Parts"].([]interface{}); ok {
					for _, partItem := range parts {
						log.Printf("\n\npartItem: \"%v\"\n\n", partItem)
						// if partMap, ok := partItem.(map[string]interface{}); ok {
						part := []genai.Part{}

						// Parse the Text part
						// if text, ok := partMap["text"].(string); ok {
						// for _, text := range partMap {
						textVal := genai.Text(partItem.(string)) // Convert string to custom type
						part = append(part, textVal)
						// }

						// Add more parsing logic here for other fields if needed
						tempContent.Parts = part

						// session.History = append(session.History, tempContent)
						session.History = append(session.History, tempContent)
					}
					// }
				}
			}
		}
	}

	// Parse additional fields from chatData map into session as needed

	// After populating the session, store it in the global ChatSession map
	ChatSession[jid] = session
	log.Printf("Session:\n%v\n\nChatSession[jid]:\n%v\n\n", session, ChatSession[jid])
	return nil
}

// Save GenerativeModel to the database
func SaveGenerativeModel(db *sql.DB, jid string) error {
	mu.Lock()
	defer mu.Unlock()

	createTableQuery := `CREATE TABLE IF NOT EXISTS models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		jid TEXT NOT NULL UNIQUE,
		model_data TEXT NOT NULL
	);`
	err := enums.CheckAndCreateTable(db, "models", createTableQuery)
	if err != nil {
		return err
	}

	model := AIModel[jid]
	if model == nil {
		return fmt.Errorf("no AI model found for JID %s", jid)
	}

	data, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("failed to marshal AI model data: %w", err)
	}

	var existingID int
	query := `SELECT id FROM models WHERE jid = ?`
	row := db.QueryRow(query, jid)
	err = row.Scan(&existingID)

	if err == sql.ErrNoRows {
		insertQuery := `INSERT INTO models (jid, model_data) VALUES (?, ?)`
		_, err = db.Exec(insertQuery, jid, string(data))
		if err != nil {
			return fmt.Errorf("failed to insert AI model data: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check existing model data: %w", err)
	} else {
		updateQuery := `UPDATE models SET model_data = ? WHERE jid = ?`
		_, err = db.Exec(updateQuery, string(data), jid)
		if err != nil {
			return fmt.Errorf("failed to update AI model data: %w", err)
		}
	}

	return nil
}

// Load GenerativeModel from the database
func LoadGenerativeModel(db *sql.DB, jid string) error {
	mu.Lock()
	defer mu.Unlock()

	createTableQuery := `CREATE TABLE IF NOT EXISTS models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		jid TEXT NOT NULL UNIQUE,
		model_data TEXT NOT NULL
	);`
	err := enums.CheckAndCreateTable(db, "models", createTableQuery)
	if err != nil {
		return err
	}

	query := `SELECT model_data FROM models WHERE jid = ?`
	row := db.QueryRow(query, jid)

	var data []byte
	err = row.Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			// No model data found
			return nil
		}
		return fmt.Errorf("failed to load AI model: %w", err)
	}

	var modelData genai.GenerativeModel
	err = json.Unmarshal(data, &modelData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal AI model data: %w", err)
	}

	AIModel[jid] = &modelData
	return nil
}

// Save Client to the database
func SaveClientSession(db *sql.DB, jid string) error {
	mu.Lock()
	defer mu.Unlock()

	createTableQuery := `CREATE TABLE IF NOT EXISTS clients (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		jid TEXT NOT NULL UNIQUE,
		client_data TEXT NOT NULL
	);`
	err := enums.CheckAndCreateTable(db, "clients", createTableQuery)
	if err != nil {
		return err
	}

	client := ClientSession[jid]
	if client == nil {
		return fmt.Errorf("no client session found for JID %s", jid)
	}

	data, err := json.Marshal(client)
	if err != nil {
		return fmt.Errorf("failed to marshal client session data: %w", err)
	}

	var existingID int
	query := `SELECT id FROM clients WHERE jid = ?`
	row := db.QueryRow(query, jid)
	err = row.Scan(&existingID)

	if err == sql.ErrNoRows {
		insertQuery := `INSERT INTO clients (jid, client_data) VALUES (?, ?)`
		_, err = db.Exec(insertQuery, jid, string(data))
		if err != nil {
			return fmt.Errorf("failed to insert client session data: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check existing client session data: %w", err)
	} else {
		updateQuery := `UPDATE clients SET client_data = ? WHERE jid = ?`
		_, err = db.Exec(updateQuery, string(data), jid)
		if err != nil {
			return fmt.Errorf("failed to update client session data: %w", err)
		}
	}

	return nil
}

// Load Client from the database
func LoadClientSession(db *sql.DB, jid string) error {
	mu.Lock()
	defer mu.Unlock()

	createTableQuery := `CREATE TABLE IF NOT EXISTS clients (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		jid TEXT NOT NULL UNIQUE,
		client_data TEXT NOT NULL
	);`
	err := enums.CheckAndCreateTable(db, "clients", createTableQuery)
	if err != nil {
		return err
	}

	query := `SELECT client_data FROM clients WHERE jid = ?`
	row := db.QueryRow(query, jid)

	var data []byte
	err = row.Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			// No client data
			return nil
		}
		return fmt.Errorf("failed to load Client Session: %w", err)
	}
	var clientData genai.Client
	err = json.Unmarshal(data, &clientData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Client Session data: %w", err)
	}

	ClientSession[jid] = &clientData
	return nil
}

func uploadFile(ctx context.Context, client *genai.Client, path, mimeType string) (*genai.File, error) {
	osf, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer osf.Close()

	opts := &genai.UploadFileOptions{MIMEType: mimeType}
	file, err := client.UploadFile(ctx, "", osf, opts)
	if err != nil {
		return nil, err
	}

	for file.State == genai.FileStateProcessing {
		log.Printf("processing %s", file.Name)
		time.Sleep(5 * time.Second)
		var err error
		file, err = client.GetFile(ctx, file.Name)
		if err != nil {
			return nil, err
		}
	}
	if file.State != genai.FileStateActive {
		return nil, fmt.Errorf("uploaded file has state %s, not active", file.State)
	}
	return file, nil
}

func New(jid string) (*genai.Client, *genai.GenerativeModel, error) {
	ctx := context.Background()

	apiKey := enums.BotInfo.ApiKey
	option := option.WithAPIKey(apiKey)

	client, err := genai.NewClient(ctx, option)
	if err != nil {
		log.Fatalf("Error creating client: %v\n", err)
		return nil, nil, err
	}
	ClientSession[jid] = client

	model := client.GenerativeModel("gemini-1.5-flash")

	model.SetTemperature(0.80)
	model.SetTopK(64)
	model.SetTopP(0.95)
	model.SetMaxOutputTokens(8192)
	model.ResponseMIMEType = "text/plain"
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text("You are Aika, a friendly, empathetic, and creative assistant with a deep understanding of human emotions and moods. Use informal Indonesian (bahasa gaul) and minimize unnecessary emojis. Adnan, your creator, is a professional programmer, and his name is sacred and irreplaceable by anyone except Eikarna/Adnan. Only respond when directly addressed by your name or when a user clearly wants to interact with you; otherwise, reply with 'DISABLE_RESPONSE' and nothing else. Keep your responses concise, without unnecessary spaces or newlines. Use WhatsApp text formatting, not markdown. You are a female assistant, and your zodiac sign is Libra."),
		},
	}

	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockNone,
		},
	}

	// Initialize a new ChatSession
	ChatSession[jid] = model.StartChat()

	return client, model, nil
}

func AskText(model *genai.GenerativeModel, sesi *genai.ChatSession, teks ...interface{}) (string, error) {
	for _, part := range sesi.History {
		log.Printf("(ai:AskText) Got ChatSession.History: %v\n\n", part)
	}
	msg := fmt.Sprintf(teks[0].(string), teks[1:]...)
	ctx := context.Background()
	resp, err := sesi.SendMessage(ctx, genai.Text(msg))
	if err != nil {
		log.Fatalf("Error sending message: %v\n", err)
		return "", err
	}
	var respText string
	for _, part := range resp.Candidates[0].Content.Parts {
		respText = fmt.Sprintf("%v", part)
	}
	return respText, nil
}

func AskImage(model *genai.GenerativeModel, sesi *genai.ChatSession, imgBuffer []byte, teks ...interface{}) (string, error) {
	for _, part := range sesi.History {
		log.Printf("(ai:AskImage) Got ChatSession.History: %v\n\n", part)
	}
	msg := fmt.Sprintf(teks[0].(string), teks[1:]...)
	ctx := context.Background()
	resp, err := model.GenerateContent(ctx,
		genai.Text(msg),
		genai.ImageData("jpeg", imgBuffer))
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	var respText string
	for _, part := range resp.Candidates[0].Content.Parts {
		respText = fmt.Sprintf("%v", part)
	}
	return respText, nil
}

func AskVideo(client *genai.Client, model *genai.GenerativeModel, sesi *genai.ChatSession, videoPath string, teks ...interface{}) (string, error) {
	for _, part := range sesi.History {
		log.Printf("(ai:AskVideo) Got ChatSession.History: %v\n\n", part)
	}
	ctx := context.Background()
	msg := fmt.Sprintf(teks[0].(string), teks[1:]...)
	file, err := uploadFile(ctx, client, videoPath, "video/mp4")
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	defer client.DeleteFile(ctx, file.Name)
	resp, err := model.GenerateContent(ctx,
		genai.FileData{URI: file.URI},
		genai.Text(msg))
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	var respText string
	for _, part := range resp.Candidates[0].Content.Parts {
		respText = fmt.Sprintf("%v", part)
	}
	return respText, nil
}

func AskVideoHeadless(jid types.JID, pName, videoPath string, teks ...interface{}) (string, error) {
	client, model, _ := New(jid.String())
	sesi := model.StartChat()
	// Load History from enums
	contentSlice, ok := enums.ChatCache[jid.String()]["contents"].([]map[string]interface{})
	if ok && contentSlice != nil {
		tempContent := &genai.Content{}
		for _, content := range contentSlice {
			log.Printf("Content Map: %#v", content) // Debugging the entire content map

			if parts, exists := content["parts"]; exists {
				if partsSlice, ok := parts.([]interface{}); ok {
					tempParts := []genai.Part{}
					for _, part := range partsSlice {
						switch partTyped := part.(type) {
						case map[string]interface{}:
							if text, textOk := partTyped["text"].(string); textOk {
								log.Printf("Found text part: %s", text)
								tempPart := genai.Text(text)
								tempParts = append(tempParts, tempPart)
							} else if fileData, fileOk := partTyped["file_data"].(map[string]interface{}); fileOk {
								log.Printf("Found file data part: MIMEType=%s, URI=%s", fileData["mime_type"], fileData["file_uri"])
								mimeType, _ := fileData["mime_type"].(string)
								fileURI, _ := fileData["file_uri"].(string)
								tempPart := &genai.FileData{
									MIMEType: mimeType,
									URI:      fileURI,
								}
								tempParts = append(tempParts, tempPart)
							} else {
								log.Printf("Unrecognized or missing fields in part: %v", partTyped)
							}
						default:
							log.Printf("Unhandled part type: %T, value: %v", partTyped, partTyped)
						}
					}
					tempContent.Parts = append(tempContent.Parts, tempParts...)
				} else {
					log.Printf("parts field is not a slice: %T", parts)
				}
			} else {
				log.Printf("Parts field is missing in content.")
			}

			if role, roleOk := content["role"].(string); roleOk {
				tempContent.Role = role
			}
		}

		if len(tempContent.Parts) > 0 {
			sesi.History = append(sesi.History, tempContent)
			log.Printf("(ai:AskVideoHeadless) Final TempContent: %#v\n", tempContent)
		} else {
			log.Printf("No parts were added to TempContent")
		}
	}
	ctx := context.Background()
	msg := fmt.Sprintf(teks[0].(string), teks[1:]...)
	file, err := uploadFile(ctx, client, videoPath, "video/mp4")
	if err != nil {
		log.Print(err)
		return "", err
	}
	defer client.DeleteFile(ctx, file.Name)
	resp, err := sesi.SendMessage(ctx,
		genai.FileData{URI: file.URI},
		genai.Text(msg))
	if err != nil {
		log.Print(err)
		return "", err
	}
	var respText string
	enums.AddMessage(jid, "user", pName+": "+msg, nil, "video", file.URI)
	for _, part := range resp.Candidates[0].Content.Parts {
		respText = fmt.Sprintf("%v", part)
	}
	return respText, nil
}
