package bank

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	cibURL    = "https://personalbank.cib.com.cn/pers/main/pubinfo/ifxQuotationQuery.do"
	cibAPIURL = "https://personalbank.cib.com.cn/pers/main/pubinfo/ifxQuotationQuery/list?_search=false&dataSet.nd=%d&dataSet.rows=80&dataSet.page=1&dataSet.sidx=&dataSet.sord=asc"
	// ua 在 boc.go 里定义了
)

type CIBRate struct {
	Name        string // 币种中文名
	Symbol      string // 币种符号
	BuySpot     string // 现汇买入价
	BuyCash     string // 现钞买入价
	SellSpot    string // 现汇卖出价
	SellCash    string // 现钞卖出价
	ReleaseTime string // 汇率发布时间
}

// GetCIBRate 通过“代码/中文名/模糊匹配”获取单币种牌价
// 返回：rate，found，error
func GetCIBRate(ctx context.Context, query string) (*CIBRate, bool, error) {
	// 1) 获取 HTML 以拿 Cookie 和更新时间
	html, cookie, err := fetchCIBHTML(ctx)
	if err != nil {
		return nil, false, err
	}
	releaseTime := parseCIBReleaseTime(html)

	// 2) 请求 JSON 列表
	rows, err := fetchCIBRows(ctx, cookie)
	if err != nil {
		return nil, false, err
	}

	target := normalizeQueryCIB(query)

	// 3) 遍历匹配
	for _, cells := range rows {
		if len(cells) < 7 {
			continue
		}

		// 代码与名称直接来自接口
		code := strings.TrimSpace(toStr(cells[1]))
		name := strings.TrimSpace(toStr(cells[0]))
		if name == "" {
			name = code
		}

		if !matchCIBCurrency(name, code, target) {
			continue
		}

		rate := &CIBRate{
			Name:        name,
			Symbol:      code,
			BuySpot:     nz(strings.TrimSpace(toStr(cells[3])), "-"),
			SellSpot:    nz(strings.TrimSpace(toStr(cells[4])), "-"),
			BuyCash:     nz(strings.TrimSpace(toStr(cells[5])), "-"),
			SellCash:    nz(strings.TrimSpace(toStr(cells[6])), "-"),
			ReleaseTime: nz(releaseTime, "-"),
		}
		return rate, true, nil
	}

	return nil, false, nil
}

// 使用默认传输层的简单 HTTP 客户端（保留超时）
// 照抄的时候懒得重构了(x
func cibClient() *http.Client {
	return &http.Client{Timeout: 12 * time.Second}
}

func fetchCIBHTML(ctx context.Context) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cibURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Connection", "close")

	client := cibClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("CIB request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("CIB response returned status: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// 聚合 Cookie
	setCookies := resp.Header.Values("Set-Cookie")
	cookie := joinSetCookie(setCookies)

	return body, cookie, nil
}

func fetchCIBRows(ctx context.Context, cookie string) ([][]any, error) {
	url := fmt.Sprintf(cibAPIURL, time.Now().UnixMilli())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json,text/javascript,*/*;q=0.1")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	client := cibClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CIB list request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CIB list response returned status: %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Rows []struct {
			Cell []any `json:"cell"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("CIB list json parse error: %v", err)
	}

	out := make([][]any, 0, len(payload.Rows))
	for _, r := range payload.Rows {
		out = append(out, r.Cell)
	}
	return out, nil
}

func parseCIBReleaseTime(html []byte) string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return ""
	}
	raw := strings.TrimSpace(doc.Find(".labe_text").First().Text())
	if raw == "" {
		return ""
	}

	// 清洗：去掉换行/制表/前缀，压缩多空格
	s := strings.ReplaceAll(raw, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "日期：", "")
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")

	// 去掉“星期x”
	parts := strings.Fields(s)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.Contains(p, "星期") {
			continue
		}
		out = append(out, p)
	}
	s = strings.Join(out, " ")

	// 年月日 -> 标准
	s = strings.ReplaceAll(s, "年", "-")
	s = strings.ReplaceAll(s, "月", "-")
	s = strings.ReplaceAll(s, "日", "")
	s = strings.TrimSpace(s)

	return s
}

func joinSetCookie(items []string) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	for i, c := range items {
		seg := c
		if idx := strings.Index(seg, ";"); idx >= 0 {
			seg = seg[:idx]
		}
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		if b.Len() > 0 && i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(seg)
	}
	return b.String()
}

func toStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	default:
		return fmt.Sprint(v)
	}
}

func normalizeQueryCIB(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return q
	}
	lq := strings.ToLower(q)
	lq = strings.ReplaceAll(lq, "_", "")
	lq = strings.ReplaceAll(lq, "-", "")
	lq = strings.ReplaceAll(lq, "/", "")
	return lq
}

func matchCIBCurrency(name, code, target string) bool {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	target = strings.TrimSpace(target)
	if name == "" || target == "" {
		return false
	}
	// 直接相等（中英文）
	if strings.EqualFold(name, target) || strings.EqualFold(code, target) {
		return true
	}
	// 模糊包含
	return strings.Contains(name, target) ||
		strings.Contains(target, name) ||
		strings.Contains(strings.ToLower(code), strings.ToLower(target)) ||
		strings.Contains(strings.ToLower(target), strings.ToLower(code))
}
