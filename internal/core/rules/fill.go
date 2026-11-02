package rules

// How full the session is (spc-2610111627213138 step 5, "How full the session
// is"). The prompt hook's payload carries no measure of the context window, so
// the loader learns it from two sources, the status line's first, as the
// facilitator ruled (2026-10-11):
//
//  1. The status line's fill file. The `statusline` verb, on the path that
//     renders abcd's own row, records the payload's context percentage for its
//     session in ~/.abcd.noindex/fill/<sha256(session)>.json. It is the one
//     file the status line may write (the product thinker's ruling of
//     2026-10-11, "Allow one fill file"), and this file owns it: its name, its
//     writer, its reader, its reset and its sweep are all here.
//  2. The transcript's tail. Without a fill file for the session, the hook
//     reads at most the last TranscriptTailBytes of the transcript its payload
//     names, takes the last response's input tokens, and divides them by the
//     window the per-model table below declares for that response's model.
//
// When neither tells, the fill is unknown, and says why. Step 5 reports the
// fill and acts on nothing; the bracket table that acts on it is step 6's.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The fill's sources, as the hook's --json envelope names them.
const (
	FillSourceStatusLine = "status-line"
	FillSourceTranscript = "transcript"
	FillSourceUnknown    = "unknown"
)

// TranscriptTailBytes bounds the transcript read: at most this many bytes,
// from the end, are read, however long the session's transcript has grown.
// The figure is the spec's starting point, set here alone.
const TranscriptTailBytes = 256 * 1024

// maxFillFileBytes caps a fill file on read. The file holds two short fields;
// anything near this size is not one the status line wrote.
const maxFillFileBytes = 4 * 1024

// fillRel is the fill directory, relative to the account's home.
var fillRel = abcdhome.Rel("fill")

// FillDirDisplay is the fill directory in the tilde form a message or a help
// text names it by.
var FillDirDisplay = abcdhome.Display("fill") + "/"

// ErrNoFillDirectory is WriteFill's answer when the fill directory is not
// there: the session-start reset makes it, and the status line never does, so
// a machine whose session-start hook is not installed, or whose folder was
// swept, keeps no fill. A caller skips the write with no notice: the status
// line is stateless and cannot say it once.
var ErrNoFillDirectory = errors.New("rules: the fill directory is not there, so no fill is kept")

// Fill is how full the session is, and where that came from. Source is one
// of the FillSource constants. UsedPercentage is the share of the context
// window in use, 0 to 100, and nil exactly when the source is unknown. Model
// is the window table's name for the model a transcript-derived fill was
// divided by; At is when the status line recorded its figure. Reason says
// why the fill is unknown, in abcd's own words: no path and no text from the
// files read is ever echoed in it.
type Fill struct {
	Source         string   `json:"source"`
	UsedPercentage *float64 `json:"used_percentage,omitempty"`
	Model          string   `json:"model,omitempty"`
	At             string   `json:"at,omitempty"`
	Reason         string   `json:"reason,omitempty"`
}

// Known reports whether the fill was learned from a source.
func (f Fill) Known() bool { return f.UsedPercentage != nil }

// fillRecord is the fill file's whole content: the two fields, and nothing
// else (TestStatuslineWritesOnlyTheFillFile holds the status line to them).
type fillRecord struct {
	UsedPercentage *float64 `json:"used_percentage"`
	At             string   `json:"at"`
}

// validPercentage reports whether p is a percentage of a window: finite and
// within 0 to 100.
func validPercentage(p float64) bool {
	return !math.IsNaN(p) && !math.IsInf(p, 0) && p >= 0 && p <= 100
}

