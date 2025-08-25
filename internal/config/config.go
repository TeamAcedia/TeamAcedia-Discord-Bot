package config

import (
	"errors"
	"strings"
	"teamacedia/discord-bot/internal/models"

	"gopkg.in/ini.v1"
)

var Config *models.Config

// LoadConfig loads Config from an INI file
func LoadConfig(path string) (*models.Config, error) {
	cfgFile, err := ini.Load(path)
	if err != nil {
		return nil, err
	}

	reactionRoles, err := ParseReactionRoles(cfgFile.Section("").Key("ReactionRoles").String())
	if err != nil {
		return nil, err
	}

	cfg := &models.Config{
		Token:                  cfgFile.Section("").Key("Token").String(),
		AppID:                  cfgFile.Section("").Key("AppID").String(),
		GuildID:                cfgFile.Section("").Key("GuildID").String(),
		ReactionRolesChannelID: cfgFile.Section("").Key("ReactionRolesChannelID").String(),
		LogChannelID:           cfgFile.Section("").Key("LogChannelID").String(),
		MemberRoleID:           cfgFile.Section("").Key("MemberRoleID").String(),
		ReactionRoles:          reactionRoles,
	}

	return cfg, nil
}

// ParseReactionRoles parses a |-delimited string of reaction roles.
// Format: ROLEID,ROLENAME,ROLEEMOJI|ROLEID2,ROLENAME2,ROLEEMOJI2|...
func ParseReactionRoles(data string) ([]models.ReactionRole, error) {
	if strings.TrimSpace(data) == "" {
		return nil, errors.New("input string is empty")
	}

	parts := strings.Split(data, "|")
	roles := make([]models.ReactionRole, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		fields := strings.Split(part, ",")
		if len(fields) != 3 {
			return nil, errors.New("invalid format: expected ROLEID,ROLENAME,ROLEEMOJI")
		}

		id := strings.TrimSpace(fields[0])
		name := strings.TrimSpace(fields[1])
		emoji := strings.TrimSpace(fields[2])

		if id == "" || name == "" || emoji == "" {
			return nil, errors.New("invalid entry: one or more fields are empty")
		}

		roles = append(roles, models.ReactionRole{
			ID:    id,
			Name:  name,
			Emoji: emoji,
		})
	}

	return roles, nil
}
