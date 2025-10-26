package anonimize

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"teamacedia/discord-bot/internal/config"
	"time"

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

// OnMessageCreate handles anonymous reposting with proper file support
func OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	if m.Author == nil || m.Author.Bot {
		return false
	}

	// Only handle anon channel
	if m.ChannelID != config.Config.AnonChannelID {
		return false
	}

	id, token, err := SplitWebhookURL(config.Config.AnonWebhook)
	if err != nil {
		return false
	}

	content := strings.TrimSpace(m.Content)

	// Handle replies or forwards
	if m.MessageReference != nil && m.MessageReference.MessageID != "" {
		// Create a link to the replied message
		link := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", m.GuildID, m.MessageReference.ChannelID, m.MessageReference.MessageID)
		content = fmt.Sprintf("Reply > %s\n%s", link, content)
	}

	params := &discordgo.WebhookParams{
		Content:   content,
		Username:  m.Author.DisplayName(),
		AvatarURL: m.Author.AvatarURL(""),
	}

	// Download attachments before deletion
	client := &http.Client{Timeout: 10 * time.Second}

	for _, a := range m.Attachments {
		resp, err := client.Get(a.URL)
		if err != nil {
			params.Content += "\n" + a.URL
			continue
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || len(data) == 0 {
			params.Content += "\n" + a.URL
			continue
		}

		params.Files = append(params.Files, &discordgo.File{
			Name:   a.Filename,
			Reader: bytes.NewReader(data),
		})
	}

	if params.Content == "" && len(params.Files) == 0 {
		return false
	}

	if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
		return false
	}

	if _, err := s.WebhookExecute(id, token, false, params); err != nil {
		return false
	}

	return true
}
