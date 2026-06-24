package authentication

import (
	"testing"

	security "github.com/xiehqing/hiauthx/rsax"
)

func TestNormalizeRSAKeyBits(t *testing.T) {
	if got := normalizeRSAKeyBits(3072); got != 3072 {
		t.Fatalf("expected 3072, got %d", got)
	}
	if got := normalizeRSAKeyBits(1024); got != security.DefaultRSAKeyBits {
		t.Fatalf("unsupported bits should default to %d, got %d", security.DefaultRSAKeyBits, got)
	}
}
