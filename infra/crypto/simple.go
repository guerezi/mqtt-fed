package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/xdg-go/pbkdf2"
)

// GenerateSessionKey generates a session key from a seed using PBKDF2
func GenerateSessionKey(seed string) []byte {
	// A salt to use for the key derivation. This can be changed to a random value for each key generation.
	salt := []byte("fixed_salt_value")

	// TODO: GENERATE COMES ONLY FROM THE TOPOLOGY
	// VER COMO FAZER O CORE GERAR A CHAVE, mAS O TOPOLOGY DISTRIBUIR
	// THE PASSWORD IS THE SAME FOR ALL NODES
	// THE PASSWORD MUST BE PASSED DOWN TO THE NODES
	// WHITELISTE DE TOPICO TAMBËM, por NÓ
	// VER QUAL A DIFICULDADE DE LIDAR COM NÓS INTERMEDIARIOS, SEM CHILD

	// COMO ENVIAR MENSAGEM PRO TOPOLOGY MANAGER?
	// CHAVE DE SESSAO NO SIPHASH

	// COMO BUSCAR MENSAGENS QUE ESTÃO NO BANCO DO NÓ? QUALIDADE 2
	// CRIPTOGRAFAR O ROUTED PUB?

	iterations := 10000
	// Use PBKDF2 to generate the key based on the seed
	key := pbkdf2.Key([]byte(seed), salt, iterations, 32, sha256.New)

	fmt.Println("Generated key: ", key)

	return key
}

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
