package crypto

import (
	"crypto/ecdsa"
	"math/big"
)

// ConvertECDSAPrivateKeyToBytes converts an ECDSA private key to a byte slice
func ConvertECDSAPrivateKeyToBytes(privateKey *ecdsa.PrivateKey) []byte {
	return privateKey.D.Bytes()
}

// ConvertECDSAPublicKeyToBytes converts an ECDSA public key to a byte slice (concatenating X and Y coordinates)
func ConvertECDSAPublicKeyToBytes(publicKey *ecdsa.PublicKey) []byte {
	xBytes := publicKey.X.Bytes()
	yBytes := publicKey.Y.Bytes()

	// Concatenate X and Y to make the public key as a byte slice
	return append(xBytes, yBytes...)
}

// ConvertBytesToECDSAPublicKey converts a byte slice back to an ECDSA public key
func ConvertBytesToECDSAPublicKey(privateKey *ecdsa.PrivateKey, pubKeyBytes []byte) (*ecdsa.PublicKey, error) {
	keyLen := len(pubKeyBytes) / 2
	x := new(big.Int).SetBytes(pubKeyBytes[:keyLen])
	y := new(big.Int).SetBytes(pubKeyBytes[keyLen:])

	publicKey := &ecdsa.PublicKey{
		Curve: privateKey.Curve,
		X:     x,
		Y:     y,
	}
	return publicKey, nil
}
