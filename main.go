package main

import (
	"context"
	"os"
	"os/signal"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/joho/godotenv"
)

func main() {
	// 仅当本地存在 `.env` 时加载，容器里通常没有
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			LogError("加载 .env 失败: %v", err)
			os.Exit(1)
		}
		LogInfo("已从 .env 加载环境变量")
	} else {
		LogInfo("未发现 .env，使用环境变量")
	}

	botToken := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if botToken == "" {
		LogError("缺少环境变量 TELEGRAM_BOT_TOKEN")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(HandleCommand),
	}

	b, err := bot.New(botToken, opts...)
	if err != nil {
		LogError("创建 Bot 时出错: %v", err)
	} else {
		LogInfo("Bot 创建完毕")
	}
	b.Start(ctx)
}
