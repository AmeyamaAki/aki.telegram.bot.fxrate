package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"aki.telegram.bot.fxrate/bank"
)

func HandleCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	fields := strings.Fields(update.Message.Text)
	if len(fields) == 0 {
		return
	}

	cmd := fields[0]
	if atIndex := strings.Index(cmd, "@"); atIndex != -1 {
		cmd = cmd[:atIndex]
	}

	userID := update.Message.From.ID

	switch cmd {
	case "/start":
		CommandStart(ctx, b, update)
		setCommandsForUser(ctx, b, userID)
	case "/boc":
		HandleBOCCommand(ctx, b, update)
	case "/cib":
		HandleCIBCommand(ctx, b, update)
	default:
		return
	}
}

func CommandStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	nickname := getUserNickName(update)
	startReply := fmt.Sprintf(
		"Welcome, %s!\n\n目前可用的指令:\n"+
			"/start - 显示这条消息，更新命令列表\n"+
			"/boc [币种] - 查询中国银行牌价\n"+
			"/cib [币种] - 查询兴业银行牌价\n\n"+
			"Enjoy~ 💖", nickname,
	)
	SendMessage(ctx, b, update.Message.Chat.ID, startReply, update.Message.MessageThreadID, "")
}

func setCommandsForUser(ctx context.Context, b *bot.Bot, userID int64) {
	userCommands := []models.BotCommand{
		{Command: "start", Description: "启动~ 顺便更新一下命令列表w"},
		{Command: "boc", Description: "查询中国银行牌价"},
		{Command: "cib", Description: "查询兴业银行牌价"},
	}
	params := &bot.SetMyCommandsParams{
		Commands: userCommands,
		Scope: &models.BotCommandScopeChat{
			ChatID: userID,
		},
	}
	b.SetMyCommands(ctx, params)
}

func HandleBOCCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	fields := strings.Fields(update.Message.Text)
	if len(fields) < 2 {
		SendMessage(ctx, b, update.Message.Chat.ID, "用法: /boc [币种]，例如: /boc hkd 或 /boc 港币", update.Message.MessageThreadID, "")
		return
	}

	rate, found, err := bank.GetBOCRate(ctx, fields[1])
	if err != nil {
		LogError("BOC fetch error: %v", err)
		SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
		return
	}
	if !found || rate == nil {
		SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请尝试币种代码（如: USD/HKD）或中文名。", update.Message.MessageThreadID, "")
		return
	}

	msg := fmt.Sprintf(
		"中国银行外汇牌价 — %s\n\n"+
			"现汇买入价: %s\n"+
			"现钞买入价: %s\n"+
			"现汇卖出价: %s\n"+
			"现钞卖出价: %s\n"+
			"中行折算价: %s\n\n"+
			"发布时间: %s",
		rate.Name, rate.BuySpot, rate.BuyCash, rate.SellSpot, rate.SellCash, rate.BankRate, rate.ReleaseTime,
	)

	SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
}

func HandleCIBCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	fields := strings.Fields(update.Message.Text)
	if len(fields) < 2 {
		SendMessage(ctx, b, update.Message.Chat.ID, "用法: /cib [币种]，例如: /cib hkd 或 /cib 港币", update.Message.MessageThreadID, "")
		return
	}

	rate, found, err := bank.GetCIBRate(ctx, fields[1])
	if err != nil {
		LogError("BOC fetch error: %v", err)
		SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
		return
	}
	if !found || rate == nil {
		SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请尝试币种代码（如: USD/HKD）或中文名。", update.Message.MessageThreadID, "")
		return
	}

	msg := fmt.Sprintf(
		"兴业银行外汇牌价 — %s (%s)\n\n"+
			"现汇买入价: %s\n"+
			"现钞买入价: %s\n"+
			"现汇卖出价: %s\n"+
			"现钞卖出价: %s\n\n"+
			"发布时间: %s",
		rate.Name, rate.Symbol, rate.BuySpot, rate.BuyCash, rate.SellSpot, rate.SellCash, rate.ReleaseTime,
	)

	SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
}
