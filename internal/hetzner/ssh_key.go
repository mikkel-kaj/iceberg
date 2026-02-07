package hetzner

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"net/http"

	"golang.org/x/crypto/ssh"
)

type SSHKey struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
}

func GenerateSSHKeyPair() (publicKey string, privateKey []byte, err error) {
	pub, prv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", nil, err
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", nil, err
	}
	encodedPriv, err := ssh.MarshalPrivateKey(prv, "")
	if err != nil {
		return "", nil, err
	}
	block := &pem.Block{Type: encodedPriv.Type, Bytes: encodedPriv.Bytes}
	return string(ssh.MarshalAuthorizedKey(sshPub)), pem.EncodeToMemory(block), nil
}

func (c *Client) CreateSSHKey(ctx context.Context, name string, publicKey string) (*SSHKey, error) {
	if name == "" || publicKey == "" {
		return nil, errors.New("name and public key are required")
	}
	var resp struct {
		SSHKey SSHKey `json:"ssh_key"`
	}
	if err := c.do(ctx, http.MethodPost, "/ssh_keys", map[string]any{"name": name, "public_key": publicKey}, &resp); err != nil {
		return nil, err
	}
	return &resp.SSHKey, nil
}

func (c *Client) DeleteSSHKey(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, "/ssh_keys/"+itoa64(id), nil, nil)
}
