package fnConsensus

func safeCopyBytes(originalBytes []byte) []byte {
	if originalBytes == nil {
		return nil
	}

	copiedBytes := make([]byte, len(originalBytes))
	copy(copiedBytes, originalBytes)
	return copiedBytes
}

func safeCopyDoubleArray(originalArray [][]byte) [][]byte {
	if originalArray == nil {
		return nil
	}

	copiedArray := make([][]byte, len(originalArray))

	for i := 0; i < len(copiedArray); i++ {
		copiedArray[i] = safeCopyBytes(originalArray[i])
	}

	return copiedArray
}
