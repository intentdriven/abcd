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

// HostOutputCap is the host's cap on one hook's output, in characters (the hooks
// reference, read 2026-11-04): above it the host saves the output to a file,
// shows a short preview and does not ask the agent to read the rest. Another host
// may differ, and this is the one place the figure sits.
const HostOutputCap = 10_000

// hostOutputMargin is kept free under the cap, the margin the session-start hook
// keeps too. Bytes never undercount characters, so the margin covers only the
// blank lines that join the blocks.
const hostOutputMargin = 500

// InjectionBudgetBytes bounds everything the prompt hook adds on one turn: the
// rules, the memory notes and pages, the status block and the away block
// (iss-2611041419354106). The status and away blocks are read first and the rest
// share what they leave; the memory recall takes what the rules actually left
// (spc-2610111627213138, "Budget"), never a budget of its own.
const InjectionBudgetBytes = HostOutputCap - hostOutputMargin

// MemoryFloorBytes is what the rules leave for memory notes and pages however
// many domains match, so a long rule set never silences what was learned. It
// holds a note at the longest body a note may have, on lines of ordinary length,
// with its header and line.
// What the notes leave of it goes back to the rules (cli.injectWithRecall).
const MemoryFloorBytes = 2_500

// RulesBudget is what the rules may take of the turn once reserved bytes have
// gone to the blocks read first and the floor to memory. It is never below 0.
func RulesBudget(reserved int) int {
	return max(0, InjectionBudgetBytes-reserved-MemoryFloorBytes)
}

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
	// Shrunk names, in order, each injected domain sent as its heading and
	// first rule rather than in full (renderShrunk), because of its bracket or
	// because the turn could not hold it.
	Shrunk []string
	// Pointed names, in order, each injected domain sent as its heading and the
	// command alone, because even its first rule did not fit (renderPointer).
	Pointed []string
	// Bracket is the bracket the rules were injected in.
	Bracket Bracket
	State   SessionState
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
	// FillUnknownSaid records that the hook has said, once this session, that
	// how full the session is is unknown (UnknownFillNotice). The resets
	// clear it with the ledgers, since they remove the file; the fixed
	// backstop does not, so it is said once a session and not once a cycle.
	FillUnknownSaid bool `json:"fill_unknown_said,omitempty"`
}

