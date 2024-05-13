package acrypt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
)

func GenerateRSAKeypairPKCS8() ([]byte, error) {
	new_rsa_key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	rsa_priv_pkcs8, err := x509.MarshalPKCS8PrivateKey(new_rsa_key)
	if err != nil {
		return nil, err
	}

	return rsa_priv_pkcs8, nil
}
