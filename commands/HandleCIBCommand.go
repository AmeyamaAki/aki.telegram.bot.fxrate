package commands

import (
	"context"
	"fmt"
	"strconv"
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
		tools.SendMessage(ctx, b, update.Message.Chat.ID,
			"用法: /cib [币种] [金额] [目标币种]\n"+
				"示例:\n"+
				"/cib hkd - 查询港币（HKD）外汇牌价\n"+
				"/cib hkd 100 - 计算 100HKD 换算成 CNY\n"+
				"/cib cny 100 hkd - 计算 100CNY 换算成 HKD",
			update.Message.MessageThreadID, "")
		return
	}

	// 查询模式
	if len(fields) == 2 {
		handleCIBLookup(ctx, b, update, fields[1])
		return
	}

	// 换算模式: /cib <from> <amount> [to]
	from := fields[1]
	amountStr := fields[2]
	to := "cny"
	if len(fields) >= 4 {
		to = fields[3]
	}
	amount, ok := cibParseAmount(amountStr)
	if !ok {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "金额格式不正确，请输入数字，例如: 100 或 100.5", update.Message.MessageThreadID, "")
		return
	}
	handleCIBConvert(ctx, b, update, from, to, amount)
}

func handleCIBLookup(ctx context.Context, b *bot.Bot, update *models.Update, q string) {
	rate, found, err := bank.GetCIBRate(ctx, q)
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

// ====== 换算实现（与 BOC 一致的策略）======

func cibParseAmount(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	s = strings.ReplaceAll(s, ",", "")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func cibParseRate(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0, false
	}
	s = strings.ReplaceAll(s, ",", "")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func cibIsCNY(code string) bool {
	c := strings.ToLower(strings.TrimSpace(code))
	c = strings.ReplaceAll(c, "_", "")
	c = strings.ReplaceAll(c, "-", "")
	c = strings.ReplaceAll(c, "/", "")
	switch c {
	case "cny", "rmb", "renminbi", "人民币":
		return true
	default:
		return false
	}
}

func cibUpper(code string) string { return strings.ToUpper(strings.TrimSpace(code)) }

// 规则：
// 外币 -> CNY：优先“现汇买入价”，缺失回退“现钞买入价”
// CNY -> 外币：优先“现汇卖出价”，缺失回退“现钞卖出价”
// 牌价单位为“每100外币”
func handleCIBConvert(ctx context.Context, b *bot.Bot, update *models.Update, from, to string, amount float64) {
	if amount < 0 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "金额不能为负数。", update.Message.MessageThreadID, "")
		return
	}

	fromCode := cibUpper(from)
	toCode := cibUpper(to)
	unit := 100.0

	if strings.EqualFold(fromCode, toCode) {
		msg := fmt.Sprintf("%.2f %s = %.2f %s (同币种，无需换算)", amount, fromCode, amount, toCode)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	// CNY -> 外币
	if cibIsCNY(fromCode) && !cibIsCNY(toCode) {
		rate, found, err := bank.GetCIBRate(ctx, toCode)
		if err != nil {
			tools.LogError("CIB fetch error: %v", err)
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
			return
		}
		if !found || rate == nil {
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请检查输入的目标币种代码。", update.Message.MessageThreadID, "")
			return
		}
		label := "现汇卖出价"
		rateStr := rate.SellSpot
		rateVal, ok := cibParseRate(rate.SellSpot)
		if !ok || rateVal <= 0 {
			if v, ok2 := cibParseRate(rate.SellCash); ok2 && v > 0 {
				label = "现钞卖出价"
				rateStr = rate.SellCash
				rateVal = v
			} else {
				tools.SendMessage(ctx, b, update.Message.Chat.ID, "目标币种缺少有效的卖出价（现汇/现钞），无法换算。", update.Message.MessageThreadID, "")
				return
			}
		}
		out := amount / rateVal * unit
		msg := fmt.Sprintf(
			"按兴业银行牌价换算: %s -> %s\n\n"+
				"%.2f CNY ≈ %.2f %s\n\n"+
				"使用牌价: %s %s\n"+
				"发布时间: %s",
			"CNY", rate.Name, amount, out, toCode, label, rateStr, rate.ReleaseTime,
		)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	// 外币 -> CNY
	if !cibIsCNY(fromCode) && cibIsCNY(toCode) {
		rate, found, err := bank.GetCIBRate(ctx, fromCode)
		if err != nil {
			tools.LogError("CIB fetch error: %v", err)
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
			return
		}
		if !found || rate == nil {
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请检查输入的源币种代码。", update.Message.MessageThreadID, "")
			return
		}
		label := "现汇买入价"
		rateStr := rate.BuySpot
		rateVal, ok := cibParseRate(rate.BuySpot)
		if !ok || rateVal <= 0 {
			if v, ok2 := cibParseRate(rate.BuyCash); ok2 && v > 0 {
				label = "现钞买入价"
				rateStr = rate.BuyCash
				rateVal = v
			} else {
				tools.SendMessage(ctx, b, update.Message.Chat.ID, "源币种缺少有效的买入价（现汇/现钞），无法换算。", update.Message.MessageThreadID, "")
				return
			}
		}
		out := amount * rateVal / unit
		msg := fmt.Sprintf(
			"按兴业银行牌价换算: %s -> %s\n\n"+
				"%.2f %s ≈ %.2f CNY\n\n"+
				"使用牌价: %s %s\n"+
				"发布时间: %s",
			rate.Name, "CNY", amount, fromCode, out, label, rateStr, rate.ReleaseTime,
		)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	tools.SendMessage(ctx, b, update.Message.Chat.ID, "暂不支持~", update.Message.MessageThreadID, "")
}
