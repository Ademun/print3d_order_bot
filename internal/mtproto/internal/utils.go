package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
)

func parseBase64RLE(s string) ([]byte, error) {
	rle, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	decoded := make([]byte, 0, len(rle))
	i := 0
	for i < len(rle) {
		if rle[i] == 0x00 && i+1 < len(rle) {
			count := int8(rle[i+1])
			if count == 0 {
				count = 1
			}
			for range count {
				decoded = append(decoded, 0x00)
			}
			i += 2
			continue
		}
		decoded = append(decoded, rle[i])
		i += 1
	}

	return decoded, nil
}

func readTLBytes(data []byte) ([]byte, int, error) {
	dataLen := int(data[0])
	if dataLen > 254 {
		return nil, 0, errors.New("tl string size length too big")
	}

	var startPos, paddingSize int
	if dataLen < 254 {
		startPos = 1
		paddingSize = posmod(-(dataLen + startPos), 4)
	}

	if dataLen == 254 {
		if len(data) < 4 {
			return nil, 0, errors.New("tl string size too small")
		}
		startPos = 4
		var trueLen int32
		if err := binary.Read(bytes.NewBuffer(data[1:4]), binary.LittleEndian, &trueLen); err != nil {
			return nil, 0, err
		}
		paddingSize = posmod(-(int(trueLen) + startPos), 4)
		dataLen = int(trueLen)
	}

	if len(data[startPos:]) < dataLen {
		return nil, 0, errors.New("tl string size too small")
	}

	return data[startPos : startPos+dataLen], startPos + dataLen + paddingSize, nil
}

func posmod(a, b int) int {
	rem := a % b
	if rem < 0 {
		rem += abs(b)
	}
	return rem
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func qptr[T any](o T) *T {
	return &o
}
