package repolint

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestTheReleaseEntryDoesNotAcceptBareSnapshot holds the release glossary
// entry off the bare word "snapshot" as an alias (iss-2610262037128325). The
// lab chapter defines a lab's snapshot as a standalone clone detached at its
// pin, so a bare "snapshot" alias made the word an accepted name for two
// different things. "published snapshot" stays: it names a release only.
func TestTheReleaseEntryDoesNotAcceptBareSnapshot(t *testing.T) {
	const entry = ".abcd/development/brief/glossary/distribution/release.md"
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test working directory")
		}
		dir = parent
	}
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(entry)))
	if err != nil {
		t.Fatalf("read %s: %v", entry, err)
	}
	line := regexp.MustCompile(`(?m)^aliases:\s*\[(.*)\]\s*$`).FindStringSubmatch(string(raw))
	if line == nil {
		t.Fatalf("%s has no aliases: line", entry)
	}
	var aliases []string
	for _, a := range strings.Split(line[1], ",") {
		aliases = append(aliases, strings.Trim(strings.TrimSpace(a), `"`))
	}
	var published bool
	for _, a := range aliases {
		if strings.EqualFold(a, "snapshot") {
			t.Errorf("%s lists the bare alias %q; a lab's snapshot is a pinned clone, not a release", entry, a)
		}
		if a == "published snapshot" {
			published = true
		}
	}
	if !published {
		t.Errorf("%s no longer lists the alias \"published snapshot\" (aliases: %q)", entry, aliases)
	}
}
