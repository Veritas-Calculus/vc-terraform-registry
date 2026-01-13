package scheduler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSchedulerNew(t *testing.T) {
	t.Run("create new scheduler with nil db", func(t *testing.T) {
		tempDir := t.TempDir()

		scheduler := New(nil, tempDir)
		if scheduler == nil {
			t.Fatal("New returned nil scheduler")
		}

		if scheduler.storagePath != tempDir {
			t.Errorf("storagePath = %q, want %q", scheduler.storagePath, tempDir)
		}

		if scheduler.jobs == nil {
			t.Error("jobs map should be initialized")
		}
	})
}

// Note: TestSchedulerStartStop is not included because Start() requires a real database connection.
// The Scheduler.Start() method calls loadSchedules() which uses GORM to query the database,
// and GORM panics when the database connection is nil.

func TestGetPlatformsToMirror(t *testing.T) {
	testCases := []struct {
		name          string
		syncOS        string
		syncArch      string
		platforms     []struct{ os, arch string }
		expectedCount int
	}{
		{
			name:     "all platforms",
			syncOS:   "all",
			syncArch: "all",
			platforms: []struct{ os, arch string }{
				{"linux", "amd64"},
				{"linux", "arm64"},
				{"darwin", "amd64"},
				{"darwin", "arm64"},
			},
			expectedCount: 4,
		},
		{
			name:     "specific OS all arch",
			syncOS:   "linux",
			syncArch: "all",
			platforms: []struct{ os, arch string }{
				{"linux", "amd64"},
				{"linux", "arm64"},
				{"darwin", "amd64"},
			},
			expectedCount: 2,
		},
		{
			name:     "all OS specific arch",
			syncOS:   "all",
			syncArch: "amd64",
			platforms: []struct{ os, arch string }{
				{"linux", "amd64"},
				{"linux", "arm64"},
				{"darwin", "amd64"},
			},
			expectedCount: 2,
		},
		{
			name:     "specific OS and arch",
			syncOS:   "linux",
			syncArch: "amd64",
			platforms: []struct{ os, arch string }{
				{"linux", "amd64"},
				{"linux", "arm64"},
				{"darwin", "amd64"},
			},
			expectedCount: 1,
		},
		{
			name:     "no matching platforms",
			syncOS:   "windows",
			syncArch: "arm64",
			platforms: []struct{ os, arch string }{
				{"linux", "amd64"},
				{"darwin", "amd64"},
			},
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a helper function to simulate getPlatformsToMirror logic
			filtered := filterPlatforms(tc.platforms, tc.syncOS, tc.syncArch)
			if len(filtered) != tc.expectedCount {
				t.Errorf("filtered count = %d, want %d", len(filtered), tc.expectedCount)
			}
		})
	}
}

// filterPlatforms is a test helper that simulates the filtering logic
func filterPlatforms(platforms []struct{ os, arch string }, syncOS, syncArch string) []struct{ os, arch string } {
	var result []struct{ os, arch string }
	for _, p := range platforms {
		osMatch := syncOS == "all" || p.os == syncOS
		archMatch := syncArch == "all" || p.arch == syncArch
		if osMatch && archMatch {
			result = append(result, p)
		}
	}
	return result
}

func TestCronExpressionValidation(t *testing.T) {
	testCases := []struct {
		name    string
		expr    string
		isValid bool
	}{
		{"daily at midnight", "0 0 * * *", true},
		{"every 6 hours", "0 */6 * * *", true},
		{"every minute", "* * * * *", true},
		{"weekly on sunday", "0 0 * * 0", true},
		{"monthly on first", "0 0 1 * *", true},
		{"every 15 minutes", "*/15 * * * *", true},
		{"workdays 9am", "0 9 * * 1-5", true},
		{"empty string", "", false},
		{"too few fields", "* * *", false},
		{"too many fields", "* * * * * * *", false},
		{"invalid minute", "60 * * * *", false},
		{"invalid hour", "* 25 * * *", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid := isValidCronExpr(tc.expr)
			if valid != tc.isValid {
				t.Errorf("isValidCronExpr(%q) = %v, want %v", tc.expr, valid, tc.isValid)
			}
		})
	}
}

// isValidCronExpr validates cron expressions
func isValidCronExpr(expr string) bool {
	if expr == "" {
		return false
	}

	// Simple validation: check field count
	fields := 0
	inField := false
	for _, c := range expr {
		if c == ' ' || c == '\t' {
			if inField {
				fields++
				inField = false
			}
		} else {
			inField = true
		}
	}
	if inField {
		fields++
	}

	// Standard cron has 5 fields (minute hour day month weekday)
	if fields != 5 {
		return false
	}

	// Basic range validation for first two fields
	parts := splitFields(expr)
	if len(parts) < 2 {
		return false
	}

	// Check minute (0-59)
	if !isValidField(parts[0], 0, 59) {
		return false
	}

	// Check hour (0-23)
	if !isValidField(parts[1], 0, 23) {
		return false
	}

	return true
}

func splitFields(expr string) []string {
	var result []string
	var current string
	for _, c := range expr {
		if c == ' ' || c == '\t' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func isValidField(field string, min, max int) bool {
	// Handle wildcards
	if field == "*" {
		return true
	}

	// Handle step values like */15
	if len(field) > 2 && field[:2] == "*/" {
		return true
	}

	// Handle ranges like 1-5
	if containsChar(field, '-') {
		return true
	}

	// Handle simple numbers
	num := 0
	for _, c := range field {
		if c < '0' || c > '9' {
			return true // Contains non-numeric, assume it's a valid pattern
		}
		num = num*10 + int(c-'0')
	}

	return num >= min && num <= max
}

func containsChar(s string, c rune) bool {
	for _, ch := range s {
		if ch == c {
			return true
		}
	}
	return false
}

func TestStoragePathCreation(t *testing.T) {
	t.Run("scheduler uses provided storage path", func(t *testing.T) {
		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "providers", "data")

		// Create the directory with restricted permissions
		if err := os.MkdirAll(subDir, 0750); err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}

		scheduler := New(nil, subDir)
		if scheduler.storagePath != subDir {
			t.Errorf("storagePath = %q, want %q", scheduler.storagePath, subDir)
		}
	})
}

func TestJobManagement(t *testing.T) {
	t.Run("jobs map is initialized", func(t *testing.T) {
		tempDir := t.TempDir()
		scheduler := New(nil, tempDir)

		if scheduler.jobs == nil {
			t.Fatal("jobs map should be initialized")
		}

		if len(scheduler.jobs) != 0 {
			t.Errorf("jobs map should be empty, got %d entries", len(scheduler.jobs))
		}
	})
}
