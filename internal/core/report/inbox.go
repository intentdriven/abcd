package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// inboxRelPath is the inbox's root relative to the caller's home.
const inboxRelPath = ".abcd/inbox"

// Names inside the inbox.
const (
	promotedDirName = "promoted"
	promotedLogName = "promoted.jsonl"
	lockFileName    = ".lock"
)

// storeDirPerm and fileMode keep the inbox the account's own business.
const (
	storeDirPerm = 0o700
	fileMode     = 0o600
)

// lockTimeout bounds how long a promotion waits for another one.
const lockTimeout = 3 * time.Second

// idFamily is the report id's family tag: a report is `rpt-<stamp>`, where the
// stamp is the shared record-id mint's time-ordered sixteen digits (adr-45).
// The id is an inbox handle, not a record in any repository.
const idFamily = "rpt"

// GenericSender is the description that stands in for a sender's name in
// anything abcd files from a report (decision 4: identity in the inbox,
// fingerprint in the record).
const GenericSender = "a managed repository"

// The states an inbox entry can be in.
const (
	// StateWaiting is a report nobody has acted on.
	StateWaiting = "waiting"
	// StatePromoted is a report filed as a capture; it is kept, not deleted.
	StatePromoted = "promoted"
	// StateUnreadable is a waiting file this abcd cannot read: a later template
	// version, or bytes that are not a report. It is listed, never dropped.
	StateUnreadable = "unreadable"
)

// now is the clock the id and the received stamp read. Tests pin it.
var now = time.Now

// fileNameRe is a report's file name: the received stamp and the sender key.
var fileNameRe = regexp.MustCompile(`^([0-9]{16})-([0-9a-f]{40}|[0-9a-f]{64})\.md$`)

// idRe is a report id as a caller names it.
var idRe = regexp.MustCompile(`^rpt-([0-9]{16})$`)

// Sender is the repository a report comes from: its root-commit key, which is
// what anything committed carries, and its name, which only the inbox shows.
type Sender struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// SenderOf resolves the sender for the checkout at root: the full root-commit
// SHA, and the name of the repository's main checkout directory (a worktree
// reports under its repository's name, not its own directory's). A repository
// with no commit has no key and cannot report.
func SenderOf(root string) (Sender, error) {
	key := gitutil.RootCommit(root)
	if !gitutil.IsFullSHA(key) {
		return Sender{}, fmt.Errorf("%w: the repository has no root commit to key the report on (a repository with no commits cannot report)", ErrRefused)
	}
	return Sender{Key: key, Name: repoName(root)}, nil
}

// repoName is the main checkout's directory name, held to senderNameRe.
func repoName(root string) string {
	dir := root
	if common, err := gitutil.Run(root, "rev-parse", "--path-format=absolute", "--git-common-dir"); err == nil && common != "" {
		if filepath.Base(common) == ".git" {
			dir = filepath.Dir(common)
		}
	}
	name := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			return r
		}
		return '-'
	}, filepath.Base(dir))
	name = strings.TrimLeft(name, ".-_")
	if len(name) > 64 {
		name = name[:64]
	}
	if !senderNameRe.MatchString(name) {
		return "unnamed"
	}
	return name
}

// inboxDir resolves the inbox under the caller's home without touching disk.
func inboxDir() (home, dir string, err error) {
	home, err = os.UserHomeDir()
	if err != nil || home == "" {
		return "", "", fmt.Errorf("cannot resolve the caller's home directory: %v", err)
	}
	return home, filepath.Join(home, filepath.FromSlash(inboxRelPath)), nil
}

// ensureInbox creates the inbox and its promoted folder, one real directory at
// a time and never through a symlink.
func ensureInbox() (string, error) {
	home, dir, err := inboxDir()
	if err != nil {
		return "", err
	}
	if err := fsutil.EnsureRealDirAll(home, inboxRelPath+"/"+promotedDirName, storeDirPerm); err != nil {
		return "", fmt.Errorf("cannot create the inbox: %w", err)
	}
	return dir, nil
}

// peekInbox returns the inbox directory, or "" when it does not exist yet. A
// path occupied by anything but a real directory is refused.
func peekInbox() (string, error) {
	_, dir, err := inboxDir()
	if err != nil {
		return "", err
	}
	if fsutil.IsRealDir(dir) {
		return dir, nil
	}
	if ok, _ := fsutil.ExistsNoFollow(dir); ok {
		return "", fmt.Errorf("the inbox path is not a real directory (a symlink or a file occupies it); refusing")
	}
	return "", nil
}

// Filed is where a report landed.
type Filed struct {
	ID string `json:"id"`
	// Path is the report's location with the home directory written as `~`, so
	// no output carries the account's absolute path.
	Path string `json:"path"`
}

