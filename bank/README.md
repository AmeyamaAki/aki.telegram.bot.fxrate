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

#### 寰宇人生借记卡

> [寰宇人生借记卡](https://mobile.cib.com.cn/app/abroad/intro/debit.html)面向境内外客户发行，满足客户外汇投资、境外旅游、出国留学、跨境薪酬结算、跨境支付提现等需求，并提供多种外汇服务权益。

```go
type CIBLifeRate struct {
	Name        string // 币种中文名
	Symbol      string // 币种符号
	BuySpot     string // 现汇买入价
	BuyCash     string // 现钞买入价
	SellSpot    string // 现汇卖出价
	SellCash    string // 现钞卖出价
	ReleaseTime string // 汇率发布时间
}
```

Use `bank.GetCIBLifeRate(ctx, "usd")` to get the USD exchange rate from 寰宇人生借记卡 with CIB.

The example is the same as above.

**Note: The exchange rate maybe different for that show in their app. Pls refer to the real transaction.**

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

---

## citic.go

```go
type CITICRate struct {
	Name        string // 币种中文名
	Symbol      string // 代码
	BuySpot     string // 结汇
	SellSpot    string // 购汇
	ReleaseTime string // 发布时间
}
```

use `bank.GETCITICRate(ctx,"usd")` to get the USD rate from CITIC

same as above

---

## cgb.go

```go
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
```

use `bank.GETCGBRate(ctx,"usd")` to get the USD rate from CITIC

same as above

---


## uniopay.go

```go
type UniopayRate struct {
	BaseCur     string // 基准币种代码（扣账币种）
	BaseName    string // 基准币种中文名
	TransCur    string // 目标币种代码（交易币种）
	TransName   string // 目标币种中文名
	Rate        string // 汇率（1 BaseCur = Rate TransCur）
	ReleaseTime string // 汇率发布时间
}
```

Use `bank.GetUnionPayRate(ctx, debit, trans)` to get the exchange rate from Uniopay.

Same as above.

Note: When the exchange rate is not published, an attempt is made to obtain the previous day's.