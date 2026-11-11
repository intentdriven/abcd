package history

// resumable.go — a staged transcript redacted in parts, so a drain that its
// hook's bound stops keeps what it did (iss-2611101302324425).
//
// The prompt hook drains under a bound below the host's limit, and one staged
// transcript can need more redaction time than any one prompt has. Redacted
// whole, such a transcript was never finished: every prompt started it again
// and the host ended the hook first, discarding its output. So a drain redacts
// a transcript in PARTS, each a run of whole transcript lines, and keeps every
// part it finishes on disk until the last one is done and the record is
// written; the next drain picks up at the first byte no kept part covers.
//
// WHAT A PASS DOES WITH ONE ENTRY. The first part of every entry runs
// whatever its size and whatever the pass's context, as the whole transcript
// did before parts existed, so a transcript that cannot be split is redacted
// at the same pass as before. A later part starts whenever the context has not
// ended: that context is the hook's bound, which sits under the host's limit.
// Past the bound no part starts; what the pass has not reached is left, with
// the parts kept before it, for the next pass. A part has no upper size (a
// part is whole transcript lines, so one long line is one part), and a part
// in progress when the bound passes runs to its end. A part, first or later,
// that runs past the host's limit ends the hook there without its output, as
// redacting the transcript whole did before; nothing of that part is kept,
// and at the next pass it is the first part. So the hook's output is delivered
// when the part in progress at the bound ends inside the margin. The one case
// stored later than before is a transcript whose drain would have ended inside
// the margin, between the bound and the host's limit: it is finished at the
// next pass. A pass whose context never ends (the drain verb) finishes every
// entry it reaches.
//
// When nothing the pass does can be picked up by a later one (a part could
// not be kept, the binary has a stamp unique to the process, or gitleaks
// reads the text whole), stopping saves nothing, so the entry runs to its
// record as it did before parts existed: the bound stops no later part, and
// the records lock is waited for as long as a capture waits.
//
// THE ON-DISK STATE, and why a drain killed after a part is kept, after its
// parts are done, after the record is written or after the staged copy is
// removed leaves the staged original or the whole record and never neither
// (the four boundaries drainStep names, each tested):
//
//   - The staged .raw is the only place unredacted text lives, as before. It
//     is removed only after the record is on disk (Drain's
//     removeStagedIfUnchanged), and a kill before that leaves it in place.
//   - A part is written only after its text passed BOTH redaction stages (the
//     re-scan of redactWith included), atomically, under
//     staging/progress/<staged name>.parts/. A kill mid-write leaves no part or
//     the whole part. A part is never under records/, so no reader of the
//     store (List, Read, the reconstructors) can mistake parts for a
//     transcript.
//   - The record is written once, atomically, from parts that cover the whole
//     framed text, under the records lock. A kill before the rename leaves no
//     record; after it, the whole record.
//
// A part that cannot be kept (its directory refused or unwritable, the disk
// full) does not fail the entry: the pass carries on in memory to the record,
// as a whole capture would, and names the fault once in DrainResult.KeepFault.
//
// WHEN A KEPT PART IS REUSED. A part is reused only when everything that
// decided its content is unchanged, so the record is the one a single pass
// over the whole text writes. Its header records the sha256 of the exact input
// bytes it covers, whether it ended at the end of the text as the text then
// stood, and the fingerprint of the scanner it was redacted with (partsScanner:
// the pattern set, the identity, the augmenter, the caller's home and the
// running binary's bytes). The parts must run without a gap from byte zero; a
// part that did not end at the end of the text must end at a safe cut of the
// current text; one that did is reused only while the text is still exactly
// that long, since a line appended after it could close a block its last line
// opened. A part that fails any of that is removed, with every part after it,
// and that stretch is redacted again. So parts kept under another
// configuration (another binary, home, pattern set or identity, as two
// sessions on two builds have) are redone, never mixed in. When the scanner
// reads the text whole (an augmenter is armed), no kept part is used at all
// and the pass starts from byte zero.
//
// A part holds redacted text only, so a part left behind (its staged file
// discarded, quarantined or stored by an older binary) costs disk and no
// privacy; a drain removes the parts whose staged file is gone unless that
// file's lock is held or the pass's bound has passed, when a later drain does,
// and a discard removes its file's parts. A part read back is held to the same trust
// as the staged file beside it: both live in the owner-only staging directory,
// and a process that can write a forged part there could as well write the
// staged bytes or the record itself. Every directory on the way to a part is
// proved a real directory, with the store's own primitives
// (fsutil.EnsureRealDirAll to create, fsutil.ProbeRealDirAll to read,
// fsutil.OpenRealDir to pin), before anything in it is read, written or
// removed.
//
// WHERE A PART MAY END. The scanner's detection is line-local, and redaction is
// too with one exception: a PEM private-key header consumes the body-shaped
// lines after it (scanner/pem.go). A part therefore ends only between two
// transcript records, before a line B where B and the two lines before it, A-1
// and A, each start with '{' and end with '}'. A '{' line is never body-shaped
// and never continues an armour header, so no block opened above A runs past A.
// A header ON line A could still close on an END marker on B, so before a part
// is redacted its last line is scanned on its own (detection being
// line-local), and a part whose last line carries a PEM header runs on to the
// next cut. With an augmenter (gitleaks) armed nothing is split: an augmenter
// reads the whole text and may match across lines, so the transcript is one
// part, as it was before parts existed. A transcript with no safe cut (not
// line-delimited JSON) is one part too.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/sessionkind"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// redactionPartBytes is the size a part aims at: about a fifth of a second of
// redaction on an idle machine. A part is whole transcript lines, so one line
// longer than this (a pasted file, an image) is a part of its own size.
var redactionPartBytes = 256 << 10

