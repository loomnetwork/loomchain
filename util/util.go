package util

func PrefixKey(prefix, key []byte) []byte {
	buf := make([]byte, 0, len(prefix)+len(key)+1)
	buf = append(buf, prefix...)
	buf = append(buf, 0)
	buf = append(buf, key...)
	return buf
}
