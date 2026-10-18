package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"offer-hub/backend/src/config"
)

func TestRunReportsFailedStep(t *testing.T) {
	old := config.Conf
	t.Cleanup(func() { config.Conf = old })
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	})
	t.Setenv("APP_ENV", "startup-test")
	err = run()
	if err == nil || !strings.Contains(err.Error(), "config.Init") {
		t.Fatalf("expected config.Init error, got %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config-startup-test.toml"), []byte("[COMMON]\nport = 8180\n"), 0600); err != nil {
		t.Fatal(err)
	}
	err = run()
	if err == nil || !strings.Contains(err.Error(), "data.NewData") {
		t.Fatalf("expected data.NewData error, got %v", err)
	}
}

func TestMainLogsAndExitsOnStartupFailure(t *testing.T) {
	if os.Getenv("OFFER_HUB_TEST_MAIN_EXIT") == "1" {
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestMainLogsAndExitsOnStartupFailure$")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "OFFER_HUB_TEST_MAIN_EXIT=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(string(output), "config.Init") {
		t.Fatalf("missing initialization error log: %s", output)
	}
}
