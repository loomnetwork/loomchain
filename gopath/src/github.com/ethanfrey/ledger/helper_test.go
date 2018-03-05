package ledger

type EchoDevice struct {
	chunk int
	data  [][]byte
}

func NewEcho(chunk int) *EchoDevice {
	return &EchoDevice{
		chunk: chunk,
	}
}

func (e *EchoDevice) Write(input []byte) error {
	for len(input) > e.chunk {
		e.data = append(e.data, input[:e.chunk])
		input = input[e.chunk:]
	}
	pad := len(input) - e.chunk
	if pad > 0 {
		input = append(input, make([]byte, pad)...)
	}
	e.data = append(e.data, input)
	return nil
}

func (e *EchoDevice) Close()           {}
func (e *EchoDevice) ReadError() error { return nil }

func (e *EchoDevice) ReadCh() <-chan []byte {
	output := make(chan []byte, 3)
	go func() {
		buf := e.data
		for i := 0; i < len(buf); i++ {
			output <- buf[i]
		}
	}()
	return output
}
