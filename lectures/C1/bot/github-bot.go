package main

import (
	"encoding/json"
	"fmt"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
	"net/http"
	"sync"
)

const (
	BotToken = "888183279:AAE12HB93EyAPEtxZ4h1_Y98j48rUqPAJQI"
)

var MU = sync.RWMutex{}
var chatIds = make([]int64, 0, 8)

type BotHandler struct {
	bot *tgbotapi.BotAPI
}

func (bh BotHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	msg := make(map[string]interface{})
	err := decoder.Decode(&msg)
	if err != nil {
		http.Error(rw, "can't read body", 500)
		return
	}

	MU.RLock()
	for _, chatId := range chatIds {
		if _, err := bh.bot.Send(tgbotapi.NewMessage(chatId, msg["zen"].(string))); err != nil {
			fmt.Println(err)
		}
	}
	MU.RUnlock()

}

func main() {
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		panic(err)
	}
	response, err := bot.RemoveWebhook()
	if err != nil {
		panic(err)
	}

	fmt.Println(response.Description)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	http.Handle("/", BotHandler{bot: bot})
	go http.ListenAndServe(":8080", nil)
	fmt.Println("server started")

	for update := range updates {
		if "start" == update.Message.Text {
			MU.Lock()
			chatIds = append(chatIds, update.Message.Chat.ID)
			MU.Unlock()
			fmt.Println("add new chat")
		}
	}
}
