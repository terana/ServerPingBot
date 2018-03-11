package pingbot

import (
	"github.com/tatsushid/go-fastping"
	"github.com/valyala/fasthttp"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"net"
	"log"
	"time"
	"fmt"
	"os"
	"strings"
)

type PingBot struct {
	*tgbotapi.BotAPI
	ServerURL           string
	ServerResponse      string
	HostAddress         string
	Masters             []*tgbotapi.User
	Delay               time.Duration
	Chats               map[int64]*tgbotapi.Chat
	IsControllingServer bool
	IsControllingHost   bool
	IsStarted           bool
}

func CreateBot(botToken string, masters []*tgbotapi.User,
	hostAddress string, serverResponse string, delay time.Duration, url string) *PingBot {
	var err error
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		panic(err)
	}

	bot.Self, err = bot.GetMe()
	if err != nil {
		panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)
	pingBot := PingBot{
		BotAPI:              bot,
		Masters:             masters,
		HostAddress:         hostAddress,
		ServerResponse:      serverResponse,
		Delay:               delay,
		ServerURL:           url,
		IsControllingServer: false,
		IsControllingHost:   false,
		IsStarted:           false,
		Chats:               make(map[int64]*tgbotapi.Chat),
	}

	return &pingBot
}

func (o *PingBot) ListenForUpdates() {
	updConfig := tgbotapi.NewUpdate(0)
	updConfig.Timeout = 60

	updates, err := o.GetUpdatesChan(updConfig)
	if err != nil {
		panic(err)
	}

	chMain := make(chan int, 1)
	chHost := make(chan int)
	chServer := make(chan int)
	for update := range updates {
		log.Println(update)
		msg := update.Message
		log.Println(msg)
		if msg == nil {
			continue
		}

		log.Printf("[%d] %s  %s", msg.From.ID, msg.From.UserName, msg.Text)
		o.ReportToMasters(fmt.Sprintf("[%s] %s", msg.From.UserName, msg.Text))

		if msg.IsCommand() {
			switch msg.Command() {
			case "hello":
				{
					o.AnswerHello(msg)
					continue
				}
			case "start":
				{
					if o.IsStarted == false {
						go o.StartDispatcher(chMain, chHost, chServer)
						o.IsStarted = true
					}

					if o.IsControllingHost == false {
						go o.ControlHost(chHost)
						o.IsControllingHost = true
					}

					if o.IsControllingServer == false {
						go o.ControlServer(chServer)
						o.IsControllingServer = true
					}
					continue
				}
			case "goodbye":
				{
					o.AnswerGoodbye(msg)
					continue
				}
			case "stop":
				{
					o.AnswerStop(msg)
					chMain <- 1
					continue
				}
			case "die":
				{
					o.AnswerDie(msg)
				}
			default:
				o.AnswerUnexpected(msg)
			}
		} else if o.IsMessageToMe(*msg) {
			o.AnswerSomething(msg)
		}
	}
}

func (o *PingBot) AnswerStop(msg *tgbotapi.Message) {
	o.ReportToEverybody(fmt.Sprintf("%s (%s) ordered to stop controlling server",
		msg.From.FirstName, msg.From.UserName))
}

func (o *PingBot) AnswerDie(msg *tgbotapi.Message) {
	o.ReportToEverybody(fmt.Sprintf("%s (%s) killed me",
		msg.From.FirstName, msg.From.UserName))
	os.Exit(0)
}

func (o *PingBot) AnswerHello(msg *tgbotapi.Message) {
	o.Chats[msg.Chat.ID] = msg.Chat
	msgConfig := tgbotapi.NewMessage(int64(msg.Chat.ID), "Well, hello there.")
	_, err := o.Send(msgConfig)
	if err != nil {
		log.Println("Error answering on hello: ", err)
	}
}

func (o *PingBot) AnswerGoodbye(msg *tgbotapi.Message) {
	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, "Bye =****")
	_, err := o.Send(msgConfig)
	if err != nil {
		log.Println("Error saing goodbye: ", err)
	}

	delete(o.Chats, msg.Chat.ID)
	o.ReportToMasters(fmt.Sprintf("Stopped to communicate with %s", msg.From.UserName))
}

func (o *PingBot) AnswerUnexpected(msg *tgbotapi.Message) {
	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, "It's impossible to understand you(")
	_, err := o.Send(msgConfig)
	if err != nil {
		log.Println("Error sending default message: ", err)
	}
}

