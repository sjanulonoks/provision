package server

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func validFile(t *testing.T, bits, cert, key string) {
	if err := buildKeys(bits, cert, key); err != nil {
		t.Errorf("Should not have failed: %s", err)
		return
	}

	if _, err := os.Stat(cert); os.IsNotExist(err) {
		t.Errorf("Failed to create cert file: %s", cert)
	} else {
		t.Logf("Cert file correctly created")
	}

	if _, err := os.Stat(key); os.IsNotExist(err) {
		t.Errorf("Failed to create cert file: %s", key)
	} else {
		t.Logf("Key file correctly created")
	}

	os.Remove(cert)
	os.Remove(key)
}

func TestCert(t *testing.T) {
	certFile := fmt.Sprintf("%s/c1.pem", tmpDir)
	keyFile := fmt.Sprintf("%s/k1.pem", tmpDir)

	err := buildKeys("P600", certFile, keyFile)
	if err == nil {
		t.Errorf("Should have failed with bad curve")
	} else {
		if err.Error() != "Unrecognized elliptic curve: \"P600\"" {
			t.Errorf("Should have failed with error: %s", err)
		}
	}

	err = buildKeys("Fred", certFile, keyFile)
	if err == nil {
		t.Errorf("Should have failed with bad bits")
	} else {
		if err.Error() != "strconv.Atoi: parsing \"Fred\": invalid syntax" {
			t.Errorf("Should have failed with error: %s", err)
		}
	}

	err = buildKeys("P384", tmpDir, keyFile)
	if err == nil {
		t.Errorf("Should have failed with bad filenames")
	} else {
		if !strings.HasPrefix(err.Error(), "failed to open cert.pem for writing: open") {
			t.Errorf("Should have failed with error: %s", err)
		}
	}

	err = buildKeys("P384", certFile, tmpDir)
	if err == nil {
		t.Errorf("Should have failed with bad filenames")
	} else {
		if !strings.HasPrefix(err.Error(), "failed to open key.pem for writing: open") {
			t.Errorf("Should have failed with error: %s", err)
		}
	}

	os.Remove(certFile)
	os.Remove(keyFile)

	validFile(t, "RSA", certFile, keyFile)
	validFile(t, "P224", certFile, keyFile)
	validFile(t, "P256", certFile, keyFile)
	validFile(t, "P384", certFile, keyFile)
	validFile(t, "P521", certFile, keyFile)
	validFile(t, "2048", certFile, keyFile)
	validFile(t, "1024", certFile, keyFile)
}