// progressDirName is the directory under staging/ holding the parts of the
// transcripts a drain left part-redacted. listStaged skips directories, so it
// is never read as a staged transcript.
const progressDirName = "progress"

// partsDirSuffix ends the name of every per-transcript parts directory.
const partsDirSuffix = ".parts"

// partSuffix ends every part's file name.
const partSuffix = ".part"

// partSchema is the part header's schema version. A part of any other version
// is not reused.
const partSchema = 2

// maxPartBytes bounds one part file read back: a part is at most the whole
// framed transcript, and redaction's placeholders can lengthen a line.
const maxPartBytes = 4 * maxTranscriptBytes

// minLockWait is the least a drain under a deadline waits for the records
// lock once its parts are done, so a pass that finished its last part right at
// the deadline can still store it.
const minLockWait = 250 * time.Millisecond

// partHeader is the first line of a part file; the redacted text follows it.
type partHeader struct {
	Schema int `json:"schema_version"`
	Start  int `json:"start"`
	End    int `json:"end"`
	// AtEnd is true when the part ended at the end of the framed text as the
	// text then stood. Such a part is reused only while the text is exactly
	// that long.
	AtEnd bool `json:"at_end"`
	// Scanner is partsScanner's fingerprint of the scanner the part was
	// redacted with; a part is reused only under the same one.
	Scanner string `json:"scanner"`
	// SHA256 is the sha256 of the input bytes the part covers.
	SHA256 string `json:"sha256"`
	// TextSHA256 is the sha256 of the redacted text the part holds, so a part
	// cut short on disk is not taken for a whole one.
	TextSHA256 string `json:"text_sha256"`
	Secrets    int    `json:"secrets"`
	HomePaths  int    `json:"home_paths"`
	ScanGap    string `json:"scan_gap,omitempty"`
}

// redactionPart is one finished part: the header and its redacted text.
type redactionPart struct {
	partHeader
	Text string
}

// DrainProgress is one staged transcript a drain pass left part-redacted.
type DrainProgress struct {
	SessionID string `json:"session_id"`
	AgentID   string `json:"agent_id,omitempty"`
	// Done and Total are bytes of the framed transcript: redacted and kept,
	// where the next pass continues from, and in all.
	Done  int64 `json:"done_bytes"`
	Total int64 `json:"total_bytes"`
}

