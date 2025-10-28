package main

import (
	"context"
	"fmt"
	"strings"

	"aki.telegram.bot.fxrate/tools"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

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
		commands.HandleCIBLifeCommand(ctx, b, update)
	case "/cmb":
		commands.HandleCMBCommand(ctx, b, update)
	case "/citic":
		commands.HandleCITICCommand(ctx, b, update)
	case "/cgb":
		commands.HandleCGBCommand(ctx, b, update)
	case "/uniopay":
		commands.HandleUnionPayCommand(ctx, b, update)
	case "/xhmr":
		commands.HandleXHMRCommand(ctx, b, update)
	case "/jh":
		commands.HandleXHMRCommand(ctx, b, update)
	case "/xhmc":
		commands.HandleXHMCCommand(ctx, b, update)
	case "/gh":
		commands.HandleXHMCCommand(ctx, b, update)
	default:
		return
	}
}

func CommandStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	nickname := tools.GetUserNickName(update)
	startReply := fmt.Sprintf(
		"Welcome, %s!\n\n目前可用的指令:\n"+
			"/start - 显示这条消息，更新命令列表\n"+
			"/boc - 中国银行\n"+
			"/cib - 兴业银行\n"+
			"/cgb - 广发银行\n"+
			"/citic - 中信银行\n"+
			"/hy  - 寰宇人生借记卡\n"+
			"/cmb - 招商银行\n\n"+
			"/uniopay - 银联\n\n"+
			"/xhmr [币种] [筛选数|银行] - 现汇买入对比\n"+
			"也可以使用 /jh\n\n"+
			"/xhmc [币种] [筛选数|银行] - 现汇卖出对比\n"+
			"也可以使用 /gh\n\n"+
			"Enjoy~ 💖", nickname,
	)
	tools.SendMessage(ctx, b, update.Message.Chat.ID, startReply, update.Message.MessageThreadID, "")
}

func setCommandsForUser(ctx context.Context, b *bot.Bot, userID int64) {
	userCommands := []models.BotCommand{
		{Command: "start", Description: "启动~ 顺便更新一下命令列表w"},
		{Command: "boc", Description: "中国银行"},
		{Command: "cib", Description: "兴业银行"},
		{Command: "cgb", Description: "广发银行"},
		{Command: "citic", Description: "中信银行"},
		{Command: "hy", Description: "寰宇人生借记卡"},
		{Command: "cmb", Description: "招商银行"},
		{Command: "uniopay", Description: "银联"},
		{Command: "xhmr", Description: "现汇买入对比"},
		{Command: "xhmc", Description: "现汇卖出对比"},
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
