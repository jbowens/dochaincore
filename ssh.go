package dochaincore

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"time"

	"golang.org/x/crypto/ssh"
)

type sshKeyPair struct {
	privateKey    *rsa.PrivateKey
	publicKey     ssh.PublicKey
	privateKeyPEM []byte
	authorizedKey []byte
}

func createSSHKeyPair() (*sshKeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}

	// generate and write private key as PEM
	var pembuf bytes.Buffer
	err = pem.Encode(&pembuf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		return nil, err
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}
	authorizedKey := ssh.MarshalAuthorizedKey(publicKey)

	return &sshKeyPair{
		privateKey:    privateKey,
		publicKey:     publicKey,
		privateKeyPEM: pembuf.Bytes(),
		authorizedKey: authorizedKey,
	}, nil
}

func connect(ctx context.Context, host string, keypair *sshKeyPair) (*ssh.Session, error) {
	signer, err := ssh.NewSignerFromKey(keypair.privateKey)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	// propogate the context's deadline as a timeout on the config
	if deadline, ok := ctx.Deadline(); ok {
		config.Timeout = deadline.Sub(time.Now())
	}

	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		return nil, err
	}
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	return session, nil
}
