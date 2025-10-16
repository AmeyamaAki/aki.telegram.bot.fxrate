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

func HandleCIBLifeCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	fields := strings.Fields(update.Message.Text)
	if len(fields) < 2 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID,
			"用法: /hy [币种] [金额] [目标币种]\n"+
				"示例:\n"+
				"/hy hkd - 查询寰宇人生优惠价港币（HKD）牌价\n"+
				"/hy hkd 100 - 计算 100HKD 换算成 CNY\n"+
				"/hy cny 100 hkd - 计算 100CNY 换算成 HKD",
			update.Message.MessageThreadID, "")
		return
	}

	// 仅 1 参数：查询
	if len(fields) == 2 {
		handleCIBLifeLookup(ctx, b, update, fields[1])
		return
	}

	// 3 参数以上：换算 /hy <from> <amount> [to]
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
	handleCIBLifeConvert(ctx, b, update, from, to, amount)
}

func handleCIBLifeLookup(ctx context.Context, b *bot.Bot, update *models.Update, q string) {
	rate, found, err := bank.GetCIBLifeRate(ctx, q)
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

// 换算逻辑（现汇缺失回退现钞；单位每100外币）
func handleCIBLifeConvert(ctx context.Context, b *bot.Bot, update *models.Update, from, to string, amount float64) {
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
		rate, found, err := bank.GetCIBLifeRate(ctx, toCode)
		if err != nil {
			tools.LogError("CIBLife fetch error: %v", err)
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
		msg := FormatCNYToFX("寰宇人生借记卡", rate.Name, toCode, amount, out, label, rateStr, rate.ReleaseTime)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	// 外币 -> CNY
	if !IsCNY(fromCode) && IsCNY(toCode) {
		rate, found, err := bank.GetCIBLifeRate(ctx, fromCode)
		if err != nil {
			tools.LogError("CIBLife fetch error: %v", err)
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
			return
		}
		if !found || rate == nil {
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请检查输入的源币种代码。", update.Message.MessageThreadID, "")
			return
		}
		label := "现汇买入价"
		rateStr := rate.BuySpot
		rateVal, ok := ParseRate(rate.BuySpot)
		if !ok || rateVal <= 0 {
			if v, ok2 := ParseRate(rate.BuyCash); ok2 && v > 0 {
				label = "现钞买入价"
				rateStr = rate.BuyCash
				rateVal = v
			} else {
				tools.SendMessage(ctx, b, update.Message.Chat.ID, "源币种缺少有效的买入价（现汇/现钞），无法换算。", update.Message.MessageThreadID, "")
				return
			}
		}
		out := amount * rateVal / unit
		msg := FormatFXToCNY("寰宇人生借记卡", rate.Name, fromCode, amount, out, label, rateStr, rate.ReleaseTime)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	tools.SendMessage(ctx, b, update.Message.Chat.ID, "暂不支持~", update.Message.MessageThreadID, "")
}
