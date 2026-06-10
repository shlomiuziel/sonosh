package cli

import (
	"os"
	"regexp"
	"testing"
)

func TestVersionMatchesLatestChangelogRelease(t *testing.T) {
	body, err := os.ReadFile("../../CHANGELOG.md")
	if err != nil {
		t.Fatalf("read changelog: %v", err)
	}

	re := regexp.MustCompile(`(?m)^## \[([0-9]+\.[0-9]+\.[0-9]+)\] - `)
	match := re.FindSubmatch(body)
	if match == nil {
		t.Fatal("missing released changelog heading")
	}
	if got, want := Version, string(match[1]); got != want {
		t.Fatalf("Version = %q, latest changelog release = %q", got, want)
	}
}
