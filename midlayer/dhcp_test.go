package midlayer

import (
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/pinger"
)

/*
DHCP test layout:

dhcp-tests/0000-test-name/
     0000.request
     0000.response-expect
     0000.response-actual
     0000.logs-expect
     0000.logs-actual
     0000.delay
     ....

*/

func diff(expect, actual string) (string, error) {
	cmd := exec.Command("diff", "-NZBu", expect, actual)
	res, err := cmd.CombinedOutput()
	return string(res), err
}

func rt(t *testing.T) *DhcpRequest {
	return &DhcpRequest{
		Logger: logger.New(nil).Log("dhcp").SetLevel(logger.Info),
		idxMap: map[int][]*net.IPNet{
			1: []*net.IPNet{&net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.IPv4Mask(255, 0, 0, 0)}},
			2: []*net.IPNet{&net.IPNet{IP: net.IPv4(192, 168, 124, 1), Mask: net.IPv4Mask(255, 255, 255, 0)}},
			3: []*net.IPNet{&net.IPNet{IP: net.IPv4(10, 0, 0, 10), Mask: net.IPv4Mask(255, 0, 0, 0)}},
		},
		nameMap: map[int]string{1: "lo", 2: "eno1", 3: "eno2"},
		pinger:  pinger.Fake(false),
		handler: dhcpHandler,
	}
}

func TestDHCPCases(t *testing.T) {
	ents, err := filepath.Glob("dhcp-tests/*/*.request")
	if err != nil || len(ents) == 0 {
		t.Errorf("No tests to run")
		return
	}
	sort.Strings(ents)
	for _, ent := range ents {
		testPath := path.Dir(ent)
		testFailed := false
		req, err := ioutil.ReadFile(ent)
		if err != nil {
			t.Errorf("FAIL: %s: cannot read %s: %v", testPath, path.Base(ent), err)
			return
		}
		part := strings.TrimSuffix(path.Base(ent), ".request")
		testPart := path.Join(testPath, part)
		respName := path.Join(testPart + ".response-expect")
		logName := path.Join(testPart + ".logs-expect")
		actualResp := path.Join(testPart + ".response-actual")
		actualLog := path.Join(testPart + ".logs-actual")

		if info, err := os.Stat(respName); err != nil || !info.Mode().IsRegular() {
			t.Errorf("FAIL: %s: missing %s.response-expect", testPath, part)
			testFailed = true
		}
		if info, err := os.Stat(logName); err != nil || !info.Mode().IsRegular() {
			t.Errorf("FAIL: %s: missing %s.logs-expect", testPath, part)
			testFailed = true
		}
		request := rt(t)
		if err := request.UnmarshalText(req); err != nil {
			t.Errorf("FAIL: %s: Error parsing request: %v", testPart, err)
			return
		}
		response := request.PrintOutgoing(request.Process())
		if err := ioutil.WriteFile(actualResp, []byte(response), 0644); err != nil {
			t.Errorf("FAIL: %s: Error saving response: %v", testPart, err)
			return
		}
		logBuf := []string{}
		lines := request.Logger.Buffer().Lines(-1)
		for _, line := range lines {
			logBuf = append(logBuf, line.Message)
		}
		if err := ioutil.WriteFile(actualLog, []byte(strings.Join(logBuf, "\n")), 0644); err != nil {
			t.Errorf("FAIL: %s: Error saving logs: %v", testPart, err)
			return
		}
		respDiff, err := diff(respName, actualResp)
		if err != nil || strings.TrimSpace(respDiff) != "" {
			t.Errorf("FAIL: %s: Diff from expected response:\n%s", testPart, respDiff)
			testFailed = true
		}
		logDiff, err := diff(logName, actualLog)
		if err != nil || strings.TrimSpace(logDiff) != "" {
			t.Errorf("FAIL: %s: Diff from expected logs:\n%s", testPart, logDiff)
			testFailed = true
		}
		if testFailed {
			continue
		}
		delay, err := ioutil.ReadFile(testPart + ".delay")
		if delaySecs, _ := strconv.Atoi(string(delay)); err == nil && delaySecs > 0 {
			time.Sleep(time.Duration(delaySecs) * time.Second)
		}
		t.Logf("PASS: %s", testPart)
	}
}
