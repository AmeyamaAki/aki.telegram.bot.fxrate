package bank

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const citicURL = "https://etrade.citicbank.com/portalweb/cms/getForeignExchRate.htm"

type CITICRate struct {
	Name        string // 币种中文名
	Symbol      string // 代码
	BuySpot     string // 结汇
	SellSpot    string // 购汇
	ReleaseTime string // 发布时间
}

func GetCITICRate(ctx context.Context, query string) (*CITICRate, bool, error) {
	rows, err := fetchCITICRows(ctx)
	if err != nil {
		return nil, false, err
	}

	target := strings.TrimSpace(query)
	if target == "" {
		return nil, false, nil
	}
	lt := strings.ToLower(target)
	lt = strings.ReplaceAll(lt, "_", "")
	lt = strings.ReplaceAll(lt, "-", "")
	lt = strings.ReplaceAll(lt, "/", "")

	for _, r := range rows {
		name := strings.TrimSpace(r.CurName)

		code3 := citicNameToCode(name)
		if !(strings.Contains(name, target) || strings.EqualFold(code3, lt)) {
			continue
		}

		ts := composeCITICTime(r.QuotePriceDate, r.QuotePriceTime)
		rate := &CITICRate{
			Name:        nz(name, "-"),
			Symbol:      nz(code3, "-"),
			BuySpot:     nz(strings.TrimSpace(r.CstexcBuyPrice), "-"),
			SellSpot:    nz(strings.TrimSpace(r.CstexcSellPrice), "-"),
			ReleaseTime: nz(ts, "-"),
		}
		return rate, true, nil
	}

	return nil, false, nil
}

type citicResponse struct {
	RetCode string `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Content struct {
		ResultList []citicItem `json:"resultList"`
	} `json:"content"`
}

type citicItem struct {
	QuotePriceDate  string `json:"quotePriceDate"`
	QuotePriceTime  string `json:"quotePriceTime"`
	CurName         string `json:"curName"`
	CurCode         string `json:"curCode"`
	CstexcBuyPrice  string `json:"cstexcBuyPrice"`
	CstexcSellPrice string `json:"cstexcSellPrice"`
}

func fetchCITICRows(ctx context.Context) ([]citicItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, citicURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", ua)

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CITIC request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CITIC response returned status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload citicResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("CITIC json parse error: %v", err)
	}
	if !strings.EqualFold(payload.RetCode, "AAAAAAA") {
		return nil, fmt.Errorf("CITIC api retCode=%s, retMsg=%s", payload.RetCode, payload.RetMsg)
	}
	return payload.Content.ResultList, nil
}

func composeCITICTime(d, t string) string {
	d = strings.TrimSpace(d)
	t = strings.TrimSpace(t)
	if d == "" && t == "" {
		return ""
	}
	s := d
	s = strings.ReplaceAll(s, "年", ".")
	s = strings.ReplaceAll(s, "月", ".")
	s = strings.ReplaceAll(s, "日", "")
	s = strings.TrimSuffix(s, ".")
	s = strings.TrimSpace(s)
	if t != "" {
		if s != "" {
			s += " "
		}
		s += t
	}
	return s
}

func citicNameToCode(cn string) string {
	name := strings.TrimSpace(cn)
	if name == "" {
		return ""
	}
	// 查找现有的
	for code, n := range codeToCN {
		if strings.EqualFold(n, name) {
			return strings.ToUpper(code)
		}
	}
	// 别的名字或者说没有的
	switch name {
	case "韩元":
		return "KRW"
	case "坚戈":
		return "KZT"
	default:
		return ""
	}
}