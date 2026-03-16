package collector

type Config struct {
	ListenAddr string
	StatePath  string
}

func DefaultConfig() Config {
	return Config{
		ListenAddr: ":50051",
		StatePath:  "agents.json",
	}
}
