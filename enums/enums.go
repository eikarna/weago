package enums

import (
	"go.mau.fi/whatsmeow"
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