// File stamps a parsed report with its sender and the received time and writes
// it into the inbox by an exclusive create. The file holds what Parse accepted,
// written back by abcd, so nothing the reporter wrote chooses a byte of its name
// or reaches the file unparsed.
func File(r Report, s Sender) (Filed, error) {
	if !senderKeyRe.MatchString(s.Key) {
		return Filed{}, fmt.Errorf("%w: the sender has no root-commit key", ErrRefused)
	}
	if !senderNameRe.MatchString(s.Name) {
		s.Name = "unnamed"
	}
	dir, err := ensureInbox()
	if err != nil {
		return Filed{}, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return Filed{}, err
	}
	defer root.Close()
	minter := recordid.Minter{Now: now}
	for range 8 {
		id, err := minter.Mint(idFamily)
		if err != nil {
			return Filed{}, err
		}
		stamp := strings.TrimPrefix(id, idFamily+"-")
		r.ReceivedAt = now().UTC().Truncate(time.Second).Format(time.RFC3339)
		r.SenderKey, r.SenderName = s.Key, s.Name
		name := stamp + "-" + s.Key + ".md"
		err = fsutil.CreateExclusiveIn(root, name, serialize(r), fileMode)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return Filed{}, fmt.Errorf("cannot file the report: %w", err)
		}
		return Filed{ID: id, Path: "~/" + inboxRelPath + "/" + name}, nil
	}
	return Filed{}, fmt.Errorf("cannot file the report: every id drawn this second is taken; try again")
}

