package rules

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// fillHome is a home whose fill directory exists, as the session-start reset
// leaves it, owner-only.
func fillHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	if err := ResetFill(home, ""); err != nil {
		t.Fatalf("ResetFill: %v", err)
	}
	return home
}

func fillFile(home, session string) string {
	return abcdhome.Path(home, "fill", ledgerLeaf(session))
}

// assistantLine is one transcript line of a response from model with the
// given input, cache-creation and cache-read token counts.
func assistantLine(model string, input, created, read int64) string {
	return fmt.Sprintf(`{"type":"assistant","message":{"id":"msg_%d","role":"assistant","model":%q,"content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":%d,"cache_creation_input_tokens":%d,"cache_read_input_tokens":%d,"output_tokens":7}}}`,
		input+created+read, model, input, created, read)
}

const userLine = `{"type":"user","message":{"role":"user","content":"hello"}}`

func writeTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(realDir(t), "session.jsonl")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// realDir is a temporary directory by its resolved path, so a path built in
// it names no symlink on the way (macOS's /var is one).
func realDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func wantPct(t *testing.T, f Fill, source string, pct float64) {
	t.Helper()
	if f.Source != source {
		t.Fatalf("source = %q (reason %q), want %q", f.Source, f.Reason, source)
	}
	if f.UsedPercentage == nil {
		t.Fatalf("used_percentage absent, want %v", pct)
	}
	if d := *f.UsedPercentage - pct; d > 1e-9 || d < -1e-9 {
		t.Fatalf("used_percentage = %v, want %v", *f.UsedPercentage, pct)
	}
}

func wantUnknown(t *testing.T, f Fill, reason string) {
	t.Helper()
	if f.Source != FillSourceUnknown || f.UsedPercentage != nil || f.Known() {
		t.Fatalf("fill = %+v, want unknown", f)
	}
	if !strings.Contains(f.Reason, reason) {
		t.Fatalf("reason = %q, want it to say %q", f.Reason, reason)
	}
}

