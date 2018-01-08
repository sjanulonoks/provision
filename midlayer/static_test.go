package midlayer

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/provision/backend"
)

func TestStaticFiles(t *testing.T) {
	locallogger := log.New(os.Stderr, "", log.LstdFlags)
	l := logger.New(locallogger).Log("static")
	svr, hh := ServeStatic(":3235235", backend.NewFS(".", l), l, backend.NewPublishers(locallogger))
	if hh != nil {
		if hh.Error() != "listen tcp: address 3235235: invalid port" {
			t.Errorf("Expected a different error: %v", hh.Error())
		}
	} else {
		t.Errorf("Should have returned an error")
	}

	svr, hh = ServeStatic(":32134", backend.NewFS(".", l), l, backend.NewPublishers(locallogger))
	if hh != nil {
		t.Errorf("Should not have returned an error: %v", hh)
	}

	response, err := http.Get("http://127.0.0.1:32134/dhcp.go")
	count := 0
	for err != nil && count < 10 {
		t.Logf("Failed to get file: %v", err)
		time.Sleep(1 * time.Second)
		count++
		response, err = http.Get("http://127.0.0.1:32134/dhcp.go")
	}
	if count == 10 {
		t.Errorf("Should have served the file: missing content")
	}
	buf, _ := ioutil.ReadAll(response.Body)
	if !strings.HasPrefix(string(buf), "package midlayer") {
		t.Errorf("Should have served the file: missing content")
	}

	if err := svr.Shutdown(context.Background()); err != nil {
		t.Errorf("Static server shutdown failed! %v", err)
	}
}