// WriteFill records the context percentage the status line was handed for a
// session, in the session's fill file, mode 0600, atomically, through the home
// scope's guarded walk, so never behind a symlinked ~/.abcd.noindex. It is
// the status line's one write, and the only writer of the file.
//
// It creates nothing but the file: the directory is the session-start
// reset's to make (ResetFill). A directory that is not there is
// ErrNoFillDirectory, decided before any level is judged, so a symlinked
// ~/.abcd.noindex holding no fill directory is a quiet skip like any other
// absence; one that IS there behind a link, or in a directory another account
// can change, is refused and the error says so. With no home or no session
// there is nothing to name the file by, and nothing is written.
func WriteFill(home, session string, usedPct float64, at time.Time) error {
	if home == "" || session == "" {
		return ErrNoFillDirectory
	}
	if !validPercentage(usedPct) {
		return fmt.Errorf("rules: a context percentage of %v is not within 0 to 100, so it was not recorded", usedPct)
	}
	if _, err := os.Stat(abcdhome.Path(home, "fill")); err != nil {
		if os.IsNotExist(err) {
			return ErrNoFillDirectory
		}
		return fmt.Errorf("rules: %s cannot be looked at: %w", FillDirDisplay, pathCause(err))
	}
	dir, err := fsutil.OpenHomeDeclarationScope(home, fillRel)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNoFillDirectory
		}
		return fmt.Errorf("rules: %s was refused: %w", FillDirDisplay, err)
	}
	defer dir.Close()
	data, err := json.Marshal(fillRecord{UsedPercentage: &usedPct, At: at.UTC().Format(time.RFC3339)})
	if err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomicInRoot(dir, sessionLeaf(session), data, abcdhome.FileMode); err != nil {
		return fmt.Errorf("rules: the session's fill file in %s was not written: %w", FillDirDisplay, pathCause(err))
	}
	return nil
}

// ResetFill is the fill's half of the reset triggers. It makes the fill
// directory (0700) when it is not there, through the home scope's guarded
// walk, so a symlinked ~/.abcd.noindex is refused and no fill directory lands
// under its target, and it deletes the session's fill file, so a compaction
// does not leave the hook reading the full session's figure until the status
// line next refreshes. With no home it does nothing.
//
// A symlinked ~/.abcd.noindex is refused quietly: the error wraps
// fsutil.ErrHomeScopeSymlinked, which the hook does not report, since a
// dotfiles home is the machine's layout and the status line simply keeps no
// fill there. Any other failure is returned for the hook to name.
func ResetFill(home, session string) error {
	if home == "" {
		return nil
	}
	dir, err := fsutil.EnsureHomeDeclarationScope(home, fillRel, abcdhome.DirMode)
	if err != nil {
		return fmt.Errorf("rules: %s could not be made: %w", FillDirDisplay, err)
	}
	defer dir.Close()
	if session == "" {
		return nil
	}
	if err := dir.Remove(sessionLeaf(session)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("rules: the session's fill file in %s was not deleted: %w", FillDirDisplay, pathCause(err))
	}
	return nil
}

// pruneFill is PruneState's sweep of the fill directory, by the same age.
// Best-effort like it: a missing or refused directory, and a per-entry error,
// are ignored.
func pruneFill(home string, maxAge time.Duration) {
	if home == "" {
		return
	}
	dir, err := fsutil.OpenHomeDeclarationScope(home, fillRel)
	if err != nil {
		return
	}
	defer dir.Close()
	pruneRoot(dir, maxAge)
}

// ReadFill says how full the session is: from the session's fill file when
// there is one, otherwise from the tail of the transcript the hook's payload
// names, otherwise unknown with the reason. It writes nothing and opens only
// those two files and the directories on the way to them.
func ReadFill(home, session, transcriptPath string) Fill {
	if f, ok := readFillFile(home, session); ok {
		return f
	}
	if transcriptPath == "" {
		return unknownFill("no fill file for this session and no transcript named")
	}
	return fillFromTranscript(transcriptPath)
}

func unknownFill(reason string) Fill {
	return Fill{Source: FillSourceUnknown, Reason: reason}
}