// UnknownFillNotice is the one line the hook says when how full the session
// is is unknown, and only the first time this session (the technical
// facilitator's ruling of 2026-10-11, "Rules full, notes short": the hook
// says the fill is unknown on stderr on the first prompt of each session).
// It records in st that it was said, so a caller saves st after it. It is ""
// when the fill is known or the line was already said this session. The
// reason is the fill reader's, in abcd's own words.
func UnknownFillNotice(st *SessionState, f Fill) string {
	if f.Known() || st.FillUnknownSaid {
		return ""
	}
	st.FillUnknownSaid = true
	reason := f.Reason
	if reason == "" {
		reason = "no source told"
	}
	return "rules: how full this session is, is unknown (" + reason + "); rules come back in full where they fit and memory notes and pages as one line each while it stays unknown, and `abcd rules --prompt` shows any of them in full"
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
//
// bracket is the context bracket the session's fill puts the prompt in
// (brackets.go, BracketFor). In a bracket whose row shrinks rules, each
// matched domain is sent as its heading, its first rule and a line naming
// how many rules it holds and the command that renders the rest
// (renderShrunk); BracketUnknown, the zero value, a caller with no fill
// passes, keeps every domain in full, as before the brackets. The ledger
// keys on a domain's content, never on the form it was sent in (Signature
// hashes the full block), so a domain sent shrunk is not sent again in full
// when the bracket changes, nor the other way round, until a reset clears
// the ledger.
//
// Inject is InjectWithin with nothing reserved, the budget of a turn with no
// status or away block.
func Inject(rs RuleSet, prompt string, prev SessionState, backstop int, bracket Bracket) InjectResult {
	return InjectWithin(rs, prompt, prev, backstop, bracket, RulesBudget(0))
}

// InjectWithin is Inject under budget bytes for the rules. A domain that does not
// fit in full goes in its short form and is recorded as sent, the same form and
// the same record as in a bracket that shrinks, since the form names the command
// that renders the rest.
func InjectWithin(rs RuleSet, prompt string, prev SessionState, backstop int, bracket Bracket, budget int) InjectResult {
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

	// The budget holds across ALL matched domains. Only a domain with no room
	// even for its pointer is left out of the ledger, so it is tried again next
	// turn; every form sent is recorded, by the domain's content (Signature
	// hashes the full block), so a change of form never sends it again.
	text, kept := renderWithinBudget(fresh, rs.rulesFrom, bracket.Row().ShrinkRules, budget)
	var injected, shrunk, pointed []string
	sources := make(map[string]string, len(kept))
	for _, k := range kept {
		ledger[k.domain.Name] = Signature(k.domain)
		injected = append(injected, k.domain.Name)
		sources[k.domain.Name] = k.domain.Source
		switch k.form {
		case formShrunk:
			shrunk = append(shrunk, k.domain.Name)
		case formPointer:
			pointed = append(pointed, k.domain.Name)
		}
	}
	sort.Strings(injected)
	sort.Strings(shrunk)
	sort.Strings(pointed)

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
		Shrunk:   shrunk,
		Pointed:  pointed,
		Bracket:  bracket,
		State:    SessionState{Count: count, Ledger: ledger, Recalled: recalled, FillUnknownSaid: prev.FillUnknownSaid},
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

// The forms a domain can be sent in, and none: left out this turn.
type sentForm int

const (
	formNone sentForm = iota
	formPointer
	formShrunk
	formFull
)

// sentDomain is one domain and the form it goes in.
type sentDomain struct {
	domain ResolvedDomain
	form   sentForm
	block  string
}

// noticeReserve is kept for the truncation notice when a domain is left out.
const noticeReserve = 512

// renderWithinBudget renders fresh within budget bytes. When every domain fits in
// its row's form that is the whole answer, byte-identical to Render in the row
// that keeps rules full. Otherwise the room goes up a ladder: every domain first
// gets its pointer, then what room is left widens domains to their first rule, then
// to full, the cheapest first at each step, so one long domain never holds back the
// next and as many as possible arrive whole; a row that shrinks rules stops at the
// first rule. Whole blocks only, which keeps the "## NAME" contract: never half a
// rule. A domain with no room even for a pointer is left out, and a notice says so
// where the notice itself fits. It returns the text and the domains sent with their
// forms, so the caller records exactly those. rulesFrom maps a domain to the file
// its rules came from (RuleSet.rulesFrom), which the notice names.
func renderWithinBudget(fresh []ResolvedDomain, rulesFrom map[string]string, shrinks bool, budget int) (string, []sentDomain) {
	if len(fresh) == 0 {
		return "", nil
	}
	whole := make([]sentDomain, len(fresh))
	size := len(domainsHeader(len(fresh)))
	for i, d := range fresh {
		whole[i] = rowForm(d, shrinks)
		size += len(whole[i].block)
	}
	if size <= budget {
		return compose(whole, ""), whole
	}

	left := budget - len(domainsHeader(len(fresh)))
	sent, omitted, room := packPointers(fresh, left)
	if omitted > 0 {
		// Room for the notice, unless keeping it would send nothing at all.
		if s, o, r := packPointers(fresh, left-noticeReserve); o < len(fresh) {
			sent, omitted, room = s, o, r
		}
	}
	widen(sent, &room, func(d ResolvedDomain) sentDomain { return rowForm(d, true) })
	if !shrinks {
		widen(sent, &room, func(d ResolvedDomain) sentDomain { return rowForm(d, false) })
	}
	var kept []sentDomain
	for _, s := range sent {
		if s.form != formNone {
			kept = append(kept, s)
		}
	}
	if len(kept) == 0 {
		return "", nil
	}
	notice := ""
	if omitted > 0 {
		notice = fmt.Sprintf("\n# abcd rules — %s: %d of %d matched domain(s) did not fit the %d bytes left for rules this turn and are tried again on the next. %s\n",
			injectionTruncatedMarker, omitted, len(fresh), budget, budgetCause(fresh, rulesFrom))
	}
	text := compose(kept, notice)
	if len(text) > budget {
		text = compose(kept, "")
	}
	return text, kept
}

// domainsHeader is the line every rules block opens with.
func domainsHeader(n int) string { return fmt.Sprintf("# abcd rules — %d domain(s) active\n", n) }

// compose joins the header, the blocks and the notice.
func compose(sent []sentDomain, notice string) string {
	var b strings.Builder
	b.WriteString(domainsHeader(len(sent)))
	for _, s := range sent {
		b.WriteString(s.block)
	}
	b.WriteString(notice)
	return b.String()
}

// rowForm is d in the form its bracket's row gives it, room allowing.
func rowForm(d ResolvedDomain, shrinks bool) sentDomain {
	if shrinks {
		if block, ok := renderShrunk(d); ok {
			return sentDomain{d, formShrunk, block}
		}
	}
	return sentDomain{d, formFull, renderDomain(d)}
}

// packPointers gives each domain, in order, its pointer while one fits. It returns
// the forms, how many got nothing and the room left over.
func packPointers(fresh []ResolvedDomain, room int) (sent []sentDomain, omitted, left int) {
	sent = make([]sentDomain, len(fresh))
	for i, d := range fresh {
		block := renderPointer(d)
		if len(block) > room {
			sent[i] = sentDomain{domain: d, form: formNone}
			omitted++
			continue
		}
		room -= len(block)
		sent[i] = sentDomain{d, formPointer, block}
	}
	return sent, omitted, room
}

// widen spends room moving domains up to the form target gives them, the cheapest
// first. A domain left out stays out, since a later turn sends it.
func widen(sent []sentDomain, room *int, target func(ResolvedDomain) sentDomain) {
	next := make([]sentDomain, len(sent))
	order := make([]int, 0, len(sent))
	for i, s := range sent {
		next[i] = target(s.domain)
		if s.form != formNone && next[i].form > s.form {
			order = append(order, i)
		}
	}
	extra := func(i int) int { return len(next[i].block) - len(sent[i].block) }
	sort.SliceStable(order, func(a, b int) bool { return extra(order[a]) < extra(order[b]) })
	for _, i := range order {
		if e := extra(i); e <= *room {
			*room -= e
			sent[i] = next[i]
		}
	}
}

// budgetCause names the files whose words matched this turn, largest share
// first, charging each domain's full block to the file its rules came from and
// the bundled words to no file. Naming one file always, .abcd/rules.json, sent a
// reader to the wrong file when the repository's guard.json or the user's
// rules.json held the bulk.
func budgetCause(fresh []ResolvedDomain, rulesFrom map[string]string) string {
	cost := map[string]int{}
	for _, d := range fresh {
		if file := rulesFrom[d.Name]; file != "" {
			cost[file] += len(renderDomain(d))
		}
	}
	if len(cost) == 0 {
		return "`abcd rules` lists them."
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
	return "Most of the words come from " + strings.Join(files, " or ") + "; `abcd rules` lists them."
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
// of the state folder across many sessions, and sweeps the fill directory
// (~/.abcd.noindex/fill/, fill.go) by the same age. Best-effort housekeeping:
// a missing folder or a per-entry error is ignored.
func PruneState(maxAge time.Duration) {
	pruneFill(UserHome(), maxAge)
	dir, _, _, err := stateFolder(false)
	if err != nil {
		return
	}
	defer dir.Close()
	pruneRoot(dir, maxAge)
}

// pruneRoot removes every entry directly in dir, other than a directory, last
// modified before maxAge ago. A per-entry error is ignored.
func pruneRoot(dir *os.Root, maxAge time.Duration) {
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
