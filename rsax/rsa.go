package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"strings"
)

const DefaultRSAKeyBits = 2048

type RSAKeyPair struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

func GenerateRSAKeyPair(bits int) (*RSAKeyPair, error) {
	if bits <= 0 {
		bits = DefaultRSAKeyBits
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	if err := privateKey.Validate(); err != nil {
		return nil, err
	}

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return &RSAKeyPair{
		PublicKey: string(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyDER,
		})),
		PrivateKey: string(pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKeyDER,
		})),
	}, nil
}

func NormalizePublicKeyPEM(value string) string {
	value = normalizePEMText(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "-----BEGIN PUBLIC KEY-----") {
		return value
	}
	return "-----BEGIN PUBLIC KEY-----\n" + value + "\n-----END PUBLIC KEY-----"
}

func RSADecryptOAEPBase64(cipherTextBase64, privateKey string) (string, error) {
	rsaPrivateKey, err := ParseRSAPrivateKey(privateKey)
	if err != nil {
		return "", err
	}

	cipherText, err := DecodeBase64(cipherTextBase64)
	if err != nil {
		return "", err
	}

	plainText, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, rsaPrivateKey, cipherText, nil)
	if err != nil {
		return "", err
	}
	return string(plainText), nil
}

func ParseRSAPrivateKey(value string) (*rsa.PrivateKey, error) {
	value = normalizePEMText(value)
	if value == "" {
		return nil, errors.New("RSA 私钥不能为空")
	}

	var der []byte
	if block, _ := pem.Decode([]byte(value)); block != nil {
		der = block.Bytes
	} else {
		var err error
		der, err = DecodeBase64(value)
		if err != nil {
			return nil, err
		}
	}

	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		privateKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("不是合法的 RSA 私钥")
		}
		return privateKey, nil
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

func DecodeBase64(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	decoders := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, decoder := range decoders {
		result, err := decoder.DecodeString(value)
		if err == nil {
			return result, nil
		}
	}
	return nil, errors.New("Base64 格式不合法")
}

func normalizePEMText(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, `\n`, "\n")
	return value
}
