// +build !evm

package utils

func GetId() string {
	return ""
}

func UnmarshalEthFilter(query []byte) (EthFilter, error) {
	return EthFilter{}, nil
}
