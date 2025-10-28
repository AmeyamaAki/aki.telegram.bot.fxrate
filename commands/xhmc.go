package commands

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"aki.telegram.bot.fxrate/bank"
	"aki.telegram.bot.fxrate/tools"
)

func HandleXHMCCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}

	fields := strings.Fields(update.Message.Text)
	if len(fields) < 2 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "用法: /xhmc [币种] [数字|银行...]，例如:\n/xhmc hkd\n/xhmc hkd 3\n/xhmc hkd boc cmb", update.Message.MessageThreadID, "")
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
		case "boc", "cib", "cmb", "hy", "cgb", "citic":
			bankKeys = append(bankKeys, t)
		}
	}
	bankKeys = dedup(bankKeys)
	if len(bankKeys) == 0 {
		bankKeys = []string{"boc", "cib", "cmb", "hy", "cgb", "citic"}
	}

	waitMsgID, _ := tools.SendMessage(ctx, b, update.Message.Chat.ID,
		fmt.Sprintf("正在查询和比对 %s 的现汇卖出价，请稍候…", ccy),
		update.Message.MessageThreadID, "")

	// 并发拉取数据（每个银行一个 goroutine），设置单请求超时
	resultsCh := make(chan *bankRate, len(bankKeys))
	timeoutsCh := make(chan string, len(bankKeys))
	var wg sync.WaitGroup
	for _, key := range bankKeys {
		k := key
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctxFetch, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			var r *bankRate
			switch k {
			case "boc":
				r = fetchBOC(ctxFetch, ccy)
			case "cib":
				r = fetchCIB(ctxFetch, ccy)
			case "cmb":
				r = fetchCMB(ctxFetch, ccy)
			case "hy":
				r = fetchCIBLife(ctxFetch, ccy)
			case "cgb":
				r = fetchCGB(ctxFetch, ccy)
			case "citic":
				r = fetchCITIC(ctxFetch, ccy)
			}
			if r != nil {
				resultsCh <- r
			} else if ctxFetch.Err() == context.DeadlineExceeded {
				timeoutsCh <- k
			}
		}()
	}
	wg.Wait()
	close(resultsCh)
	close(timeoutsCh)
	var results []bankRate
	for r := range resultsCh {
		results = append(results, *r)
	}
	var timeoutKeys []string
	for k := range timeoutsCh {
		timeoutKeys = append(timeoutKeys, k)
	}

	if len(results) == 0 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, "未找到该币种的现汇买入价，请尝试币种代码（如: USD/HKD）或中文名。", update.Message.MessageThreadID, "")
		// 若有超时，额外提醒
		if len(timeoutKeys) > 0 {
			tools.SendMessage(ctx, b, update.Message.Chat.ID, fmt.Sprintf("提醒：以下银行查询超时（>10s）：%s", strings.Join(mapBankNames(timeoutKeys), ", ")), update.Message.MessageThreadID, "")
		}
		return
	}

	// 排序（从低到高）
	sort.Slice(results, func(i, j int) bool { return results[i].BuySpotVal < results[j].BuySpotVal })

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
	sb.WriteString(fmt.Sprintf("现汇卖出最优排序 — %s\n", currencyDesc))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s: %s（发布时间: %s）\n", i+1, r.BankNameCN, r.BuySpotRaw, r.ReleaseTime))
	}
	// 成功获取结果后，先删除等待提示消息（忽略删除错误）
	_ = tools.DeleteMessage(ctx, b, update.Message.Chat.ID, waitMsgID)

	tools.SendMessage(ctx, b, update.Message.Chat.ID, sb.String(), update.Message.MessageThreadID, "")
	// 若有超时，额外提醒
	if len(timeoutKeys) > 0 {
		tools.SendMessage(ctx, b, update.Message.Chat.ID, fmt.Sprintf("提醒：以下银行查询超时（>10s）：%s", strings.Join(mapBankNames(timeoutKeys), ", ")), update.Message.MessageThreadID, "")
	}
}