func (o *PingBot) AnswerSomething(msg *tgbotapi.Message) {
	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, "I'm pinging the server...")
	_, err := o.Send(msgConfig)
	if err != nil {
		log.Println("Error answering on message to me: ", err)
	}
}

func (o *PingBot) StartDispatcher(chMain <-chan int, chHost chan int, chServer chan int) {
	defer func() {
		o.IsStarted = false
		o.IsControllingHost = false
		o.IsControllingServer = false
	}()

	for {
		select {
		case <-chMain:
			log.Println("Got cancel sinal from MAIN goroutine")
			chHost <- 1
			chServer <- 1
			o.IsControllingServer = false
			o.IsControllingHost = false
			o.ReportToEverybody("Stopping controlling server and host...")

		case <-chHost:
			log.Println("Got cancel sinal from HOST goroutine")
			o.IsControllingHost = false
			o.ReportToEverybody("Stopping controlling host...")

		case <-chServer:
			log.Println("Got cancel sinal from SERVER goroutine")
			o.IsControllingServer = false
			o.ReportToEverybody("Stopping controlling server...")
		}
	}
}

func (o *PingBot) ControlHost(ch chan int) {
	defer func() {
		if err := recover(); err != nil {
			o.ReportToEverybody(fmt.Sprintf("Something went wrong with host!\n Got ERROR: %s", err))
			ch <- 1
			return
		}
	}()

	pinger := fastping.NewPinger()
	pinger.MaxRTT = time.Second
	ipAddr, err := net.ResolveIPAddr("ip4:icmp", o.HostAddress)
	if err != nil {
		panic(err)
	}

	pinger.AddIPAddr(ipAddr)
	pinger.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		log.Printf("IP Addr: %s receive, RTT: %v\n", addr.String(), rtt)
	}
	firstTime := true
	var msg int
	for {
		err = pinger.Run()
		if err != nil {
			panic(err)
		}

		if firstTime {
			firstTime = false
			o.ReportToEverybody("Host is responding!")
		}

		select {
		case msg = <-ch:
			if msg != 0 {
				log.Println("Stopped controlling host")
				o.ReportToEverybody("Stopped controlling host")
				return
			}
		default:
		}

		time.Sleep(o.Delay * time.Second)
	}
}

func (o *PingBot) ControlServer(ch chan int) {
	defer func() {
		if err := recover(); err != nil {
			o.ReportToEverybody(fmt.Sprintf("Something went wrong with Server!\nGot ERROR: %s", err))
			ch <- 1
			return
		}
	}()
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod("GET")
	req.SetRequestURI(o.ServerURL)

	client := fasthttp.Client{Name: "PingBot"}
	firstTime := true
	var msg int

	for {
		err := client.Do(req, resp)
		if err != nil {
			panic(err)
		}

		if resp.Header.StatusCode() != 200 {
			panic("Server is down")
		}

		if string(resp.Body()) != o.ServerResponse {
			panic(fmt.Sprintf("Invalid responce: %s", resp.Body()))
		}

		if firstTime {
			firstTime = false
			o.ReportToEverybody("Server is working!")
		}

		log.Println("Server is working")

		select {
		case msg = <-ch:
			if msg != 0 {
				log.Println("Stopped controlling Server")
				o.ReportToEverybody("Stopped controlling Server")
				return
			}
		default:
		}

		time.Sleep(o.Delay * time.Second)
	}
}

func (o *PingBot) ReportToMasters(text string) {
	var allNames string
	for _, usr := range o.Masters {
		msgConfig := tgbotapi.NewMessage(int64(usr.ID), text)
		_, err := o.Send(msgConfig)
		if err != nil {
			log.Println(err)
		}
		allNames = strings.Join([]string{allNames, " "}, usr.UserName)
	}
	log.Printf("Report to masters (%s):\n %s\n", allNames, text)
}

func (o *PingBot) ReportToEverybody(text string) {
	o.ReportToMasters(text)
	for _, chat := range o.Chats {
		msgConfig := tgbotapi.NewMessage(chat.ID, text)
		_, err := o.Send(msgConfig)
		if err != nil {
			log.Println(err)
		}
	}
	log.Printf("Report to everybody:\n %s\n", text)
}
