package dochaincore

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os/user"

	"golang.org/x/crypto/ssh"
)

type sshKeyPair struct {
	privateKey    *rsa.PrivateKey
	privateKeyPEM []byte
	authorizedKey []byte
}

func createSSHKeyPair() (*sshKeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}

	// generate and write private key as PEM
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	var pembuf bytes.Buffer
	err = pem.Encode(&pembuf, privateKeyPEM)
	if err != nil {
		return nil, err
	}

	// TODO(jackson): Don't write the private key to the fs at all.
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(usr.HomeDir+"/dochaincore.pem", pembuf.Bytes(), 0400)
	if err != nil {
		return nil, err
	}

	// generate and write public key
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}
	authorizedKey := ssh.MarshalAuthorizedKey(pub)

	return &sshKeyPair{
		privateKey:    privateKey,
		privateKeyPEM: pembuf.Bytes(),
		authorizedKey: authorizedKey,
	}, nil
}
