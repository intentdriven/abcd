package lab

// pairs.go is the decision adapter's shadow record (itd-2609221009495079,
// spc-2609221011151661 scopes 3 and 4): while a judgement type runs in shadow
// the host decides and the adapter answers beside it, and each pair (the
// host's answer, the adapter's, its probability and the state's hash) is
// appended to pairs.jsonl in a lab home, in the machine-scoped store and never
// the repository. The harvest reads the file back as the agreement report a
// research note turns a type on with.
//
// A pair goes to the newest lab minted for the repository: a lab is minted to
// answer one question, and the shadow run is the question "does the adapter
// agree with the host?", so the person mints the lab before configuring a type
// for shadow, and the lab the latest mint opened collects the pairs.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// pairsName is the shadow record inside a lab home.
const pairsName = "pairs.jsonl"

// maxPairBytes bounds one recorded line: a pair carries answers and a hash,
// never the state itself.
const maxPairBytes = 8 << 10

// SkippedWindow marks a pair whose state exceeded the adapter's window, so
// only the host answered; SkippedFailed one whose adapter call failed.
const (
	SkippedWindow = "window"
	SkippedFailed = "adapter-failed"
)

// ErrNoLab is RecordPair's answer when the repository has no lab to record
// in: the host has decided, and the pair is not kept.
var ErrNoLab = errors.New("no lab minted for this repository")

