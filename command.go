package main

import (
	"context"
	"fmt"
	"strings"

	"aki.telegram.bot.fxrate/tools"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"aki.telegram.bot.fxrate/bank"
	"aki.telegram.bot.fxrate/commands"
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
		commands.HandleBOCCommand(ctx, b, update)
	case "/cib":
		commands.HandleCIBCommand(ctx, b, update)
	case "/hy":
		HandleCIBLifeCommand(ctx, b, update)
	case "/cmb":
		HandleCMBCommand(ctx, b, update)
	case "/xhmr":
		commands.HandleXHMRCommand(ctx, b, update)
	default:
		return
	}
}

func CommandStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	nickname := tools.GetUserNickName(update)
	startReply := fmt.Sprintf(
		"Welcome, %s!\n\n目前可用的指令:\n"+
			"/start - 显示这条消息，更新命令列表\n"+
			"/boc - 中国银行牌价相关功能\n"+
			"/cib - 兴业银行牌价相关功能\n"+
			"/hy [币种] - 寰宇人生借记卡汇率\n"+
			"/cmb [币种] - 招商银行牌价\n\n"+
			"/xhmr [币种] [数字|银行} - 现汇买入对比\n\n"+
			"Enjoy~ 💖", nickname,
	)
	tools.SendMessage(ctx, b, update.Message.Chat.ID, startReply, update.Message.MessageThreadID, "")
}

func setCommandsForUser(ctx context.Context, b *bot.Bot, userID int64) {
	userCommands := []models.BotCommand{
		{Command: "start", Description: "启动~ 顺便更新一下命令列表w"},
		{Command: "boc", Description: "中国银行牌价相关"},
		{Command: "cib", Description: "兴业银行牌价相关"},
		{Command: "hy", Description: "寰宇人生借记卡汇率"},
		{Command: "cmb", Description: "招商银行牌价"},
		{Command: "xhmr", Description: "现汇买入对比"},
	}
	params := &bot.SetMyCommandsParams{
		Commands: userCommands,
		Scope: &models.BotCommandScopeChat{
			ChatID: userID,
		},
	}

	_, err := b.SetMyCommands(ctx, params)
	if err != nil {
		tools.LogError("setting commands error for user: %v", err)
		return
	}
}

func HandleCIBLifeCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	fields := strings.Fields(update.Message.Text)
	if len(fields) < 2 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "用法: /hy [币种]，例如: /hy hkd 或 /hy 港币", update.Message.MessageThreadID, "")
		return
	}

	rate, found, err := bank.GetCIBLifeRate(ctx, fields[1])
	if err != nil {
		tools.LogError("CIB Universal Life Debit Card fetch error: %v", err)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
		return
	}
	if !found || rate == nil {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请尝试币种代码（如: USD/HKD）或中文名。", update.Message.MessageThreadID, "")
		return
	}

	msg := fmt.Sprintf(
		"寰宇人生借记卡外汇牌价 — %s (%s)\n\n"+
			"现汇买入价: %s\n"+
			// "现钞买入价: %s\n"+
			"现汇卖出价: %s\n\n"+
			// "现钞卖出价: %s\n\n"+
			"发布时间: %s",
		rate.Name, rate.Symbol, rate.BuySpot, rate.SellSpot, rate.ReleaseTime,
	)

	tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
}

func HandleCMBCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	fields := strings.Fields(update.Message.Text)
	if len(fields) < 2 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "用法: /cmb [币种]，例如: /cmb hkd 或 /cmb 港币", update.Message.MessageThreadID, "")
		return
	}

	rate, found, err := bank.GetCMBRate(ctx, fields[1])
	if err != nil {
		tools.LogError("CMB fetch error: %v", err)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
		return
	}
	if !found || rate == nil {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请尝试币种代码（如: USD/HKD）或中文名。", update.Message.MessageThreadID, "")
		return
	}

	msg := fmt.Sprintf(
		"招商银行外汇牌价 — %s (%s)\n\n"+
			"现汇买入价: %s\n"+
			"现钞买入价: %s\n"+
			"现汇卖出价: %s\n"+
			"现钞卖出价: %s\n"+
			"招行折算价: %s\n\n"+
			"发布时间: %s",
		rate.Name, rate.Symbol, rate.BuySpot, rate.BuyCash, rate.SellSpot, rate.SellCash, rate.BankRate, rate.ReleaseTime,
	)

	tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
}
