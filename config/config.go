package config

type Config struct {
	MetaTitle string
	BodyTitle string
	Repos     []Repository `mapstructure:"repositories"`
}
