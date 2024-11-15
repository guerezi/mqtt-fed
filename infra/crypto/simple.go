package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// DecryptSimple decrypts a message using the provided key
// It uses the AES block cipher in CFB mode
// Only used in the topic messages, in symmetric encryption
func DecryptSimple(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println("Error creating new cipher: ", err)
		return nil, err
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// Decrypt using CFB mode
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}

// EncryptSimple encrypts a message using the provided key
// It uses the AES block cipher in CFB mode
// Only used in the topic messages, in symmetric encryption
func EncryptSimple(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println("Error creating new cipher: ", err)
		return nil, err
	}

	// Generate a new IV for encryption
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		fmt.Println("Error reading random bytes: ", err)
		return nil, err
	}

	// Encrypt using CFB mode
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}
