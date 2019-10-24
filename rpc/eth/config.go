package eth

type Web3Config struct {
	// GetLogsMaxBlockRange defines the maximum number of block range in one request
	GetLogsMaxBlockRange int64
}

func DefaultWeb3Config() *Web3Config {
	return &Web3Config{
		GetLogsMaxBlockRange: 20,
	}
}
