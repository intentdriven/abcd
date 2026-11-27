package spotcheck

// groups.go is what a window's populations are drawn from
// (spc-2610301925429250 approach 4): the landings the reader classes drain
// and automatic, and the hand-backs and decisions origin's copy holds, each
// a group of its own.
//
// The landings reader lives in internal/core/landings, which imports
// internal/core/landing, which imports this package, so this package cannot
// call it: the reader registers itself here (RegisterLandingsReader) when its
// package is linked, and a draw with no reader registered is refused, as one
// whose forge answer was not read is.

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

// The four groups, in the order a cycle holds them.
const (
	GroupDrainLanded          = "drain-landed"
	GroupAutomaticMerges      = "automatic-merges"
	GroupDrainHandBacks       = "drain-hand-backs"
	GroupFacilitatorDecisions = "facilitator-decisions"
)

// groupOrder is the order of the groups in a cycle.
var groupOrder = []string{GroupDrainLanded, GroupAutomaticMerges, GroupDrainHandBacks, GroupFacilitatorDecisions}

// The roles an item is for.
const (
	RoleFacilitator      = "facilitator"
	RoleHumanFacilitator = "human-facilitator"
	RoleProductThinker   = "product-thinker"
)

// The landings classes the draw reads, as the reader names them.
const (
	ClassDrain     = "drain"
	ClassAutomatic = "automatic"
)

// LandingRow is one landing as the draw copies it into an item: the reader's
// row in typed fields alone. The reader's own words for why it classed a row
// as it did stay with the reader.
type LandingRow struct {
	Commit     string    `json:"commit"`
	When       time.Time `json:"when"`
	PR         int       `json:"pr,omitempty"`
	MergedHead string    `json:"merged_head,omitempty"`
	Class      string    `json:"class"`
	RuleKind   string    `json:"rule_kind,omitempty"`
	RuleID     string    `json:"rule_id,omitempty"`
	RulePath   string    `json:"rule_path,omitempty"`
	Lane       string    `json:"lane,omitempty"`
	Mode       string    `json:"mode,omitempty"`
	Resolves   []string  `json:"resolves,omitempty"`
	Delivers   []string  `json:"delivers,omitempty"`
	Unattended bool      `json:"unattended"`
	// ForgeUnread says the reader could not settle the row's approval from
	// history and could not read the forge's answer: such a row refuses the
	// draw, so a person's approval is never drawn as unattended on a guess.
	ForgeUnread bool `json:"-"`
}

// LandingsReader reads the landings of the default branch from (exclusive)
// to (inclusive), both full object names, under ctx.
type LandingsReader func(ctx context.Context, root, from, to string) ([]LandingRow, error)

var (
	readerMu       sync.Mutex
	landingsReader LandingsReader
)

// RegisterLandingsReader is how the landings reader makes itself known to the
// draw, once, as its package is linked.
func RegisterLandingsReader(r LandingsReader) {
	readerMu.Lock()
	defer readerMu.Unlock()
	landingsReader = r
}

// registeredReader is the reader the draw reads landings through, nil when
// none is linked.
func registeredReader() LandingsReader {
	readerMu.Lock()
	defer readerMu.Unlock()
	return landingsReader
}

// errNoReader is a draw with no landings reader linked into the binary.
var errNoReader = errors.New("no landings reader is linked into this binary, so the window's landings cannot be read")

