# aki.telegram.bot.fxrate/bank

This directory contains the code related to banking functionality in the `aki.telegram.bot.fxrate` project.

well... The code is a mess and needs to be sorted out later

read .go file you want to use to get you want to use functions.

## Usage

### boc.go

```go
type BOCRate struct {
	Name        string // 币种中文名
	BuySpot     string // 现汇买入价
	BuyCash     string // 现钞买入价
	SellSpot    string // 现汇卖出价
	SellCash    string // 现钞卖出价
	BankRate    string // 中行折算价
	ReleaseTime string // 汇率发布时间
}
```

```go
package main

import (
	"context"
	"fmt"

	"aki.telegram.bot.fxrate/bank"
)

func main() {
	ctx := context.Background()

	// 查询港元汇率
	rate, found, err := bank.GetBOCRate(ctx, "hkd") // or you can use "港元" or "HKD"
	if err != nil {
		fmt.Errorf("查询失败: %v", err)
		return
	}
	if !found {
		fmt.Errorf("未找到相关币种")
		return
	}

	fmt.Printf("%s: BuySpot=%s, SellSpot=%s, ReleaseTime=%s\n", rate.Name, rate.BuySpot, rate.SellSpot, rate.ReleaseTime)
}
```

### cib.go

```go
type CIBRate struct {
	Name        string // 币种中文名
	Symbol      string // 币种符号
	BuySpot     string // 现汇买入价
	BuyCash     string // 现钞买入价
	SellSpot    string // 现汇卖出价
	SellCash    string // 现钞卖出价
	ReleaseTime string // 汇率发布时间
}
```

Use `bank.GetCIBRate(ctx, "usd")` to get the USD exchange rate from CIB.

The example is the same as above.

---

## cmb.go

```go
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
```

Use `bank.GetCMBRate(ctx, "usd")` to get the USD exchange rate from CMB.

Same as above.