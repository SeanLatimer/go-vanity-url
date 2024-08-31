package config

type Repository struct {
	Url         string
	Branch      string
	Protocol    Protocol
	Path        string
	VanityUrl   string `mapstructure:"vanity-url"`
	Description string
	Hidden      bool
}
