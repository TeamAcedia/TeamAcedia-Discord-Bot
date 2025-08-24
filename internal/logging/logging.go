package logging

import (
	"fmt"
	"teamacedia/discord-bot/internal/config"
	"time"

	"github.com/bwmarrin/discordgo"
)

type CachedMessage struct {
	Content  string
	AuthorID string
	Author   string
}

var messageCache = make(map[string]CachedMessage)

// Message Create Handler
func OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot {
		return
	}

	messageCache[m.ID] = CachedMessage{
		Content:  m.Content,
		AuthorID: m.Author.ID,
		Author:   fmt.Sprintf("<@%s> (%s#%s)", m.Author.ID, m.Author.Username, m.Author.Discriminator),
	}
}

// Message Update Handler
func OnMessageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
	if m.Author == nil || m.Author.Bot {
		return
	}

	channel, _ := s.Channel(m.ChannelID)
	channelName := "Unknown"
	clickableLink := "Unknown"
	if channel != nil {
		messageLink := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", channel.GuildID, channel.ID, m.ID)
		channelName = channel.Name
		clickableLink = fmt.Sprintf("[%s](%s)", channelName, messageLink)
	}

	oldMsg, ok := messageCache[m.ID]
	oldContent := ""
	if ok {
		oldContent = oldMsg.Content
	}

	messageCache[m.ID] = CachedMessage{
		Content:  m.Content,
		AuthorID: m.Author.ID,
		Author:   fmt.Sprintf("<@%s> (%s#%s)", m.Author.ID, m.Author.Username, m.Author.Discriminator),
	}

	embed := &discordgo.MessageEmbed{
		Title: "Message Edited",
		Color: 0xffff00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Channel", Value: clickableLink, Inline: true},
			{Name: "Author", Value: oldMsg.Author, Inline: true},
			{Name: "Old Content", Value: oldContent, Inline: false},
			{Name: "New Content", Value: m.Content, Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, _ = s.ChannelMessageSendEmbed(config.Config.LogChannelID, embed)
}

// Message Delete Handler
func OnMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	cached, ok := messageCache[m.ID]
	if !ok {
		return
	} else {
		delete(messageCache, m.ID)
	}

	channel, _ := s.Channel(m.ChannelID)
	channelName := "Unknown"
	clickableLink := "Unknown"
	if channel != nil {
		messageLink := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", channel.GuildID, channel.ID, m.ID)
		channelName = channel.Name
		clickableLink = fmt.Sprintf("[%s](%s)", channelName, messageLink)
	}

	embed := &discordgo.MessageEmbed{
		Title: "Message Deleted",
		Color: 0xff0000,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Channel", Value: clickableLink, Inline: true},
			{Name: "Author", Value: cached.Author, Inline: true},
			{Name: "Content", Value: cached.Content, Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, _ = s.ChannelMessageSendEmbed(config.Config.LogChannelID, embed)
}
