package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/eikarna/weago/enums"
	"github.com/eikarna/weago/functions"
	"github.com/eikarna/weago/handler/message"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func eventHandler(evt interface{}) {
	// Handle event
	switch v := evt.(type) {
	case *events.Message:
		message.MessageHandler(v)
	case *events.Receipt:
		fmt.Println("Received a receipt!")
	default:
		fmt.Printf("Unhandled event: %#v", v)
	}
}

func setupDatabase() error {
	var err error

	// Open Chat Database
	enums.ChatDB, err = sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on&mode=rwc&cache=shared&_sync=1", enums.BotInfo.ChatPath))
	if err != nil {
		return fmt.Errorf("failed to open chat database: %w", err)
	}

	return nil
}

func main() {
	// Load config.json
	err := enums.LoadConfig()
	if err != nil {
		panic(err)
	}
	// Setup Chat Database
	err = setupDatabase()
	if err != nil {
		panic(err)
	}

	dbLog := waLog.Stdout("Database", "DEBUG", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New("sqlite3", "file:database/session.db?_foreign_keys=on&mode=rwc&cache=shared&_sync=1", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	enums.Client = whatsmeow.NewClient(deviceStore, clientLog)
	enums.EventHandlerID = enums.Client.AddEventHandler(eventHandler)

	if enums.Client.Store.ID == nil {
		// No ID stored, new login
		// enable this if you want QR Pairing
		qrChan, _ := enums.Client.GetQRChannel(context.Background())
		err = enums.Client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				enums.Once.Do(func() {
					paircode, _ := enums.Client.PairPhone(enums.BotInfo.NumberString, true, 6, "Safari (IOS)")
					fmt.Println("PAIRING CODE:", paircode)
				})
				// fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = enums.Client.Connect()
		if err != nil {
			panic(err)
		}
		// Start AutoSave
		enums.Once.Do(functions.SaveSystem)
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	enums.Client.Disconnect()
}
