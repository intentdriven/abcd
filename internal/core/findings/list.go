// Package findings is the list of findings abcd holds a session and a landing
// to (itd-2611251831138387, spc-2611252115253843): one JSON file for each
// finding under ~/.abcd.noindex/findings/<root-sha>/, in open/, private/ or
// settled/. The readings that put an entry here (an agent's block, the words
// of a closing text, a gate failing at a head) are the callers'; the hold and
// the landing read this list and not the readings, so a better reader changes
// neither.
//
// Every level is made and opened through fsutil's declaration scope, which
// judges each directory's owner and mode, and every file is read through
// ReadHomeDeclarationDenyingIn denying 0o077. A file is written whole by
// rename with no lock: two writers of one finding write one name. Nothing is
// removed by age, by a new session or by the worktree going.
package findings

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// SchemaVersion is the version every entry this build writes carries. A file
// of a later version is a newer abcd's: counted and shown by its handle, never
// settled here.
const SchemaVersion = 1

// The bounds of the spec's table that the store holds. The two a reading of a
// closing text holds (its bytes and its entries) are the reader's.
const (
	// MaxFileBytes is one entry file; a larger one is unreadable, named.
	MaxFileBytes = 4 << 10
	// MaxTextBytes is an entry's text, cut on a rune past it.
	MaxTextBytes = 600
	// MaxOpen is the open entries of one root; past it nothing more is put.
	MaxOpen = 1000
	// MaxFilesRead is the files one read takes; past it the list is unreadable.
	MaxFilesRead = 1000
)

// Kind is what made an entry, and decides what clears it (clears).
type Kind string

// The closed set of kinds this build knows.
const (
	KindUntested Kind = "untested"
	KindHandedUp Kind = "handed-up"
	KindDoubt    Kind = "doubt"
	KindKnown    Kind = "known"
	KindLine     Kind = "line"
	KindCheck    Kind = "check"
	KindPrivate  Kind = "private"
)

// Act is what settles an entry. All but ActReceipt are the command's clearing
// flags; ActReceipt is a landing's preflight receipt naming the offered head.
type Act string

// The acts.
const (
	ActCaptured    Act = "captured"
	ActSettled     Act = "settled"
	ActRefuted     Act = "refuted"
	ActOwn         Act = "own"
	ActNotAFinding Act = "not-a-finding"
	ActPrivate     Act = "private"
	ActReceipt     Act = "receipt"
)

// clears is the spec's kind table: the acts that clear each kind. A private
// entry's acts are the person's alone (SettleRequest.Person).
var clears = map[Kind][]Act{
	KindUntested: {ActCaptured, ActSettled, ActRefuted, ActPrivate},
	KindHandedUp: {ActCaptured, ActSettled, ActRefuted, ActPrivate},
	KindDoubt:    {ActCaptured, ActSettled},
	KindKnown:    {ActCaptured, ActSettled, ActRefuted, ActPrivate},
	KindLine:     {ActCaptured, ActSettled, ActRefuted, ActNotAFinding, ActPrivate},
	KindCheck:    {ActCaptured, ActSettled, ActOwn, ActPrivate, ActReceipt},
	KindPrivate:  {ActCaptured, ActSettled, ActRefuted},
}

// clearedBy reports whether act clears an entry of kind, as the table has it.
func clearedBy(kind Kind, act Act) bool {
	for _, a := range clears[kind] {
		if a == act {
			return true
		}
	}
	return false
}

// The refusals. Each is wrapped with what it names.
var (
	ErrFull         = errors.New("findings: the list is full")
	ErrTooManyFiles = fmt.Errorf("findings: more than %d files; the list is unreadable", MaxFilesRead)
	ErrNewer        = errors.New("findings: the entry is a newer abcd's")
	ErrUnknownKind  = errors.New("findings: a kind this build does not know")
	ErrNotCleared   = errors.New("findings: that act does not clear the entry")
	ErrAmbiguous    = errors.New("findings: the handle names more than one entry")
	ErrNoEntry      = errors.New("findings: no entry has that handle")
	ErrUnreadable   = errors.New("findings: the entry cannot be read")
)

