package crypto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	c0 = 0x736f6d6570736575
	c1 = 0x646f72616e646f6d
	c2 = 0x6c7967656e657261
	c3 = 0x7465646279746573
)

func rotateLeft(x uint64, b int) uint64 {
	return (x << b) | (x >> (64 - b))
}

func sipRound(v *[4]uint64) {
	v[0] += v[1]
	v[1] = rotateLeft(v[1], 13)
	v[1] ^= v[0]
	v[0] = rotateLeft(v[0], 32)

	v[2] += v[3]
	v[3] = rotateLeft(v[3], 16)
	v[3] ^= v[2]

	v[0] += v[3]
	v[3] = rotateLeft(v[3], 21)
	v[3] ^= v[0]

	v[2] += v[1]
	v[1] = rotateLeft(v[1], 17)
	v[1] ^= v[2]
	v[2] = rotateLeft(v[2], 32)
}

// Generates a Message Authentication Code (MAC) for the given message
// using the provided key and the SipHash-2-4 algorithm
func GenerateMAC(key [16]byte, msg []byte) []byte {
	v := [4]uint64{
		c0 ^ binary.LittleEndian.Uint64(key[0:8]),
		c1 ^ binary.LittleEndian.Uint64(key[8:16]),
		c2 ^ binary.LittleEndian.Uint64(key[0:8]),
		c3 ^ binary.LittleEndian.Uint64(key[8:16]),
	}

	// Process the message in 8-byte blocks
	for len(msg) >= 8 {
		m := binary.LittleEndian.Uint64(msg[:8])
		v[3] ^= m
		sipRound(&v)
		sipRound(&v)
		v[0] ^= m
		msg = msg[8:]
	}

	// Process the remaining bytes
	var lastBlock uint64
	switch len(msg) {
	case 7:
		lastBlock |= uint64(msg[6]) << 48
		fallthrough
	case 6:
		lastBlock |= uint64(msg[5]) << 40
		fallthrough
	case 5:
		lastBlock |= uint64(msg[4]) << 32
		fallthrough
	case 4:
		lastBlock |= uint64(msg[3]) << 24
		fallthrough
	case 3:
		lastBlock |= uint64(msg[2]) << 16
		fallthrough
	case 2:
		lastBlock |= uint64(msg[1]) << 8
		fallthrough
	case 1:
		lastBlock |= uint64(msg[0])
	}
	lastBlock |= uint64(len(msg)) << 56

	// Add the last block to the hash
	v[3] ^= lastBlock
	sipRound(&v)
	sipRound(&v)
	v[0] ^= lastBlock

	// Add the finalization block to the hash
	v[2] ^= 0xff
	sipRound(&v)
	sipRound(&v)
	sipRound(&v)
	sipRound(&v)

	// Build the final hash
	finalHash := v[0] ^ v[1] ^ v[2] ^ v[3]

	// Convert the final hash to a byte slice
	hashBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(hashBytes, finalHash)

	return hashBytes
}

// ValidateMAC verifies the given message and MAC using the provided key
func ValidateMAC(key [16]byte, message, expectedMAC []byte) bool {
	actualMAC := GenerateMAC(key, message)

	fmt.Println("Actual   MAC: ", actualMAC, string(actualMAC))
	fmt.Println("Expected MAC: ", expectedMAC, string(expectedMAC))

	return bytes.Equal(actualMAC, expectedMAC)
}
