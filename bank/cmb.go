package bank

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const cmbURL = "https://fx.cmbchina.com/api/v1/fx/rate"

type CMBRate struct {
	Name        string // 币种中文名
	Symbol      string // 币种代码
	BuySpot     string // 现汇买入价 rthBid
	BuyCash     string // 现钞买入价 rtcBid
	SellSpot    string // 现汇卖出价 rthOfr
	SellCash    string // 现钞卖出价 rtcOfr
	BankRate    string // 招行折算价 rtbBid
	ReleaseTime string // 汇率发布时间
}

// GetCMBRate 通过“代码/中文名/模糊匹配”获取单币种牌价
// 返回：rate，found，error
func GetCMBRate(ctx context.Context, query string) (*CMBRate, bool, error) {
	rows, err := fetchCMBRows(ctx)
	if err != nil {
		return nil, false, err
	}

	target := strings.TrimSpace(query)
	lt := strings.ToLower(target)

	for _, r := range rows {
		name := strings.TrimSpace(r.CcyNbr)
		eng := strings.TrimSpace(r.CcyNbrEng)
		symbol := extractCMBSymbol(eng)

		if !matchCMBCurrency(name, eng, symbol, lt) {
			continue
		}

		ts := composeCMBTime(r.RatDat, r.RatTim)

		rate := &CMBRate{
			Name:        nz(name, "-"),
			Symbol:      nz(symbol, "-"),
			BuySpot:     nz(strings.TrimSpace(r.RthBid), "-"),
			BuyCash:     nz(strings.TrimSpace(r.RtcBid), "-"),
			SellSpot:    nz(strings.TrimSpace(r.RthOfr), "-"),
			SellCash:    nz(strings.TrimSpace(r.RtcOfr), "-"),
			BankRate:    nz(strings.TrimSpace(r.RtbBid), "-"),
			ReleaseTime: nz(ts, "-"),
		}
		return rate, true, nil
	}

	return nil, false, nil
}

type cmbResponse struct {
	ReturnCode string       `json:"returnCode"`
	ErrorMsg   *string      `json:"errorMsg"`
	Body       []cmbRowItem `json:"body"`
}

type cmbRowItem struct {
	CcyNbr    string `json:"ccyNbr"`
	CcyNbrEng string `json:"ccyNbrEng"`
	RtbBid    string `json:"rtbBid"`
	RthOfr    string `json:"rthOfr"`
	RtcOfr    string `json:"rtcOfr"`
	RthBid    string `json:"rthBid"`
	RtcBid    string `json:"rtcBid"`
	RatTim    string `json:"ratTim"`
	RatDat    string `json:"ratDat"`
	CcyExc    string `json:"ccyExc"`
}

func fetchCMBRows(ctx context.Context) ([]cmbRowItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cmbURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", ua)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CMB request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CMB response returned status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload cmbResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("CMB json parse error: %v", err)
	}

	if !strings.EqualFold(payload.ReturnCode, "SUC0000") {
		return nil, fmt.Errorf("CMB api returnCode=%s, errorMsg=%v", payload.ReturnCode, payload.ErrorMsg)
	}

	return payload.Body, nil
}

func extractCMBSymbol(eng string) string {
	eng = strings.TrimSpace(eng)
	if eng == "" {
		return ""
	}
	// 形如："美元 USD" 或 "港币 HKD"，取最后一个 token 作为代码
	parts := strings.Fields(eng)
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	// 仅保留英文字母
	var b strings.Builder
	for _, r := range last {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			b.WriteRune(r)
		}
	}
	return strings.ToUpper(b.String())
}

func composeCMBTime(ratDat, ratTim string) string {
	ratDat = strings.TrimSpace(ratDat)
	ratTim = strings.TrimSpace(ratTim)
	if ratDat == "" && ratTim == "" {
		return ""
	}

	s := ratDat
	s = strings.ReplaceAll(s, "年", ".")
	s = strings.ReplaceAll(s, "月", ".")
	s = strings.ReplaceAll(s, "日", "")
	s = strings.TrimSuffix(s, ".")
	s = strings.TrimSpace(s)

	if ratTim != "" {
		if s != "" {
			s += " "
		}
		s += ratTim
	}
	return s
}

// 招行更新汇率的频率好快啊...

func matchCMBCurrency(nameCN, eng, code, ltTarget string) bool {
	if ltTarget == "" {
		return false
	}
	// 中文名直接包含
	if strings.Contains(nameCN, ltTarget) {
		return true
	}
	// 英文整串包含（不区分大小写）
	if strings.Contains(strings.ToLower(eng), ltTarget) {
		return true
	}
	// 代码完全匹配
	if strings.EqualFold(code, ltTarget) {
		return true
	}
	return false
}