// readFillFile reads the session's fill file with the guards every
// home-scoped declaration gets: the directory walked level by level and
// refused behind a symlink or where another account can change it, and the
// file a regular file this account owns that no one else can write. Any
// refusal, an absent file and one that does not hold a percentage of a
// window all read as no fill file, and the transcript is asked instead.
func readFillFile(home, session string) (Fill, bool) {
	if home == "" || session == "" {
		return Fill{}, false
	}
	dir, err := fsutil.OpenHomeDeclarationScope(home, fillRel)
	if err != nil {
		return Fill{}, false
	}
	defer dir.Close()
	data, refusal, err := fsutil.ReadHomeDeclarationDenyingIn(dir, home, path.Join(fillRel, sessionLeaf(session)), maxFillFileBytes, 0o022)
	if err != nil || refusal != fsutil.DeclarationOK {
		return Fill{}, false
	}
	var rec fillRecord
	if err := json.Unmarshal(data, &rec); err != nil || rec.UsedPercentage == nil || !validPercentage(*rec.UsedPercentage) {
		return Fill{}, false
	}
	at := ""
	if t, err := time.Parse(time.RFC3339, rec.At); err == nil {
		at = t.UTC().Format(time.RFC3339)
	}
	pct := *rec.UsedPercentage
	return Fill{Source: FillSourceStatusLine, UsedPercentage: &pct, At: at}, true
}

// transcriptLine is the part of one transcript line the fill reader looks
// at: an assistant line's model and token usage, in the shape
// internal/core/history parses, and whether the line is a side chain's.
type transcriptLine struct {
	Type        string `json:"type"`
	IsSidechain bool   `json:"isSidechain"`
	Message     *struct {
		Model string              `json:"model"`
		Usage *history.TokenUsage `json:"usage"`
	} `json:"message"`
}

// fillFromTranscript divides the last response's input by its model's window.
func fillFromTranscript(p string) Fill {
	tail, err := readTranscriptTail(p)
	if err != nil {
		return unknownFill(err.Error())
	}
	model, usage, ok := lastResponseUsage(tail)
	if !ok {
		return unknownFill("the transcript's tail holds no response's token usage")
	}
	name := windowModelName(model)
	window, ok := contextWindow(name)
	if !ok {
		return unknownFill("the transcript's last response names a model the window table does not hold")
	}
	pct := float64(usage.ContextInput()) * 100 / float64(window)
	if pct > 100 {
		// More input than the table's window: the session runs a larger
		// window than the table holds for its model. Full errs towards
		// shrinking, as the table's smaller window does.
		pct = 100
	}
	return Fill{Source: FillSourceTranscript, UsedPercentage: &pct, Model: name}
}

// lastResponseUsage scans tail from the end for the last assistant line of
// the main conversation whose usage counts any input, and returns its model
// and usage. A side chain's line is another conversation's window, and a line
// with no input (the harness's own synthetic notices) is no request at all,
// so neither says how full this session is.
func lastResponseUsage(tail []byte) (model string, usage history.TokenUsage, ok bool) {
	lines := bytes.Split(tail, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		line := bytes.TrimSpace(lines[i])
		if len(line) == 0 {
			continue
		}
		var tl transcriptLine
		if err := json.Unmarshal(line, &tl); err != nil {
			continue
		}
		if tl.Type != "assistant" || tl.IsSidechain || tl.Message == nil || tl.Message.Usage == nil {
			continue
		}
		if tl.Message.Usage.ContextInput() <= 0 {
			continue
		}
		return tl.Message.Model, *tl.Message.Usage, true
	}
	return "", history.TokenUsage{}, false
}

