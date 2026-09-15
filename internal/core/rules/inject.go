package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
// came from (SourceBundled or SourceRepo): the names alone cannot, and the
// out-of-band diagnostic must say whose words went into the context.
type InjectResult struct {
	Text     string
	Injected []string
	Sources  map[string]string
	State    SessionState
}

// Labels returns the injected names in order, each labelled the way the
// rendered heading is — "PII (repo override)" for a repo-sourced domain, the
// bare name otherwise — so the diagnostic and the block agree byte for byte.
func (r InjectResult) Labels() []string {
	out := make([]string, 0, len(r.Injected))
	for _, name := range r.Injected {
		if r.Sources[name] == SourceRepo {
			out = append(out, name+" (repo override)")
			continue
		}
		out = append(out, name)
	}
	return out
}

// SessionState is the per-session dedup ledger plus the prompt counter that
// drives the fixed-N refresh backstop.
type SessionState struct {
	Count  int               `json:"count"`
	Ledger map[string]string `json:"ledger"` // domain name -> last-injected signature
}

// Inject is the pure heart of the prompt-router hook: it recall-matches prompt,
// drops any matched domain already injected this session with an unchanged
// signature, renders the remainder, and returns the updated session state. It
// performs no I/O and never reflects prompt bytes into the output — the rendered
// text is abcd's own rule content only.
//
// backstop is the fixed-N full-refresh interval; <= 0 uses DefaultRefreshBackstop.
// When the (incremented) prompt count is a multiple of the backstop, the ledger
// is cleared first so always-relevant domains re-inject.
func Inject(rs RuleSet, prompt string, prev SessionState, backstop int) InjectResult {
	if backstop <= 0 {
		backstop = DefaultRefreshBackstop
	}
	if len(prompt) > maxPromptBytes {
		prompt = prompt[:maxPromptBytes]
	}

	ledger := map[string]string{}
	for k, v := range prev.Ledger {
		ledger[k] = v
	}
	count := prev.Count + 1
	if count%backstop == 0 {
		ledger = map[string]string{}
	}

	var fresh []ResolvedDomain
	for _, d := range rs.Match(prompt) {
		if ledger[d.Name] == Signature(d) {
			continue // already injected this session, unchanged
		}
		fresh = append(fresh, d)
	}

	// Enforce the per-repo injection budget across ALL matched domains, not just
	// the per-file read cap (iss-2608261551077971). A domain that is truncated
	// away is deliberately NOT recorded in the ledger, so it is retried next turn
	// rather than silently dropped for the session.
	text, kept := renderWithinBudget(fresh)
	var injected []string
	sources := make(map[string]string, len(kept))
	for _, d := range kept {
		ledger[d.Name] = Signature(d)
		injected = append(injected, d.Name)
		sources[d.Name] = d.Source
	}
	sort.Strings(injected)

	return InjectResult{
		Text:     text,
		Injected: injected,
		Sources:  sources,
		State:    SessionState{Count: count, Ledger: ledger},
	}
}

// renderWithinBudget renders fresh under the per-repo injection budget. When the
// full render fits it is returned byte-identical to Render (the common path is
// untouched). When it overflows, whole domain blocks are kept greedily in order
// until the next would exceed the budget, and a loud, unmistakable truncation
// notice is appended — whole-block granularity keeps the "## NAME" injection
// contract intact (never a half-rendered rule). It returns the text and the
// domains actually kept, so the caller records only those in the dedup ledger.
func renderWithinBudget(fresh []ResolvedDomain) (string, []ResolvedDomain) {
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
		"\n# abcd rules — %s: rendered rule set exceeds the %d-byte per-repo budget; %d of %d matched domain(s) omitted this turn. A bloated or hostile .abcd/rules.json is the likely cause — inspect it with `abcd rules`.\n",
		injectionTruncatedMarker, injectionBudgetBytes, len(fresh)-len(kept), len(fresh))
	return b.String(), kept
}

// stateDir is the machine-local directory holding per-session ledgers. It is
// overridable via ABCD_RULES_STATE_DIR (used by tests and by operators who want
// the state elsewhere). The default is the per-user cache dir, NOT the
// world-writable shared temp dir: a predictable path under a shared /tmp lets a
// local co-tenant pre-create or poison the session-state file and suppress rule
// injection fail-open. The uid-qualified temp fallback (only if the user cache
// dir is unavailable — a degraded env with no HOME/XDG_CACHE_HOME) still lives
// under the shared temp dir and is best-effort only: it avoids a cross-user
// collision, not co-tenant pre-creation. State is fail-open advisory dedup, so a
// poisoned fallback at worst re-injects; it never blocks or corrupts.
func stateDir() string {
	if d := os.Getenv("ABCD_RULES_STATE_DIR"); d != "" {
		return d
	}
	if cache, err := os.UserCacheDir(); err == nil {
		return filepath.Join(cache, "abcd-rules-state")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("abcd-rules-state-%d", os.Getuid()))
}

// sessionFile maps a session id to a state file. The id is hashed, so an
// attacker-supplied session id can never traverse out of the state dir.
func sessionFile(session string) string {
	sum := sha256.Sum256([]byte(session))
	return filepath.Join(stateDir(), hex.EncodeToString(sum[:])+".json")
}

// LoadState reads the ledger for a session. A missing, oversized, or malformed
// file yields the zero state (a fresh session), never an error — dedup is a
// best-effort optimisation, not a correctness gate.
func LoadState(session string) SessionState {
	data, err := readGuarded(sessionFile(session), maxStateFileBytes)
	if err != nil {
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
// fsutil.WriteFileAtomic (temp-write, fsync, rename, parent-dir fsync), so the
// write is crash-safe and a reader sees either the old file or the complete new
// one. The state dir is pre-created 0700 (WriteFileAtomic's own MkdirAll is a
// no-op once it exists); the session file is 0600.
func SaveState(session string, st SessionState) error {
	if err := os.MkdirAll(stateDir(), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(sessionFile(session), data, 0o600)
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
// prompt re-injects every matching domain). A missing file is not an error.
func ResetState(session string) error {
	err := os.Remove(sessionFile(session))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// PruneState removes session-state files older than maxAge, bounding the growth
// of the temp state dir across many sessions. Best-effort housekeeping: a
// missing dir or a per-entry error is ignored.
func PruneState(maxAge time.Duration) {
	entries, err := os.ReadDir(stateDir())
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
			_ = os.Remove(filepath.Join(stateDir(), e.Name()))
		}
	}
}
