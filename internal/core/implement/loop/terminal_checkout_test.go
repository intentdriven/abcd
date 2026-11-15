//go:build unix

package loop

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/gittest"
)

// seenByPreflight is what one preflight found in the checkout it was given.
type seenByPreflight struct {
	dir     string
	aGo     string
	goWork  bool
	strayGo bool
}

// watchPreflight records what each preflight of w finds in its checkout.
func watchPreflight(w *terminalWorld) *[]seenByPreflight {
	var seen []seenByPreflight
	w.onPreflight = func(dir string) error {
		a, _ := os.ReadFile(filepath.Join(dir, "a.go"))
		_, werr := os.Lstat(filepath.Join(dir, "go.work"))
		_, zerr := os.Lstat(filepath.Join(dir, "zz_x.go"))
		seen = append(seen, seenByPreflight{dir: dir, aGo: string(a), goWork: werr == nil, strayGo: zerr == nil})
		return nil
	}
	return &seen
}

// TestTheDrivesPreflightRunsInItsOwnCheckoutOfTheHead: whatever the lane's
// worktree holds beside the head (an index whose entry names the head's blob
// with stat data matching a changed file, an ignored go.work at its root, a
// file hidden by a widened info/exclude), each reading clean to git status
// there, the drive's preflight runs once, in a checkout of its own that holds
// the head's files and nothing else, the lane lands on the receipt it took,
// and the checkout is gone afterwards.
func TestTheDrivesPreflightRunsInItsOwnCheckoutOfTheHead(t *testing.T) {
	cases := []struct {
		name  string
		lane  func(t *testing.T, w *terminalWorld, wt *gittest.Repo)
		check func(s seenByPreflight) string
	}{
		{name: "an index whose entry names the head's blob",
			lane: func(t *testing.T, w *terminalWorld, wt *gittest.Repo) {
				blob, err := hex.DecodeString(strings.TrimSpace(wt.Git("rev-parse", w.head+":a.go")))
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(w.worktree, "a.go")
				wt.Write("a.go", "package a\n\nvar changed = 1\n")
				past := time.Now().Add(-10 * time.Second)
				if err := os.Chtimes(path, past, past); err != nil {
					t.Fatal(err)
				}
				wt.Git("add", "a.go")
				craftIndexBlob(t, strings.TrimSpace(wt.Git("rev-parse", "--path-format=absolute", "--git-path", "index")), "a.go", blob)
			},
			check: func(s seenByPreflight) string {
				if s.aGo != "package a\n" {
					return "a.go is not the head's: " + s.aGo
				}
				return ""
			}},
		{name: "an ignored go.work",
			lane: func(t *testing.T, w *terminalWorld, wt *gittest.Repo) {
				wt.Write("go.work", "go 1.22\n\nuse .\n")
			},
			check: func(s seenByPreflight) string {
				if s.goWork {
					return "a go.work stood in the checkout"
				}
				return ""
			}},
		{name: "a file a widened info/exclude hides",
			lane: func(t *testing.T, w *terminalWorld, wt *gittest.Repo) {
				exclude := filepath.Join(strings.TrimSpace(wt.Git("rev-parse", "--path-format=absolute", "--git-common-dir")), "info", "exclude")
				if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
					t.Fatal(err)
				}
				was, _ := os.ReadFile(exclude)
				if err := os.WriteFile(exclude, append(was, []byte("\nzz_*.go\n")...), 0o644); err != nil {
					t.Fatal(err)
				}
				wt.Write("zz_x.go", "package a\n\nfunc init() {}\n")
			},
			check: func(s seenByPreflight) string {
				if s.strayGo {
					return "zz_x.go stood in the checkout"
				}
				return ""
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newTerminalWorld(t)
			w.plantedReceipt = true
			wt := gittest.OpenRepo(t, w.worktree)
			c.lane(t, w, wt)
			if out := wt.Git("status", "--porcelain"); out != "" {
				t.Fatalf("the lane's worktree does not read clean, so the case shows nothing: %q", out)
			}
			seen := watchPreflight(w)
			o := oneLane(t, w)
			res, err := RunTerminal(context.Background(), w.repo.Root(), o)
			if err != nil {
				t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
			}
			if len(*seen) != 1 {
				t.Fatalf("the preflight ran %d times, want once:\n%s", len(*seen), strings.Join(w.lines, "\n"))
			}
			s := (*seen)[0]
			if s.dir == w.worktree {
				t.Fatalf("the preflight ran in the lane's worktree %s, not the drive's own checkout", s.dir)
			}
			if why := c.check(s); why != "" {
				t.Fatalf("the preflight did not run on the head's files: %s", why)
			}
			if len(lanesDone(t, w.repo)) != 1 {
				t.Fatalf("the lane did not land on the receipt the drive took: %+v\n%s", res, strings.Join(w.lines, "\n"))
			}
			if _, err := os.Lstat(filepath.Join(w.repo.Root(), filepath.FromSlash(landing.ReceiptsRelDir), w.head)); err != nil {
				t.Fatalf("the receipt is not in the main checkout's tier: %v", err)
			}
			if _, err := os.Lstat(s.dir); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("the drive's checkout %s is still there: %v", s.dir, err)
			}
			if strings.Contains(w.repo.Git("worktree", "list", "--porcelain"), filepath.Base(filepath.Dir(s.dir))) {
				t.Fatalf("git still lists the drive's checkout:\n%s", w.repo.Git("worktree", "list", "--porcelain"))
			}
		})
	}
}

