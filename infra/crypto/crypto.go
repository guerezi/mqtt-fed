package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"
)

// GenerateECDHKeyPair generates a private-public key pair using elliptic curves
func GenerateECDHKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	// Using the P256 elliptic curve
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// The public key can be derived from the private key
	pubKey := &privKey.PublicKey

	return privKey, pubKey, nil
}

// GenerateSharedSecret generates the shared secret using ECDH
func GenerateSharedSecret(privateKey *ecdsa.PrivateKey, otherPublicKey *ecdsa.PublicKey) ([]byte, error) {
	fmt.Println("Private Key: ", privateKey.Curve)

	// Use the elliptic curve scalar multiplication to generate the shared secret
	x, _ := privateKey.Curve.ScalarMult(otherPublicKey.X, otherPublicKey.Y, privateKey.D.Bytes())

	// Hash the x coordinate (shared secret) to get a fixed-length key for encryption (AES, etc.)
	hash := sha256.New()
	hash.Write(x.Bytes())
	sharedSecret := hash.Sum(nil)

	return sharedSecret, nil
}

// CheckKeys ensures that the shared secrets match
func CheckKeys(sharedSecret, neighborSharedSecret []byte) bool {
	return subtle.ConstantTimeCompare(sharedSecret, neighborSharedSecret) == 1
}

// Encrypt encrypts a message using the shared secret key provided
func Encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	fmt.Println("Encrypted message: ", string(ciphertext))

	return ciphertext, nil
}

// Decrypt decrypts a message using the private key provided
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	fmt.Println("Decrypted message: ", string(plaintext))

	return plaintext, nil
}
