package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// StateTTL bounds how long a per-session ledger lives before the reset hook
// sweeps it (sessions are one-shot; a week is generous headroom).
const StateTTL = 7 * 24 * time.Hour

// maxPromptBytes bounds how much prompt text is scanned for recall, so a huge
// pasted prompt cannot blow up matching (trust boundary).
const maxPromptBytes = 64 * 1024

// maxStateFileBytes caps a session-state ledger file on read.
const maxStateFileBytes = 256 * 1024

// injectionBudgetBytes caps the TOTAL rendered rule text injected per turn,
// summed across every matched domain (iss-2608261551077971). The 256 KiB
// per-file read cap bounds one rules.json on disk, not the injected total — and
// the payload re-lands at least every DefaultRefreshBackstop prompts and after
// each compaction, so an unbounded set is a recurring context-flood vector.
//
// 64 KiB is a quarter of the per-file read cap and equal to maxPromptBytes (the
// per-turn prompt ceiling the loader already scans against), so it is a
// consistent per-turn text budget. The entire bundled default set renders to a
// few KiB, so a legitimate multi-domain match sits an order of magnitude under
// this; only a bloated or hostile rules.json can reach it.
const injectionBudgetBytes = 64 * 1024

// InjectionBudgetBytes is the per-turn budget, exported because the memory
// recall shares it (spc-2610111627213138, "Budget"): the rules render first,
// and the recall takes what they leave, never a budget of its own.
const InjectionBudgetBytes = injectionBudgetBytes

// injectionTruncatedMarker is the unmistakable substring the loud truncation
// notice carries when the budget bites — searchable in the injected output and
// in host logs.
const injectionTruncatedMarker = "INJECTION TRUNCATED"

// DefaultRefreshBackstop is the fixed-N full-refresh backstop (D1): the primary
// refresh is event-driven (a SessionStart/PreCompact reset clears the ledger),
// and this large counter only catches always-relevant domains that never
// recall-match. It is deliberately larger than CARL's every-5.
const DefaultRefreshBackstop = 15

// InjectResult is the outcome of one prompt-router evaluation. Text is empty
// when nothing new is injected (a healthy no-match renders zero model-facing
// tokens, per D3). Sources says, per injected name, which layer the domain
// came from (SourceBundled, SourceUser or SourceRepo): the names alone cannot, and the
// out-of-band diagnostic must say whose words went into the context.
//
// Active is the FULL set of domain names in force this turn, name-sorted and
// never nil (ruling J15, iss-2608261550580260): every domain that is not
// dormant, plus a dormant one this prompt activated with *NAME, and nothing at
// all under the kill switch. Injected names only what changed; Active is the
// removal signal a client that snapshots injected rules prunes against — a
// name it holds that Active omits has stopped, whether it was deleted, renamed
// or made dormant. It is never rendered into Text, so it costs the model's
// context nothing; a front door hands it to a machine reader out of band.
type InjectResult struct {
	Text     string
	Injected []string
	Active   []string
	Sources  map[string]string
	State    SessionState
}

// Labels returns the injected names in order, each labelled the way the
// rendered heading is — "PII (repo override)" for a repo-sourced domain,
// "PII (user override)" for a user-scope one, the bare name otherwise — so the
// diagnostic and the block agree byte for byte.
func (r InjectResult) Labels() []string {
	out := make([]string, 0, len(r.Injected))
	for _, name := range r.Injected {
		out = append(out, Label(name, r.Sources[name]))
	}
	return out
}

// SessionState is the per-session dedup ledger plus the prompt counter that
// drives the fixed-N refresh backstop.
//
// Recalled is the memory recall's own ledger (internal/core/recall,
// spc-2610111627213138): item key to a signature of the item's content. It is
// a second map because Inject prunes every Ledger name that is not an active
// domain, which would forget every recalled item each turn; Inject carries it
// through untouched, and the same resets clear both — ResetState removes the
// file, and the backstop prompt clears the two together.
type SessionState struct {
	Count    int               `json:"count"`
	Ledger   map[string]string `json:"ledger"`             // domain name -> last-injected signature
	Recalled map[string]string `json:"recalled,omitempty"` // memory item key -> content signature
}

