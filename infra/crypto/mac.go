package crypto

import (
	"bytes"
	"fmt"

	"github.com/aead/siphash"
)

func GenerateMAC(key, message []byte) []byte {
	sipkey, err := siphash.New128(key)

	if err != nil {
		fmt.Println("Error generating MAC: ", err)
	}

	sipkey.Write(message)
	mac := sipkey.Sum(nil)

	return mac[:]
}

func ValidateMAC(key, message, expectedMAC []byte) bool {
	actualMAC := GenerateMAC(key, message)
	return bytes.Equal(actualMAC, expectedMAC)
}
