package branta

import (
	"errors"
	"testing"
)

func TestRoundTripWithMatchingSecret(t *testing.T) {
	ciphertext, err := Encrypt("hello world", "my-secret", false)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := Decrypt(ciphertext, "my-secret")
	if err != nil || plaintext != "hello world" {
		t.Fatalf("got %q %v", plaintext, err)
	}
}

func TestRoundTripDeterministic(t *testing.T) {
	ciphertext, err := Encrypt("hello world", "my-secret", true)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := Decrypt(ciphertext, "my-secret")
	if err != nil || plaintext != "hello world" {
		t.Fatalf("got %q %v", plaintext, err)
	}
}

func TestWrongSecretFailsToDecrypt(t *testing.T) {
	ciphertext, err := Encrypt("hello world", "my-secret", false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Decrypt(ciphertext, "wrong-secret")
	var decErr *DecryptionError
	if !errors.As(err, &decErr) {
		t.Fatalf("got %v", err)
	}
}

func TestRandomNonceProducesDifferentCiphertextEachCall(t *testing.T) {
	a, _ := Encrypt("hello world", "my-secret", false)
	b, _ := Encrypt("hello world", "my-secret", false)
	if a == b {
		t.Fatal("expected different ciphertext")
	}
}

func TestDeterministicNonceProducesIdenticalCiphertextEachCall(t *testing.T) {
	a, _ := Encrypt("hello world", "my-secret", true)
	b, _ := Encrypt("hello world", "my-secret", true)
	if a != b {
		t.Fatal("expected identical ciphertext")
	}
}

func TestDeterministicRoundTripWithHashDerivedKey(t *testing.T) {
	value := "lnbc1qsomething"
	key := ToNormalizedHash(value)
	ciphertext, err := Encrypt(value, key, true)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := Decrypt(ciphertext, key)
	if err != nil || plaintext != value {
		t.Fatalf("got %q %v", plaintext, err)
	}
}

func TestUnicodeValueRoundTrips(t *testing.T) {
	value := "hello 世界 🚀 emoji test"
	ciphertext, err := Encrypt(value, "my-secret", false)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := Decrypt(ciphertext, "my-secret")
	if err != nil || plaintext != value {
		t.Fatalf("got %q %v", plaintext, err)
	}
}

func TestTooShortBase64IsADistinctError(t *testing.T) {
	err := func() error { _, e := Decrypt("AAAA", "my-secret"); return e }()
	if !errors.Is(err, ErrEncryptedDataTooShort) {
		t.Fatalf("got %v", err)
	}
}

func TestMalformedBase64IsADecryptionFailureNotTooShort(t *testing.T) {
	_, err := Decrypt("not-valid-base64!!!", "my-secret")
	var decErr *DecryptionError
	if !errors.As(err, &decErr) {
		t.Fatalf("got %v", err)
	}
}

func TestCrossSDKFixedVectorMatchesPython(t *testing.T) {
	expected := "mPIKHc3ywVlsBHf3Lv2Rwpz2+fKE0kgUePq2m4fPIUidMuGEHVIB"
	ciphertext, err := Encrypt("hello world", "my-secret", true)
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext != expected {
		t.Fatalf("ciphertext mismatch\n got %s\nwant %s", ciphertext, expected)
	}
	plaintext, err := Decrypt(expected, "my-secret")
	if err != nil || plaintext != "hello world" {
		t.Fatalf("got %q %v", plaintext, err)
	}
}

func TestGuidSecretGenerator(t *testing.T) {
	g := GuidSecretGenerator{}
	a, b := g.Generate(), g.Generate()
	if a == b {
		t.Fatal("expected unique uuids")
	}
	if g.DeterministicNonce() {
		t.Fatal("deterministic nonce should be false")
	}
}
