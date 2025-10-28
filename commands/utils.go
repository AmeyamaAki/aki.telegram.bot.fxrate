package commands

import (
	"fmt"
	"strconv"
	"strings"
)

type bankRate struct {
	BankNameCN   string
	BankKey      string
	CurrencyDesc string
	BuySpotVal   float64
	BuySpotRaw   string
	ReleaseTime  string
}

// ParseAmount 解析输入金额，允许包含千分位逗号
func ParseAmount(s string) (float64, bool) {
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

// ParseRate 解析牌价，过滤 "-" 或空
func ParseRate(s string) (float64, bool) {
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

// IsCNY 判断是否为人民币
func IsCNY(code string) bool {
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

// UpperCurrency 规范化货币代码为大写（保留原有字符集）
func UpperCurrency(code string) string { return strings.ToUpper(strings.TrimSpace(code)) }

// FormatCNYToFX 构造 CNY -> 外币 的换算消息
func FormatCNYToFX(bankName, toName, toCode string, amountCNY, outFX float64, label, rateStr, releaseTime string) string {
	return fmt.Sprintf(
		"按%s牌价换算: %s -> %s\n\n"+
			"%.2f CNY ≈ %.2f %s\n\n"+
			"使用牌价: %s %s\n"+
			"发布时间: %s",
		bankName, "CNY", toName, amountCNY, outFX, toCode, label, rateStr, releaseTime,
	)
}

// FormatFXToCNY 构造 外币 -> CNY 的换算消息
func FormatFXToCNY(bankName, fromName, fromCode string, amountFX, outCNY float64, label, rateStr, releaseTime string) string {
	return fmt.Sprintf(
		"按%s牌价换算: %s -> %s\n\n"+
			"%.2f %s ≈ %.2f CNY\n\n"+
			"使用牌价: %s %s\n"+
			"发布时间: %s",
		bankName, fromName, "CNY", amountFX, fromCode, outCNY, label, rateStr, releaseTime,
	)
}

// 去重
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

// mapBankNames 将银行 key 列表映射为中文名
func mapBankNames(keys []string) []string {
	if len(keys) == 0 {
		return nil
	}
	m := map[string]string{
		"boc":   "中国银行",
		"cib":   "兴业银行",
		"cmb":   "招商银行",
		"hy":    "寰宇人生",
		"cgb":   "广发银行",
		"citic": "中信银行",
	}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if v, ok := m[k]; ok {
			out = append(out, v)
		} else {
			out = append(out, k)
		}
	}
	return out
}