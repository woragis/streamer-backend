package kick

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
)

var (
	ErrInvalidPublicKey = errors.New("invalid kick public key")
	ErrInvalidSignature = errors.New("invalid kick webhook signature")
)

func ParsePublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, ErrInvalidPublicKey
	}
	if block.Type != "PUBLIC KEY" {
		return nil, ErrInvalidPublicKey
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	publicKey, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, ErrInvalidPublicKey
	}
	return publicKey, nil
}

func VerifySignature(publicKey *rsa.PublicKey, messageID, timestamp string, body []byte, signatureHeader string) error {
	if publicKey == nil {
		return ErrInvalidPublicKey
	}
	signed := []byte(fmt.Sprintf("%s.%s.%s", messageID, timestamp, body))

	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(signatureHeader)))
	n, err := base64.StdEncoding.Decode(decoded, []byte(signatureHeader))
	if err != nil {
		return err
	}
	signature := decoded[:n]

	hashed := sha256.Sum256(signed)
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signature); err != nil {
		return ErrInvalidSignature
	}
	return nil
}

func DefaultPublicKeyParsed() (*rsa.PublicKey, error) {
	return ParsePublicKey([]byte(DefaultPublicKey))
}
