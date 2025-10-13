package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		LogError("Error loading .env file: %v", err)
		os.Exit(1)
	}
	LogInfo("Environment variables loaded")

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(HandleCommand),
	}

	b, err := bot.New(botToken, opts...)
	if err != nil {
		LogError("Error creating bot: %v", err)
	} else {
		LogInfo("Bot created")
	}
	b.Start(ctx)
}