// Inject is the pure heart of the prompt-router hook: it recall-matches prompt,
// drops any matched domain already injected this session with an unchanged
// signature, renders the remainder, and returns the updated session state. It
// performs no I/O and never reflects prompt bytes into the output — the rendered
// text is abcd's own rule content only.
//
// backstop is the fixed-N full-refresh interval; <= 0 uses DefaultRefreshBackstop.
// When the (incremented) prompt count is a multiple of the backstop, the ledger
// is cleared first so always-relevant domains re-inject, and the recalled
// ledger with it, so a recalled item comes back on the same prompt.
func Inject(rs RuleSet, prompt string, prev SessionState, backstop int) InjectResult {
	if backstop <= 0 {
		backstop = DefaultRefreshBackstop
	}
	prompt = CapPrompt(prompt)

	ledger := map[string]string{}
	for k, v := range prev.Ledger {
		ledger[k] = v
	}
	recalled := copyLedger(prev.Recalled)
	count := prev.Count + 1
	if count%backstop == 0 {
		ledger = map[string]string{}
		recalled = nil
	}

	matched := rs.Match(prompt)
	active := activeNames(rs, matched)
	// A domain that has left the active set is forgotten, so that when it comes
	// back its text is injected again: a snapshotting client pruned it the turn
	// it left, and deduping its return against the old signature would leave
	// that client without it for the rest of the session.
	for name := range ledger {
		if _, ok := active[name]; !ok {
			delete(ledger, name)
		}
	}

	var fresh []ResolvedDomain
	for _, d := range matched {
		if ledger[d.Name] == Signature(d) {
			continue // already injected this session, unchanged
		}
		fresh = append(fresh, d)
	}

	// Enforce the per-repo injection budget across ALL matched domains, not just
	// the per-file read cap (iss-2608261551077971). A domain that is truncated
	// away is deliberately NOT recorded in the ledger, so it is retried next turn
	// rather than silently dropped for the session.
	text, kept := renderWithinBudget(fresh, rs.rulesFrom)
	var injected []string
	sources := make(map[string]string, len(kept))
	for _, d := range kept {
		ledger[d.Name] = Signature(d)
		injected = append(injected, d.Name)
		sources[d.Name] = d.Source
	}
	sort.Strings(injected)

	names := make([]string, 0, len(active))
	for name := range active {
		names = append(names, name)
	}
	sort.Strings(names)

	return InjectResult{
		Text:     text,
		Injected: injected,
		Active:   names,
		Sources:  sources,
		State:    SessionState{Count: count, Ledger: ledger, Recalled: recalled},
	}
}