// redact is capture's ledger redactor, the seam a test replaces.
var redact = capture.RedactText

// Entry is one finding. Name is the file's, never stored in it.
type Entry struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          Kind   `json:"kind"`
	Session       string `json:"session,omitempty"`
	Agent         string `json:"agent,omitempty"`
	AgentType     string `json:"agent_type,omitempty"`
	// Checkout is the checkout the payload's directory is in; Place is that
	// checkout when a run's state names it as a lane's worktree, else "".
	Checkout string `json:"checkout,omitempty"`
	Place    string `json:"place,omitempty"`
	Run      string `json:"run,omitempty"`
	Lane     string `json:"lane,omitempty"`
	// Head names a gate's entry: set, the entry is named by kind, place and
	// head, so a failure at a new head is a new entry.
	Head     string      `json:"head,omitempty"`
	Text     string      `json:"text,omitempty"`
	Withheld bool        `json:"text_withheld,omitempty"`
	At       time.Time   `json:"at"`
	Settled  *Settlement `json:"settled,omitempty"`
	Name     string      `json:"-"`
}

// Handle is the entry's name as it is shown: f- and eight characters.
func (e Entry) Handle() string { return handleOf(e.Name) }

func handleOf(name string) string {
	if len(name) < 8 {
		return "f-" + name
	}
	return "f-" + name[:8]
}

// Settlement is what settled an entry: the act, the record it names, or the
// reason, test or run given.
type Settlement struct {
	Act    Act       `json:"act"`
	Record string    `json:"record,omitempty"`
	Reason string    `json:"reason,omitempty"`
	At     time.Time `json:"at"`
}

// Store is one root's list under one home.
type Store struct{ home, root string }

var rootSHARe = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// NewStore is the list of the repository whose root commit is rootSHA, under
// home (abcdhome.MachineHome in production).
func NewStore(home, rootSHA string) (Store, error) {
	if !rootSHARe.MatchString(rootSHA) {
		return Store{}, fmt.Errorf("findings: %q is not a root commit's object name", rootSHA)
	}
	return Store{home: home, root: rootSHA}, nil
}

const (
	dirOpen    = "open"
	dirPrivate = "private"
	dirSettled = "settled"
)

func (s Store) rel(sub string) string { return abcdhome.Rel("findings", s.root, sub) }

func (s Store) ensure(sub string) (*os.Root, error) {
	return fsutil.EnsureHomeDeclarationScope(s.home, s.rel(sub), abcdhome.DirMode)
}

// open opens sub for reading; an absent directory is (nil, nil).
func (s Store) open(sub string) (*os.Root, error) {
	r, err := fsutil.OpenHomeDeclarationScope(s.home, s.rel(sub))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return r, err
}