// Entry is one report as the inbox shows it.
type Entry struct {
	ID         string `json:"id"`
	State      string `json:"state"`
	ReceivedAt string `json:"received_at"`
	SenderKey  string `json:"sender_key"`
	// SenderName is the sender's name, shown plainly here and nowhere abcd
	// commits anything.
	SenderName string `json:"sender_name,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Severity   string `json:"severity,omitempty"`
	Title      string `json:"title,omitempty"`
	// Unreadable says why a waiting file could not be read, naming the template
	// version when that is the reason.
	Unreadable string `json:"unreadable,omitempty"`
	// PromotedTo is the capture a promoted report became.
	PromotedTo string `json:"promoted_to,omitempty"`
	// Report is the whole report, on Show only.
	Report *Report `json:"report,omitempty"`
	// written orders two reports received in the same second, which the
	// second-grained stamp cannot.
	written time.Time
}

// readEntry reads one report file inside root.
func readEntry(root *os.Root, rel, name, state string) Entry {
	m := fileNameRe.FindStringSubmatch(name)
	e := Entry{ID: idFamily + "-" + m[1], State: state, SenderKey: m[2], ReceivedAt: stampTime(m[1])}
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxFiledBytes)
	if err != nil {
		e.State = StateUnreadable
		switch {
		case errors.Is(err, fsutil.ErrTooBig):
			e.Unreadable = fmt.Sprintf("larger than the %d-byte bound a filed report is held to", maxFiledBytes)
		default:
			e.Unreadable = "not a regular file abcd can read"
		}
		return e
	}
	r, err := parseFiled(data)
	if err != nil {
		e.State = StateUnreadable
		var ve *VersionError
		var fe *FieldError
		switch {
		case errors.As(err, &ve):
			e.Unreadable = ve.Error()
		case errors.As(err, &fe):
			e.Unreadable = "not a report this abcd can read: " + fe.Error()
		default:
			e.Unreadable = "not a report this abcd can read"
		}
		return e
	}
	if r.SenderKey != e.SenderKey {
		e.State = StateUnreadable
		e.Unreadable = "its sender key does not match its file name"
		return e
	}
	e.SenderName, e.Kind, e.Severity, e.Title = r.SenderName, r.Kind, r.Severity, r.Title
	e.ReceivedAt = r.ReceivedAt
	e.Report = &r
	return e
}

// stampTime renders the leading twelve digits of a received stamp as RFC 3339.
func stampTime(stamp string) string {
	t, err := time.Parse("060102150405", stamp[:12])
	if err != nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// scan reads every report file in one folder of the inbox.
func scan(root *os.Root, sub, state string) ([]Entry, error) {
	dirRel := "."
	if sub != "" {
		dirRel = sub
	}
	f, err := root.Open(dirRel)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, name := range names {
		if !fileNameRe.MatchString(name) {
			continue
		}
		rel := name
		if sub != "" {
			rel = path.Join(sub, name)
		}
		e := readEntry(root, rel, name, state)
		if info, err := root.Lstat(rel); err == nil {
			e.written = info.ModTime()
		}
		out = append(out, e)
	}
	return out, nil
}

// newestFirst orders entries newest first: by the received second in the id,
// then, within one second, by when the file was written, then by id.
func newestFirst(es []Entry) {
	slices.SortFunc(es, func(a, b Entry) int {
		if c := strings.Compare(b.ID[:len(idFamily)+13], a.ID[:len(idFamily)+13]); c != 0 {
			return c
		}
		if c := b.written.Compare(a.written); c != 0 {
			return c
		}
		return strings.Compare(b.ID, a.ID)
	})
}

// List returns the waiting reports, the unreadable ones among them, newest
// first. It reads and never writes: an absent inbox is an empty list, and
// nothing is filed.
func List() ([]Entry, error) {
	dir, err := peekInbox()
	if err != nil || dir == "" {
		return []Entry{}, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	out, err := scan(root, "", StateWaiting)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Report = nil
	}
	newestFirst(out)
	if out == nil {
		out = []Entry{}
	}
	return out, nil
}

// Tally is the greeting's two numbers.
type Tally struct {
	Reports int `json:"reports"`
	Senders int `json:"senders"`
}

// Count returns how many reports wait and from how many repositories. It reads
// file names only, so the greeting costs one directory listing however large
// the reports are, and an unreadable report counts: it waits like any other.
func Count() (Tally, error) {
	dir, err := peekInbox()
	if err != nil || dir == "" {
		return Tally{}, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Tally{}, err
	}
	senders := map[string]bool{}
	var t Tally
	for _, e := range entries {
		m := fileNameRe.FindStringSubmatch(e.Name())
		if m == nil || e.IsDir() {
			continue
		}
		t.Reports++
		senders[m[2]] = true
	}
	t.Senders = len(senders)
	return t, nil
}

// Show returns one report, waiting or promoted, whole. The id is checked for
// its shape before anything is read, and it is only ever compared with the
// names already in the inbox: it never becomes a path.
func Show(id string) (Entry, error) {
	m := idRe.FindStringSubmatch(id)
	if m == nil {
		return Entry{}, fmt.Errorf("%w: %q is not a report id (rpt- and sixteen digits)", ErrRefused, id)
	}
	dir, err := peekInbox()
	if err != nil {
		return Entry{}, err
	}
	if dir == "" {
		return Entry{}, fmt.Errorf("%w: no report %s in the inbox", ErrRefused, id)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return Entry{}, err
	}
	defer root.Close()
	for _, where := range []struct{ sub, state string }{{"", StateWaiting}, {promotedDirName, StatePromoted}} {
		es, err := scan(root, where.sub, where.state)
		if err != nil {
			return Entry{}, err
		}
		for _, e := range es {
			if e.ID != id {
				continue
			}
			if e.State == StatePromoted {
				e.PromotedTo = promotedTo(root, id)
			}
			return e, nil
		}
	}
	return Entry{}, fmt.Errorf("%w: no report %s in the inbox", ErrRefused, id)
}

// promotion is one line of the promoted log.
type promotion struct {
	Report  string `json:"report"`
	Capture string `json:"capture"`
	At      string `json:"at"`
}

// promotedTo reads the capture id a report became from the promoted log.
func promotedTo(root *os.Root, id string) string {
	data, err := fsutil.ReadGuardedInRoot(root, promotedLogName, 4<<20)
	if err != nil {
		return ""
	}
	var to string
	for _, line := range strings.Split(string(data), "\n") {
		var p promotion
		if json.Unmarshal([]byte(line), &p) == nil && p.Report == id {
			to = p.Capture
		}
	}
	return to
}

// Promoted is what a promotion filed.
type Promoted struct {
	Report  string `json:"report"`
	Capture string `json:"capture"`
	// Path is the capture's repository-relative path.
	Path string `json:"path"`
	// Redacted and Degraded are the capture redactor's own report, relayed so
	// the person who promoted can see the text was altered.
	Redacted int    `json:"redacted,omitempty"`
	Degraded string `json:"redaction_degraded,omitempty"`
}

// Source is the capture source a promoted report is filed under.
const Source = "managed-repo"

// Promote files a waiting report as a capture in the ledger of the repository
// at ledgerRoot, through capture's own core, so the promoted report is an
// ordinary ledger record, redacted by the ledger's own scanner before it is
// written. The capture carries the sender's root-commit key and GenericSender,
// never the sender's name, and the report's id as its evidence. The report is
// then moved to the promoted folder and kept. Promotions are serialised by the
// inbox's lock, so two sessions cannot file one report twice.
func Promote(ledgerRoot, id string) (Promoted, error) {
	m := idRe.FindStringSubmatch(id)
	if m == nil {
		return Promoted{}, fmt.Errorf("%w: %q is not a report id (rpt- and sixteen digits)", ErrRefused, id)
	}
	dir, err := peekInbox()
	if err != nil {
		return Promoted{}, err
	}
	if dir == "" {
		return Promoted{}, fmt.Errorf("%w: no report %s in the inbox", ErrRefused, id)
	}
	if dir, err = ensureInbox(); err != nil {
		return Promoted{}, err
	}
	var out Promoted
	err = fsutil.WithFileLock(filepath.Join(dir, lockFileName), lockTimeout, func() error {
		root, err := os.OpenRoot(dir)
		if err != nil {
			return err
		}
		defer root.Close()
		waiting, err := scan(root, "", StateWaiting)
		if err != nil {
			return err
		}
		var e *Entry
		for i := range waiting {
			if waiting[i].ID == id {
				e = &waiting[i]
			}
		}
		if e == nil {
			if to := promotedTo(root, id); to != "" {
				return fmt.Errorf("%w: report %s was already promoted to %s", ErrRefused, id, to)
			}
			return fmt.Errorf("%w: no waiting report %s in the inbox", ErrRefused, id)
		}
		if e.State == StateUnreadable {
			return fmt.Errorf("%w: report %s is unreadable (%s) and cannot be promoted", ErrRefused, id, e.Unreadable)
		}
		req := captureRequest(ledgerRoot, id, *e.Report)
		res, err := capture.Capture(req)
		if err != nil {
			return fmt.Errorf("the capture was refused, and the report still waits: %w", err)
		}
		out = Promoted{Report: id, Capture: res.ID, Path: res.Path, Redacted: res.Redacted, Degraded: res.Degraded}
		name := m[1] + "-" + e.SenderKey + ".md"
		if err := root.Rename(name, path.Join(promotedDirName, name)); err != nil {
			return fmt.Errorf("filed %s, but the report could not be marked promoted: %w", res.ID, err)
		}
		line, err := json.Marshal(promotion{Report: id, Capture: res.ID, At: now().UTC().Format(time.RFC3339)})
		if err != nil {
			return err
		}
		return fsutil.AppendLineIn(root, promotedLogName, line, fileMode)
	})
	if errors.Is(err, fsutil.ErrLockContention) {
		return Promoted{}, fmt.Errorf("%w: another promotion holds the inbox; retry", ErrRefused)
	}
	return out, err
}

// captureRequest composes the capture a report becomes. Every free-text value
// has the sender's name replaced by GenericSender, whatever its case, so the
// name reaches no file abcd writes into a repository (criterion 7).
func captureRequest(ledgerRoot, id string, r Report) capture.CaptureRequest {
	scrub := nameScrubber(r.SenderName)
	var b strings.Builder
	b.WriteString(scrub(r.Title))
	b.WriteString("\n\n")
	b.WriteString(scrub(r.Prose))
	b.WriteString("\n")
	if r.Remedy != "" {
		fmt.Fprintf(&b, "\nRemedy the reporter proposes: %s\n", scrub(r.Remedy))
	}
	fmt.Fprintf(&b, "\nReported by %s (root commit %s) through the abcd inbox as %s, a %s against abcd %s, surface %s.\n",
		GenericSender, r.SenderKey, id, r.Kind, scrub(r.AbcdVersion), scrub(r.Surface))
	fmt.Fprintf(&b, "\nEvidence:\n\n- %s (the report, kept in the inbox)\n", id)
	for _, e := range r.Evidence {
		fmt.Fprintf(&b, "- %s\n", scrub(e))
	}
	return capture.CaptureRequest{
		RepoRoot:    ledgerRoot,
		Text:        b.String(),
		Severity:    capture.Severity(r.Severity),
		Category:    capture.Category(r.Category),
		Source:      capture.Source(Source),
		FoundDuring: fmt.Sprintf("abcd inbox report %s from %s (root commit %s)", id, GenericSender, r.SenderKey),
		FoundAt:     scrub(r.Surface),
	}
}

// nameScrubber returns a function replacing every case-insensitive occurrence
// of name that stands as a word — not run on into a letter or a digit — with
// GenericSender. A bare substring match would rewrite ordinary words that
// happen to contain a short name ("cap" inside "capture") without making the
// record any less identifying.
func nameScrubber(name string) func(string) string {
	if name == "" {
		return func(s string) string { return s }
	}
	re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(name))
	return func(s string) string {
		var b strings.Builder
		last := 0
		for _, m := range re.FindAllStringIndex(s, -1) {
			if wordByte(s, m[0]-1) || wordByte(s, m[1]) {
				continue
			}
			b.WriteString(s[last:m[0]])
			b.WriteString(GenericSender)
			last = m[1]
		}
		b.WriteString(s[last:])
		return b.String()
	}
}

// wordByte reports whether s[i] exists and is an ASCII letter or digit.
func wordByte(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return false
	}
	c := s[i]
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
