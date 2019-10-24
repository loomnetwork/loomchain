package eth

type Web3Config struct {
	// MaxBlockLimit defines the maximum number of block range in one request
	MaxBlockLimit int64
}

func DefaultWeb3Config() *Web3Config {
	return &Web3Config{
		MaxBlockLimit: 20,
	}
}
