package config

type App struct {
	LogLevel   string `env:"LOG_LEVEL" envDefault:"info"`
	Prometheus Prometheus
	Health     Health
	Database   Database
	Nats       Nats
	Inbox      Inbox
	Core       Core
}
