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

func HandleBOCCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	fields := strings.Fields(update.Message.Text)
	if len(fields) < 2 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID,
			"用法: /boc [币种] [金额] [目标币种]\n"+
				"示例:\n"+
				"/boc hkd - 查询港币（HKD）外汇牌价\n"+
				"/boc hkd 100 - 计算 100HKD 换算成 CNY\n"+
				"/boc cny 100 hkd - 计算 100CNY 换算成 HKD",
			update.Message.MessageThreadID, "")
		return
	}

	// 仅命令 + 1参数 => 查询某币种牌价
	if len(fields) == 2 {
		handleBOCLookup(ctx, b, update, fields[1])
		return
	}

	// >= 3 个参数 => 进行换算
	// 约定：
	// /boc <from> <amount> [to]
	// 若省略 to，则默认为 CNY
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
	handleBOCConvert(ctx, b, update, from, to, amount)
}

// 处理牌价查询
func handleBOCLookup(ctx context.Context, b *bot.Bot, update *models.Update, q string) {
	rate, found, err := bank.GetBOCRate(ctx, q)
	if err != nil {
		tools.LogError("BOC fetch error: %v", err)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
		return
	}
	if !found || rate == nil {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请尝试币种代码（如: USD/HKD）或中文名。", update.Message.MessageThreadID, "")
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

	tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
}

// ===== 货币换算 =====

// handleBOCConvert 使用中行牌价进行换算：
// 外币 -> CNY 使用现汇买入价（结汇），若缺失则用现钞买入价
// CNY -> 外币 使用现汇卖出价（购汇），若缺失则同上
// 牌价单位为“每100外币”，按 100 为基准进行换算。
func handleBOCConvert(ctx context.Context, b *bot.Bot, update *models.Update, from, to string, amount float64) {
	if amount < 0 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "金额不能为负数。", update.Message.MessageThreadID, "")
		return
	}

	fromCode := UpperCurrency(from)
	toCode := UpperCurrency(to)
	unit := 100.0

	// 同币种
	if strings.EqualFold(fromCode, toCode) {
		msg := fmt.Sprintf("%.2f %s = %.2f %s (同币种，无需换算)", amount, fromCode, amount, toCode)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	// CNY -> 外币
	if IsCNY(fromCode) && !IsCNY(toCode) {
		// 获取目标外币的卖出价，无现汇则退回到现钞
		rate, found, err := bank.GetBOCRate(ctx, toCode)
		if err != nil {
			tools.LogError("BOC fetch error: %v", err)
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
			return
		}
		if !found || rate == nil {
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请检查输入的目标币种代码。", update.Message.MessageThreadID, "")
			return
		}
		// 优先现汇卖出价
		usedLabel := "现汇卖出价"
		usedRateStr := rate.SellSpot
		usedRateVal, ok := ParseRate(rate.SellSpot)
		if !ok || usedRateVal <= 0 {
			// 回退现钞卖出价
			if v, ok2 := ParseRate(rate.SellCash); ok2 && v > 0 {
				usedLabel = "现钞卖出价"
				usedRateStr = rate.SellCash
				usedRateVal = v
			} else {
				tools.SendMessage(ctx, b, update.Message.Chat.ID, "目标币种缺少有效的卖出价（现汇/现钞），无法换算。", update.Message.MessageThreadID, "")
				return
			}
		}
		out := amount / usedRateVal * unit
		msg := FormatCNYToFX("中国银行", rate.Name, toCode, amount, out, usedLabel, usedRateStr, rate.ReleaseTime)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	// 外币 -> CNY
	if !IsCNY(fromCode) && IsCNY(toCode) {
		rate, found, err := bank.GetBOCRate(ctx, fromCode)
		if err != nil {
			tools.LogError("BOC fetch error: %v", err)
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
			return
		}
		if !found || rate == nil {
			tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请检查输入的源币种代码。", update.Message.MessageThreadID, "")
			return
		}
		// 优先现汇买入价
		usedLabel := "现汇买入价"
		usedRateStr := rate.BuySpot
		usedRateVal, ok := ParseRate(rate.BuySpot)
		if !ok || usedRateVal <= 0 {
			// 回退现钞买入价
			if v, ok2 := ParseRate(rate.BuyCash); ok2 && v > 0 {
				usedLabel = "现钞买入价"
				usedRateStr = rate.BuyCash
				usedRateVal = v
			} else {
				tools.SendMessage(ctx, b, update.Message.Chat.ID, "源币种缺少有效的买入价（现汇/现钞），无法换算。", update.Message.MessageThreadID, "")
				return
			}
		}
		out := amount * usedRateVal / unit
		msg := FormatFXToCNY("中国银行", rate.Name, fromCode, amount, out, usedLabel, usedRateStr, rate.ReleaseTime)
		tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
		return
	}

	// 外币 -> 外币（不支持，按需提示）
	tools.SendMessage(ctx, b, update.Message.Chat.ID, "暂不支持~", update.Message.MessageThreadID, "")
}
