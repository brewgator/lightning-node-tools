package testutils

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTestDBPath creates a temporary SQLite database file path for testing
func CreateTestDBPath(t *testing.T) string {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()
	return filepath.Join(tmpDir, "test.db")
}

// AssertNoError is a helper to check for no error
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// AssertError is a helper to check for expected error
func AssertError(t *testing.T, err error, expectedMsg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("Expected error containing '%s', got nil", expectedMsg)
	}
	// We can add more specific error message checking here if needed
}

// AssertEqual checks if two values are equal
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// AssertNotEqual checks if two values are not equal
func AssertNotEqual(t *testing.T, got, notWant interface{}) {
	t.Helper()
	if got == notWant {
		t.Errorf("got %v, expected it to be different", got)
	}
}

// CleanupTestFiles removes test files and directories
func CleanupTestFiles(paths ...string) {
	for _, path := range paths {
		os.RemoveAll(path)
	}
}
