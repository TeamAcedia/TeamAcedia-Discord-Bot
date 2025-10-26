package anonimize

import (
	"fmt"
	"strings"
	"teamacedia/discord-bot/internal/config"

	"github.com/bwmarrin/discordgo"
)

// Helper function to split webhook URL into ID and Token so I can use the WebhookExecute function
func SplitWebhookURL(url string) (string, string, error) {
	url = strings.TrimSuffix(url, "/")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid webhook URL")
	}
	id := parts[len(parts)-2]
	token := parts[len(parts)-1]
	return id, token, nil
}

// Message Create Handler
func OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	if m.Author == nil || m.Author.Bot {
		return false
	}

	// Check if the message is in the anon channel
	if m.ChannelID != config.Config.AnonChannelID {
		return false
	}

	id, token, err := SplitWebhookURL(config.Config.AnonWebhook)
	if err != nil {
		return false
	}

	// Send the message to the anon webhook but use the same username and avatar as the message sender
	_, err = s.WebhookExecute(id, token, false, &discordgo.WebhookParams{
		Content:   m.Content,
		Username:  m.Author.DisplayName(),
		AvatarURL: m.Author.AvatarURL(""),
	})

	if err != nil {
		return false
	}

	// Delete the original message
	s.ChannelMessageDelete(m.ChannelID, m.ID)

	return true
}
