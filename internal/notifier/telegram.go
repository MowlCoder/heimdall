package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/MowlCoder/heimdall/internal/domain"
)

type TelegramNotifier struct {
	ChatID   string
	BotToken string
}

func NewTelegramNotifier(chatId string, botToken string) *TelegramNotifier {
	return &TelegramNotifier{
		ChatID:   chatId,
		BotToken: botToken,
	}
}

func (n *TelegramNotifier) Notify(serviceErr *domain.ServiceError) error {
	sb := strings.Builder{}

	sb.WriteString("🚨 <b>Service Alert</b> 🚨\n\n")
	sb.WriteString(fmt.Sprintf("🔧 Service: %s\n", serviceErr.Name))

	if serviceErr.StatusCode != 0 {
		sb.WriteString(fmt.Sprintf("📊 Status Code: %d\n", serviceErr.StatusCode))
	}

	if len(serviceErr.Body) > 0 {
		sb.WriteString(fmt.Sprintf("📄 Response body: %s\n", serviceErr.Body))
	}

	if serviceErr.Error != nil {
		sb.WriteString(fmt.Sprintf("❌ Error: %v\n", serviceErr.Error))
	}

	body := struct {
		ChatID    string `json:"chat_id"`
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode"`
	}{
		ChatID:    n.ChatID,
		Text:      sb.String(),
		ParseMode: "HTML",
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.BotToken),
		"application/json",
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error when sending message to telegram: %s", respBody)
	}

	return nil
}
