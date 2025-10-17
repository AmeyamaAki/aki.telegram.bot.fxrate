package bank

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const unionpayURL = "https://m.unionpayintl.com/jfimg/"

type UniopayRate struct {
	BaseCur     string // 基准币种代码（扣账币种）
	BaseName    string // 基准币种中文名
	TransCur    string // 目标币种代码（交易币种）
	TransName   string // 目标币种中文名
	Rate        string // 汇率（1 BaseCur = Rate TransCur）
	ReleaseTime string // 汇率发布时间
}

// ErrUnionPayRateNotFound 未找到对应的直接汇率
var ErrUnionPayRateNotFound = errors.New("unionpay: direct rate not found for given debit and transaction currency")

// UnionpayResponse 银联API返回的JSON结构（导出供外部使用）
type UnionpayResponse struct {
	ExchangeRateJson []UnionpayExchangeRate `json:"exchangeRateJson"`
	CurDate          string                 `json:"curDate"`
}

// UnionpayExchangeRate 单条汇率数据（导出供外部使用）
type UnionpayExchangeRate struct {
	TransCur string  `json:"transCur"` // 目标币种（这里表示被等式左侧的币种）
	BaseCur  string  `json:"baseCur"`  // 基准币种（等式右侧的币种）
	RateData float64 `json:"rateData"` // 汇率：1 TransCur = RateData BaseCur
}

// UnionpayCodeToCN 币种代码到中文名的映射
var UnionpayCodeToCN = map[string]string{
	"CNY": "人民币",
	"USD": "美元",
	"HKD": "港币",
	"EUR": "欧元",
	"GBP": "英镑",
	"JPY": "日元",
	"AUD": "澳大利亚元",
	"CAD": "加拿大元",
	"SGD": "新加坡元",
	"NZD": "新西兰元",
	"CHF": "瑞士法郎",
	"THB": "泰国铢",
	"TWD": "新台币",
	"KRW": "韩国元",
	"PHP": "菲律宾比索",
	"IDR": "印尼卢比",
	"INR": "印度卢比",
	"RUB": "卢布",
	"ZAR": "南非兰特",
	"AED": "阿联酋迪拉姆",
	"SAR": "沙特里亚尔",
	"MYR": "马来西亚林吉特",
	"VND": "越南盾",
	"BRL": "巴西雷亚尔",
	"MXN": "墨西哥比索",
	"TRY": "土耳其里拉",
}

// GetCurrencyName 获取币种中文名，如果没有映射则返回代码本身
func GetCurrencyName(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if name, ok := UnionpayCodeToCN[code]; ok {
		return name
	}
	return code
}

// GetUnionPayRate 获取指定货币对的汇率（扣账币种 -> 交易币种）
// debitCur: 扣账币种，transCur: 交易币种
// 仅使用JSON中的直接汇率（TransCur=debitCur 且 BaseCur=transCur）。未找到则返回 ErrUnionPayRateNotFound。
func GetUnionPayRate(ctx context.Context, debitCur, transCur string) (*UniopayRate, bool, error) {
	debitCur = strings.ToUpper(strings.TrimSpace(debitCur))
	transCur = strings.ToUpper(strings.TrimSpace(transCur))

	if transCur == "" {
		return nil, false, fmt.Errorf("unionpay: empty transaction currency")
	}
	if debitCur == "" {
		return nil, false, fmt.Errorf("unionpay: empty debit currency")
	}
	if debitCur == transCur {
		return nil, false, ErrUnionPayRateNotFound
	}

	resp, err := fetchUnionPayRates(ctx)
	if err != nil {
		return nil, false, err
	}

	for _, it := range resp.ExchangeRateJson {
		if strings.EqualFold(it.TransCur, debitCur) && strings.EqualFold(it.BaseCur, transCur) {
			// 直接数据：1 debitCur = RateData transCur
			rate := &UniopayRate{
				BaseCur:     debitCur,
				BaseName:    GetCurrencyName(debitCur),
				TransCur:    transCur,
				TransName:   GetCurrencyName(transCur),
				Rate:        fmt.Sprintf("%g", it.RateData),
				ReleaseTime: resp.CurDate,
			}
			return rate, true, nil
		}
	}
	return nil, false, ErrUnionPayRateNotFound
}

// fetchUnionPayRates 从银联API获取原始JSON数据
func fetchUnionPayRates(ctx context.Context) (*UnionpayResponse, error) {
	// 构建URL：获取今天的日期，格式为YYYYMMDD
	today := time.Now()

	// 尝试最多2天的数据（今天和昨天）
	for daysBack := 0; daysBack < 2; daysBack++ {
		targetDate := today.AddDate(0, 0, -daysBack)
		dateStr := targetDate.Format("20060102") // Go的时间格式：20060102 = YYYYMMDD
		url := fmt.Sprintf("%s%s.json", unionpayURL, dateStr)

		// 创建HTTP请求
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %w", err)
		}

		// 设置User-Agent
		req.Header.Set("User-Agent", ua)

		// 发起请求
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			if daysBack == 1 {
				return nil, fmt.Errorf("请求失败: %w", err)
			}
			continue
		}
		defer resp.Body.Close()

		// 检查响应状态码
		if resp.StatusCode == http.StatusNotFound {
			if daysBack == 0 {
				continue // 今天未发布，试昨天
			}
			return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			if daysBack == 1 {
				return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
			}
			continue
		}

		// 读取响应体
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			if daysBack == 1 {
				return nil, fmt.Errorf("读取响应失败: %w", err)
			}
			continue
		}

		// 解析JSON
		var response UnionpayResponse
		if err := json.Unmarshal(body, &response); err != nil {
			if daysBack == 1 {
				return nil, fmt.Errorf("解析JSON失败: %w", err)
			}
			continue
		}

		// 成功获取数据
		return &response, nil
	}

	return nil, fmt.Errorf("获取汇率数据失败")
}
