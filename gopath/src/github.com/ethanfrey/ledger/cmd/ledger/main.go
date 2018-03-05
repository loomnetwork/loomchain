package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/ethanfrey/ledger"
)

func main() {
	ledger, err := ledger.FindLedger()
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		return
	}

	data, err := hex.DecodeString(os.Args[1])
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		return
	}
	fmt.Printf("Sending %X\n\n", data)

	resp, err := ledger.Exchange(data, 100)
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		return
	}
	fmt.Printf("Response: %X\n", resp)
}
