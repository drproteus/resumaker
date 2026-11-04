package envelope

import (
	"bytes"
	"testing"
)

func TestEncryptRoundTrip(t *testing.T) {
	dek, err := NewDataKey()
	if err != nil {
		t.Fatal(err)
	}
	ct, err := Encrypt(dek, []byte("secret resume"), []byte("rev-1"))
	if err != nil {
		t.Fatal(err)
	}
	pt, err := Decrypt(dek, ct, []byte("rev-1"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pt, []byte("secret resume")) {
		t.Fatalf("got %q", pt)
	}
}

func TestDecryptRejectsWrongAssociatedData(t *testing.T) {
	dek, _ := NewDataKey()
	ct, _ := Encrypt(dek, []byte("secret"), []byte("rev-1"))
	if _, err := Decrypt(dek, ct, []byte("rev-2")); err != ErrDecrypt {
		t.Fatalf("expected ErrDecrypt, got %v", err)
	}
}

func TestWrapDataKeyRoundTrip(t *testing.T) {
	master := bytes.Repeat([]byte{7}, KeySize)
	kr, err := NewKeyring(master)
	if err != nil {
		t.Fatal(err)
	}
	dek, _ := NewDataKey()
	wrapped, err := kr.WrapDataKey(dek, []byte("user-1"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := kr.UnwrapDataKey(wrapped, []byte("user-1"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, dek) {
		t.Fatal("unwrapped key does not match")
	}
	if _, err := kr.UnwrapDataKey(wrapped, []byte("user-2")); err != ErrDecrypt {
		t.Fatalf("expected ErrDecrypt for other user, got %v", err)
	}
}

func TestNewKeyringRequiresKeySize(t *testing.T) {
	if _, err := NewKeyring(make([]byte, 16)); err == nil {
		t.Fatal("expected error for short master key")
	}
}