// stagedOutcome is what one staged entry's capture came to. keepFault is the
// first fault keeping a part met, which the pass carried on past.
type stagedOutcome struct {
	result    CaptureResult
	done      bool
	progress  DrainProgress
	keepFault error
}

// recordShaped reports whether one transcript line is a whole JSON object on
// its own line: '{' first and '}' last, a trailing CR aside.
func recordShaped(line string) bool {
	line = strings.TrimSuffix(line, "\r")
	return len(line) >= 2 && line[0] == '{' && line[len(line)-1] == '}'
}

// lineBefore returns the line that ends at end-1 (text[end-1] is its '\n').
func lineBefore(text string, end int) (line string, start int) {
	start = strings.LastIndexByte(text[:end-1], '\n') + 1
	return text[start : end-1], start
}

// safeCut reports whether a part may end at byte at: just after a newline,
// before a record-shaped line, with the two lines before it record-shaped too.
func safeCut(text string, at int) bool {
	if at <= 0 || at >= len(text) || text[at-1] != '\n' {
		return false
	}
	next := text[at:]
	if i := strings.IndexByte(next, '\n'); i >= 0 {
		next = next[:i]
	}
	if !recordShaped(next) {
		return false
	}
	a, aStart := lineBefore(text, at)
	if !recordShaped(a) || aStart == 0 {
		return false
	}
	prev, _ := lineBefore(text, aStart)
	return recordShaped(prev)
}

// nextCut returns where the part starting at start ends: the end of text when
// the rest fits in target bytes, else the last safe cut within target bytes,
// else the first safe cut past it, else the end of text.
func nextCut(text string, start, target int) int {
	if len(text)-start <= target {
		return len(text)
	}
	best := -1
	for pos := start; pos < len(text); {
		i := strings.IndexByte(text[pos:], '\n')
		if i < 0 {
			break
		}
		c := pos + i + 1
		if c >= len(text) {
			break
		}
		if safeCut(text, c) {
			if c-start <= target {
				best = c
			} else {
				if best < 0 {
					best = c
				}
				break
			}
		}
		pos = c
	}
	if best < 0 {
		return len(text)
	}
	return best
}

// pemHeaderOn reports whether line, scanned on its own, carries a PEM
// private-key header: the one finding whose block can run past the line it is
// on. Detection is line-local, so this is the finding a scan of the whole part
// reports on that line.
func pemHeaderOn(sc *scanner.Scanner, line string) bool {
	for _, f := range sc.ScanText(line, "transcript") {
		if strings.HasSuffix(f.Kind, "pem_private_key") {
			return true
		}
	}
	return false
}

// partsScanner is the fingerprint a part is stamped with: the scanner's own
// (scanner.Fingerprint), the caller's home that the literal home-path sweep
// collapses, and the running binary (binaryStamp), whose code decides what the
// scanner's inputs produce. A part made under any other is not reused.
func partsScanner(sc *scanner.Scanner) string {
	var b strings.Builder
	b.WriteString(sc.Fingerprint())
	b.WriteByte(0)
	b.WriteString(scanner.CallerHome())
	b.WriteByte(0)
	b.WriteString(binaryStamp())
	return digest(b.String())
}

// binaryStamp names the running binary in partsScanner: the sha256 of the
// executable's bytes, computed once per process, and only by a pass that has
// parts to read or write. A test replaces it.
var binaryStamp = sync.OnceValue(func() string {
	exe, err := os.Executable()
	if err != nil {
		return processStamp()
	}
	return executableStamp(exe, exeAtStart)
})

// exeAtStart is the executable file as the process found it when it started,
// so a binary swapped in at the same path afterwards (an update) is not taken
// for the one running. nil when it could not be read.
var exeAtStart = func() os.FileInfo {
	exe, err := os.Executable()
	if err != nil {
		return nil
	}
	fi, err := os.Stat(exe)
	if err != nil {
		return nil
	}
	return fi
}()

