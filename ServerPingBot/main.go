package main

import (
	. "pingbot"
	"log"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"fmt"
	"settings"
	"time"
)

func main() {
	fmt.Print("Host address: ")
	hostAddr := ""
	fmt.Scanln(&hostAddr)
	if hostAddr == "" {
		hostAddr = settings.HostAddress
	}

	fmt.Print("Server URL: ")
	url := ""
	fmt.Scanln(&url)
	if url == "" {
		url = settings.ServerURL
	}

	fmt.Print("Response string: ")
	response := ""
	fmt.Scanln(&response)
	if response == "" {
		response = settings.ServerResponse
	}

	var delay time.Duration = 0
	fmt.Print("Delay in seconds: ")
	fmt.Scanln(&delay)
	if delay == 0 {
		delay = 300
	}

	pingBot := CreateBot(settings.BotToken, []*tgbotapi.User{&settings.Terana}, hostAddr, response, delay, url)
	defer func() {
		if err := recover(); err != nil {
			log.Printf("PingBot stopped because of panic %s\n", err)
			pingBot.ReportToMasters(fmt.Sprintf("PingBot error: %s", err))
			pingBot.ListenForUpdates()
		}
	}()

	pingBot.ListenForUpdates()
}
