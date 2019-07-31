package loomchain

var (
	Version            = "2.0.0"
	Build              = ""
	BuildVariant       = "generic"
	GitSHA             = ""
	GoLoomGitSHA       = ""
	EthGitSHA          = ""
	HashicorpGitSHA    = ""
	BtcdGitSHA         = ""
	TransferGatewaySHA = ""
)

func FullVersion() string {
	lastPart := "b" + Build
	if Build == "" {
		lastPart = "dev"
		if GitSHA != "" {
			lastPart += "+" + GitSHA[:8]
		}
	}

	return Version + "+" + lastPart
}
