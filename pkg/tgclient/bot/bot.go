package bot

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/KSerditov/Trading/pkg/broker/orders"
	"github.com/KSerditov/Trading/pkg/tgclient/botuser"
	"github.com/KSerditov/Trading/pkg/tgclient/brokerclient"
	"github.com/KSerditov/Trading/pkg/tgclient/oauth"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type TgBot struct {
	BotToken         string
	Debug            bool
	AuthProvider     oauth.OauthProvider
	BrokerClient     brokerclient.BrokerClient
	TgUserRepository botuser.TgUserRepository

	bot *tgbotapi.BotAPI
}

func (t *TgBot) ListenAndServe() {
	bot, err := tgbotapi.NewBotAPI(t.BotToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = t.Debug
	t.bot = bot

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := t.bot.GetUpdatesChan(u)
	if err != nil {
		fmt.Printf("failed to get updates channel: %v", err)
	}

	for update := range updates {
		fmt.Printf("NEW UPDATE: %v\n", update)
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		switch update.Message.Command() {
		case "start":
			t.Start(update)
		case "register":
			t.Register(update)
		case "history":
			t.History(update)
		case "positions":
			t.Positions(update)
		case "buy":
			t.AddDeal(update, true)
		case "sell":
			t.AddDeal(update, false)
		case "cancel":
			t.CancelDeal(update)
		default:
			t.Default(update)
		}
	}
}

func (t *TgBot) Start(upd tgbotapi.Update) {
	var msg tgbotapi.MessageConfig
	if upd.Message.CommandArguments() == "" {
		t.sendAuthMsg(upd.Message.Chat.ID)
		return
	}

	token, err := t.AuthProvider.GetToken(upd.Message.CommandArguments())
	if err != nil {
		errmsg := tgbotapi.NewMessage(upd.Message.Chat.ID, fmt.Sprintf("authentication error %v", err.Error()))
		t.bot.Send(errmsg)
		return
	}

	fmt.Printf("TOKEN:\n%v\n", token.AccessToken)
	_, adderr := t.TgUserRepository.AddUser(
		upd.Message.Chat.ID,
		token,
		t.AuthProvider.GetProviderName(),
	)
	if adderr != nil {
		text := fmt.Sprintf("Authentication failed: %v", adderr.Error())
		msg = tgbotapi.NewMessage(upd.Message.Chat.ID, text)
		t.bot.Send(msg)
		return
	}

	text := fmt.Sprintf("Hi, userid: %v email: %v", token.Extra("user_id"), token.Extra("email"))
	msg = tgbotapi.NewMessage(upd.Message.Chat.ID, text)
	t.bot.Send(msg)
}

func (t *TgBot) History(upd tgbotapi.Update) {
	ticker := strings.TrimSpace(upd.Message.CommandArguments())

	u, err := t.TgUserRepository.GetUser(upd.Message.Chat.ID)
	if err != nil {
		t.sendAuthMsg(upd.Message.Chat.ID)
		return
	}

	history, err := t.BrokerClient.History(ticker, u.Fquid)
	if err != nil {
		t.sendError(upd.Message.Chat.ID, err)
		return
	}
	msg := tgbotapi.NewMessage(upd.Message.Chat.ID, upd.Message.Text)
	msg.ParseMode = "HTML"

	builder := strings.Builder{}
	builder.WriteString("<pre>\n")
	builder.WriteString("|    Time      |  Open  |  High  |  Low  |  Close  |Volume|\n")
	for i, v := range history.Body.Prices {
		builder.WriteString(fmt.Sprintf("| %v | %v | %v | %v | %v | %v |\n", time.Unix(int64(v.Time), 0).Format("Jan _2 15:04:05.000"), v.Open, v.High, v.Low, v.Close, v.Volume))
		if i > 9 {
			break
		}
	}
	builder.WriteString("</pre>")
	msg.Text = builder.String()
	t.bot.Send(msg)
}

func (t *TgBot) Positions(upd tgbotapi.Update) {
	u, err := t.TgUserRepository.GetUser(upd.Message.Chat.ID)
	if err != nil {
		t.sendAuthMsg(upd.Message.Chat.ID)
		return
	}

	positions, err := t.BrokerClient.Positions(u.Fquid)
	if err != nil {
		t.sendError(upd.Message.Chat.ID, err)
		return
	}
	msg := tgbotapi.NewMessage(upd.Message.Chat.ID, upd.Message.Text)

	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Balance: %v\n", positions.Body.Balance))
	builder.WriteString("Positions:\n")
	for _, v := range positions.Body.Positions {
		builder.WriteString(fmt.Sprintf("T: %v V: %v\n", v.Ticker, v.Volume))
	}
	builder.WriteString("Open deals:\n")
	for _, v := range positions.Body.OpenOrders {
		builder.WriteString(fmt.Sprintf("T: %v P: %v T: %v V: %v\n", v.Ticker, v.Price, v.Type, v.Volume))
	}
	msg.Text = builder.String()
	t.bot.Send(msg)
}

func (t *TgBot) Register(upd tgbotapi.Update) {
	u, err := t.TgUserRepository.GetUser(upd.Message.Chat.ID)
	if err != nil {
		t.sendAuthMsg(upd.Message.Chat.ID)
		return
	}

	err = t.BrokerClient.Register(u.Fquid)
	if err != nil {
		t.sendError(upd.Message.Chat.ID, err)
		return
	}
	msg := tgbotapi.NewMessage(upd.Message.Chat.ID, upd.Message.Text)
	msg.Text = fmt.Sprintf("User %v registered successfully", u.Fquid)
	t.bot.Send(msg)
}

func (t *TgBot) AddDeal(upd tgbotapi.Update, is_buy bool) {
	dealparams := strings.TrimSpace(upd.Message.CommandArguments())
	if dealparams == "" {
		t.sendError(upd.Message.Chat.ID, errors.New("use /buy Ticker Volume Price"))
		return
	}
	params := strings.Split(dealparams, " ")
	if len(params) != 3 {
		t.sendError(upd.Message.Chat.ID, errors.New("use /buy Ticker Volume Price"))
		return
	}
	volume, err := strconv.Atoi(params[1])
	if err != nil {
		t.sendError(upd.Message.Chat.ID, errors.New("valid int volume is required"))
		return
	}

	price, err := strconv.Atoi(params[2])
	if err != nil {
		t.sendError(upd.Message.Chat.ID, errors.New("valid int price is required"))
		return
	}

	d := &orders.Deal{
		Ticker: params[0],
		Volume: int32(volume),
		Price:  int32(price),
	}

	if is_buy {
		d.Type = "BUY"
	} else {
		d.Type = "SELL"
	}

	u, err := t.TgUserRepository.GetUser(upd.Message.Chat.ID)
	if err != nil {
		t.sendAuthMsg(upd.Message.Chat.ID)
		return
	}

	deal, err := t.BrokerClient.Deal(d, u.Fquid)
	if err != nil {
		t.sendError(upd.Message.Chat.ID, err)
		return
	}
	msg := tgbotapi.NewMessage(upd.Message.Chat.ID, upd.Message.Text)
	msg.Text = fmt.Sprintf("Deal placed to exchange with id: %v", deal.Body.ID)
	t.bot.Send(msg)
}

func (t *TgBot) CancelDeal(upd tgbotapi.Update) {
	deailidStr := strings.TrimSpace(upd.Message.CommandArguments())
	dealid, err := strconv.ParseInt(deailidStr, 10, 64)
	if err != nil || deailidStr == "" {
		t.sendError(upd.Message.Chat.ID, errors.New("valid int64 deal id is required"))
		return
	}

	u, err := t.TgUserRepository.GetUser(upd.Message.Chat.ID)
	if err != nil {
		t.sendAuthMsg(upd.Message.Chat.ID)
		return
	}

	cancelled, err := t.BrokerClient.Cancel(dealid, u.Fquid)
	if err != nil {
		t.sendError(upd.Message.Chat.ID, err)
		return
	}
	msg := tgbotapi.NewMessage(upd.Message.Chat.ID, upd.Message.Text)
	if cancelled {
		msg.Text = fmt.Sprintf("Deal id: %v is cancelled", dealid)
	} else {
		msg.Text = fmt.Sprintf("Deal id: %v cannot be cancelled", dealid)
	}
	t.bot.Send(msg)
}

func (t *TgBot) Default(upd tgbotapi.Update) {
	msg := tgbotapi.NewMessage(upd.Message.Chat.ID, upd.Message.Text)
	msg.Text = "Supported commands: /start /register /buy /sell /positions /history"
	t.bot.Send(msg)
}

func (t *TgBot) sendAuthMsg(chatid int64) {
	msg := tgbotapi.NewMessage(chatid, "Please authorize yourself\nClick START once you are redirected back to Telegram")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("VK", t.AuthProvider.GetOauthURL()),
		),
	)
	t.bot.Send(msg)
}

func (t *TgBot) sendError(chatid int64, err error) {
	msg := tgbotapi.NewMessage(chatid,
		fmt.Sprintf("Error processing command: %v\nPlease try again or use /start for restart", err.Error()))
	t.bot.Send(msg)
}
