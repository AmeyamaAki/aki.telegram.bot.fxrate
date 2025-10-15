// message.go
package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func parseModeFromString(mode string) models.ParseMode {
	switch mode {
	case "Markdown":
		return models.ParseModeMarkdown
	case "MarkdownV1":
		return models.ParseModeMarkdownV1
	case "HTML":
		return models.ParseModeHTML
	default:
		return models.ParseModeHTML
	}
}

func GetUserNickName(update *models.Update) string {
	if update.Message == nil || update.Message.From == nil {
		return ""
	}

	firstName := update.Message.From.FirstName
	lastName := update.Message.From.LastName

	if lastName != "" {
		return fmt.Sprintf("%s %s", lastName, firstName)
	}
	return firstName
}

func getUserNicknameByID(ctx context.Context, b *bot.Bot, userID int64) (string, error) {
	chat, err := b.GetChat(ctx, &bot.GetChatParams{
		ChatID: userID,
	})
	if err != nil {
		return "", err
	}

	firstName := chat.FirstName
	lastName := chat.LastName

	if lastName != "" {
		return fmt.Sprintf("%s %s", firstName, lastName), nil
	}
	return firstName, nil
}

func SendMessage(ctx context.Context, b *bot.Bot, chatID int64, message string, messageThreadID int, parseMode string) {
	params := &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      message,
		ParseMode: parseModeFromString(parseMode),
	}
	if messageThreadID > 0 {
		params.MessageThreadID = messageThreadID
	}

	_, err := b.SendMessage(ctx, params)
	if err != nil {
		LogError("Error sending message: %v", err)
	}
}

func SendDocument(ctx context.Context, b *bot.Bot, chatID int64, topicID *int, filePath string) {
	fileData, errReadFile := os.ReadFile(filePath)
	if errReadFile != nil {
		LogError("Error reading file: %v", errReadFile)
		return
	}

	file := &models.InputFileUpload{
		Filename: filePath,
		Data:     bytes.NewReader(fileData),
	}

	params := &bot.SendDocumentParams{
		ChatID:   chatID,
		Document: file,
	}

	if topicID != nil {
		params.MessageThreadID = *topicID
	}

	_, err := b.SendDocument(ctx, params)
	if err != nil {
		LogError("Error sending document: %v", err)
	}
}