// copyLedger copies the memory recall's ledger, nil for an empty one, so the
// state Inject hands back never shares a map with the caller's.
func copyLedger(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// activeNames is the set of domain names in force this turn: every domain the
// rule set holds active, plus every matched one — the matched set adds only a
// dormant domain the prompt activated explicitly, which is in force for the
// turn that names it. A disabled set matches nothing and has nothing active.
func activeNames(rs RuleSet, matched []ResolvedDomain) map[string]struct{} {
	out := map[string]struct{}{}
	for _, d := range rs.Active() {
		out[d.Name] = struct{}{}
	}
	for _, d := range matched {
		out[d.Name] = struct{}{}
	}
	return out
}

// renderWithinBudget renders fresh under the per-repo injection budget. When the
// full render fits it is returned byte-identical to Render (the common path is
// untouched). When it overflows, whole domain blocks are kept greedily in order
// until the next would exceed the budget, and a loud, unmistakable truncation
// notice is appended — whole-block granularity keeps the "## NAME" injection
// contract intact (never a half-rendered rule). It returns the text and the
// domains actually kept, so the caller records only those in the dedup ledger.
// rulesFrom maps a domain to the file its rules came from (RuleSet.rulesFrom);
// the notice names those files as the likely cause.
func renderWithinBudget(fresh []ResolvedDomain, rulesFrom map[string]string) (string, []ResolvedDomain) {
	full := Render(fresh)
	if len(full) <= injectionBudgetBytes {
		return full, fresh
	}

	var kept []ResolvedDomain
	size := 0
	// Reserve room for the header line and the truncation notice so the composed
	// output is provably within the budget regardless of how many blocks fit.
	const reserve = 512
	for _, d := range fresh {
		block := renderDomain(d)
		if size+len(block) > injectionBudgetBytes-reserve {
			break
		}
		kept = append(kept, d)
		size += len(block)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# abcd rules — %d domain(s) active\n", len(kept))
	for _, d := range kept {
		b.WriteString(renderDomain(d))
	}
	fmt.Fprintf(&b,
		"\n# abcd rules — %s: rendered rule set exceeds the %d-byte per-repo budget; %d of %d matched domain(s) omitted this turn. %s\n",
		injectionTruncatedMarker, injectionBudgetBytes, len(fresh)-len(kept), len(fresh), budgetCause(fresh, rulesFrom))
	return b.String(), kept
}

// budgetCause names the files whose words filled the budget, largest share
// first: each matched domain's rendered block is charged to the file its rules
// came from, and the bundled words are charged to no file. Naming one file
// always — .abcd/rules.json — sent a reader to the wrong file when the
// repository's guard.json or the user's rules.json held the bulk.
func budgetCause(fresh []ResolvedDomain, rulesFrom map[string]string) string {
	cost := map[string]int{}
	for _, d := range fresh {
		if file := rulesFrom[d.Name]; file != "" {
			cost[file] += len(renderDomain(d))
		}
	}
	if len(cost) == 0 {
		return "No override file wrote these rules; inspect them with `abcd rules`."
	}
	files := make([]string, 0, len(cost))
	for f := range cost {
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool {
		if cost[files[i]] != cost[files[j]] {
			return cost[files[i]] > cost[files[j]]
		}
		return files[i] < files[j]
	})
	return "A bloated or hostile " + strings.Join(files, " or ") + " is the likely cause — inspect it with `abcd rules`."
}

// ErrNoStateHome is SaveState's answer when the account has no home folder:
// no ledger is kept, so every prompt re-injects each domain it matches. It is
// returned, not swallowed, so the caller can say why on its diagnostic stream.
var ErrNoStateHome = errors.New("no home folder to keep the session ledger in, so every prompt re-injects what it matches")

// stateRel is where per-session ledgers are kept, relative to the account's
// home: a folder of abcd's own home store.
var stateRel = abcdhome.Rel("rules-state")

// stateFolder opens the folder holding per-session ledgers, creating it when
// create is set, and returns it with the home and the folder relative to it
// that a read names the file by. ABCD_RULES_STATE_DIR overrides it (tests, and
// operators who want the state elsewhere). Otherwise it is the account's own
// home store, walked by the guarded helpers: never a predictable name under
// the shared temporary folder, where another account can make the folder
// first and plant a ledger that silences every rule a session matches. With
// no home there is no private place for it, so ErrNoStateHome is returned and
// no ledger is kept: each prompt then starts fresh, which re-injects what it
// matches and never silences anything. Reads and writes alike open the folder with
// OpenHomeDeclarationScope, which refuses one another account owns or can
// write, so a save into such a folder fails out loud rather than writing a
// ledger every later read ignores.
func stateFolder(create bool) (dir *os.Root, home, rel string, err error) {
	if d := os.Getenv("ABCD_RULES_STATE_DIR"); d != "" {
		if create {
			if err := os.MkdirAll(d, 0o700); err != nil {
				return nil, "", "", err
			}
		}
		dir, err := os.OpenRoot(d)
		return dir, d, ".", err
	}
	home, err = os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return nil, "", "", ErrNoStateHome
	}
	if create {
		made, err := fsutil.EnsureHomeScope(home, stateRel, abcdhome.DirMode)
		if err != nil {
			return nil, "", "", err
		}
		made.Close()
	}
	dir, err = fsutil.OpenHomeDeclarationScope(home, stateRel)
	return dir, home, stateRel, err
}

// sessionLeaf maps a session id to its ledger's file name. The id is hashed,
// so an attacker-supplied session id can never name a path.
func sessionLeaf(session string) string {
	sum := sha256.Sum256([]byte(session))
	return hex.EncodeToString(sum[:]) + ".json"
}

// LoadState reads the ledger for a session. A missing, oversized, or malformed
// file yields the zero state (a fresh session), never an error — dedup is a
// best-effort optimisation, not a correctness gate. So does a file this
// account does not own, one others can write, one reached through a link, and
// one in a folder another account owns or can write: none is this account's
// word, and a fresh session only re-injects.
func LoadState(session string) SessionState {
	dir, home, rel, err := stateFolder(false)
	if err != nil {
		return SessionState{}
	}
	defer dir.Close()
	data, refusal, err := fsutil.ReadHomeDeclarationDenyingIn(dir, home, path.Join(rel, sessionLeaf(session)), maxStateFileBytes, 0o022)
	if err != nil || refusal != fsutil.DeclarationOK {
		return SessionState{}
	}
	var st SessionState
	if err := json.Unmarshal(data, &st); err != nil {
		return SessionState{}
	}
	if st.Ledger == nil {
		st.Ledger = map[string]string{}
	}
	return st
}

// SaveState durably persists the ledger for a session through the canonical
// fsutil.WriteFileAtomicInRoot (temp-write, fsync, rename, parent-dir fsync)
// into the folder stateFolder opened, so the write is crash-safe, a reader
// sees either the old file or the complete new one, and it never follows a
// link out of that folder. The folder is made 0700 and the file 0600. With no
// home it keeps nothing and returns ErrNoStateHome.
func SaveState(session string, st SessionState) error {
	dir, _, _, err := stateFolder(true)
	if err != nil {
		return err
	}
	defer dir.Close()
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(dir, sessionLeaf(session), data, 0o600)
}

// LoadBackstop reads rules.force_refresh_every_n from <repoRoot>/.abcd/config.json
// — the fixed-N full-refresh backstop (D1). Event-driven reset is the primary
// refresh, so this only bounds always-relevant domains; a missing file/key or a
// non-positive value falls back to DefaultRefreshBackstop.
func LoadBackstop(repoRoot string) int {
	path := filepath.Join(repoRoot, ".abcd", "config.json")
	data, err := readGuarded(path, maxRulesFileBytes)
	if err != nil {
		return DefaultRefreshBackstop
	}
	var cfg struct {
		Rules struct {
			ForceRefreshEveryN int `json:"force_refresh_every_n"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil || cfg.Rules.ForceRefreshEveryN <= 0 {
		return DefaultRefreshBackstop
	}
	return cfg.Rules.ForceRefreshEveryN
}

// ResetState clears a session's ledger (the event-driven refresh: the next
// prompt re-injects every matching domain). A missing file or folder, or no
// home, is not an error; a folder stateFolder refuses is, so the hook never
// reports a reset it did not make.
func ResetState(session string) error {
	dir, _, _, err := stateFolder(false)
	if errors.Is(err, ErrNoStateHome) || os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer dir.Close()
	err = dir.Remove(sessionLeaf(session))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// PruneState removes session-state files older than maxAge, bounding the growth
// of the state folder across many sessions. Best-effort housekeeping: a
// missing folder or a per-entry error is ignored.
func PruneState(maxAge time.Duration) {
	dir, _, _, err := stateFolder(false)
	if err != nil {
		return
	}
	defer dir.Close()
	f, err := dir.Open(".")
	if err != nil {
		return
	}
	entries, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = dir.Remove(e.Name())
		}
	}
}
