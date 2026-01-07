package collector

type Config struct {
	ListenAddr string
}

func DefaultConfig() Config {
	return Config{
		ListenAddr: ":50051",
	}
}
