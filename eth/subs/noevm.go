// +build !evm

package subs

type EthSubscriptions struct {
}

func (s *EthSubscriptions) Add(filter string) (string, error) {
	return "", nil
}

func (s *EthSubscriptions) Poll(id string) ([]byte, error) {
	return nil, nil
}

func (s *EthSubscriptions) Remove(id string) (bool, error) {
	return true, nil
}
