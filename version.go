package loomchain

var (
	Version = "1.0.0"
	Build   = ""
	GitSHA  = ""
)

func FullVersion() string {
	if Build == "" {
		return Version + "+dev" + GitSHA[:8]
	}

	return Version + "+" + "b" + Build
}
