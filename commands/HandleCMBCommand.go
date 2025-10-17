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

func HandleCMBCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	fields := strings.Fields(update.Message.Text)
	if len(fields) < 2 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID,
			"用法: /cmb [币种] [金额] [目标币种]\n"+
				"示例:\n"+
				"/cmb hkd - 查询寰宇人生优惠价港币（HKD）牌价\n"+
				"/cmb hkd 100 - 计算 100HKD 换算成 CNY\n"+
				"/cmb cny 100 hkd - 计算 100CNY 换算成 HKD",
			update.Message.MessageThreadID, "")
		return
	}

	// 查询汇率
	if len(fields) == 2 {
		handleCMBLookup(ctx, b, update, fields[1])
		return
	}

	// 汇率换算 /cmb <from> <amount> [to]
	from := fields[1]
	amountStr := fields[2]
	to := "cny"
	if len(fields) >= 4 {
		to = fields[3]
	}
	amount, ok := ParseAmount(amountStr)
	if !ok {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "金额格式不正确，请输入数字，例如: 100 或 100.5", update.Message.MessageThreadID, "")
		return
	}
	handleCMBConvert(ctx, b, update, from, to, amount)
}

func handleCMBLookup(ctx context.Context, b *bot.Bot, update *models.Update, q string) {
	rate, found, err := bank.GetCMBRate(ctx, q)
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

func handleCMBConvert(ctx context.Context, b *bot.Bot, update *models.Update, from, to string, amount float64) {
	if amount < 0 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "金额不能为负数。", update.Message.MessageThreadID, "")
		return
	}

	fromCode := UpperCurrency(from)
	toCode := UpperCurrency(to)
	unit := 100.0

	if strings.EqualFold(fromCode, toCode) {
		msg := fmt.Sprintf("%.2f %s = %.2f %s (同币种，无需换算)", amount, fromCode, amount, toCode)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	// CNY -> 外币
	if IsCNY(fromCode) && !IsCNY(toCode) {
		rate, found, err := bank.GetCMBRate(ctx, toCode)
		if err != nil {
			tools.LogError("CMB fetch error: %v", err)
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
			return
		}
		if !found || rate == nil {
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请检查输入的目标币种代码。", update.Message.MessageThreadID, "")
			return
		}
		label := "现汇卖出价"
		rateStr := rate.SellSpot
		rateVal, ok := ParseRate(rate.SellSpot)
		if !ok || rateVal <= 0 {
			if v, ok2 := ParseRate(rate.SellCash); ok2 && v > 0 {
				label = "现钞卖出价"
				rateStr = rate.SellCash
				rateVal = v
			} else {
				tools.SendMessage(ctx, b, update.Message.Chat.ID, "目标币种缺少有效的卖出价（现汇/现钞），无法换算。", update.Message.MessageThreadID, "")
				return
			}
		}
		out := amount / rateVal * unit
		msg := FormatCNYToFX("招商银行", rate.Name, toCode, amount, out, label, rateStr, rate.ReleaseTime)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	// 外币 -> CNY
	if !IsCNY(fromCode) && IsCNY(toCode) {
		rate, found, err := bank.GetCMBRate(ctx, fromCode)
		if err != nil {
			tools.LogError("CMB fetch error: %v", err)
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
			return
		}
		if !found || rate == nil {
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请检查输入的源币种代码。", update.Message.MessageThreadID, "")
			return
		}
		label := "现汇卖出价"
		rateStr := rate.SellSpot
		rateVal, ok := ParseRate(rate.SellSpot)
		if !ok || rateVal <= 0 {
			if v, ok2 := ParseRate(rate.SellCash); ok2 && v > 0 {
				label = "现钞卖出价"
				rateStr = rate.SellCash
				rateVal = v
			} else {
				tools.SendMessage(ctx, b, update.Message.Chat.ID, "源币种缺少有效的买入价（现汇/现钞），无法换算。", update.Message.MessageThreadID, "")
				return
			}
		}
		out := amount * rateVal / unit
		msg := FormatFXToCNY("招商银行", rate.Name, fromCode, amount, out, label, rateStr, rate.ReleaseTime)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	tools.SendMessage(ctx, b, update.Message.Chat.ID, "暂不支持~", update.Message.MessageThreadID, "")
}
