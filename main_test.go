package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDirCarriesTheOldFolderOver(t *testing.T) {
	base := t.TempDir()
	old := filepath.Join(base, legacyName)
	os.MkdirAll(old, 0o700)
	os.WriteFile(filepath.Join(old, "config.json"), []byte("token"), 0o600)

	dir := dataDir(base)
	if b, _ := os.ReadFile(filepath.Join(dir, "config.json")); string(b) != "token" {
		t.Fatalf("config not carried over to %s", dir)
	}
	if _, err := os.Stat(old); err == nil {
		t.Error("old folder still there")
	}

	// A folder under the new name is never replaced by an old one that reappears.
	os.MkdirAll(old, 0o700)
	os.WriteFile(filepath.Join(old, "config.json"), []byte("stale"), 0o600)
	if b, _ := os.ReadFile(filepath.Join(dataDir(base), "config.json")); string(b) != "token" {
		t.Error("existing folder was replaced")
	}

	// New install: no old folder, no error.
	if got := dataDir(t.TempDir()); filepath.Base(got) != "abysstracker-autosubmit" {
		t.Errorf("got %s", got)
	}
}
