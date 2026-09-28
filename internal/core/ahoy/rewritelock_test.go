package ahoy

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// rivalWindow is how long the simulated concurrent writer holds the rewrite
// lock with the rival started: far longer than any of the rewrites under test
// takes when nothing makes it wait, so an unguarded rival always lands inside
// the window.
const rivalWindow = 300 * time.Millisecond

// raceRewrite plays a concurrent writer of path against rival, one of ahoy's
// load-modify-writes of the same file (iss-127). The writer takes path's
// rewrite lock the way every writer of the file does, reads the file, starts
// rival, waits rivalWindow and then writes edit applied to the bytes it read.
// A rival that takes the lock waits for the writer and then rewrites from the
// writer's bytes, so both changes survive; a rival that does not lands inside
// the window and the writer's stale write erases its change.
func raceRewrite(t *testing.T, path string, edit func([]byte) []byte, rival func()) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	err := fsutil.WithFileLock(rewriteLockPath(path), 5*time.Second, func() error {
		before, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		go func() {
			defer close(done)
			rival()
		}()
		time.Sleep(rivalWindow)
		return os.WriteFile(path, edit(before), 0o644)
	})
	if err != nil {
		t.Fatalf("the concurrent writer failed: %v", err)
	}
	<-done
}

// setJSON returns an edit that sets section.key to v in a JSON object.
func setJSON(t *testing.T, section, key string, v any) func([]byte) []byte {
	return func(raw []byte) []byte {
		m := map[string]any{}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Errorf("the concurrent writer read unparseable config: %v", err)
			}
		}
		setSub(m, section, key, v)
		out, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			t.Errorf("marshal: %v", err)
		}
		return append(out, '\n')
	}
}

// appendLine returns an edit that appends line to a text file.
func appendLine(line string) func([]byte) []byte {
	return func(raw []byte) []byte { return append(append([]byte{}, raw...), line+"\n"...) }
}

func readConfigT(t *testing.T, dir string) map[string]any {
	t.Helper()
	m, err := readConfig(dir)
	if err != nil {
		t.Fatalf("readConfig: %v", err)
	}
	return m
}

// TestConfigRewritesKeepAConcurrentWritersChange is the iss-127 detector for
// .abcd/config.json: every ahoy read-modify-write of the file, raced against a
// writer that changes another key, must keep both changes.
func TestConfigRewritesKeepAConcurrentWritersChange(t *testing.T) {
	cases := []struct {
		name  string
		rival func(dir string)
		// landed reports whether the rival's own change is on disk.
		landed func(m map[string]any) bool
	}{
		{
			name: "attribution opt-in",
			rival: func(dir string) {
				(&applyCtx{cwd: dir}).recordAttributionOptIn()
			},
			landed: func(m map[string]any) bool {
				v, ok := boolVal(subMap(m, "attribution"), "hook")
				return ok && v
			},
		},
		{
			name: "version stamp",
			rival: func(dir string) {
				(&applyCtx{
					cwd:        dir,
					approved:   map[GapCategory]bool{SafeAutocreate: true},
					gapPresent: map[string]bool{"install_meta.missing": true},
				}).stepVersionStamp()
			},
			landed: func(m map[string]any) bool {
				_, ok := subMap(m, "meta")["setup_version"]
				return ok
			},
		},
		{
			name: "config values",
			rival: func(dir string) {
				(&applyCtx{
					cwd:        dir,
					approved:   map[GapCategory]bool{},
					gapPresent: map[string]bool{},
					overrides:  map[string]string{"visibility": "public"},
					prompter:   RefusingPrompter{},
					autoYes:    true,
				}).stepConfigValues()
			},
			landed: func(m map[string]any) bool {
				v, _ := stringVal(subMap(m, "repo"), "visibility")
				return v == "public"
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeValidConfig(t, dir, "private", "both", "host-delegated")
			raceRewrite(t, configPath(dir), setJSON(t, "scan", "deep", false), func() { tc.rival(dir) })

			m := readConfigT(t, dir)
			if !tc.landed(m) {
				t.Errorf("the rewrite's own change was lost to the concurrent writer: %v", m)
			}
			if v, ok := boolVal(subMap(m, "scan"), "deep"); !ok || v {
				t.Errorf("the concurrent writer's scan.deep=false was lost to the rewrite: %v", m)
			}
			assertLockRetired(t, configPath(dir))
		})
	}
}

