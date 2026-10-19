// Package config reads the exporter's settings from flags, environment variables, and the targets file.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Target is one WorldAPI to read.
type Target struct {
	URL       string `yaml:"url"`
	Token     string `yaml:"token"`
	TokenFile string `yaml:"token_file"`
	// Players overrides the exporter's default for this target.
	Players *bool `yaml:"players"`
}

// File is the targets file behind /probe.
type File struct {
	Targets map[string]Target `yaml:"targets"`
}

// LoadFile reads the targets file and checks every target.
func LoadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file File
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(file.Targets) == 0 {
		return nil, fmt.Errorf("%s lists no targets", path)
	}
	for name, target := range file.Targets {
		if err := CheckURL(target.URL); err != nil {
			return nil, fmt.Errorf("%s, target %q: %w", path, name, err)
		}
		if target.Token != "" && target.TokenFile != "" {
			return nil, fmt.Errorf("%s, target %q: set token or token_file, not both", path, name)
		}
	}
	return &file, nil
}

// CheckURL accepts a WorldAPI address like http://localhost:8110/api/v1.
func CheckURL(raw string) error {
	if raw == "" {
		return errors.New("no WorldAPI URL")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%q isn't an http or https URL", raw)
	}
	if !strings.HasSuffix(strings.TrimRight(u.Path, "/"), "/api/v1") {
		return fmt.Errorf("%q should end in /api/v1, like http://localhost:8110/api/v1", raw)
	}
	return nil
}

// ReadToken returns token, or the first line of tokenFile when token is empty.
func ReadToken(token, tokenFile string) (string, error) {
	if token != "" || tokenFile == "" {
		return token, nil
	}
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// Duration reads a duration like 60s, or plain seconds.
func Duration(raw string) (time.Duration, error) {
	if d, err := time.ParseDuration(raw); err == nil {
		return d, nil
	}
	d, err := time.ParseDuration(raw + "s")
	if err != nil {
		return 0, fmt.Errorf("%q isn't a duration like 60s", raw)
	}
	return d, nil
}
