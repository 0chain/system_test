package config

type RequiredConfig struct {
	DNSHostName *string `yaml:"dnshostname"`
	LogLevel    *string `yaml:"loglevel"`
}