// TestSkeletonNeverReplacesAConfigWrittenMeanwhile: the starter settings file
// is planted when detection found none, and a config another writer created
// since is kept rather than replaced by the seed.
func TestSkeletonNeverReplacesAConfigWrittenMeanwhile(t *testing.T) {
	dir := t.TempDir()
	path := configPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	err := fsutil.WithFileLock(rewriteLockPath(path), 5*time.Second, func() error {
		if err := os.WriteFile(path, []byte(`{"docs": {"target": "agents_md"}}`+"\n"), 0o644); err != nil {
			return err
		}
		go func() {
			defer close(done)
			(&applyCtx{
				cwd:        dir,
				approved:   map[GapCategory]bool{SafeAutocreate: true},
				gapPresent: map[string]bool{"skeleton.config_missing": true},
			}).stepSkeleton()
		}()
		time.Sleep(rivalWindow)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	<-done
	if v, _ := stringVal(subMap(readConfigT(t, dir), "docs"), "target"); v != "agents_md" {
		t.Errorf("the seed replaced a config written after detection: docs.target = %q", v)
	}
}

// TestRulesSkeletonNeverReplacesARulesFileWrittenMeanwhile: detection sets
// rules.missing before the interactive prompts, and stepRules runs after them.
// A rules.json written in that window holds a hand-written override; the
// skeleton must keep it byte for byte and must not report a write it did not
// make (iss-2609281931185016).
func TestRulesSkeletonNeverReplacesARulesFileWrittenMeanwhile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, filepath.FromSlash(rulesRelPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	a := &applyCtx{
		cwd:        dir,
		approved:   map[GapCategory]bool{SafeAutocreate: true},
		gapPresent: map[string]bool{"rules.missing": true},
	}
	// Written after detection saw no file, before the step runs.
	written := []byte(`{"schema_version": 1, "domains": {"HOUSE": {"recall": ["house"], "rules": ["keep it"]}}}` + "\n")
	if err := os.WriteFile(path, written, 0o600); err != nil {
		t.Fatal(err)
	}
	a.stepRules()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, written) {
		t.Errorf("the skeleton replaced a rules.json written after detection:\n got %s\nwant %s", got, written)
	}
	for _, k := range a.writeKinds {
		if k == writeRules {
			t.Errorf("the receipt reports a rules write that did not happen: writes = %v", a.writes)
		}
	}
	if len(a.notes) != 0 {
		t.Errorf("a kept rules.json is not a refusal; notes = %q", a.notes)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("the kept file's mode changed: %v %v", fi.Mode().Perm(), err)
	}
}

// TestRulesSkeletonIsWrittenWhenAbsent is the other half: with no rules.json
// the skeleton is planted, at 0644 whatever the umask, and reported.
func TestRulesSkeletonIsWrittenWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	a := &applyCtx{
		cwd:        dir,
		approved:   map[GapCategory]bool{SafeAutocreate: true},
		gapPresent: map[string]bool{"rules.missing": true},
	}
	a.stepRules()

	path := filepath.Join(dir, filepath.FromSlash(rulesRelPath))
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the skeleton was not written: %v (notes %q)", err, a.notes)
	}
	var doc map[string]any
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("the skeleton is not JSON: %v\n%s", err, got)
	}
	if d, ok := doc["domains"].(map[string]any); !ok || len(d) != 0 {
		t.Errorf("the skeleton is not the empty-domains override: %s", got)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o644 {
		t.Errorf("the skeleton's mode is not 0644: %v %v", fi.Mode().Perm(), err)
	}
	if len(a.writeKinds) != 1 || a.writeKinds[0] != writeRules {
		t.Errorf("the receipt does not report the rules write: kinds = %v", a.writeKinds)
	}
}

// TestGitignoreBlockKeepsAConcurrentEdit is the iss-127 detector for the
// .gitignore block rewrite.
func TestGitignoreBlockKeepsAConcurrentEdit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(path, []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var rivalErr error
	raceRewrite(t, path, appendLine("dist/"), func() {
		_, rivalErr = applyVisibilityBlock(dir, "private")
	})
	if rivalErr != nil {
		t.Fatalf("applyVisibilityBlock: %v", rivalErr)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte(gitignoreBegin)) {
		t.Errorf("the abcd block was lost to the concurrent edit:\n%s", got)
	}
	if !bytes.Contains(got, []byte("dist/\n")) {
		t.Errorf("the concurrent edit was lost to the block rewrite:\n%s", got)
	}
	assertLockRetired(t, path)
}

// TestMarkerRewritesKeepAConcurrentEdit is the iss-127 detector for the marker
// block's install and removal.
func TestMarkerRewritesKeepAConcurrentEdit(t *testing.T) {
	t.Run("install", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "CLAUDE.md")
		if err := os.WriteFile(path, []byte("# Project\n\nbody\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		var rivalErr error
		raceRewrite(t, path, appendLine("appended meanwhile"), func() {
			_, rivalErr = installMarkerFile(path)
		})
		if rivalErr != nil {
			t.Fatalf("installMarkerFile: %v", rivalErr)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !markerBlockRe.Match(got) {
			t.Errorf("the marker block was lost to the concurrent edit:\n%s", got)
		}
		if !strings.Contains(string(got), "appended meanwhile") {
			t.Errorf("the concurrent edit was lost to the marker install:\n%s", got)
		}
		assertLockRetired(t, path)
	})
	t.Run("remove", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "AGENTS.md")
		if err := os.WriteFile(path, []byte("# Project\n\nbody\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := installMarkerFile(path); err != nil {
			t.Fatal(err)
		}
		var rivalErr error
		raceRewrite(t, path, appendLine("appended meanwhile"), func() {
			_, rivalErr = removeMarkerFile(path)
		})
		if rivalErr != nil {
			t.Fatalf("removeMarkerFile: %v", rivalErr)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if markerBlockRe.Match(got) {
			t.Errorf("the removed marker block came back through the concurrent edit:\n%s", got)
		}
		if !strings.Contains(string(got), "appended meanwhile") {
			t.Errorf("the concurrent edit was lost to the marker removal:\n%s", got)
		}
		assertLockRetired(t, path)
	})
}

// assertLockRetired: the rewrite lock beside a file the user owns is removed
// by its last holder, so a rewrite leaves no lock file behind in the tree.
func assertLockRetired(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(rewriteLockPath(path)); !os.IsNotExist(err) {
		t.Errorf("the rewrite lock %s was left behind (err=%v)", filepath.Base(rewriteLockPath(path)), err)
	}
}