var (
	ruleWordRe = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,63}$`)
	repoPathRe = regexp.MustCompile(`^[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)*$`)
	idListRe   = regexp.MustCompile(`^(iss|itd|spc)-[0-9]{1,20}$`)
)

// typedRow is r as an item keeps it: every field held to its shape, and a
// field that does not keep to it dropped. ok is false for a row that names no
// commit or no class the draw reads.
func typedRow(r LandingRow) (LandingRow, bool) {
	if !shaRe.MatchString(r.Commit) || (r.Class != ClassDrain && r.Class != ClassAutomatic) {
		return LandingRow{}, false
	}
	t := LandingRow{Commit: r.Commit, When: r.When.UTC().Truncate(time.Second), Class: r.Class, Unattended: r.Unattended, ForgeUnread: r.ForgeUnread}
	if r.PR > 0 {
		t.PR = r.PR
	}
	if shaRe.MatchString(r.MergedHead) {
		t.MergedHead = r.MergedHead
	}
	for _, f := range []struct {
		in  string
		out *string
		re  *regexp.Regexp
	}{{r.RuleKind, &t.RuleKind, ruleWordRe}, {r.RuleID, &t.RuleID, ruleWordRe}, {r.Lane, &t.Lane, ruleWordRe}, {r.Mode, &t.Mode, ruleWordRe}, {r.RulePath, &t.RulePath, repoPathRe}} {
		if f.re.MatchString(f.in) && !strings.Contains(f.in, "..") {
			*f.out = f.in
		}
	}
	for _, id := range r.Resolves {
		if idListRe.MatchString(id) {
			t.Resolves = append(t.Resolves, id)
		}
	}
	for _, id := range r.Delivers {
		if idListRe.MatchString(id) {
			t.Delivers = append(t.Delivers, id)
		}
	}
	return t, true
}

// candidate is one item a group may draw: its key and the state an item
// copies.
type candidate struct {
	key      string
	roles    []string
	landing  *LandingRow
	handBack *HandBack
	decision *Decision
}

// namedHandBack and namedDecision are records with the file name origin's
// copy keeps them under.
type namedHandBack struct {
	name string
	h    HandBack
}

type namedDecision struct {
	name string
	d    Decision
}

// recordKey is a record's population key: its file name less the hash of its
// content and .json, so records of one issue and one time are one item even
// when their content drifted.
func recordKey(name string) string {
	stem := strings.TrimSuffix(name, ".json")
	if len(stem) > hashLen+1 && stem[len(stem)-hashLen-1] == '-' {
		return stem[:len(stem)-hashLen-1]
	}
	return stem
}

// populations are the window's four groups: the reader's rows classed drain
// and automatic, and the hand-backs and decisions origin's copy holds whose
// key no earlier verified cycle's population of their group held (seen, by
// group). Membership is by arrival on the store's branch, never by the time
// a record names, so a record that reaches origin after its window was drawn
// is drawn from in the next, and every record is in exactly one population:
// windowRecords leaves out what an earlier population listed, and lists no
// more than maxListedKeys, so every record population stays listed.
// A row whose approval was not read from the forge refuses them all. Within
// a group, records of one key are one candidate, the record of the greatest
// file name copied.
func populations(rows []LandingRow, hbs []namedHandBack, ds []namedDecision, seen map[string]map[string]bool) (map[string][]candidate, error) {
	groups := map[string][]candidate{}
	for _, r := range rows {
		if r.ForgeUnread {
			return nil, fmt.Errorf("the forge's answer for the landing %s was not read, so a person's approval of it cannot be ruled out", r.Commit[:12])
		}
		t, ok := typedRow(r)
		if !ok {
			continue
		}
		g, roles := GroupDrainLanded, []string{RoleFacilitator}
		if t.Class == ClassAutomatic {
			g = GroupAutomaticMerges
			if len(t.Delivers) > 0 {
				roles = []string{RoleFacilitator, RoleProductThinker}
			}
		}
		groups[g] = append(groups[g], candidate{key: t.Commit, roles: roles, landing: &t})
	}
	pickH := map[string]namedHandBack{}
	for _, h := range hbs {
		k := recordKey(h.name)
		if seen[GroupDrainHandBacks][k] {
			continue
		}
		if old, ok := pickH[k]; !ok || h.name > old.name {
			pickH[k] = h
		}
	}
	for k, h := range pickH {
		t := h.h.typed()
		groups[GroupDrainHandBacks] = append(groups[GroupDrainHandBacks], candidate{key: k, roles: []string{RoleFacilitator}, handBack: &t})
	}
	pickD := map[string]namedDecision{}
	for _, d := range ds {
		k := recordKey(d.name)
		if seen[GroupFacilitatorDecisions][k] {
			continue
		}
		if old, ok := pickD[k]; !ok || d.name > old.name {
			pickD[k] = d
		}
	}
	for k, d := range pickD {
		t := d.d.typed()
		groups[GroupFacilitatorDecisions] = append(groups[GroupFacilitatorDecisions], candidate{key: k, roles: []string{RoleHumanFacilitator}, decision: &t})
	}
	for g := range groups {
		slices.SortFunc(groups[g], func(a, b candidate) int { return strings.Compare(a.key, b.key) })
		groups[g] = slices.CompactFunc(groups[g], func(a, b candidate) bool { return a.key == b.key })
	}
	return groups, nil
}
