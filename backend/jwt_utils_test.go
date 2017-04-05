package backend

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func TestRandString(t *testing.T) {
	r := randString(16)
	if len(r) != 16 {
		t.Errorf("Random string should be 16 bytes long: %s\n", r)
	}
}

func TestJWTUtils(t *testing.T) {
	testkey := "testhashkey01234testhashkey01234"
	jwtManager := NewJwtManager([]byte(testkey))

	if jwtManager.method != jwt.SigningMethodHS256 {
		t.Errorf("Default signing method wasn't used: %v %v\n", jwt.SigningMethodHS256, jwtManager.method)
	}
	if string(jwtManager.key) != testkey {
		t.Errorf("Key was not set: %v %v\n", testkey, string(jwtManager.key))
	}

	jwtManager = NewJwtManager([]byte(testkey), JwtConfig{Method: jwt.SigningMethodRS512})
	if jwtManager.method != jwt.SigningMethodRS512 {
		t.Errorf("Default signing method wasn't used: %v %v\n", jwt.SigningMethodRS512, jwtManager.method)
	}
	if string(jwtManager.key) != testkey {
		t.Errorf("Key was not set: %v %v\n", testkey, string(jwtManager.key))
	}

	jwtManager = NewJwtManager([]byte(randString(32)))
	tok := jwtManager.newToken("fred", 30, "all", "a", "m")
	s, e := jwtManager.sign(tok)
	if e != nil {
		t.Errorf("Failed to sign token: %v\n", e)
	}
	drpClaim, e := jwtManager.get(s)
	if e != nil {
		t.Errorf("Failed to get token: %v\n", e)
	} else {
		if drpClaim.Id != "fred" {
			t.Errorf("Claim ID doesn't match: %v %v\n", "fred", drpClaim.Id)
		}
		if drpClaim.Scope != "all" {
			t.Errorf("Claim Scope doesn't match: %v %v\n", "all", drpClaim.Scope)
		}
		if drpClaim.Action != "a" {
			t.Errorf("Claim Action doesn't match: %v %v\n", "a", drpClaim.Action)
		}
		if drpClaim.Specific != "m" {
			t.Errorf("Claim Specific doesn't match: %v %v\n", "m", drpClaim.Specific)
		}
	}

	tok = jwtManager.newToken("fred", 1, "all", "m", "a")
	s, e = jwtManager.sign(tok)
	if e != nil {
		t.Errorf("Failed to sign token: %v\n", e)
	}
	time.Sleep(1000 * 1000 * 1000 * 3)
	drpClaim, e = jwtManager.get(s)
	if e == nil {
		t.Errorf("Failed because we got a token: %v\n", drpClaim)
	}
}
