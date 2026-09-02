package paths

import (
	"os"
	"strings"
	"testing"
)

func TestHome_EnvOverride(t *testing.T) {
	t.Setenv("PIVOT_HOME", "/tmp/test-pivot-home")
	home, err := Home()
	if err != nil {
		t.Fatal(err)
	}
	if home != "/tmp/test-pivot-home" {
		t.Fatalf("expected /tmp/test-pivot-home, got %s", home)
	}
}

func TestHome_Default(t *testing.T) {
	os.Unsetenv("PIVOT_HOME")
	home, err := Home()
	if err != nil {
		t.Fatal(err)
	}
	if home == "" {
		t.Fatal("home should not be empty")
	}
}

func TestConfigFile_ContainsDotPivot(t *testing.T) {
	t.Setenv("PIVOT_HOME", "/tmp/ph")
	path, err := ConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, ".pivot") {
		t.Errorf("config path should contain .pivot, got %s", path)
	}
	if !strings.HasSuffix(path, "config.yaml") {
		t.Errorf("config path should end with config.yaml, got %s", path)
	}
}

func TestStateFile_ContainsDotPivot(t *testing.T) {
	t.Setenv("PIVOT_HOME", "/tmp/ph")
	path, err := StateFile()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, ".pivot") {
		t.Errorf("state path should contain .pivot, got %s", path)
	}
	if !strings.HasSuffix(path, "state.db") {
		t.Errorf("state path should end with state.db, got %s", path)
	}
}