// Pair is one shadow judgement: the type and its question kind, the state's
// hash and size, the host's answer, and the adapter's answer with its
// probability, or why the adapter did not answer.
type Pair struct {
	At          string   `json:"at"`
	Type        string   `json:"type"`
	Kind        string   `json:"kind"`
	StateSHA256 string   `json:"state_sha256"`
	StateTokens int      `json:"state_tokens"`
	Host        string   `json:"host"`
	Adapter     string   `json:"adapter,omitempty"`
	Probability *float64 `json:"probability,omitempty"`
	// Target is the adapter's <provider>/<model>; ModelReported the model
	// the provider said answered.
	Target        string `json:"target"`
	ModelReported string `json:"model_reported,omitempty"`
	// Skipped is SkippedWindow or SkippedFailed when the adapter gave no
	// answer, and Detail says why.
	Skipped string `json:"skipped,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

// RecordPair appends p to pairs.jsonl in the newest lab minted for the
// repository at repoRoot, stamping its time, and returns the lab's id and the
// file in tilde form. It refuses a pair without its type, kind, state hash or
// host answer, and returns ErrNoLab when the repository has no lab.
func RecordPair(repoRoot string, p Pair) (id, where string, err error) {
	if p.Type == "" || p.Kind == "" || len(p.StateSHA256) != 64 || p.Host == "" {
		return "", "", fmt.Errorf("%w: a pair names its type, kind, state hash and the host's answer", ErrRefused)
	}
	s, err := resolveStore(repoRoot)
	if err != nil {
		return "", "", err
	}
	es, _, err := s.entries()
	if err != nil {
		return "", "", err
	}
	if len(es) == 0 {
		return "", "", fmt.Errorf("%w in %s; mint one with `abcd lab mint` to record shadow pairs", ErrNoLab, s.display())
	}
	newest := es[0]
	for _, e := range es[1:] {
		if e.ID > newest.ID {
			newest = e
		}
	}
	l, err := open(repoRoot, newest.ID)
	if err != nil {
		return "", "", err
	}
	defer l.close()
	p.At = stamp()
	line, err := json.Marshal(p)
	if err != nil {
		return "", "", err
	}
	if len(line) >= maxPairBytes {
		return "", "", fmt.Errorf("%w: a pair of %d bytes exceeds the %d a line may hold", ErrRefused, len(line), maxPairBytes)
	}
	if fi, err := l.root.Lstat(pairsName); err == nil && !fi.Mode().IsRegular() {
		return "", "", fmt.Errorf("%w: %s is not a regular file", ErrRefused, l.display(pairsName))
	}
	f, err := l.root.OpenFile(pairsName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, fileMode)
	if err != nil {
		return "", "", fmt.Errorf("cannot open %s: %v", l.display(pairsName), redact(err, l.store.home))
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return "", "", fmt.Errorf("cannot append to %s: %v", l.display(pairsName), redact(err, l.store.home))
	}
	if err := f.Close(); err != nil {
		return "", "", fmt.Errorf("cannot close %s: %v", l.display(pairsName), redact(err, l.store.home))
	}
	return newest.ID, l.display(pairsName), nil
}

// Band is one probability band of a type's compared pairs.
type Band struct {
	Band     string `json:"band"`
	Compared int    `json:"compared"`
	Agreed   int    `json:"agreed"`
}

// TypeAgreement is one judgement type's agreement: every pair recorded, the
// ones the adapter answered (compared), those whose answers matched, those it
// did not answer (skipped), the rate over the compared, and the bands.
type TypeAgreement struct {
	Type     string  `json:"type"`
	Pairs    int     `json:"pairs"`
	Compared int     `json:"compared"`
	Agreed   int     `json:"agreed"`
	Skipped  int     `json:"skipped"`
	Rate     float64 `json:"rate"`
	Bands    []Band  `json:"bands"`
}

// Agreement is the harvest's report over a lab's pairs: the lab home, the
// pairs counted, the lines that did not parse, each type's agreement, and the
// line a research note filed from it carries.
type Agreement struct {
	LabHome    string          `json:"lab_home"`
	Pairs      int             `json:"pairs"`
	Unreadable int             `json:"unreadable"`
	Types      []TypeAgreement `json:"types"`
	Note       string          `json:"note"`
}

// bands are the probability bands, lowest first: [lo, hi), the last closed.
var bands = []struct {
	name   string
	lo, hi float64
}{{"0.00-0.50", 0, 0.5}, {"0.50-0.70", 0.5, 0.7}, {"0.70-0.90", 0.7, 0.9}, {"0.90-1.00", 0.9, 1.0000001}}

// agreement reads the lab's pairs.jsonl; nil when the lab recorded none.
//
// The file is read a line at a time, each line bounded by maxPairBytes, and
// never whole: RecordPair appends to it for as long as a type runs in shadow,
// so a busy lab's file outgrows any bound one document read could hold, and
// the harvest still counts every pair. A line longer than a pair may be is
// counted as one that did not parse, and the read goes on past it.
func (l *lab) agreement() (*Agreement, error) {
	f, err := l.openPairs()
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %v", l.display(pairsName), redact(err, l.store.home))
	}
	defer f.Close()
	a := &Agreement{LabHome: l.store.display(l.entry.ID), Types: []TypeAgreement{}}
	by := map[string]*TypeAgreement{}
	rd := bufio.NewReaderSize(f, maxPairBytes)
	for {
		raw, rerr := rd.ReadSlice('\n')
		if errors.Is(rerr, bufio.ErrBufferFull) {
			a.Unreadable++
			for errors.Is(rerr, bufio.ErrBufferFull) {
				_, rerr = rd.ReadSlice('\n')
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				return nil, fmt.Errorf("cannot read %s: %v", l.display(pairsName), redact(rerr, l.store.home))
			}
			continue
		}
		if rerr != nil && rerr != io.EOF {
			return nil, fmt.Errorf("cannot read %s: %v", l.display(pairsName), redact(rerr, l.store.home))
		}
		a.count(by, bytes.TrimSpace(raw))
		if rerr == io.EOF {
			break
		}
	}
	names := make([]string, 0, len(by))
	for n := range by {
		names = append(names, n)
	}
	sort.Strings(names)
	compared := 0
	for _, n := range names {
		t := by[n]
		if t.Compared > 0 {
			t.Rate = float64(t.Agreed) / float64(t.Compared)
		}
		compared += t.Compared
		a.Types = append(a.Types, *t)
	}
	a.Note = fmt.Sprintf("Shadow agreement from the lab home %s: %d pairs (%d compared) over %d judgement type(s), harvested %s.",
		a.LabHome, a.Pairs, compared, len(a.Types), stamp())
	return a, nil
}

// count adds one line of pairs.jsonl to a: a pair to its type's counts and
// bands, a blank line to nothing, and anything else as unreadable.
func (a *Agreement) count(by map[string]*TypeAgreement, line []byte) {
	if len(line) == 0 {
		return
	}
	var p Pair
	if json.Unmarshal(line, &p) != nil || p.Type == "" || p.Host == "" {
		a.Unreadable++
		return
	}
	a.Pairs++
	t := by[p.Type]
	if t == nil {
		t = &TypeAgreement{Type: p.Type, Bands: make([]Band, len(bands))}
		for i, b := range bands {
			t.Bands[i].Band = b.name
		}
		by[p.Type] = t
	}
	t.Pairs++
	if p.Skipped != "" || p.Adapter == "" || p.Probability == nil {
		t.Skipped++
		return
	}
	t.Compared++
	agreed := p.Adapter == p.Host
	if agreed {
		t.Agreed++
	}
	for i, b := range bands {
		if *p.Probability >= b.lo && *p.Probability < b.hi {
			t.Bands[i].Compared++
			if agreed {
				t.Bands[i].Agreed++
			}
		}
	}
}

// openPairs opens the lab's pairs.jsonl to read: a regular file in the lab
// home, never a link or anything else, and the file the check vetted, not one
// put in its place since; os.ErrNotExist when the lab recorded none.
func (l *lab) openPairs() (*os.File, error) {
	fi, err := l.root.Lstat(pairsName)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fsutil.ErrNotRegular
	}
	f, err := l.root.OpenFile(pairsName, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	if st, err := f.Stat(); err != nil || !os.SameFile(fi, st) || !st.Mode().IsRegular() {
		f.Close()
		return nil, fsutil.ErrNotRegular
	}
	return f, nil
}

// agreementDoc renders the harvest's agreement section.
func agreementDoc(a *Agreement) string {
	var b strings.Builder
	b.WriteString("## Agreement — decision adapter shadow pairs\n\n")
	fmt.Fprintf(&b, "From `%s/%s`: %d pairs", a.LabHome, pairsName, a.Pairs)
	if a.Unreadable > 0 {
		fmt.Fprintf(&b, ", %d line(s) that did not parse and are not counted", a.Unreadable)
	}
	b.WriteString(". A pair is compared when the adapter answered; one it did not answer (its state\nexceeded the window, or the call failed) is skipped, and the host decided alone.\n\n")
	b.WriteString("| Type | Pairs | Compared | Agreed | Skipped | Agreement |\n| --- | --- | --- | --- | --- | --- |\n")
	for _, t := range a.Types {
		rate := "n/a"
		if t.Compared > 0 {
			rate = fmt.Sprintf("%.0f%%", t.Rate*100)
		}
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %s |\n", t.Type, t.Pairs, t.Compared, t.Agreed, t.Skipped, rate)
	}
	b.WriteString("\nBy the adapter's probability (agreed/compared):\n\n")
	for _, t := range a.Types {
		parts := make([]string, 0, len(t.Bands))
		for _, bd := range t.Bands {
			parts = append(parts, fmt.Sprintf("%s: %d/%d", bd.Band, bd.Agreed, bd.Compared))
		}
		fmt.Fprintf(&b, "- %s — %s\n", t.Type, strings.Join(parts, ", "))
	}
	fmt.Fprintf(&b, "\nFor the research note that turns a type on:\n\n> %s\n", a.Note)
	return b.String()
}
