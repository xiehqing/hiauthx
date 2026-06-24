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

type GenerateRSAKeyPairRequest struct {
	Bits int `json:"bits"`
}

type RSAKeyPairResponse struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
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

func (s *Service) GenerateRSAKeyPair(ctx context.Context, req GenerateRSAKeyPairRequest) (*RSAKeyPairResponse, error) {
	bits := normalizeRSAKeyBits(req.Bits)
	pair, err := security.GenerateRSAKeyPair(bits)
	if err != nil {
		return nil, err
	}
	return &RSAKeyPairResponse{
		PublicKey:  pair.PublicKey,
		PrivateKey: pair.PrivateKey,
	}, nil
}

func (s *Service) decryptLoginPassword(ctx context.Context, password string) (string, error) {
	privateKey := configx.New(s.queries).String(ctx, entity.SecurityLoginRSAPrivateKey, "")
	return security.RSADecryptOAEPBase64(password, privateKey)
}

func normalizeRSAKeyBits(bits int) int {
	switch bits {
	case 2048, 3072, 4096:
		return bits
	default:
		return security.DefaultRSAKeyBits
	}
}
