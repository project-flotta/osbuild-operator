package sshkey

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

const (
	privateKeyBitSize = 4096
)

//go:generate mockgen -package=sshkey -destination=mock_sshkey.go . SSHKeyGenerator
type SSHKeyGenerator interface {
	GenerateSSHKeyPair() ([]byte, []byte, error)
}

type SSHKeyGeneratorImpl struct{}

func NewSSHKeyGenerator() *SSHKeyGeneratorImpl {
	return &SSHKeyGeneratorImpl{}
}

func (s *SSHKeyGeneratorImpl) GenerateSSHKeyPair() ([]byte, []byte, error) {
	privateKey, err := generatePrivateKey(privateKeyBitSize)
	if err != nil {
		return nil, nil, err
	}

	publicKeyBytes, err := generatePublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	return encodePrivateKeyToPEM(privateKey), publicKeyBytes, nil
}

// generatePrivateKey creates a RSA Private Key of specified byte size
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// encodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	return pem.EncodeToMemory(&privBlock)
}

// generatePublicKey take a rsa.PublicKey and return bytes suitable for writing to .pub file
// returns in the format "ssh-rsa ..."
func generatePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privatekey)
	if err != nil {
		return nil, err
	}
	return ssh.MarshalAuthorizedKey(publicRsaKey), nil
}
