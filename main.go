package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/eikarna/weago/enums"
	"github.com/eikarna/weago/handler/message"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func setupDatabase() error {
	var err error

	// Open Chat Database
	enums.ChatDB, err = sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on&mode=rwc&cache=shared&_sync=1", enums.BotInfo.ChatPath))
	if err != nil {
		return fmt.Errorf("failed to open chat database: %w", err)
	}

	// Open Settings Database
	enums.SettingDB, err = sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on&mode=rwc&cache=shared&_sync=1", enums.BotInfo.SettingsPath))
	if err != nil {
		return fmt.Errorf("failed to open settings database: %w", err)
	}

	/* Check and create tables if they do not exist
		chatTableQuery := `
	CREATE TABLE IF NOT EXISTS chats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		jid TEXT NOT NULL,
		chat_data TEXT NOT NULL
	);`
		settingsTableQuery := `
	CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		jid TEXT NOT NULL,
		owner_jid TEXT NOT NULL,
		settings_data TEXT NOT NULL,
		use_ai BOOLEAN DEFAULT 0,
		name TEXT NOT NULL,
		is_premium BOOLEAN DEFAULT 0,
		limit_value INTEGER DEFAULT 10
	);`

		if err := enums.CheckAndCreateTable(enums.ChatDB, enums.ChatTable, chatTableQuery); err != nil {
			return fmt.Errorf("failed to check and create chat table: %w", err)
		}
		if err := enums.CheckAndCreateTable(enums.SettingDB, enums.SettingsTable, settingsTableQuery); err != nil {
			return fmt.Errorf("failed to check and create settings table: %w", err)
		}*/

	return nil
}

func setupWhatsMeow() (*whatsmeow.Client, error) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on&mode=rwc&cache=shared&_sync=1", enums.BotInfo.SessionPath), dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQLStore container: %w", err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, fmt.Errorf("failed to get first device: %w", err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	enums.Client = client
	return client, nil
}

func handleQR(client *whatsmeow.Client) {
	qrChan, _ := client.GetQRChannel(context.Background())
	for evt := range qrChan {
		if evt.Event == "code" {
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			enums.Once.Do(func() {
				paircode, _ := client.PairPhone(enums.BotInfo.NumberString, true, 6, "Safari (IOS)")
				fmt.Println("PAIRING CODE:", paircode)
			})
		} else {
			fmt.Println("Login event:", evt.Event)
		}
	}
}

func main() {
	// Load configuration
	if err := enums.LoadConfig(); err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Setup databases
	if err := setupDatabase(); err != nil {
		fmt.Printf("Error setting up database: %v\n", err)
		os.Exit(1)
	}

	/* Load all saved chat and settings data
	if err := enums.LoadChatAndSettingsData(); err != nil {
		fmt.Printf("Error loading chat and settings data: %v\n", err)
		os.Exit(1)
	}*/

	// Setup WhatsApp client
	client, err := setupWhatsMeow()
	if err != nil {
		fmt.Printf("Error setting up WhatsApp client: %v\n", err)
		os.Exit(1)
	}

	enums.EventHandlerID = client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			message.MessageHandler(v)
		case *events.Receipt:
			fmt.Printf("Received a receipt!, %#v\n\n", v)
		default:
			fmt.Printf("Unhandled event: %#v", v)
		}
	})

	if client.Store.ID == nil {
		// New login, handle QR
		if err := client.Connect(); err != nil {
			fmt.Printf("Error connecting client: %v\n", err)
			os.Exit(1)
		}
		handleQR(client)
	} else {
		// Already logged in, just connect
		if err := client.Connect(); err != nil {
			fmt.Printf("Error connecting client: %v\n", err)
			os.Exit(1)
		}
	}

	// Wait for interrupt signal to gracefully shut down
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
