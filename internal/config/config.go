package config

type Config struct {
	Env      string `env:"KSEF_ENVIRONMENT" help:"KSeF environment (TEST, DEMO, PROD)" default:"TEST"`
	Keystore string `env:"KSEF_KEYSTORE_TYPE" help:"Keystore type for KSeF token" default:"desktop"`
}