// executableStamp is the sha256 of the file at path, when it is still the file
// atStart describes; otherwise, or when it cannot be read, a stamp unique to
// this process (processStamp), so nothing kept by another process is reused
// and nothing this one keeps is reused by another.
func executableStamp(path string, atStart os.FileInfo) string {
	f, err := os.Open(path)
	if err != nil {
		return processStamp()
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || atStart == nil || !os.SameFile(fi, atStart) ||
		fi.Size() != atStart.Size() || !fi.ModTime().Equal(atStart.ModTime()) {
		return processStamp()
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return processStamp()
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// processStampPrefix begins every processStamp.
const processStampPrefix = "process:"

// processStamp is a stamp no other process makes.
func processStamp() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s%d:%s", processStampPrefix, os.Getpid(), hex.EncodeToString(b[:]))
}

// isProcessStamp reports whether stamp is one processStamp made: parts kept
// under it are reused by no later process.
func isProcessStamp(stamp string) bool { return strings.HasPrefix(stamp, processStampPrefix) }

// partsDirName is the name of the directory under staging/progress/ holding
// the parts of the staged file at stagedPath: the staged file's name with
// partsDirSuffix. listStaged reads only plain names ending in .raw, so the
// result is never empty, never "." or "..", and holds no separator; a name
// that is anything else gets no directory (ok false).
func partsDirName(stagedPath string) (string, bool) {
	base := filepath.Base(stagedPath)
	name := base + partsDirSuffix
	if !strings.HasSuffix(base, stagedSuffix) || strings.ContainsAny(name, `/\`) || !fsutil.ValidRelPath(name) {
		return "", false
	}
	return name, true
}

// stagedNameForParts is partsDirName's inverse: the staged file name a parts
// directory belongs to, ok false for a name partsDirName never makes.
func stagedNameForParts(name string) (string, bool) {
	staged, found := strings.CutSuffix(name, partsDirSuffix)
	if !found {
		return "", false
	}
	if back, ok := partsDirName(staged); !ok || back != name {
		return "", false
	}
	return staged, true
}

// openParts opens the parts directory name under sdir's progress directory,
// after proving sdir, progress and it real directories: created when create
// (fsutil.EnsureRealDirAll), probed otherwise (fsutil.ProbeRealDirAll). An
// absent directory, when not creating, is nil and no error.
func openParts(sdir, name string, create bool) (*os.Root, error) {
	rel := progressDirName + "/" + name
	if create {
		if err := fsutil.EnsureRealDirAll(sdir, rel, storeDirPerm); err != nil {
			return nil, err
		}
	} else if ok, err := fsutil.ProbeRealDirAll(sdir, rel); !ok {
		return nil, err
	}
	return fsutil.OpenRealDir(filepath.Join(sdir, progressDirName, name))
}

// openProgress opens sdir's progress directory after proving sdir and it real
// directories; absent, it is nil and no error.
func openProgress(sdir string) (*os.Root, error) {
	if ok, err := fsutil.ProbeRealDirAll(sdir, progressDirName); !ok {
		return nil, err
	}
	return fsutil.OpenRealDir(filepath.Join(sdir, progressDirName))
}

func partName(start int) string { return fmt.Sprintf("%016d%s", start, partSuffix) }

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// loadParts reads the parts kept for the staged file at stagedPath and returns
// the run of them that covers text from byte zero without a gap, each one
// proved against the bytes it covers, the end of text and the scanner
// fingerprint fp gives (see WHEN A KEPT PART IS REUSED); fp is called only
// when there is a part to prove. Every other part file is removed. A parts
// directory that is not a real directory is not read.
func loadParts(sdir, stagedPath, text string, fp func() string) []redactionPart {
	name, ok := partsDirName(stagedPath)
	if !ok {
		return nil
	}
	root, err := openParts(sdir, name, false)
	if err != nil || root == nil {
		return nil
	}
	defer root.Close()
	dir, err := root.Open(".")
	if err != nil {
		return nil
	}
	entries, err := dir.ReadDir(-1)
	_ = dir.Close()
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasSuffix(e.Name(), partSuffix) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var parts []redactionPart
	offset := 0
	for i, n := range names {
		p, ok := readPart(root, n)
		if !ok || !p.reusable(text, offset, fp()) {
			for _, stale := range names[i:] {
				_ = root.Remove(stale)
			}
			break
		}
		parts = append(parts, p)
		offset = p.End
	}
	return parts
}

// reusable reports whether p may stand for text[offset:p.End] in a record of
// text redacted under fp.
func (p redactionPart) reusable(text string, offset int, fp string) bool {
	if p.Schema != partSchema || p.Scanner != fp || p.Start != offset || p.End <= p.Start || p.End > len(text) {
		return false
	}
	if p.AtEnd {
		if p.End != len(text) {
			return false
		}
	} else if p.End == len(text) || !safeCut(text, p.End) {
		return false
	}
	return p.SHA256 == digest(text[p.Start:p.End]) && p.TextSHA256 == digest(p.Text)
}

func readPart(root *os.Root, name string) (redactionPart, bool) {
	data, err := fsutil.ReadGuardedInRoot(root, name, maxPartBytes)
	if err != nil {
		return redactionPart{}, false
	}
	head, text, ok := bytes.Cut(data, []byte("\n"))
	if !ok {
		return redactionPart{}, false
	}
	var h partHeader
	if err := json.Unmarshal(head, &h); err != nil {
		return redactionPart{}, false
	}
	return redactionPart{partHeader: h, Text: string(text)}, true
}

// writePart keeps one finished part of the staged file at stagedPath,
// atomically, inside its parts directory proved real first.
func writePart(sdir, stagedPath string, p redactionPart) error {
	name, ok := partsDirName(stagedPath)
	if !ok {
		return fmt.Errorf("history: no parts directory for staged name %q", filepath.Base(stagedPath))
	}
	root, err := openParts(sdir, name, true)
	if err != nil {
		return err
	}
	defer root.Close()
	head, err := json.Marshal(p.partHeader)
	if err != nil {
		return err
	}
	data := make([]byte, 0, len(head)+1+len(p.Text))
	data = append(append(append(data, head...), '\n'), p.Text...)
	return fsutil.WriteFileAtomicInRoot(root, partName(p.Start), data, recordPerm)
}

// removeProgress removes the parts kept for one staged file. Only a real
// parts directory under a real progress directory is removed, and the removal
// never follows a link.
func removeProgress(sdir, stagedPath string) {
	name, ok := partsDirName(stagedPath)
	if !ok {
		return
	}
	removeParts(sdir, name)
}

func removeParts(sdir, name string) {
	if ok, err := fsutil.ProbeRealDirAll(sdir, progressDirName+"/"+name); !ok || err != nil {
		return
	}
	root, err := openProgress(sdir)
	if err != nil || root == nil {
		return
	}
	defer root.Close()
	_ = root.RemoveAll(name)
}

// sweepProgress removes the parts of every staged file that is gone. It reads
// the progress directory, not a listing of staged files taken earlier, and
// judges each parts directory under the staging lock of the file it belongs
// to, which Stage holds while it writes that file: a file staged at any point
// keeps its parts. The lock is tried once, not waited for: a parts directory
// whose lock someone holds is left for a later sweep. The lock file is retired
// once the parts are gone, as the drain retires it once a staged file is. The
// sweep stops when ctx ends. Anything in the progress directory that is not a
// parts directory partsDirName makes, and a real directory, is left alone.
func sweepProgress(ctx context.Context, sdir string) {
	if ctx.Err() != nil {
		return
	}
	root, err := openProgress(sdir)
	if err != nil || root == nil {
		return
	}
	dir, err := root.Open(".")
	if err != nil {
		root.Close()
		return
	}
	entries, err := dir.ReadDir(-1)
	_ = dir.Close()
	root.Close()
	if err != nil {
		return
	}
	locks := filepath.Join(sdir, stagingLocksDirName)
	for _, e := range entries {
		if ctx.Err() != nil {
			return
		}
		staged, ok := stagedNameForParts(e.Name())
		if !ok || !e.IsDir() {
			continue
		}
		stagedPath := filepath.Join(sdir, staged)
		if _, err := os.Lstat(stagedPath); !errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err := fsutil.EnsureRealDir(locks, storeDirPerm); err != nil {
			return
		}
		key := stagedLockKey(stagedPath)
		_ = fsutil.WithFileLock(stagingLockPath(sdir, key), 0, func() error {
			if _, err := os.Lstat(stagedPath); !errors.Is(err, os.ErrNotExist) {
				return nil
			}
			removeParts(sdir, e.Name())
			retireStagingLock(sdir, key)
			return nil
		})
	}
}

// captureStaged is one staged entry's capture under ctx: Capture's checks and
// its two-stage redaction, done in parts that are kept as they finish, then the
// record written from them. Its first part always runs; it returns done=false
// with the progress kept when ctx ended before a later part, or when the
// records lock could not be had before ctx's deadline, each only while what it
// kept can be picked up by a later pass (see WHAT A PASS DOES WITH ONE ENTRY).
// A redaction refusal is returned as Capture returns it.
func captureStaged(ctx context.Context, repoRoot, rootSHA, sdir string, s Staged, body []byte) (stagedOutcome, error) {
	meta, tdir, err := prepareCapture(repoRoot, rootSHA, s.captureMeta())
	if err != nil {
		return stagedOutcome{}, err
	}
	sourceSHA := sourceDigest(body)
	// A transcript the store already holds needs no redaction at all; the
	// check is repeated under the lock before anything is written.
	if existing, err := listRecords(tdir); err == nil {
		if r, ok := alreadyStored(existing, sourceSHA, meta); ok {
			return stagedOutcome{result: CaptureResult{Record: r}, done: true}, nil
		}
	}
	text, err := frameLineage(meta, body)
	if err != nil {
		return stagedOutcome{}, err
	}
	// The scanner is built before a kept part is looked at, because a part is
	// trusted only under the scanner it was redacted with. The build is not
	// bounded by ctx: the first part runs whatever the deadline, as the whole
	// transcript did before parts existed, and a probe cut short would refuse it.
	sc, err := redactionScanner(context.WithoutCancel(ctx), repoRoot)
	if err != nil {
		return stagedOutcome{}, err
	}
	whole := sc.Augmented()
	// The fingerprint, and with it the binary's digest, is computed only when
	// there is a part to prove or keep.
	fp := sync.OnceValue(func() string { return partsScanner(sc) })
	var parts []redactionPart
	if whole {
		// The scanner reads the text whole, so nothing kept is used: the
		// pass starts from byte zero.
		removeProgress(sdir, s.Path)
	} else {
		parts = loadParts(sdir, s.Path, text, fp)
	}
	offset := 0
	if n := len(parts); n > 0 {
		offset = parts[n-1].End
	}
	// kept is how far the parts on disk reach, which is where the next pass
	// continues; keepFault is the first fault keeping a part met, after which
	// the pass keeps nothing more and carries on in memory.
	kept := offset
	var keepFault error
	keep := func(p redactionPart) {
		if keepFault != nil {
			return
		}
		p.Scanner = fp()
		if err := writePart(sdir, s.Path, p); err != nil {
			keepFault = fmt.Errorf("history: keep a redacted part: %w", err)
			return
		}
		kept = p.End
	}
	unfinished := func() (stagedOutcome, error) {
		return stagedOutcome{progress: DrainProgress{SessionID: s.SessionID, AgentID: s.AgentID,
			Done: int64(kept), Total: int64(len(text))}, keepFault: keepFault}, nil
	}
	// resumable reports whether what this pass keeps can be picked up by a
	// later pass: the text is split, no part has failed to keep, and the
	// binary's stamp is one another process makes too. When it cannot,
	// stopping saves nothing, so nothing stops the entry short of its record.
	resumable := func() bool { return !whole && keepFault == nil && !isProcessStamp(binaryStamp()) }
	deadline, bounded := ctx.Deadline()
	// ended reports whether a later part is not started because ctx ended.
	ended := func() bool { return ctx.Err() != nil && resumable() }
	for first := true; offset < len(text); first = false {
		if !first && ended() {
			return unfinished()
		}
		cut := len(text)
		if !whole {
			cut = nextCut(text, offset, redactionPartBytes)
			// A part whose last line opens a PEM block runs on to the next
			// cut. Only the new last line is scanned each round.
			for cut < len(text) {
				if line, _ := lineBefore(text, cut); !pemHeaderOn(sc, line) {
					break
				}
				if !first && ended() {
					return unfinished()
				}
				cut = nextCut(text, cut, redactionPartBytes)
			}
		}
		red, err := redactPart(sc, text[offset:cut], "transcript")
		if err != nil {
			var resid *RedactionResidualError
			if errors.As(err, &resid) {
				// The same bytes are refused by every pass, so the entry
				// goes to quarantine and its parts are of no further use.
				removeProgress(sdir, s.Path)
				return stagedOutcome{result: CaptureResult{Residual: resid.Residual}}, err
			}
			return stagedOutcome{}, err
		}
		secrets, homePaths := countBuckets(red.Findings)
		p := redactionPart{partHeader: partHeader{Schema: partSchema, Start: offset, End: cut, AtEnd: cut == len(text),
			SHA256: digest(text[offset:cut]), TextSHA256: digest(red.Text),
			Secrets: secrets, HomePaths: homePaths, ScanGap: red.ScanGap}, Text: red.Text}
		// A transcript redacted whole in its first part goes straight to
		// the record; any other part is kept before the next one starts.
		if offset > 0 || cut < len(text) {
			keep(p)
			drainStep(stepPartKept, cut)
		}
		parts = append(parts, p)
		offset = cut
	}

	var red redactedText
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p.Text)
		red.Secrets += p.Secrets
		red.HomePaths += p.HomePaths
		if red.ScanGap == "" {
			red.ScanGap = p.ScanGap
		}
	}
	red.Text = b.String()
	drainStep(stepPartsDone, offset)
	wait := repoLockTimeout
	if bounded {
		wait = min(wait, max(time.Until(deadline), minLockWait))
	}
	var res CaptureResult
	commit := func() error {
		existing, err := listRecords(tdir)
		if err != nil {
			return err
		}
		if r, ok := alreadyStored(existing, sourceSHA, meta); ok {
			res = CaptureResult{Record: r}
			return nil
		}
		res, err = commitRedacted(rootSHA, tdir, existing, meta, sourceSHA, sessionkind.Find(body), red)
		return err
	}
	lockFrom := time.Now()
	err = withRepoLockFor(tdir, wait, commit)
	if bounded && errors.Is(err, fsutil.ErrLockContention) {
		// The redaction is done and only the lock was missing: a transcript
		// redacted whole in one part keeps that part now, so the next pass
		// commits it without redacting it again.
		if n := len(parts); n == 1 && kept == 0 && parts[0].AtEnd && resumable() {
			keep(parts[0])
		}
		if resumable() {
			return unfinished()
		}
		// Nothing this pass did can be picked up by a later one, so it waits
		// out the rest of the wait a capture has for the lock.
		err = withRepoLockFor(tdir, max(repoLockTimeout-time.Since(lockFrom), 0), commit)
	}
	if err != nil {
		return stagedOutcome{result: res, keepFault: keepFault}, err
	}
	drainStep(stepRecordWritten, offset)
	return stagedOutcome{result: res, done: true, keepFault: keepFault}, nil
}

// The step boundaries of one entry's drain, in order, at which drainStep is
// called. A test replaces drainStep to stop a pass at one of them, the way a
// kill would.
const (
	stepPartKept      = "part kept"      // a part is on disk; the next is not started
	stepPartsDone     = "parts done"     // every part is redacted; the record is not written
	stepRecordWritten = "record written" // the record is on disk; the staged copy is not removed
	stepStagedRemoved = "staged removed" // the staged copy is gone; its parts are not removed
)

// redactPart is the two-stage redaction of one part; a test counts it.
var redactPart = redactWith

// drainStep is called at each step boundary with the step and the bytes of
// the framed text covered so far.
var drainStep = func(step string, covered int) {}
