// /bank/boc.go
package bank

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	bocURL = "https://www.boc.cn/sourcedb/whpj/index.html"
	ua     = "aki.telegram.bot.fxrate/1.0 (+https://aki.cat)"
)

var codeToCN = map[string]string{
	"usd": "美元",
	"hkd": "港币",
	"eur": "欧元",
	"gbp": "英镑",
	"jpy": "日元",
	"aud": "澳大利亚元",
	"cad": "加拿大元",
	"sgd": "新加坡元",
	"nzd": "新西兰元",
	"chf": "瑞士法郎",
	"thb": "泰国铢",
	"twd": "新台币",
	"krw": "韩国元",
	"php": "菲律宾比索",
	"idr": "印尼卢比",
	"inr": "印度卢比",
	"rub": "卢布",
	"zar": "南非兰特",
	"aed": "阿联酋迪拉姆",
	"sar": "沙特里亚尔",
	"huf": "匈牙利福林",
	"czk": "捷克克朗",
	"sek": "瑞典克朗",
	"dkk": "丹麦克朗",
	"nok": "挪威克朗",
	"mxn": "墨西哥比索",
	"ils": "以色列谢克尔",
	"try": "土耳其里拉",
	"brl": "巴西里亚尔",
	"vnd": "越南盾",
	"bnd": "文莱元",
	"kwd": "科威特第纳尔",
	"npr": "尼泊尔卢比",
	"pkr": "巴基斯坦卢比",
	"qar": "卡塔尔里亚尔",
	"mnt": "蒙古图格里克",
	"mop": "澳门元",
}

// BuildBOCMessage 访问中国银行外汇牌价页面，查找目标币种，并返回格式化消息。
// 返回值：message、found、error
func BuildBOCMessage(ctx context.Context, query string) (string, bool, error) {
	target := normalizeQuery(query)

	doc, err := fetchBOCDoc(ctx)
	if err != nil {
		return "", false, err
	}

	table := locateRateTable(doc)
	if table == nil {
		return "", false, fmt.Errorf("未在页面上找到牌价表")
	}

	var (
		found  bool
		result string
	)

	table.Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		// 跳过表头
		if i == 0 {
			return true
		}
		tds := tr.Find("td")
		if tds.Length() < 2 {
			return true
		}
		name := strings.TrimSpace(tds.Eq(0).Text())
		if matchCurrency(name, target) {
			// 常见列顺序（可能变动）：币种/货币名称、现汇买入价、现钞买入价、现汇卖出价、现钞卖出价、中行折算价、发布日期、发布时间
			buySpot := getTD(tds, 1)
			buyCash := getTD(tds, 2)
			sellSpot := getTD(tds, 3)
			sellCash := getTD(tds, 4)
			refRate := getTD(tds, 5)

			// 第7列可能已包含“日期+时间”，第8列为时间；避免重复拼接
			date := getTD(tds, 6)
			timeStr := getTD(tds, 7)
			ts := strings.TrimSpace(date)
			if ts == "" {
				ts = strings.TrimSpace(timeStr)
			} else if timeStr != "" && !strings.Contains(ts, timeStr) {
				ts = strings.TrimSpace(ts + " " + timeStr)
			}

			result = fmt.Sprintf(
				"中国银行外汇牌价 — %s\n"+
					"现汇买入价: %s\n"+
					"现钞买入价: %s\n"+
					"现汇卖出价: %s\n"+
					"现钞卖出价: %s\n"+
					"中行折算价: %s\n"+
					"发布时间: %s",
				name, nz(buySpot, "-"), nz(buyCash, "-"), nz(sellSpot, "-"), nz(sellCash, "-"), nz(refRate, "-"), nz(ts, "-"),
			)
			found = true
			return false
		}
		return true
	})

	if !found {
		return "", false, nil
	}
	return result, true, nil
}

func normalizeQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return q
	}
	lq := strings.ToLower(q)
	// 去掉可能的符号与空格
	lq = strings.ReplaceAll(lq, "_", "")
	lq = strings.ReplaceAll(lq, "-", "")
	lq = strings.ReplaceAll(lq, "/", "")
	if cn, ok := codeToCN[lq]; ok {
		return cn
	}
	// 若已是中文名称或部分匹配，直接返回
	return q
}

func matchCurrency(name string, target string) bool {
	name = strings.TrimSpace(name)
	target = strings.TrimSpace(target)
	if name == "" || target == "" {
		return false
	}
	if strings.EqualFold(name, target) {
		return true
	}
	return strings.Contains(name, target) || strings.Contains(target, name)
}

func getTD(tds *goquery.Selection, idx int) string {
	if tds.Length() <= idx {
		return ""
	}
	return strings.TrimSpace(tds.Eq(idx).Text())
}

func nz(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// 仅保留 TLS1.2 + TLS_RSA_WITH_AES_256_CBC_SHA 的最小客户端

func bocClient() *http.Client {
	tr := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: false,
		TLSNextProto:      map[string]func(string, *tls.Conn) http.RoundTripper{},
		TLSClientConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			MaxVersion:   tls.VersionTLS12,
			ServerName:   "www.boc.cn",                               // SNI
			NextProtos:   []string{"http/1.1"},                       // 仅 http/1.1
			CipherSuites: []uint16{tls.TLS_RSA_WITH_AES_256_CBC_SHA}, // 仅 RSA-CBC-256
		},
	}
	return &http.Client{Transport: tr, Timeout: 12 * time.Second}
}

// FetchBOCHTML 获取并返回中国银行外汇牌价页面 HTML（按页面编码解码）
func FetchBOCHTML(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bocURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Connection", "close")

	client := bocClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BOC request failed: %v", err)
	}
	defer resp.Body.Close()

	// Ensure successful response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BOC response returned status: %d", resp.StatusCode)
	}

	htmlBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return htmlBytes, nil
}

func fetchBOCDoc(ctx context.Context) (*goquery.Document, error) {
	htmlBytes, err := FetchBOCHTML(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBytes))
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func locateRateTable(doc *goquery.Document) *goquery.Selection {
	var found *goquery.Selection
	doc.Find("table").EachWithBreak(func(i int, t *goquery.Selection) bool {
		head := strings.TrimSpace(t.Find("tr").First().Text())
		// 只要表头包含“货币名称”即可视为目标表
		if strings.Contains(head, "货币名称") {
			found = t
			return false
		}
		return true
	})
	return found
}
