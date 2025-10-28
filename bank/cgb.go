package bank

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

const CGCURL = "https://www.cgbchina.com.cn/searchExchangePrice.gsp?internal_time=14"

type CGBRate struct {
	Name        string  // 币种中文名
	Symbol      string  // 币种代码
	Unit        float64 // 基数（1 或 100）
	MiddleRate  string  // 中间价
	BuySpot     string  // 现汇买入
	BuyCash     string  // 现钞买入
	SellSpot    string  // 现汇卖出
	SellCash    string  // 现钞卖出
	ReleaseTime string  // 发布时间
}

func GetCGBRate(ctx context.Context, query string) (*CGBRate, bool, error) {
	if strings.TrimSpace(query) == "" {
		return nil, false, nil
	}

	html, err := fetchCGBHTML(ctx)
	if err != nil {
		return nil, false, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, false, err
	}

	releaseTime, err := parseCGBReleaseTime(doc)
	if err != nil {
		return nil, false, err
	}
	targetCN := normalizeQueryCGB(query)
	targetCode := normalizeCodeCGB(query)

	table := doc.Find("table.ratetable").First()
	if table.Length() == 0 {
		return nil, false, fmt.Errorf("CGB: 未找到牌价表")
	}

	var out *CGBRate
	table.Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		if i == 0 {
			return true
		}
		tds := tr.Find("td")
		if tds.Length() < 8 {
			return true
		}

		nameRaw := strings.TrimSpace(tds.Eq(0).Text()) // 美元/人民币
		codeRaw := strings.TrimSpace(tds.Eq(1).Text()) // USD/CNY
		unitRaw := strings.TrimSpace(tds.Eq(2).Text())
		middle := strings.TrimSpace(tds.Eq(3).Text())
		buySpot := strings.TrimSpace(tds.Eq(4).Text())
		buyCash := strings.TrimSpace(tds.Eq(5).Text())
		sellSpot := strings.TrimSpace(tds.Eq(6).Text())
		sellCash := strings.TrimSpace(tds.Eq(7).Text())

		nameLeft := beforeSlash(nameRaw)
		codeLeft := beforeSlash(codeRaw)

		if !matchCGB(nameLeft, codeLeft, targetCN, targetCode) {
			return true
		}

		unit := parseUnit(unitRaw)

		out = &CGBRate{
			Name:        nz(nameLeft, "-"),
			Symbol:      nz(strings.ToUpper(codeLeft), "-"),
			Unit:        unit,
			MiddleRate:  nz(middle, "-"),
			BuySpot:     nz(buySpot, "-"),
			BuyCash:     nz(buyCash, "-"),
			SellSpot:    nz(sellSpot, "-"),
			SellCash:    nz(sellCash, "-"),
			ReleaseTime: nz(releaseTime, "-"),
		}
		return false
	})

	if out == nil {
		return nil, false, nil
	}
	return out, true, nil
}

// ---- HTTP ----

func fetchCGBHTML(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, CGCURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Connection", "close")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CGB request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CGB response returned status: %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// 解码
	enc, _, _ := charset.DetermineEncoding(data, resp.Header.Get("Content-Type"))
	reader := transform.NewReader(bytes.NewReader(data), enc.NewDecoder())
	utf8Data, err := io.ReadAll(reader)
	if err != nil {
		return data, nil
	}
	return utf8Data, nil
}

// ---- parsing helpers ----

func parseCGBReleaseTime(doc *goquery.Document) (string, error) {
	s := strings.TrimSpace(doc.Find("span._times").First().Text())
	s = strings.TrimPrefix(s, "发布时间为：")
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("CGB: 未找到发布时间")
	}
	return s, nil
}

func beforeSlash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if idx := strings.Index(s, "/"); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

func parseUnit(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 100 // 怎么就你用 1 单位啊？
	}
	var v float64 = 100
	for _, r := range s {
		if r < '0' || r > '9' {
			return v
		}
	}
	if s == "1" {
		return 1
	}
	if s == "100" {
		return 100
	}
	return 100
}

func normalizeQueryCGB(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return q
	}
	lq := strings.ToLower(q)
	lq = strings.ReplaceAll(lq, "_", "")
	lq = strings.ReplaceAll(lq, "-", "")
	lq = strings.ReplaceAll(lq, "/", "")
	if cn, ok := codeToCN[lq]; ok {
		// 允许用代码匹配到中文名
		return cn
	}
	return q
}

func matchCGB(nameLeft, codeLeft, targetCN, targetCode string) bool {
	nameCN := unifyCN(strings.TrimSpace(nameLeft))
	code := strings.TrimSpace(codeLeft)
	tCN := unifyCN(strings.TrimSpace(targetCN))
	tCode := strings.ToUpper(strings.TrimSpace(targetCode))

	if tCode != "" && strings.EqualFold(code, tCode) {
		return true
	}
	if tCN != "" && (strings.Contains(nameCN, tCN) || strings.Contains(tCN, nameCN)) {
		return true
	}
	return false
}

// 返回币种代码
func normalizeCodeCGB(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	lq := strings.ToLower(q)
	lq = strings.ReplaceAll(lq, "_", "")
	lq = strings.ReplaceAll(lq, "-", "")
	lq = strings.ReplaceAll(lq, "/", "")
	// 仅保留字母
	var b strings.Builder
	for _, r := range lq {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if len(s) == 3 { // 常见三字母代码
		return strings.ToUpper(s)
	}
	return ""
}

func unifyCN(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "港币", "港元")
	s = strings.ReplaceAll(s, "澳门币", "澳门元")
	s = strings.ReplaceAll(s, "台币", "新台币")
	return s
}