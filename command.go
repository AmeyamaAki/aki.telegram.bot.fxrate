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
	default:
		return
	}
}

func CommandStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	nickname := getUserNickName(update)
	// botInfo, err := b.GetMe(ctx)
	// if err != nil {
	//	LogError("Error getting bot info: %v", err)
	//	return
	//}
	// botUsername := botInfo.Username

	var startReply string
	startReply = fmt.Sprintf("Welcome, %s!\n\n目前可用的指令:\n"+
		"/start - 显示这条消息，更新命令列表\n"+
		"/boc [币种] - 查询中国银行牌价\n\n"+
		"Enjoy~ 💖", nickname)
	SendMessage(ctx, b, update.Message.Chat.ID, startReply, update.Message.MessageThreadID, "")
}

func setCommandsForUser(ctx context.Context, b *bot.Bot, userID int64) {

	userCommands := []models.BotCommand{
		{Command: "start", Description: "启动~ 顺便更新一下命令列表w"},
		{Command: "boc", Description: "查询中国银行牌价"},
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
	msg, found, err := bank.BuildBOCMessage(ctx, fields[1])
	if err != nil {
		LogError("BOC fetch error: %v", err)
		SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
		return
	}
	if !found {
		SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请尝试币种代码（如: USD/HKD）或中文名。", update.Message.MessageThreadID, "")
		return
	}
	SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
}
