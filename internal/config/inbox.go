package config

type Inbox struct {
	Bind string `env:"INBOX_API_GRPC_SERVER_BIND" envDefault:":11000"`

	StorageAddress string `env:"INBOX_API_STORAGE_ADDRESS" envDefault:"inbox-storage:11000"`
}