// 将银行 key 列表映射为中文名
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

func fetchBOC(ctx context.Context, ccy string) *bankRate {
	r, found, err := bank.GetBOCRate(ctx, ccy)
	if err != nil || !found || r == nil {
		return nil
	}
	val, ok := ParseRate(r.SellSpot)
	if !ok {
		return nil
	}
	return &bankRate{
		BankNameCN:   "中国银行",
		BankKey:      "boc",
		CurrencyDesc: r.Name,
		BuySpotVal:   val,
		BuySpotRaw:   r.SellSpot,
		ReleaseTime:  r.ReleaseTime,
	}
}

func fetchCIB(ctx context.Context, ccy string) *bankRate {
	r, found, err := bank.GetCIBRate(ctx, ccy)
	if err != nil || !found || r == nil {
		return nil
	}
	val, ok := ParseRate(r.SellSpot)
	if !ok {
		return nil
	}
	return &bankRate{
		BankNameCN:   "兴业银行",
		BankKey:      "cib",
		CurrencyDesc: r.Name,
		BuySpotVal:   val,
		BuySpotRaw:   r.SellSpot,
		ReleaseTime:  r.ReleaseTime,
	}
}

func fetchCMB(ctx context.Context, ccy string) *bankRate {
	r, found, err := bank.GetCMBRate(ctx, ccy)
	if err != nil || !found || r == nil {
		return nil
	}
	val, ok := ParseRate(r.SellSpot)
	if !ok {
		return nil
	}
	return &bankRate{
		BankNameCN:   "招商银行",
		BankKey:      "cmb",
		CurrencyDesc: r.Name,
		BuySpotVal:   val,
		BuySpotRaw:   r.SellSpot,
		ReleaseTime:  r.ReleaseTime,
	}
}

// 寰宇人生（兴业银行优惠）：直接使用已折算后的现汇买入价（每100外币）
func fetchCIBLife(ctx context.Context, ccy string) *bankRate {
	r, found, err := bank.GetCIBLifeRate(ctx, ccy)
	if err != nil || !found || r == nil {
		return nil
	}
	val, ok := ParseRate(r.SellSpot)
	if !ok {
		return nil
	}
	return &bankRate{
		BankNameCN:   "寰宇人生",
		BankKey:      "hy",
		CurrencyDesc: r.Name,
		BuySpotVal:   val,
		BuySpotRaw:   r.SellSpot,
		ReleaseTime:  r.ReleaseTime,
	}
}

// 广发银行：统一以 100 单位为准
func fetchCGB(ctx context.Context, ccy string) *bankRate {
	r, found, err := bank.GetCGBRate(ctx, ccy)
	if err != nil || !found || r == nil {
		return nil
	}
	val, ok := ParseRate(r.SellSpot)
	if !ok {
		return nil
	}
	unit := r.Unit
	if unit <= 0 {
		unit = 100
	}
	scale := 100.0 / unit
	valPer100 := val * scale
	return &bankRate{
		BankNameCN:   "广发银行",
		BankKey:      "cgb",
		CurrencyDesc: r.Name,
		BuySpotVal:   valPer100,
		BuySpotRaw:   fmt.Sprintf("%.4f", valPer100),
		ReleaseTime:  r.ReleaseTime,
	}
}

func fetchCITIC(ctx context.Context, ccy string) *bankRate {
	r, found, err := bank.GetCITICRate(ctx, ccy)
	if err != nil || !found || r == nil {
		return nil
	}
	val, ok := ParseRate(r.SellSpot)
	if !ok {
		return nil
	}
	return &bankRate{
		BankNameCN:   "中信银行",
		BankKey:      "citic",
		CurrencyDesc: r.Name,
		BuySpotVal:   val,
		BuySpotRaw:   r.SellSpot,
		ReleaseTime:  r.ReleaseTime,
	}
}
 
