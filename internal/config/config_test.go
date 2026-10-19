package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckURL(t *testing.T) {
	for raw, ok := range map[string]bool{
		"http://localhost:8110/api/v1":         true,
		"https://example.com/worldapi/api/v1/": true,
		"http://localhost:8110":                false,
		"ftp://localhost:8110/api/v1":          false,
		"":                                     false,
		"localhost:8110/api/v1":                false,
	} {
		if err := CheckURL(raw); (err == nil) != ok {
			t.Errorf("CheckURL(%q) = %v, want ok=%v", raw, err, ok)
		}
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	write := func(text string) string {
		path := filepath.Join(dir, "targets.yml")
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	file, err := LoadFile(write("targets:\n  survival:\n    url: http://survival:8110/api/v1\n    players: false\n"))
	if err != nil {
		t.Fatal(err)
	}
	if target := file.Targets["survival"]; target.Players == nil || *target.Players {
		t.Errorf("players override lost: %+v", target)
	}

	for text, want := range map[string]string{
		"targets: {}\n": "no targets",
		"targets:\n  a:\n    url: http://a:8110\n":                                         "/api/v1",
		"targets:\n  a:\n    url: http://a:8110/api/v1\n    tokn: x\n":                     "tokn",
		"targets:\n  a:\n    url: http://a:8110/api/v1\n    token: x\n    token_file: y\n": "not both",
	} {
		if _, err := LoadFile(write(text)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("LoadFile(%q) = %v, want an error about %q", text, err, want)
		}
	}
}

func TestReadToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if token, _ := ReadToken("", path); token != "secret" {
		t.Errorf("token from file = %q", token)
	}
	if token, _ := ReadToken("given", path); token != "given" {
		t.Errorf("token = %q, want the given one", token)
	}
}

func TestDuration(t *testing.T) {
	if d, _ := Duration("90"); d != 90*time.Second {
		t.Errorf("Duration(90) = %v", d)
	}
	if d, _ := Duration("2m"); d != 2*time.Minute {
		t.Errorf("Duration(2m) = %v", d)
	}
	if _, err := Duration("soon"); err == nil {
		t.Error("Duration(soon) passed")
	}
}
