package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// sendTelegram sends a message to the configured Telegram chat
func sendTelegram(message string) {
	telegramMsg := TelegramMessage{
		ChatID:    config.ChatID,
		Text:      message,
		ParseMode: "HTML",
	}

	jsonData, err := json.Marshal(telegramMsg)
	if err != nil {
		log.Printf("Failed to marshal telegram message: %v", err)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.BotToken)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to send telegram message: %v", err)
		return
	}
	defer resp.Body.Close()
}