// TestAHeldSettingChangedBeforeThePreflightHoldsTheLanding: a setting that
// changes what a checkout writes, changed after the drive started and before
// the lane's preflight, holds the landing naming it, and no preflight runs.
func TestAHeldSettingChangedBeforeThePreflightHoldsTheLanding(t *testing.T) {
	w := newTerminalWorld(t)
	w.plantedReceipt = true
	for i := range w.stages {
		if w.stages[i].Name == StageValidate {
			run := w.stages[i].Run
			w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
				w.repo.Git("config", "core.autocrlf", "input")
				return run(c, l)
			}
		}
	}
	o := oneLane(t, w)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	said := strings.Join(w.lines, "\n") + "\n" + res.Why
	if err != nil {
		said += "\n" + err.Error()
	}
	if len(lanesDone(t, w.repo)) != 0 {
		t.Fatalf("a lane landed after a held setting changed: %+v\n%s", res, said)
	}
	if len(w.preflights) != 0 {
		t.Fatalf("the preflight ran after a held setting changed: %q", w.preflights)
	}
	if !strings.Contains(said, "core.autocrlf") {
		t.Fatalf("the hold does not name the setting:\n%s", said)
	}
}

// TestAPreflightThatChangesTheCheckoutLeavesNoReceiptTaken: a checkout that no
// longer holds the head's content once the preflight ran (a tracked file
// changed, or an ignored file left outside the local tier) holds the landing,
// and the lane does not land.
func TestAPreflightThatChangesTheCheckoutLeavesNoReceiptTaken(t *testing.T) {
	for name, change := range map[string]func(dir string) error{
		"a tracked file changed": func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n\nvar later = 1\n"), 0o644)
		},
		"an ignored file left": func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.22\n"), 0o644)
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := newTerminalWorld(t)
			w.plantedReceipt = true
			w.onPreflight = change
			o := oneLane(t, w)
			res, err := RunTerminal(context.Background(), w.repo.Root(), o)
			if err != nil {
				t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
			}
			if len(lanesDone(t, w.repo)) != 0 {
				t.Fatalf("a lane landed on a checkout its preflight changed: %+v\n%s", res, strings.Join(w.lines, "\n"))
			}
			if len(w.preflights) == 0 {
				t.Fatal("no preflight ran")
			}
			if !strings.Contains(strings.Join(w.lines, "\n"), "once its preflight ran") {
				t.Fatalf("the hold does not say why:\n%s", strings.Join(w.lines, "\n"))
			}
		})
	}
}

// craftIndexBlob sets the object id of name's entry in the index file at path
// to blob, leaving the entry's stat data as it is, and writes the index's
// trailing checksum again: version 2 or 3, SHA-1.
func craftIndexBlob(t *testing.T, path, name string, blob []byte) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 32 || string(raw[:4]) != "DIRC" || len(blob) != sha1.Size {
		t.Fatalf("%s is not a SHA-1 index this helper reads", path)
	}
	version := binary.BigEndian.Uint32(raw[4:8])
	if version != 2 && version != 3 {
		t.Fatalf("index version %d is not one this helper reads", version)
	}
	found := false
	off := 12
	for i := uint32(0); i < binary.BigEndian.Uint32(raw[8:12]); i++ {
		flags := binary.BigEndian.Uint16(raw[off+60 : off+62])
		header := 62
		if version == 3 && flags&0x4000 != 0 {
			header += 2
		}
		n := int(flags & 0x0fff)
		if string(raw[off+header:off+header+n]) == name {
			copy(raw[off+40:off+60], blob)
			found = true
		}
		off += (header + n + 8) &^ 7
	}
	if !found {
		t.Fatalf("the index holds no entry %s", name)
	}
	sum := sha1.Sum(raw[:len(raw)-sha1.Size])
	copy(raw[len(raw)-sha1.Size:], sum[:])
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
