package authentication

import (
	"context"
	"github.com/xiehqing/hiauthx/configx"
	"github.com/xiehqing/hiauthx/db/entity"
	security "github.com/xiehqing/hiauthx/rsax"
)

const loginEncryptAlgorithm = "RSA-OAEP-SHA256"

type EncryptConfigResponse struct {
	Enabled   bool   `json:"enabled"`
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"publicKey"`
}

func (s *Service) GetEncryptConfig(ctx context.Context) (*EncryptConfigResponse, error) {
	config := configx.New(s.queries)
	enabled := config.Bool(ctx, entity.SecurityLoginEncryptEnable, false)
	if !enabled {
		return &EncryptConfigResponse{
			Enabled:   false,
			Algorithm: loginEncryptAlgorithm,
		}, nil
	}

	publicKey := security.NormalizePublicKeyPEM(config.String(ctx, entity.SecurityLoginRSAPublicKey, ""))
	if publicKey == "" {
		return nil, ErrInvalidRSAKey
	}

	return &EncryptConfigResponse{
		Enabled:   true,
		Algorithm: loginEncryptAlgorithm,
		PublicKey: publicKey,
	}, nil
}

func (s *Service) decryptLoginPassword(ctx context.Context, password string) (string, error) {
	privateKey := configx.New(s.queries).String(ctx, entity.SecurityLoginRSAPrivateKey, "")
	return security.RSADecryptOAEPBase64(password, privateKey)
}
