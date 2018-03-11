package main

import (
	. "pingbot"
	"log"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"fmt"
)

const BotToken = "504698082:AAGZ8xxIEGLfGQi5TIiMpkLraKfSBjdVHeY"

var (
	terana = tgbotapi.User{
		ID:        216762478,
		FirstName: "Anastasia",
		LastName:  "Terenteva",
		UserName:  "im_terana"}
)

func main() {
	pingBot := CreateBot(BotToken, []*tgbotapi.User{&terana})
	defer func() {
		if err := recover(); err != nil {

			log.Printf("PingBot stopped because of panic %s\n", err)
			pingBot.ReportToMasters(fmt.Sprintf("PingBot is panicing: %s", err))
			pingBot.ListenForUpdates()
		}
	}()

	pingBot.ListenForUpdates()
}
