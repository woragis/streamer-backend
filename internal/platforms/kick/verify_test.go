package kick

import (
	"testing"
)

func TestVerifySignatureRejectsInvalid(t *testing.T) {
	pub, err := DefaultPublicKeyParsed()
	if err != nil {
		t.Fatal(err)
	}
	err = VerifySignature(pub, "msg1", "2025-01-01T00:00:00Z", []byte(`{"x":1}`), "invalid")
	if err == nil {
		t.Fatal("expected invalid signature error")
	}
}
