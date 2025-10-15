# aki.telegram.bot.fxrate/bank

This directory contains the code related to banking functionality in the `aki.telegram.bot.fxrate` project.

well... The code is a mess and needs to be sorted out later

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
	rate, found, err := bank.GetBOCRate(ctx, "hkd")
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