package server

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jessevdk/go-flags"
)

var (
	tmpDir string
)

func generateArgs(args []string) *ProgOpts {
	var c_opts ProgOpts

	parser := flags.NewParser(&c_opts, flags.Default)
	if _, err := parser.ParseArgs(args); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	return &c_opts
}

func TestServer(t *testing.T) {

	testArgs := []string{
		"--data-root", tmpDir + "/digitalrebar",
		"--file-root", tmpDir + "/tftpboot",
		"--tls-key", tmpDir + "/server.key",
		"--tls-cert", tmpDir + "/server.crt",
		"--api-port", "10001",
		"--static-port", "10002",
		"--tftp-port", "10003",
		"--disable-dhcp",
	}

	c_opts := generateArgs(testArgs)
	go Server(c_opts)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	_, apierr := client.Get("https://127.0.0.1:10001/api/v3/subnets")
	count := 0
	for apierr != nil && count < 30 {
		t.Logf("Failed to get file: %v", apierr)
		time.Sleep(1 * time.Second)
		count++
		_, apierr = client.Get("https://127.0.0.1:10001/api/v3/subnets")
	}
	if count == 30 {
		t.Errorf("Server failed to start in time allowed")
	}

	// test presences of all the above
	if _, err := os.Stat(c_opts.TlsCertFile); os.IsNotExist(err) {
		t.Errorf("Failed to create cert file: %s", c_opts.TlsCertFile)
	} else {
		t.Logf("Cert file correctly created")
	}

	if _, err := os.Stat(c_opts.TlsKeyFile); os.IsNotExist(err) {
		t.Errorf("Failed to create cert file: %s", c_opts.TlsKeyFile)
	} else {
		t.Logf("Key file correctly created")
	}

	if _, err := os.Stat(c_opts.DataRoot); os.IsNotExist(err) {
		t.Errorf("Failed to create data dir: %s", c_opts.DataRoot)
	} else {
		t.Logf("DataRoot directory correctly created")
	}

	if _, err := os.Stat(c_opts.FileRoot); os.IsNotExist(err) {
		t.Errorf("Failed to create data dir: %s", c_opts.FileRoot)
	} else {
		t.Logf("FileRoot directory correctly created")
	}

	// Extract assets handle separately.

}

func TestMain(m *testing.M) {
	var err error
	tmpDir, err = ioutil.TempDir("", "server-")
	if err != nil {
		log.Printf("Creating temp dir for file root failed: %v", err)
		os.Exit(1)
	}

	ret := m.Run()

	err = os.RemoveAll(tmpDir)
	if err != nil {
		log.Printf("Creating temp dir for file root failed: %v", err)
		os.Exit(1)
	}

	os.Exit(ret)
}