// readTranscriptTail reads at most the last TranscriptTailBytes of the
// transcript at p through the guarded read: opened once, read-only, never
// through a symlinked leaf (O_NOFOLLOW) and never blocking on a FIFO or a
// device, and judged on that same descriptor to be a regular file this account
// owns that no one else can write. A tail that starts inside a line drops the
// partial line. The errors are abcd's own words and never repeat the path.
func readTranscriptTail(p string) ([]byte, error) {
	if !filepath.IsAbs(p) {
		return nil, errors.New("the transcript path is not absolute, so the transcript was not read")
	}
	f, fi, err := fsutil.OpenRegular(p)
	if err != nil {
		if errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP) {
			return nil, errors.New("the transcript was refused: it is a symlink or not a regular file")
		}
		if os.IsNotExist(err) {
			return nil, errors.New("the transcript named is not there")
		}
		return nil, errors.New("the transcript could not be opened")
	}
	defer f.Close()
	if err := fsutil.CallersAlone(p, fi); err != nil {
		return nil, errors.New("the transcript was refused: it is not this account's own, or another account can write it")
	}
	if sys, ok := fi.Sys().(*syscall.Stat_t); !ok || sys.Uid != uint32(os.Getuid()) {
		return nil, errors.New("the transcript was refused: it is not this account's own, or another account can write it")
	}
	off := fi.Size() - TranscriptTailBytes
	if off < 0 {
		off = 0
	}
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return nil, errors.New("the transcript could not be read")
	}
	data, err := io.ReadAll(io.LimitReader(f, TranscriptTailBytes))
	if err != nil {
		return nil, errors.New("the transcript could not be read")
	}
	if off > 0 {
		// The read began inside a line, which is not a whole entry.
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[i+1:]
		} else {
			data = nil
		}
	}
	return data, nil
}

// contextWindows is the per-model window table, in tokens, beside the bracket
// table step 6 adds. A model offered with more than one window size lists
// each, and contextWindow takes the smaller, so a doubt about the window
// shrinks recall rather than flooding the context. A model not listed has no
// window here, and a fill read for it is unknown. The keys are the models'
// API names without a snapshot date (windowModelName).
//
// Grounds: the Claude 3, 3.5, 3.7 and 4 families each offer a 200,000-token
// window, and Claude Sonnet 4 and 4.5 offer a 1,000,000-token window beside
// it; claude-haiku-5-5's 1,000,000 is the window the host itself reports for
// it (internal/core/runner/testdata/claude-stream-2.1.302.jsonl,
// modelUsage.contextWindow). A model is added here with its source.
var contextWindows = map[string][]int64{
	"claude-3-haiku":    {200_000},
	"claude-3-sonnet":   {200_000},
	"claude-3-opus":     {200_000},
	"claude-3-5-haiku":  {200_000},
	"claude-3-5-sonnet": {200_000},
	"claude-3-7-sonnet": {200_000},
	"claude-sonnet-4":   {200_000, 1_000_000},
	"claude-opus-4":     {200_000},
	"claude-opus-4-1":   {200_000},
	"claude-sonnet-4-5": {200_000, 1_000_000},
	"claude-haiku-4-5":  {200_000},
	"claude-opus-4-5":   {200_000},
	"claude-haiku-5-5":  {1_000_000},
}

// contextWindow is the window the table declares for model, in tokens: the
// smallest size the model is offered with. ok is false for a model the table
// does not name.
func contextWindow(model string) (window int64, ok bool) {
	sizes, ok := contextWindows[model]
	if !ok || len(sizes) == 0 {
		return 0, false
	}
	window = sizes[0]
	for _, s := range sizes[1:] {
		if s < window {
			window = s
		}
	}
	return window, true
}

// snapshotDate is the dated snapshot suffix an API model name carries
// ("claude-sonnet-4-5-20250929").
var snapshotDate = regexp.MustCompile(`-[0-9]{8}$`)

// windowModelName is the window table's key for a transcript's model name:
// lower-cased, with its snapshot date dropped, since every snapshot of a
// model is offered with that model's windows.
func windowModelName(model string) string {
	return snapshotDate.ReplaceAllString(strings.ToLower(strings.TrimSpace(model)), "")
}

// pathCause strips the path from a *os.PathError, so an error a hook prints
// names what failed and not the home it failed under.
func pathCause(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}
