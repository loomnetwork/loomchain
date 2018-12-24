package pubsub

// stringInSlice returns whether string is in slice :-D
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
