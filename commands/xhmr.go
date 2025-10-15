package commands

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"aki.telegram.bot.fxrate/bank"
	"aki.telegram.bot.fxrate/tools"
)

type bankRate struct {
	BankNameCN   string
	BankKey      string
	CurrencyDesc string
	BuySpotVal   float64
	BuySpotRaw   string
	ReleaseTime  string
}

func HandleXHMRCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}

	fields := strings.Fields(update.Message.Text)
	if len(fields) < 2 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "用法: /xhmr [币种] [数字|银行...]，例如:\n/xhmr hkd\n/xhmr hkd 3\n/xhmr hkd boc cmb", update.Message.MessageThreadID, "")
		return
	}

	ccy := strings.ToUpper(fields[1])

	// 解析可选参数：TopN（数字）与指定银行列表
	var topN int
	var bankKeys []string
	for _, t := range fields[2:] {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if n, err := strconv.Atoi(t); err == nil && n > 0 {
			topN = n
			continue
		}
		switch t {
		case "boc", "cib", "cmb":
			bankKeys = append(bankKeys, t)
		}
	}
	bankKeys = dedup(bankKeys)
	if len(bankKeys) == 0 {
		bankKeys = []string{"boc", "cib", "cmb"}
	}

	// 拉取数据
	var results []bankRate
	for _, key := range bankKeys {
		switch key {
		case "boc":
			if r := fetchBOC(ctx, ccy); r != nil {
				results = append(results, *r)
			}
		case "cib":
			if r := fetchCIB(ctx, ccy); r != nil {
				results = append(results, *r)
			}
		case "cmb":
			if r := fetchCMB(ctx, ccy); r != nil {
				results = append(results, *r)
			}
		}
	}

	if len(results) == 0 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种的现汇买入价，请尝试币种代码（如: USD/HKD）或中文名。", update.Message.MessageThreadID, "")
		return
	}

	// 排序（从高到低）
	sort.Slice(results, func(i, j int) bool { return results[i].BuySpotVal > results[j].BuySpotVal })

	// 截取 Top N
	if topN > 0 && topN < len(results) {
		results = results[:topN]
	}

	// 货币展示名
	currencyDesc := ccy
	for _, r := range results {
		if strings.TrimSpace(r.CurrencyDesc) != "" {
			currencyDesc = r.CurrencyDesc
			break
		}
	}

	// 组装消息
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("现汇买入最优排序 — %s\n", currencyDesc))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s: %s（发布时间: %s）\n", i+1, bankCN(r.BankKey), r.BuySpotRaw, r.ReleaseTime))
	}

	tools.SendMessage(ctx, b, update.Message.Chat.ID, sb.String(), update.Message.MessageThreadID, "")
}

func fetchBOC(ctx context.Context, ccy string) *bankRate {
	r, found, err := bank.GetBOCRate(ctx, ccy)
	if err != nil || !found || r == nil {
		return nil
	}
	val, ok := parsePrice(r.BuySpot)
	if !ok {
		return nil
	}
	return &bankRate{
		BankNameCN:   "中国银行",
		BankKey:      "boc",
		CurrencyDesc: safeStr(r.Name, ccy),
		BuySpotVal:   val,
		BuySpotRaw:   r.BuySpot,
		ReleaseTime:  r.ReleaseTime,
	}
}

func fetchCIB(ctx context.Context, ccy string) *bankRate {
	r, found, err := bank.GetCIBRate(ctx, ccy)
	if err != nil || !found || r == nil {
		return nil
	}
	val, ok := parsePrice(r.BuySpot)
	if !ok {
		return nil
	}
	return &bankRate{
		BankNameCN:   "兴业银行",
		BankKey:      "cib",
		CurrencyDesc: safeStr(r.Name, ccy),
		BuySpotVal:   val,
		BuySpotRaw:   r.BuySpot,
		ReleaseTime:  r.ReleaseTime,
	}
}

func fetchCMB(ctx context.Context, ccy string) *bankRate {
	r, found, err := bank.GetCMBRate(ctx, ccy)
	if err != nil || !found || r == nil {
		return nil
	}
	val, ok := parsePrice(r.BuySpot)
	if !ok {
		return nil
	}
	return &bankRate{
		BankNameCN:   "招商银行",
		BankKey:      "cmb",
		CurrencyDesc: safeStr(r.Name, ccy),
		BuySpotVal:   val,
		BuySpotRaw:   r.BuySpot,
		ReleaseTime:  r.ReleaseTime,
	}
}

func parsePrice(s string) (float64, bool) {
	t := strings.TrimSpace(s)
	if t == "" || t == "-" {
		return 0, false
	}
	// 移除可能的千位分隔符
	t = strings.ReplaceAll(t, ",", "")
	v, err := strconv.ParseFloat(t, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

func safeStr(v string, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func dedup(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func bankCN(key string) string {
	switch key {
	case "boc":
		return "中国银行"
	case "cib":
		return "兴业银行"
	case "cmb":
		return "招商银行"
	default:
		return key
	}
}