// nameOf is the entry's name: a hash of the kind, the place and the head for a
// gate's entry, and of the kind, the normalised text, the place and the
// session for a reading's.
func nameOf(e Entry) string {
	parts := []string{"reading", string(e.Kind), normalise(e.Text), e.Place, e.Session}
	if e.Head != "" {
		parts = []string{"gate", string(e.Kind), e.Place, e.Head}
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func normalise(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// clean is the stored text: capture's redactor, termsafe, the home redacted,
// cut to MaxTextBytes. withheld is true when the redactor cannot run.
func clean(checkout, text string) (out string, withheld bool) {
	red, err := redact(checkout, text)
	if err != nil {
		return "", true
	}
	out = fsutil.RedactHome(termsafe.SanitizeBlock(red))
	if len(out) > MaxTextBytes {
		out = strings.ToValidUTF8(out[:MaxTextBytes], "")
	}
	return out, false
}

func subOf(k Kind) string {
	if k == KindPrivate {
		return dirPrivate
	}
	return dirOpen
}

// Put puts e on the list and returns it as stored. An entry already settled
// stays settled and is returned so, unless a receipt settled it; one already
// listed is written again whole, under the same name.
func (s Store) Put(e Entry) (Entry, error) {
	if _, ok := clears[e.Kind]; !ok {
		return Entry{}, fmt.Errorf("%w: %q", ErrUnknownKind, e.Kind)
	}
	e.SchemaVersion = SchemaVersion
	e.Name = nameOf(e)
	e.Settled = nil
	e.Text, e.Withheld = clean(e.Checkout, e.Text)

	settled, err := s.ensure(dirSettled)
	if err != nil {
		return Entry{}, err
	}
	defer settled.Close()
	prior, err := s.readOne(settled, dirSettled, e.Name)
	switch {
	case err == nil && prior.Settled != nil && prior.Settled.Act != ActReceipt:
		return prior, nil
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return Entry{}, err
	}
	receiptSettled := err == nil

	if e.Kind != KindPrivate {
		if held, err := s.readOne(nil, dirPrivate, e.Name); err == nil {
			return held, nil
		}
	}
	dir, err := s.ensure(subOf(e.Kind))
	if err != nil {
		return Entry{}, err
	}
	defer dir.Close()
	if e.Kind != KindPrivate {
		names, err := listNames(dir, MaxOpen+1)
		if err != nil {
			return Entry{}, err
		}
		if len(names) >= MaxOpen && !contains(names, e.Name) {
			return Entry{}, fmt.Errorf("%w: %d open entries for this repository", ErrFull, len(names))
		}
	}
	if err := write(dir, e); err != nil {
		return Entry{}, err
	}
	if receiptSettled {
		// Written open first, so a crash here leaves the settled one winning
		// and the put to be made again, never neither.
		if err := settled.Remove(e.Name + ".json"); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return Entry{}, err
		}
	}
	return e, nil
}

func write(dir *os.Root, e Entry) error {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(dir, e.Name+".json", append(data, '\n'), abcdhome.FileMode)
}

func contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

var entryFileRe = regexp.MustCompile(`^([0-9a-f]{64})\.json$`)

// listNames is the entry names in dir, at most limit of them read.
func listNames(dir *os.Root, limit int) ([]string, error) {
	f, err := dir.Open(".")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ents, err := f.ReadDir(limit)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	var out []string
	for _, d := range ents {
		if m := entryFileRe.FindStringSubmatch(d.Name()); m != nil {
			out = append(out, m[1])
		}
	}
	return out, nil
}

// readOne reads one entry of sub through dir, or through a fresh open of sub
// when dir is nil. An absent file is fs.ErrNotExist; a refused one is
// ErrUnreadable naming the guard; a later schema or an unknown kind ErrNewer.
func (s Store) readOne(dir *os.Root, sub, name string) (Entry, error) {
	if dir == nil {
		d, err := s.open(sub)
		if err != nil {
			return Entry{}, err
		}
		if d == nil {
			return Entry{}, fs.ErrNotExist
		}
		defer d.Close()
		dir = d
	}
	raw, refusal, err := fsutil.ReadHomeDeclarationDenyingIn(dir, s.home, s.rel(sub)+"/"+name+".json", MaxFileBytes, 0o077)
	if refusal == fsutil.DeclarationAbsent {
		return Entry{}, fs.ErrNotExist
	}
	if err != nil {
		return Entry{}, fmt.Errorf("%w: %v", ErrUnreadable, err)
	}
	var head struct {
		SchemaVersion int  `json:"schema_version"`
		Kind          Kind `json:"kind"`
	}
	if err := json.Unmarshal(raw, &head); err != nil || head.SchemaVersion < 1 {
		return Entry{}, fmt.Errorf("%w: not an entry of any schema (%v)", ErrUnreadable, err)
	}
	if _, known := clears[head.Kind]; head.SchemaVersion > SchemaVersion || !known {
		return Entry{}, ErrNewer
	}
	var e Entry
	if err := json.Unmarshal(raw, &e); err != nil {
		return Entry{}, fmt.Errorf("%w: %v", ErrUnreadable, err)
	}
	e.Name = name
	return e, nil
}

// Unreadable is a file the read refused, by handle and why.
type Unreadable struct {
	Handle string `json:"handle"`
	Reason string `json:"reason"`
}

// View is the list as one read found it.
type View struct {
	Open       []Entry      `json:"open"`
	Private    []Entry      `json:"private"`
	Newer      []string     `json:"newer"`
	Unreadable []Unreadable `json:"unreadable"`
	// Full is true once the open entries reach MaxOpen.
	Full bool `json:"full"`
}

// Ordered is the open entries with checkout's first, each group by time.
func (v View) Ordered(checkout string) []Entry {
	out := append([]Entry(nil), v.Open...)
	sort.SliceStable(out, func(i, j int) bool {
		hi, hj := out[i].Checkout == checkout, out[j].Checkout == checkout
		if hi != hj {
			return hi
		}
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.Before(out[j].At)
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// Read is the open and private entries, past those a settled file answers.
// An absent store is an empty view, and nothing is made. A directory refused
// is the error; a file refused is named in Unreadable.
func (s Store) Read() (View, error) {
	var v View
	// A crash inside a settle or a move to private/ leaves the entry where it
	// went and where it was; where it went wins.
	settled, err := s.open(dirSettled)
	if err != nil {
		return View{}, err
	}
	if settled != nil {
		defer settled.Close()
	}
	private, err := s.open(dirPrivate)
	if err != nil {
		return View{}, err
	}
	if private != nil {
		defer private.Close()
	}
	movedOn := func(sub, name string) bool {
		for _, d := range []*os.Root{settled, private} {
			if d == nil || d == private && sub == dirPrivate {
				continue
			}
			if _, err := d.Lstat(name + ".json"); err == nil {
				return true
			}
		}
		return false
	}
	budget := MaxFilesRead
	for _, sub := range []string{dirOpen, dirPrivate} {
		dir, err := s.open(sub)
		if err != nil {
			return View{}, err
		}
		if dir == nil {
			continue
		}
		names, err := listNames(dir, budget+1)
		if err == nil && len(names) > budget {
			err = ErrTooManyFiles
		}
		if err != nil {
			dir.Close()
			return View{}, err
		}
		budget -= len(names)
		if sub == dirOpen && len(names) >= MaxOpen {
			v.Full = true
		}
		for _, name := range names {
			if movedOn(sub, name) {
				continue
			}
			e, err := s.readOne(dir, sub, name)
			switch {
			case errors.Is(err, ErrNewer):
				v.Newer = append(v.Newer, handleOf(name))
			case errors.Is(err, fs.ErrNotExist):
			case err != nil:
				v.Unreadable = append(v.Unreadable, Unreadable{Handle: handleOf(name), Reason: err.Error()})
			case sub == dirPrivate:
				v.Private = append(v.Private, e)
			default:
				v.Open = append(v.Open, e)
			}
		}
		dir.Close()
	}
	return v, nil
}

// SettledView is the last n settled entries, newest first.
func (s Store) SettledView(n int) ([]Entry, error) {
	dir, err := s.open(dirSettled)
	if err != nil || dir == nil {
		return nil, err
	}
	defer dir.Close()
	names, err := listNames(dir, -1)
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, name := range names {
		e, err := s.readOne(dir, dirSettled, name)
		if err != nil || e.Settled == nil {
			continue
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Settled.At.After(out[j].Settled.At) })
	if len(out) > n {
		out = out[:n]
	}
	return out, nil
}

// Answer is what a record says to a claim on it: whether it holds, the state
// found, and where it was found.
type Answer struct {
	Holds bool   `json:"holds"`
	State string `json:"state"`
	Where string `json:"where,omitempty"`
}

// Answerer answers a captured or settled claim on a record id; *Records is
// the one production uses.
type Answerer interface {
	Answer(act Act, id string) (Answer, error)
}

// SettleRequest settles entries by handle with one act.
type SettleRequest struct {
	Handles []string
	Act     Act
	// Record is the record a captured or settled act names, judged by Records.
	Record  string
	Records Answerer
	// Reason is the reason, test or run a not-a-finding, refuted, own or
	// receipt act gives.
	Reason string
	// Person is true where the caller is a person at a terminal, the one
	// caller that may settle a private entry.
	Person bool
	At     time.Time
}

var handleRe = regexp.MustCompile(`^f-([0-9a-f]{8,64})$`)

type located struct {
	e   Entry
	sub string
}

// Settle settles every handle or none: each handle, the act and its record are
// judged first, and a refusal writes nothing. Each settled entry is written to
// settled/ and then removed from where it was.
func (s Store) Settle(req SettleRequest) ([]Entry, error) {
	if len(req.Handles) == 0 {
		return nil, fmt.Errorf("%w: no handle given", ErrNoEntry)
	}
	switch req.Act {
	case ActCaptured, ActSettled:
		if req.Record == "" || req.Records == nil {
			return nil, fmt.Errorf("%w: --%s names a record", ErrNotCleared, req.Act)
		}
	case ActRefuted, ActOwn, ActNotAFinding, ActReceipt:
		if strings.TrimSpace(req.Reason) == "" {
			return nil, fmt.Errorf("%w: --%s takes its reason, test or run", ErrNotCleared, req.Act)
		}
	case ActPrivate:
	default:
		return nil, fmt.Errorf("%w: %q is not an act", ErrNotCleared, req.Act)
	}
	var found []located
	for _, h := range req.Handles {
		l, err := s.locate(h)
		if err != nil {
			return nil, err
		}
		if !clearedBy(l.e.Kind, req.Act) || l.e.Kind == KindPrivate && !req.Person {
			return nil, fmt.Errorf("%w: %s is %s, which --%s does not clear", ErrNotCleared, h, l.e.Kind, req.Act)
		}
		found = append(found, l)
	}
	if req.Act == ActCaptured || req.Act == ActSettled {
		ans, err := req.Records.Answer(req.Act, req.Record)
		if err != nil {
			return nil, err
		}
		if !ans.Holds {
			return nil, fmt.Errorf("%w: %s is %s on %s, which does not answer --%s", ErrNotCleared, req.Record, ans.State, orNowhere(ans.Where), req.Act)
		}
	}
	reason, withheld := clean("", req.Reason)
	if withheld {
		reason = "(withheld: the scanner could not be made)"
	}
	var out []Entry
	for _, l := range found {
		e := l.e
		to := dirSettled
		if req.Act == ActPrivate {
			e.Kind, to = KindPrivate, dirPrivate
		} else {
			e.Settled = &Settlement{Act: req.Act, Record: req.Record, Reason: reason, At: req.At}
		}
		if err := s.move(e, l.sub, to); err != nil {
			return out, err
		}
		out = append(out, e)
	}
	return out, nil
}

func orNowhere(where string) string {
	if where == "" {
		return "no branch or checkout read"
	}
	return where
}

func (s Store) move(e Entry, from, to string) error {
	dst, err := s.ensure(to)
	if err != nil {
		return err
	}
	defer dst.Close()
	if err := write(dst, e); err != nil {
		return err
	}
	src, err := s.ensure(from)
	if err != nil {
		return err
	}
	defer src.Close()
	if err := src.Remove(e.Name + ".json"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// locate finds the one open or private entry handle names.
func (s Store) locate(handle string) (located, error) {
	m := handleRe.FindStringSubmatch(handle)
	if m == nil {
		return located{}, fmt.Errorf("%w: %q is not a handle (f- and eight characters)", ErrNoEntry, handle)
	}
	var hits []located
	var newer bool
	for _, sub := range []string{dirOpen, dirPrivate} {
		dir, err := s.open(sub)
		if err != nil {
			return located{}, err
		}
		if dir == nil {
			continue
		}
		names, err := listNames(dir, MaxFilesRead+1)
		if err != nil {
			dir.Close()
			return located{}, err
		}
		for _, name := range names {
			if !strings.HasPrefix(name, m[1]) {
				continue
			}
			e, err := s.readOne(dir, sub, name)
			switch {
			case errors.Is(err, ErrNewer):
				newer = true
				hits = append(hits, located{})
			case err != nil:
				dir.Close()
				return located{}, fmt.Errorf("%s: %w", handle, err)
			default:
				hits = append(hits, located{e: e, sub: sub})
			}
		}
		dir.Close()
	}
	switch {
	case len(hits) > 1:
		return located{}, fmt.Errorf("%w: %s", ErrAmbiguous, handle)
	case newer:
		return located{}, fmt.Errorf("%w: %s; no command of this build settles it", ErrNewer, handle)
	case len(hits) == 0:
		return located{}, fmt.Errorf("%w: %s", ErrNoEntry, handle)
	}
	return hits[0], nil
}
