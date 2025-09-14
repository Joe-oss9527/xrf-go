package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Joe-oss9527/xrf-go/pkg/utils"
)

// Test deriveVEEncryption for X25519-based decryption string (no external xray needed)
func TestDeriveVEEncryption_X25519(t *testing.T) {
	priv, pub, err := utils.GenerateX25519KeyPair()
	if err != nil {
		t.Fatalf("GenerateX25519KeyPair failed: %v", err)
	}
	dec := fmt.Sprintf("mlkem768x25519plus.native.600s.%s", priv)
	enc, err := deriveVEEncryption(dec)
	if err != nil {
		t.Fatalf("deriveVEEncryption failed: %v", err)
	}
	// Expected 0rtt on client and same prefix/mode
	want := fmt.Sprintf("mlkem768x25519plus.native.0rtt.%s", pub)
	if strings.TrimSpace(enc) != want {
		t.Errorf("encryption mismatch\n got: %s\nwant: %s", enc, want)
	}
}
