package models

type Config struct {
	Token                  string
	AppID                  string
	GuildID                string
	ReactionRolesChannelID string
	ReactionRoles          []ReactionRole
	LogChannelID           string
	MemberRoleID           string
	AnonWebhook            string
	AnonChannelID          string
}

type ReactionRole struct {
	ID    string
	Name  string
	Emoji string
}
