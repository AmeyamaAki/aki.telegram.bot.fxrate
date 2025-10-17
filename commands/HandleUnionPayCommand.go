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

func HandleUnionPayCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	fields := strings.Fields(update.Message.Text)
	if len(fields) < 2 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID,
			"用法: /unionpay [币种] [金额] [目标币种]\n"+
				"语义:\n"+
				"/unionpay hkd           -> 查询 1 HKD = ? CNY\n"+
				"/unionpay hkd 100       -> 100 HKD 换算成 CNY\n"+
				"/unionpay hkd 100 usd   -> 100 HKD 换算成 USD\n",
			update.Message.MessageThreadID, "")
		return
	}

	// 查询 /unionpay <fx>   => <fx> -> CNY
	if len(fields) == 2 {
		handleUnionPayLookup(ctx, b, update, fields[1])
		return
	}

	// 换算 /unionpay <fx> <amount> [to]
	from := fields[1]
	amountStr := fields[2]
	to := ""
	if len(fields) >= 4 {
		to = fields[3]
	}
	amount, ok := ParseAmount(amountStr)
	if !ok {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "金额格式不正确，请输入数字，例如: 100 或 100.5", update.Message.MessageThreadID, "")
		return
	}
	handleUnionPayConvert(ctx, b, update, from, to, amount)
}

// handleUnionPayLookup 查询单个币种（<q> -> CNY）
func handleUnionPayLookup(ctx context.Context, b *bot.Bot, update *models.Update, q string) {
	debit := UpperCurrency(strings.TrimSpace(q))
	trans := "CNY"

	rate, found, err := bank.GetUnionPayRate(ctx, debit, trans)
	if err != nil {
		if errorsIsRateNotFound(err) || !found {
			tools.SendMessage(ctx, b, update.Message.Chat.ID, fmt.Sprintf("未找到 %s -> %s 的直接汇率。", debit, trans), update.Message.MessageThreadID, "")
			return
		}
		tools.LogError("UnionPay fetch error: %v", err)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
		return
	}

	msg := fmt.Sprintf(
		"银联国际汇率 — %s -> %s\n\n"+
			"1 %s = %s %s\n\n"+
			"发布时间: %s",
		bank.GetCurrencyName(debit), bank.GetCurrencyName(trans),
		debit, rate.Rate, trans,
		rate.ReleaseTime,
	)
	tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
}

// handleUnionPayConvert 汇率换算
// 语义：
// - /unionpay <fx> <amount>         =>  debit=<fx>, trans=CNY
// - /unionpay <fx> <amount> <to>    =>  debit=<fx>, trans=<to>
func handleUnionPayConvert(ctx context.Context, b *bot.Bot, update *models.Update, from, to string, amount float64) {
	if amount < 0 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "金额不能为负数。", update.Message.MessageThreadID, "")
		return
	}

	fromCode := UpperCurrency(from)
	toCode := UpperCurrency(to)

	// 将 CNY 同义词规范为 CNY
	if IsCNY(fromCode) {
		fromCode = "CNY"
	}
	if strings.TrimSpace(to) == "" {
		toCode = "CNY"
	} else if IsCNY(toCode) {
		toCode = "CNY"
	}

	debit := fromCode
	trans := toCode

	// 同币种
	if strings.EqualFold(debit, trans) {
		msg := fmt.Sprintf("%.2f %s = %.2f %s (同币种，无需换算)", amount, debit, amount, trans)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	rate, found, err := bank.GetUnionPayRate(ctx, debit, trans)
	if err != nil {
		if errorsIsRateNotFound(err) || !found {
			tools.SendMessage(ctx, b, update.Message.Chat.ID, fmt.Sprintf("未找到 %s -> %s 的直接汇率。", debit, trans), update.Message.MessageThreadID, "")
			return
		}
		tools.LogError("UnionPay fetch error: %v", err)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
		return
	}

	rateVal := mustParseRate(rate.Rate)
	out := amount * rateVal

	// 使用 utils 的标准格式：仅在 CNY <-> 外币 时使用
	if IsCNY(trans) && !IsCNY(debit) {
		// 外币 -> CNY
		msg := FormatFXToCNY("银联国际", bank.GetCurrencyName(debit), debit, amount, out, "汇率", rate.Rate, rate.ReleaseTime)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}
	if IsCNY(debit) && !IsCNY(trans) {
		// CNY -> 外币
		msg := FormatCNYToFX("银联国际", bank.GetCurrencyName(trans), trans, amount, out, "汇率", rate.Rate, rate.ReleaseTime)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	// 外币 -> 外币：保留原先的通用格式
	msg := fmt.Sprintf(
		"按银联国际汇率换算: %s -> %s\n\n"+
			"%.2f %s ≈ %.2f %s\n\n"+
			"使用汇率: %s (1 %s = %s %s)\n"+
			"发布时间: %s",
		bank.GetCurrencyName(debit), bank.GetCurrencyName(trans),
		amount, debit, out, trans,
		rate.Rate, debit, rate.Rate, trans,
		rate.ReleaseTime,
	)
	tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
}

// errorsIsRateNotFound 判断是否为未找到直接汇率
func errorsIsRateNotFound(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "direct rate not found")
}

// mustParseRate 将字符串汇率解析为浮点数（假定一定可用）
func mustParseRate(s string) float64 {
	v, _ := ParseRate(s)
	return v
}
