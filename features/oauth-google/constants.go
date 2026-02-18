package oauthgoogle

type Prompt string

const (
	PromptConsent       Prompt = "consent"
	PromptSelectAccount Prompt = "select_account"
)

type AccessType string

const (
	AccessTypeOffline AccessType = "offline"
	AccessTypeOnline  AccessType = "online"
)