// TestWriteFillRecordsTheTwoFields: the fill file is the session's, named by
// the hash of its id, 0600, and holds the percentage and the time alone.
func TestWriteFillRecordsTheTwoFields(t *testing.T) {
	home := fillHome(t)
	at := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	if err := WriteFill(home, "s1", 42.5, at); err != nil {
		t.Fatalf("WriteFill: %v", err)
	}
	fi, err := os.Lstat(fillFile(home, "s1"))
	if err != nil {
		t.Fatalf("the fill file is not named by the session's hash: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %04o, want 0600", perm)
	}
	data, err := os.ReadFile(fillFile(home, "s1"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["used_percentage"] != 42.5 || got["at"] != "2026-11-01T12:00:00Z" {
		t.Errorf("fill file = %s, want used_percentage 42.5 and at alone", data)
	}
	wantPct(t, ReadFill(home, "s1", ""), FillSourceStatusLine, 42.5)
	if f := ReadFill(home, "s1", ""); f.At != "2026-11-01T12:00:00Z" {
		t.Errorf("at = %q", f.At)
	}
	// Another session's fill is not this one's.
	wantUnknown(t, ReadFill(home, "s2", ""), "no fill file")
}

// TestWriteFillRefusesAFigureNotOfAWindow: a percentage outside 0 to 100 is
// not recorded.
func TestWriteFillRefusesAFigureNotOfAWindow(t *testing.T) {
	home := fillHome(t)
	for _, p := range []float64{-1, 100.5} {
		if err := WriteFill(home, "s1", p, time.Now()); err == nil || errors.Is(err, ErrNoFillDirectory) {
			t.Errorf("WriteFill(%v) = %v, want a refusal", p, err)
		}
	}
	if _, err := os.Lstat(fillFile(home, "s1")); !os.IsNotExist(err) {
		t.Errorf("a refused figure left a file: %v", err)
	}
}

// TestAbsentFillDirectoryIsASilentSkip: the status line never makes the
// fill directory, so where it is not there the write is ErrNoFillDirectory,
// which the caller skips quietly, and nothing is created: not the directory,
// and not ~/.abcd.noindex either.
func TestAbsentFillDirectoryIsASilentSkip(t *testing.T) {
	home := t.TempDir()
	if err := WriteFill(home, "s1", 10, time.Now()); !errors.Is(err, ErrNoFillDirectory) {
		t.Fatalf("WriteFill with no abcd home = %v, want ErrNoFillDirectory", err)
	}
	if _, err := os.Lstat(abcdhome.Path(home)); !os.IsNotExist(err) {
		t.Errorf("the write created ~/.abcd.noindex: %v", err)
	}
	if err := os.Mkdir(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := WriteFill(home, "s1", 10, time.Now()); !errors.Is(err, ErrNoFillDirectory) {
		t.Fatalf("WriteFill with no fill directory = %v, want ErrNoFillDirectory", err)
	}
	if _, err := os.Lstat(abcdhome.Path(home, "fill")); !os.IsNotExist(err) {
		t.Errorf("the write created the fill directory: %v", err)
	}
	// No home, or no session, names no file.
	if err := WriteFill("", "s1", 10, time.Now()); !errors.Is(err, ErrNoFillDirectory) {
		t.Errorf("WriteFill with no home = %v", err)
	}
	if err := WriteFill(fillHome(t), "", 10, time.Now()); !errors.Is(err, ErrNoFillDirectory) {
		t.Errorf("WriteFill with no session = %v", err)
	}
}

// symlinkedHome is a home whose ~/.abcd.noindex is a symlink to a real
// directory elsewhere (a dotfiles layout), and that target.
func symlinkedHome(t *testing.T) (home, target string) {
	t.Helper()
	home = t.TempDir()
	target = t.TempDir()
	if err := os.Symlink(target, abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	return home, target
}

// TestFillFileIsRefusedBehindASymlinkedHome: a fill directory behind a
// symlinked ~/.abcd.noindex is neither written nor read, and the write says
// so rather than skipping as an absent directory does.
func TestFillFileIsRefusedBehindASymlinkedHome(t *testing.T) {
	home, target := symlinkedHome(t)
	if err := os.Mkdir(filepath.Join(target, "fill"), 0o700); err != nil {
		t.Fatal(err)
	}
	err := WriteFill(home, "s1", 10, time.Now())
	if err == nil || errors.Is(err, ErrNoFillDirectory) || !errors.Is(err, fsutil.ErrHomeScopeSymlinked) {
		t.Fatalf("WriteFill behind a symlinked home = %v, want the symlink refused", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(target, "fill")); len(entries) != 0 {
		t.Errorf("the write landed behind the link: %v", entries)
	}
	// A fill file planted there is not read.
	if err := os.WriteFile(filepath.Join(target, "fill", ledgerLeaf("s1")), []byte(`{"used_percentage": 12, "at": ""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	wantUnknown(t, ReadFill(home, "s1", ""), "no fill file")
	// And with no fill directory behind the link, the write is the quiet
	// skip of an absent one.
	home2, _ := symlinkedHome(t)
	if err := WriteFill(home2, "s1", 10, time.Now()); !errors.Is(err, ErrNoFillDirectory) {
		t.Errorf("WriteFill behind a symlinked home with no fill directory = %v, want ErrNoFillDirectory", err)
	}
}

// TestFillFileOthersCanWriteIsNotRead: a fill file another account could
// have written, or one in a directory another account can change, is not this
// account's word and is not read.
func TestFillFileOthersCanWriteIsNotRead(t *testing.T) {
	home := fillHome(t)
	if err := WriteFill(home, "s1", 30, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(fillFile(home, "s1"), 0o666); err != nil {
		t.Fatal(err)
	}
	wantUnknown(t, ReadFill(home, "s1", ""), "no fill file")
	if err := os.Chmod(fillFile(home, "s1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(abcdhome.Path(home, "fill"), 0o777); err != nil {
		t.Fatal(err)
	}
	wantUnknown(t, ReadFill(home, "s1", ""), "no fill file")
	// Nor is a file holding something other than a percentage of a window.
	if err := os.Chmod(abcdhome.Path(home, "fill"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{"at": "x"}`, `{"used_percentage": 140}`, `not json`} {
		if err := os.WriteFile(fillFile(home, "s1"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		wantUnknown(t, ReadFill(home, "s1", ""), "no fill file")
	}
}

// TestFillPrefersTheStatusLineFile: with both a fill file for the session
// and a transcript that would say otherwise, the status line's figure wins.
func TestFillPrefersTheStatusLineFile(t *testing.T) {
	home := fillHome(t)
	if err := WriteFill(home, "s1", 12, time.Now()); err != nil {
		t.Fatal(err)
	}
	tr := writeTranscript(t, userLine, assistantLine("claude-opus-4-1", 10, 100_000, 50_000))
	wantPct(t, ReadFill(home, "s1", tr), FillSourceStatusLine, 12)
	// Without it, the transcript says.
	wantPct(t, ReadFill(home, "s2", tr), FillSourceTranscript, 150_010.0*100/200_000)
}

// TestFillFromTheTranscriptTail: with no fill file, the last response's
// whole input (uncached, cache-created and cache-read) over its model's
// window, the model named by its table name with the snapshot date dropped.
func TestFillFromTheTranscriptTail(t *testing.T) {
	tr := writeTranscript(t,
		userLine,
		assistantLine("claude-opus-4-1-20250805", 5, 1000, 1000),
		userLine,
		assistantLine("claude-opus-4-1-20250805", 2, 30_000, 70_000),
	)
	f := ReadFill(t.TempDir(), "s1", tr)
	wantPct(t, f, FillSourceTranscript, 100_002.0*100/200_000)
	if f.Model != "claude-opus-4-1" {
		t.Errorf("model = %q, want claude-opus-4-1", f.Model)
	}
	// With no session id and no home, the transcript still says.
	wantPct(t, ReadFill("", "", tr), FillSourceTranscript, 100_002.0*100/200_000)
}

// TestTranscriptFillSkipsWhatIsNotThisSessionsRequest: a side chain's
// response is another conversation's window, and a response with no input
// (the harness's synthetic notice) is no request, so the reader passes both
// for the last real response before them.
func TestTranscriptFillSkipsWhatIsNotThisSessionsRequest(t *testing.T) {
	side := strings.Replace(assistantLine("claude-opus-4-1", 1, 1, 190_000), `{"type":"assistant"`, `{"type":"assistant","isSidechain":true`, 1)
	tr := writeTranscript(t,
		assistantLine("claude-opus-4-1", 0, 0, 20_000),
		side,
		assistantLine("<synthetic>", 0, 0, 0),
		`{"type":"system","content":"x"}`,
		`not json at all`,
	)
	wantPct(t, ReadFill("", "", tr), FillSourceTranscript, 10)
}

// TestFillOfAFullerWindowThanTheTableIsFull: input past the table's window
// (a session running a larger window than the table holds) reads as full.
func TestFillOfAFullerWindowThanTheTableIsFull(t *testing.T) {
	tr := writeTranscript(t, assistantLine("claude-sonnet-4-5", 0, 0, 600_000))
	wantPct(t, ReadFill("", "", tr), FillSourceTranscript, 100)
}

// TestTranscriptReadIsBoundedToTheTail: only the last TranscriptTailBytes
// are read. A response whose usage lies before them is never seen, so a
// transcript whose tail holds none is unknown; one whose tail holds one is
// read from it, and the partial line the tail starts inside is dropped.
func TestTranscriptReadIsBoundedToTheTail(t *testing.T) {
	pad := `{"type":"user","message":{"role":"user","content":"` + strings.Repeat("x", 1000) + `"}}`
	var head []string
	head = append(head, assistantLine("claude-opus-4-1", 0, 0, 40_000))
	for n := 0; n*len(pad) < TranscriptTailBytes+4096; n++ {
		head = append(head, pad)
	}
	wantUnknown(t, ReadFill("", "", writeTranscript(t, head...)), "holds no response's token usage")

	withTail := append(append([]string{}, head...), assistantLine("claude-opus-4-1", 0, 0, 60_000))
	wantPct(t, ReadFill("", "", writeTranscript(t, withTail...)), FillSourceTranscript, 30)

	// A tail that starts inside a usage line does not read the severed line.
	long := assistantLine("claude-opus-4-1", 0, 0, 80_000)
	long = strings.Replace(long, `"text":"ok"`, `"text":"`+strings.Repeat("y", TranscriptTailBytes)+`"`, 1)
	wantUnknown(t, ReadFill("", "", writeTranscript(t, long, pad)), "holds no response's token usage")
}

// TestRefusedTranscriptYieldsUnknownFill: a transcript the guarded read
// refuses is unknown, and the reason never repeats its path.
func TestRefusedTranscriptYieldsUnknownFill(t *testing.T) {
	tr := writeTranscript(t, assistantLine("claude-opus-4-1", 0, 0, 40_000))
	link := filepath.Join(filepath.Dir(tr), "link.jsonl")
	if err := os.Symlink(tr, link); err != nil {
		t.Fatal(err)
	}
	f := ReadFill("", "", link)
	wantUnknown(t, f, "refused")
	if strings.Contains(f.Reason, filepath.Dir(tr)) {
		t.Errorf("the reason repeats the path: %q", f.Reason)
	}
	t.Run("foreign owner", func(t *testing.T) {
		restore := fsutil.SwapOwnerUIDForTest(func(string) (uint32, error) { return uint32(os.Getuid()) + 1, nil })
		defer restore()
		wantUnknown(t, ReadFill("", "", tr), "not this account's own")
	})
	t.Run("writable by others", func(t *testing.T) {
		other := writeTranscript(t, assistantLine("claude-opus-4-1", 0, 0, 40_000))
		if err := os.Chmod(other, 0o666); err != nil {
			t.Fatal(err)
		}
		wantUnknown(t, ReadFill("", "", other), "another account can write it")
	})
	t.Run("a directory", func(t *testing.T) {
		wantUnknown(t, ReadFill("", "", realDir(t)), "refused")
	})
	t.Run("absent", func(t *testing.T) {
		wantUnknown(t, ReadFill("", "", filepath.Join(realDir(t), "gone.jsonl")), "not there")
	})
	t.Run("relative", func(t *testing.T) {
		wantUnknown(t, ReadFill("", "", "session.jsonl"), "not absolute")
	})
	// And the plain read of the same file is admitted.
	wantPct(t, ReadFill("", "", tr), FillSourceTranscript, 20)
}

// TestUnnamedModelYieldsUnknownFill: a model the window table does not name
// has no window to divide by, and the reason does not echo the model.
func TestUnnamedModelYieldsUnknownFill(t *testing.T) {
	f := ReadFill("", "", writeTranscript(t, assistantLine("someone-elses-model-9", 0, 0, 40_000)))
	wantUnknown(t, f, "does not hold")
	if strings.Contains(f.Reason, "someone-elses") {
		t.Errorf("the reason echoes the model: %q", f.Reason)
	}
	wantUnknown(t, ReadFill("", "", ""), "no transcript named")
}

// TestAmbiguousWindowTakesTheSmaller: a model offered with more than one
// window size is divided by the smaller, whatever order the table lists them
// in, so a doubt shrinks recall rather than flooding the context.
func TestAmbiguousWindowTakesTheSmaller(t *testing.T) {
	if w, ok := contextWindow("claude-sonnet-4-5"); !ok || w != 200_000 {
		t.Errorf("contextWindow(claude-sonnet-4-5) = %d, %v; want 200000", w, ok)
	}
	contextWindows["test-model-two-windows"] = []int64{1_000_000, 200_000}
	defer delete(contextWindows, "test-model-two-windows")
	if w, ok := contextWindow("test-model-two-windows"); !ok || w != 200_000 {
		t.Errorf("contextWindow = %d, %v; want the smaller, 200000", w, ok)
	}
	tr := writeTranscript(t, assistantLine("claude-sonnet-4-5-20250929", 0, 0, 100_000))
	wantPct(t, ReadFill("", "", tr), FillSourceTranscript, 50)
	if _, ok := contextWindow("claude-sonnet-4-5-20250929"); ok {
		t.Error("the table is keyed on the dated name; windowModelName drops the date")
	}
}

// TestEveryTableEntryHasAWindow: no model is listed with no size.
func TestEveryTableEntryHasAWindow(t *testing.T) {
	for m, sizes := range contextWindows {
		if len(sizes) == 0 {
			t.Errorf("%s lists no window", m)
		}
		for _, s := range sizes {
			if s <= 0 {
				t.Errorf("%s lists a window of %d", m, s)
			}
		}
		if windowModelName(m) != m {
			t.Errorf("%s is not a table key windowModelName can produce", m)
		}
	}
}

// TestResetDeletesTheFillFile: the reset deletes the session's fill file and
// no other session's, so a compaction does not leave the old figure read.
func TestResetDeletesTheFillFile(t *testing.T) {
	home := fillHome(t)
	for _, s := range []string{"s1", "s2"} {
		if err := WriteFill(home, s, 80, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	if err := ResetFill(home, "s1"); err != nil {
		t.Fatalf("ResetFill: %v", err)
	}
	if _, err := os.Lstat(fillFile(home, "s1")); !os.IsNotExist(err) {
		t.Errorf("the session's fill file survived the reset: %v", err)
	}
	wantUnknown(t, ReadFill(home, "s1", ""), "no fill file")
	wantPct(t, ReadFill(home, "s2", ""), FillSourceStatusLine, 80)
	// A reset with no file to delete is not an error.
	if err := ResetFill(home, "s1"); err != nil {
		t.Errorf("a second reset: %v", err)
	}
}

// TestResetCreatesTheFillDirectoryAndRefusesASymlinkedHome: the reset makes
// ~/.abcd.noindex/fill/ owner-only, and behind a symlinked ~/.abcd.noindex it
// refuses, with the symlink refusal the hook keeps quiet, and makes nothing
// under the link's target.
func TestResetCreatesTheFillDirectoryAndRefusesASymlinkedHome(t *testing.T) {
	home := t.TempDir()
	if err := ResetFill(home, "s1"); err != nil {
		t.Fatalf("ResetFill: %v", err)
	}
	fi, err := os.Lstat(abcdhome.Path(home, "fill"))
	if err != nil || !fi.IsDir() {
		t.Fatalf("the reset did not make the fill directory: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o700 {
		t.Errorf("fill directory mode = %04o, want 0700", perm)
	}
	linked, target := symlinkedHome(t)
	err = ResetFill(linked, "s1")
	if !errors.Is(err, fsutil.ErrHomeScopeSymlinked) {
		t.Fatalf("ResetFill behind a symlinked home = %v, want the symlink refusal", err)
	}
	if _, err := os.Lstat(filepath.Join(target, "fill")); !os.IsNotExist(err) {
		t.Errorf("the reset made the fill directory under the link's target: %v", err)
	}
	// No home: nothing to make.
	if err := ResetFill("", "s1"); err != nil {
		t.Errorf("ResetFill with no home = %v", err)
	}
}

// TestStaleSweepReachesTheFillDirectory: the sweep of stale session ledgers
// sweeps the fill directory by the same age, and keeps a fresh fill.
func TestStaleSweepReachesTheFillDirectory(t *testing.T) {
	home := fillHome(t)
	t.Setenv("HOME", home)
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	for _, s := range []string{"fresh", "old"} {
		if err := WriteFill(home, s, 5, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	past := time.Now().Add(-StateTTL - time.Hour)
	if err := os.Chtimes(fillFile(home, "old"), past, past); err != nil {
		t.Fatal(err)
	}
	PruneState(StateTTL)
	if _, err := os.Lstat(fillFile(home, "old")); !os.IsNotExist(err) {
		t.Error("a stale fill file survived the sweep")
	}
	if _, err := os.Lstat(fillFile(home, "fresh")); err != nil {
		t.Errorf("a fresh fill file was swept: %v", err)
	}
}
