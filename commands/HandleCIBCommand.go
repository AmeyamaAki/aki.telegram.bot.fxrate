package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"aki.telegram.bot.fxrate/bank"
	"aki.telegram.bot.fxrate/tools"
)

func HandleCIBCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	fields := strings.Fields(update.Message.Text)
	if len(fields) < 2 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "用法: /cib [币种]，例如: /cib hkd 或 /cib 港币", update.Message.MessageThreadID, "")
		return
	}

	rate, found, err := bank.GetCIBRate(ctx, fields[1])
	if err != nil {
		tools.LogError("CIB fetch error: %v", err)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
		return
	}
	if !found || rate == nil {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请尝试币种代码（如: USD/HKD）或中文名。", update.Message.MessageThreadID, "")
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

	tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
}
