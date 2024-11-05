package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

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
