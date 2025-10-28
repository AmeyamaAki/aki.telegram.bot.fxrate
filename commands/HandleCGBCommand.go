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

func HandleCGBCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
    if update.Message == nil {
        return
    }
    fields := strings.Fields(update.Message.Text)
    if len(fields) < 2 {
        tools.SendMessage(ctx, b, update.Message.Chat.ID,
            "用法: /cgb [币种] [金额] [目标币种]\n"+
                "示例:\n"+
                "/cgb hkd - 查询广发银行港币（HKD）牌价\n"+
                "/cgb hkd 100 - 计算 100HKD 换算成 CNY\n"+
                "/cgb cny 100 hkd - 计算 100CNY 换算成 HKD",
            update.Message.MessageThreadID, "")
        return
    }

    if len(fields) == 2 {
        handleCGBLookup(ctx, b, update, fields[1])
        return
    }
	
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
    handleCGBConvert(ctx, b, update, from, to, amount)
}

func handleCGBLookup(ctx context.Context, b *bot.Bot, update *models.Update, q string) {
    rate, found, err := bank.GetCGBRate(ctx, q)
    if err != nil {
        tools.LogError("CGB fetch error: %v", err)
        tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
        return
    }
    if !found || rate == nil {
        tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请尝试币种代码（如: USD/HKD）或中文名。", update.Message.MessageThreadID, "")
        return
    }

    // 展示按“每100外币”为单位的牌价
    unit := rate.Unit
    if unit <= 0 {
        unit = 100
    }
    buySpotDisp := scaleRateToPer100(rate.BuySpot, unit)
    buyCashDisp := scaleRateToPer100(rate.BuyCash, unit)
    sellSpotDisp := scaleRateToPer100(rate.SellSpot, unit)
    sellCashDisp := scaleRateToPer100(rate.SellCash, unit)
    middleDisp := scaleRateToPer100(rate.MiddleRate, unit)

    msg := fmt.Sprintf(
        "广发银行外汇牌价 — %s (%s)\n\n"+
            "现汇买入价: %s\n"+
            "现钞买入价: %s\n"+
            "现汇卖出价: %s\n"+
            "现钞卖出价: %s\n"+
            "中间价: %s\n\n"+
            "发布时间: %s",
        rate.Name, rate.Symbol, buySpotDisp, buyCashDisp, sellSpotDisp, sellCashDisp, middleDisp, rate.ReleaseTime,
    )
    tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
}

// 外币 -> CNY 与 CNY -> 外币使用“现汇卖出价”，缺失回落“现钞卖出价”；
func handleCGBConvert(ctx context.Context, b *bot.Bot, update *models.Update, from, to string, amount float64) {
    if amount < 0 {
        tools.SendMessage(ctx, b, update.Message.Chat.ID, "金额不能为负数。", update.Message.MessageThreadID, "")
        return
    }

    fromCode := UpperCurrency(from)
    toCode := UpperCurrency(to)

    if strings.EqualFold(fromCode, toCode) {
        msg := fmt.Sprintf("%.2f %s = %.2f %s (同币种，无需换算)", amount, fromCode, amount, toCode)
        tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
        return
    }

    // CNY -> 外币
    if IsCNY(fromCode) && !IsCNY(toCode) {
        rate, found, err := bank.GetCGBRate(ctx, toCode)
        if err != nil {
            tools.LogError("CGB fetch error: %v", err)
            tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
            return
        }
        if !found || rate == nil {
            tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请检查输入的目标币种代码。", update.Message.MessageThreadID, "")
            return
        }
        
        unit := rate.Unit
        if unit <= 0 {
            unit = 100
        }
        scale := 100.0 / unit
        label := "现汇卖出价"
        rateVal, ok := ParseRate(rate.SellSpot)
        if !ok || rateVal <= 0 {
            if v, ok2 := ParseRate(rate.SellCash); ok2 && v > 0 {
                label = "现钞卖出价"
                rateVal = v
            } else {
                tools.SendMessage(ctx, b, update.Message.Chat.ID, "目标币种缺少有效的卖出价（现汇/现钞），无法换算。", update.Message.MessageThreadID, "")
                return
            }
        }
        
        rateValPer100 := rateVal * scale
        out := amount / rateValPer100 * 100.0
        
        rateStrDisp := fmt.Sprintf("%.4f", rateValPer100)
        msg := FormatCNYToFX("广发银行", rate.Name, toCode, amount, out, label, rateStrDisp, rate.ReleaseTime)
        tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
        return
    }

    // 外币 -> CNY
    if !IsCNY(fromCode) && IsCNY(toCode) {
        rate, found, err := bank.GetCGBRate(ctx, fromCode)
        if err != nil {
            tools.LogError("CGB fetch error: %v", err)
            tools.SendMessage(ctx, b, update.Message.Chat.ID, "查询失败，请稍后再试。", update.Message.MessageThreadID, "")
            return
        }
        if !found || rate == nil {
            tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种，请检查输入的源币种代码。", update.Message.MessageThreadID, "")
            return
        }
        unit := rate.Unit
        if unit <= 0 {
            unit = 100
        }
        scale := 100.0 / unit
        label := "现汇卖出价"
        rateVal, ok := ParseRate(rate.SellSpot)
        if !ok || rateVal <= 0 {
            if v, ok2 := ParseRate(rate.SellCash); ok2 && v > 0 {
                label = "现钞卖出价"
                rateVal = v
            } else {
                tools.SendMessage(ctx, b, update.Message.Chat.ID, "源币种缺少有效的卖出价（现汇/现钞），无法换算。", update.Message.MessageThreadID, "")
                return
            }
        }
        
        rateValPer100 := rateVal * scale
        out := amount * rateValPer100 / 100.0
        
        rateStrDisp := fmt.Sprintf("%.4f", rateValPer100)
        msg := FormatFXToCNY("广发银行", rate.Name, fromCode, amount, out, label, rateStrDisp, rate.ReleaseTime)
        tools.SendMessage(ctx, b, update.Message.Chat.ID, msg, update.Message.MessageThreadID, "")
        return
    }

    tools.SendMessage(ctx, b, update.Message.Chat.ID, "暂不支持~", update.Message.MessageThreadID, "")
}

// scaleRateToPer100 将页面给定的牌价（按 unit=1/100）折算为“每100外币”的字符串；
// 解析失败或为空时返回 "-"。
func scaleRateToPer100(s string, unit float64) string {
    v, ok := ParseRate(s)
    if !ok || v <= 0 {
        return "-"
    }
    if unit <= 0 {
        unit = 100
    }
    scale := 100.0 / unit
    return fmt.Sprintf("%.4f", v*scale)
}
