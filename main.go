package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"golang.org/x/net/proxy"
)

var (
	socksLoc = os.Getenv("SOCKS5_URL")
	botToken = os.Getenv("TGBOT_TOKEN")
)

type meta struct {
	replyTo int
	chatID  int64
}

func main() {
	if botToken == "" {
		log.Fatal("set TGBOT_TOKEN environment variable")
	}

	var bot *tgbotapi.BotAPI
	var err error
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil && socksLoc != "" {
		u, err := url.Parse(socksLoc)
		if err != nil {
			log.Fatal(err)
		}

		client, err := socks5Client(u)
		if err != nil {
			log.Fatal(err)
		}

		bot, err = tgbotapi.NewBotAPIWithClient(botToken, client)
	}
	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		body, err := json.MarshalIndent(update, "", "  ")
		if err != nil {
			body = []byte(err.Error())
		}

		fb := tgbotapi.FileBytes{
			Name:  fmt.Sprintf("update-%d.json", update.UpdateID),
			Bytes: body,
		}

		m, err := newMeta(&update)
		if err != nil {
			os.Stdout.WriteString(string(body))
			continue
		}

		msg := tgbotapi.NewDocumentUpload(m.chatID, fb)
		msg.ReplyToMessageID = m.replyTo
		msg.ParseMode = tgbotapi.ModeMarkdown

		_, err = bot.Send(msg)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func socks5Client(u *url.URL) (*http.Client, error) {
	dialer, err := proxy.FromURL(u, proxy.Direct)
	if err != nil {
		return nil, err
	}

	return &http.Client{Transport: &http.Transport{Dial: dialer.Dial}}, nil
}

func newMeta(upd *tgbotapi.Update) (*meta, error) {
	if upd.Message != nil {
		return &meta{
			chatID:  upd.Message.Chat.ID,
			replyTo: upd.Message.MessageID,
		}, nil
	}

	if upd.EditedMessage != nil {
		return &meta{
			chatID:  upd.EditedMessage.Chat.ID,
			replyTo: upd.EditedMessage.MessageID,
		}, nil
	}

	if upd.ChannelPost != nil {
		return &meta{
			chatID:  upd.ChannelPost.Chat.ID,
			replyTo: upd.ChannelPost.MessageID,
		}, nil
	}

	if upd.EditedChannelPost != nil {
		return &meta{
			chatID:  upd.EditedChannelPost.Chat.ID,
			replyTo: upd.EditedChannelPost.MessageID,
		}, nil
	}

	if upd.CallbackQuery != nil {
		return &meta{
			chatID:  upd.CallbackQuery.Message.Chat.ID,
			replyTo: upd.CallbackQuery.Message.ReplyToMessage.MessageID,
		}, nil
	}

	return nil, errors.New("unsupported update type")
}
