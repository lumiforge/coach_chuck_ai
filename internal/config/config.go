package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	PostgreSQL struct {
		Username string `yaml:"username" env:"CHUCK_AI_POSTGRES_USER" env-required:"true"`
		Password string `yaml:"password" env:"CHUCK_AI_POSTGRES_PASSWORD" env-required:"true"`
		Host     string `yaml:"host" env:"CHUCK_AI_POSTGRES_HOST" env-required:"true"`
		Port     string `yaml:"port" env:"CHUCK_AI_POSTGRES_PORT" env-required:"true"`
		Database string `yaml:"database" env:"CHUCK_AI_POSTGRES_DB" env-required:"true"`
	} `yaml:"postgresql"`

	App struct {
		LogLevel string `yaml:"log_level" env:"LOG_LEVEL" env-default:"info"`
	} `yaml:"app"`

	CoachAgent struct {
		ModelName string `yaml:"model_name" env:"COACH_AGENT_MODEL_NAME" env-required:"true"`
		OpenAI    struct {
			APIKey  string `yaml:"api_key" env:"HYDRA_AI_API_KEY" env-required:"true"`
			BaseURL string `yaml:"base_url" env:"HYDRA_AI_BASE_URL"`
		} `yaml:"openai"`
	} `yaml:"coach_agent"`

	A2A struct {
		Port string `yaml:"port" env:"A2A_PORT" env-default:"8000"`
	} `yaml:"a2a"`
}

const (
	EnvConfigPathName  = "CONFIG-PATH"
	FlagConfigPathName = "config"
)

var instance *Config
var once sync.Once

func GetConfig() *Config {
	once.Do(func() {
		log.Print("config init")

		instance = &Config{}

		if err := cleanenv.ReadEnv(instance); err != nil {
			helpText := "Environment variables:"
			help, _ := cleanenv.GetDescription(instance, &helpText)
			log.Print(help)
			log.Fatal(err)
		}
	})

	return instance
}
