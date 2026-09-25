// Package rules is abcd's transport-agnostic modular rules loader (itd-3). It
// owns the whole capability behind two front doors: the `abcd rules [domain]`
// CLI verb and the Claude Code prompt-router hook. Nothing here writes to stdout
// or knows about a harness event — the front doors under internal/surface and
// the hook entrypoint marshal these results for their transport.
//
// The model is a small set of binary-bundled default domains (embedded below)
// merged with two optional override layers, in order: the user scope's
// ~/.abcd/rules.json (one per machine, spc-23) and then the per-repo
// <repoRoot>/.abcd/rules.json, so the repo wins a field both set. Each
// domain carries recall keywords + aliases and a list of rules; a prompt is
// recall-matched against the active domains and only the matching rules are
// rendered for injection. A leading *<DOMAIN> star-command activates a domain
// unconditionally (overriding a dormant state, but never the top-level kill
// switch).
//
// Validation is hand-rolled Go (zero new dependencies): a rules.json that fails
// to parse or validate is a fail-closed error the caller surfaces loudly — it
// never silently degrades to zero injection.
package rules

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"syscall"
)

// RepoRelPath is the per-repo override file, relative to the repo worktree.
const RepoRelPath = ".abcd/rules.json"

// UserRelPath is the user-scope override file, relative to the home directory:
// the machine's own conventions, layered between the bundled defaults and every
// repo's override (itd-117, spc-23).
const UserRelPath = ".abcd/rules.json"

// UserDisplayPath is how the user-scope file is NAMED in a diagnostic: the tilde
// form, never the expanded path, so no message carries the developer-identity
// home path (iss-81, fsutil.RedactHome) and a refusal a user pastes still names
// the file.
const UserDisplayPath = "~/" + UserRelPath

// maxRulesFileBytes caps each rules.json, user scope and repo alike (trust
// boundary).
const maxRulesFileBytes = 256 * 1024

// Domain state values. An empty string is treated as active.
const (
	StateActive  = "active"
	StateDormant = "dormant"
)

// Domain is one keyed rule domain. The map key in RuleSet.Domains is its name;
// the name is therefore not a serialized field on the value.
type Domain struct {
	State   string   `json:"state,omitempty"`
	Recall  []string `json:"recall,omitempty"`
	Aliases []string `json:"aliases,omitempty"`
	Rules   []string `json:"rules,omitempty"`
}

// RuleSet is the merged, validated rule model: the bundled defaults overlaid
// with the per-repo override.
type RuleSet struct {
	SchemaVersion int               `json:"schema_version"`
	Disabled      bool              `json:"disabled"`
	Domains       map[string]Domain `json:"domains"`
	// origins records, per domain name, the layer whose override last named
	// it (SourceUser or SourceRepo). It is derived by the merge, never
	// declared in rules.json, and absent means the bundled default is
	// untouched. Unexported so the on-disk schema does not grow a field a file
	// could forge.
	origins map[string]string
	// killedBy records, in layer order, every layer whose file set the kill
	// switch, so a front door reporting "disabled" names the file to edit
	// rather than guessing. Derived like origins, and read through
	// KillSwitchSources.
	killedBy []string
	// notes carries the load-time diagnostics a front door must surface: one
	// line per domain Load dropped rather than failing the whole file over.
	// Unexported and non-serialized for the same reason as origins — it is
	// derived from this load, not declared by anyone — and read through
	// Notes().
	notes []string
}

// Notes returns the diagnostics this load produced, in a stable order: a
// front door prints them out-of-band (stderr, never the injected context) so a
// dropped domain is loud rather than silently missing. An empty result is the
// normal case.
func (rs RuleSet) Notes() []string { return append([]string(nil), rs.notes...) }

// KillSwitchSources returns the layers whose file set the kill switch, in layer
// order (SourceUser before SourceRepo); empty when the set is not disabled. The
// switch is sticky, so every layer named here has to clear it before anything
// injects again.
func (rs RuleSet) KillSwitchSources() []string { return append([]string(nil), rs.killedBy...) }

// LayerPath is the display path of the file a layer's source label names:
// UserDisplayPath for SourceUser, RepoRelPath for SourceRepo, and "" for the
// bundled defaults, which have no file.
func LayerPath(source string) string {
	switch source {
	case SourceUser:
		return UserDisplayPath
	case SourceRepo:
		return RepoRelPath
	}
	return ""
}

// Source labels for ResolvedDomain.Source: where a domain's effective content
// came from. An override that names a domain — replacing its rules, changing
// its state, or declaring it outright — makes the domain that override's layer,
// conservatively: the effective behaviour is that layer's choice even when only
// the state moved. The LAST layer to name a domain labels it, so a domain the
// user scope and the repo both name is SourceRepo.
const (
	SourceBundled = "bundled"
	SourceUser    = "user"
	SourceRepo    = "repo"
)

// ResolvedDomain pairs a domain with its name for ordered rendering and dedup,
// and with its Source so every consumer can say whose words these are
// (GHSA-22f8-qf5r-gjgq): the renderer marks a repo-sourced heading, the hook
// diagnostic labels the name, and `rules --json` carries the field.
type ResolvedDomain struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Domain
}

// domainNameRe constrains domain keys so a custom domain id can never be used
// to build a filesystem path (path-traversal defence) — uppercase, starting
// with a letter.
var domainNameRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// starCommandRe finds a candidate *<DOMAIN> token. Go's RE2 has no lookahead,
// so the pinned boundary semantics `(?:^|\s)\*([A-Z][A-Z0-9_]*)(?=$|\s)` are
// enforced by checking the surrounding bytes in parseStarCommands.
var starCommandRe = regexp.MustCompile(`\*([A-Z][A-Z0-9_]*)`)

// nonAlnumRe collapses every run of non-alphanumeric characters to one space so
// recall matching is word-bounded (no substring false positives).
var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

//go:embed defaults/rules.json
var defaultsJSON []byte

// defaultRuleSet is parsed once at init; a malformed embedded asset is a build
// error surfaced as a panic (it can never happen at runtime).
var defaultRuleSet = mustParseDefaults()

func mustParseDefaults() RuleSet {
	var rs RuleSet
	if err := json.Unmarshal(defaultsJSON, &rs); err != nil {
		panic("rules: bundled defaults are malformed: " + err.Error())
	}
	if err := Validate(rs); err != nil {
		panic("rules: bundled defaults fail validation: " + err.Error())
	}
	return rs
}

// Defaults returns a deep copy of the binary-bundled default rule set, safe for
// the caller to mutate.
func Defaults() RuleSet { return cloneRuleSet(defaultRuleSet) }

var (
	errNotRegular = errors.New("not a regular file")
	errTooBig     = errors.New("exceeds size cap")
)

// readGuarded opens path once, read-only, with O_NOFOLLOW (refuse a symlinked
// leaf) and O_NONBLOCK (a FIFO/device leaf returns immediately instead of
// blocking the open forever), then validates on the SAME file descriptor that it
// is a regular file within limit bytes before reading through a LimitReader — so
// no symlink swap between stat and read, no non-regular leaf, and no size overrun
// can reach the caller. The raw open error is returned so callers can test
// os.IsNotExist / syscall.ELOOP; a non-regular or oversize file returns the
// errNotRegular / errTooBig sentinel.
func readGuarded(path string, limit int64) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, errNotRegular
	}
	if fi.Size() > limit {
		return nil, errTooBig
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		// The file grew past the cap between fstat and read (a size TOCTOU).
		return nil, errTooBig
	}
	return data, nil
}

// Load returns the bundled defaults merged with the user scope's
// ~/.abcd/rules.json and then with <repoRoot>/.abcd/rules.json — bundled, then
// user, then repo, each layer overriding per field (itd-117, spc-23). An absent
// file contributes nothing, so a machine with neither file gets the defaults
// unchanged; a present file that cannot be read, parsed or validated is a
// fail-closed error naming that file, and no partial set is returned with it.
func Load(repoRoot string) (RuleSet, error) {
	home, _ := userHomeDir()
	user, haveUser, err := readUserLayer(home)
	if err != nil {
		return RuleSet{}, err
	}
	repo, haveRepo, err := readRepoLayer(repoRoot)
	if err != nil {
		return RuleSet{}, err
	}
	if !haveUser && !haveRepo {
		return Defaults(), nil
	}
	merged := Defaults()
	if haveUser {
		merged = mergeFrom(merged, user, SourceUser)
		// The user layer is validated on its own before the repo layer lands,
		// so a defect in it is reported against ITS file — never blamed on the
		// repo's — and is refused even where a repo override would have
		// replaced the offending field. The rules-present check is left to the
		// final set: a domain one layer declares without rules may gain them
		// from the next, and a ruleless survivor is skipped with a note below.
		if err := validate(merged, false); err != nil {
			return RuleSet{}, fmt.Errorf("rules: %s: %w", UserDisplayPath, err)
		}
	}
	if haveRepo {
		merged = Merge(merged, repo)
	}
	merged = dropRulelessDomains(merged)
	if err := Validate(merged); err != nil {
		// The user layer passed on its own, so what fails here arrived with the
		// repo's file.
		return RuleSet{}, fmt.Errorf("rules: %s: %w", RepoRelPath, err)
	}
	return merged, nil
}

// userHomeDir is the package's view of os.UserHomeDir, held as a var for the
// same reason fsutil keeps its owner lookup: a test suite must be able to keep
// the developer's own ~/.abcd/rules.json out of every test that did not lay one
// out, and only a substitution can do that for tests that never set HOME.
var userHomeDir = os.UserHomeDir

// SwapUserHomeForTest substitutes the home lookup Load reads the user layer
// through and returns the restore. It is exported because the front-door tests
// that load rules live in another package. Tests only; never called in
// production code, and never safe to call from a parallel test.
func SwapUserHomeForTest(fn func() (string, error)) (restore func()) {
	prev := userHomeDir
	userHomeDir = fn
	return func() { userHomeDir = prev }
}

// readRepoLayer reads and parses <repoRoot>/.abcd/rules.json. ok is false when
// the file is absent; every other failure is an error naming the file.
func readRepoLayer(repoRoot string) (over RuleSet, ok bool, err error) {
	// Refuse a symlinked .abcd directory component before touching the leaf, so a
	// swapped .abcd cannot redirect the read (trust boundary).
	if di, err := os.Lstat(filepath.Join(repoRoot, ".abcd")); err == nil && di.Mode()&os.ModeSymlink != 0 {
		return RuleSet{}, false, fmt.Errorf("rules: .abcd is a symlink (refusing to follow)")
	}
	path := filepath.Join(repoRoot, ".abcd", "rules.json")
	data, err := readGuarded(path, maxRulesFileBytes)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			return RuleSet{}, false, nil
		case errors.Is(err, syscall.ELOOP):
			return RuleSet{}, false, fmt.Errorf("rules: %s is a symlink (refusing to follow)", RepoRelPath)
		case errors.Is(err, errNotRegular):
			return RuleSet{}, false, fmt.Errorf("rules: %s is not a regular file", RepoRelPath)
		case errors.Is(err, errTooBig):
			return RuleSet{}, false, fmt.Errorf("rules: %s exceeds the %d-byte cap", RepoRelPath, maxRulesFileBytes)
		default:
			return RuleSet{}, false, fmt.Errorf("rules: reading %s: %w", RepoRelPath, err)
		}
	}
	over, err = parseLayer(data, RepoRelPath)
	if err != nil {
		return RuleSet{}, false, err
	}
	return over, true, nil
}

// readUserLayer reads and parses the user scope's ~/.abcd/rules.json. ok is
// false when there is no such file — or no home to hold one — which is the
// ordinary case and costs nothing: nothing is created, nothing is reported.
//
// The file injects text into every session on the machine, so it is read as the
// caller's WORD, through the same primitive as the other home-scoped
// declarations (fsutil.ReadDeclaration): a regular file — never a symlink,
// FIFO or device — within the size cap, owned by this session's uid and
// writable by nobody else. That is the ownership rule the repo root's trust
// bound applies to a foreign-uid checkout (root.go, foreignOwnerRefusal): a
// file another account could have written is not this account's convention.
// And the ~/.abcd directory itself must not be a symlink when a rules.json sits
// behind it, the pre-check the repo's .abcd gets — checked only once a file is
// there, because a machine whose ~/.abcd is a dotfiles symlink holding no
// rules.json reads nothing and must keep behaving exactly as it did.
//
// Every refusal is an error naming the file in tilde form, never a silent
// fallback to the defaults: an unreadable user layer that degraded to a partial
// injection is the silent shape loud-staging exists to prevent.
func readUserLayer(home string) (over RuleSet, ok bool, err error) {
	// No home, or a relative one, names no user scope: a relative HOME would
	// resolve ~/.abcd against whatever directory the session happens to start
	// in, which is not the machine's scope but a guess at one.
	if home == "" || !filepath.IsAbs(home) {
		return RuleSet{}, false, nil
	}
	dir := filepath.Join(home, ".abcd")
	path := filepath.Join(home, filepath.FromSlash(UserRelPath))
	data, refusal, err := fsutil.ReadDeclaration(path, maxRulesFileBytes)
	if refusal == fsutil.DeclarationAbsent && (os.IsNotExist(err) || errors.Is(err, syscall.ENOTDIR)) {
		return RuleSet{}, false, nil
	}
	if di, lerr := os.Lstat(dir); lerr == nil && di.Mode()&os.ModeSymlink != 0 {
		return RuleSet{}, false, fmt.Errorf("rules: ~/.abcd is a symlink (refusing to follow it to %s)", UserDisplayPath)
	}
	switch refusal {
	case fsutil.DeclarationOK:
	case fsutil.DeclarationNotRegular:
		return RuleSet{}, false, fmt.Errorf("rules: %s is not a regular file (a symlink, FIFO or device is refused)", UserDisplayPath)
	case fsutil.DeclarationWritableByOthers:
		return RuleSet{}, false, fmt.Errorf("rules: %s is writable by others, so its rules are not necessarily yours (chmod go-w it): %w", UserDisplayPath, err)
	case fsutil.DeclarationForeignOwner:
		return RuleSet{}, false, fmt.Errorf("rules: %s is not owned by this session's uid, so its rules are not this account's: %w", UserDisplayPath, err)
	default:
		if errors.Is(err, fsutil.ErrTooBig) {
			return RuleSet{}, false, fmt.Errorf("rules: %s exceeds the %d-byte cap", UserDisplayPath, maxRulesFileBytes)
		}
		if errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP) {
			return RuleSet{}, false, fmt.Errorf("rules: %s is not a regular file (a symlink, FIFO or device is refused)", UserDisplayPath)
		}
		// The raw error can carry the expanded path; the message names the
		// file in tilde form and keeps only the reason.
		why := "unknown error"
		var pe *os.PathError
		switch {
		case errors.As(err, &pe):
			why = pe.Err.Error()
		case err != nil:
			why = termsafe.Sanitize(fsutil.RedactHome(err.Error()))
		}
		return RuleSet{}, false, fmt.Errorf("rules: %s could not be read (%s)", UserDisplayPath, why)
	}
	over, err = parseLayer(data, UserDisplayPath)
	if err != nil {
		return RuleSet{}, false, err
	}
	return over, true, nil
}

// parseLayer turns one rules.json's bytes into its override set, naming the
// file (display) in every refusal.
func parseLayer(data []byte, display string) (RuleSet, error) {
	// encoding/json silently resolves a duplicate object key last-wins, so two
	// blocks for the same domain in rules.json would drop the first with no
	// diagnostic — an easy state to reach after a merge (iss-2608261550498779).
	// A token-level scan before the unmarshal refuses it loudly, mirroring
	// capture/parse.go's duplicate-key refusal (adapted to JSON's token stream).
	if err := checkNoDuplicateKeys(data); err != nil {
		return RuleSet{}, fmt.Errorf("rules: %s: %w", display, err)
	}
	var over RuleSet
	if err := json.Unmarshal(data, &over); err != nil {
		return RuleSet{}, fmt.Errorf("rules: %s is not valid JSON: %w", display, err)
	}
	return over, nil
}

// dropRulelessDomains removes every domain whose merged rules are empty and
// records one note per removal, naming the file of the layer that last named
// the domain (the user scope's or the repo's).
//
// A domain with no rules renders as a heading-only "## NAME" block —
// suppression wearing the domain's name, which an agent reads as a domain that
// says nothing (GHSA-22f8-qf5r-gjgq sibling). Validate refuses the shape, and
// keeps refusing it: it is what guards the BUNDLED defaults, where a ruleless
// domain is a build error with a build to fix it.
//
// A repo's rules.json is not that. `{"rules": []}` is a plausible way to have
// tried to silence a domain, the file already exists on somebody's disk, and
// failing the load stops EVERY domain injecting — the safety-shaped ones
// included — over one malformed entry, on an upgrade that changed the rules
// under a config that worked yesterday. That is not proportionate to a
// heading-only block. Dropping the domain removes the misleading render, keeps
// everything else working, and the note names the domain and the deliberate
// route ("state": "dormant"), which is the thing the author actually wanted.
func dropRulelessDomains(rs RuleSet) RuleSet {
	var dropped []string
	for name, d := range rs.Domains {
		if len(d.Rules) == 0 {
			dropped = append(dropped, name)
		}
	}
	if len(dropped) == 0 {
		return rs
	}
	sort.Strings(dropped) // deterministic diagnostics
	for _, name := range dropped {
		file := LayerPath(rs.origins[name])
		if file == "" {
			file = RepoRelPath
		}
		delete(rs.Domains, name)
		delete(rs.origins, name)
		rs.notes = append(rs.notes, fmt.Sprintf(
			"rules: %s: domain %q has no rules and was SKIPPED (it would inject a heading-only block, which reads as a domain that says nothing); "+
				"give it at least one rule, or set \"state\": \"dormant\" to silence a domain deliberately", file, name))
	}
	return rs
}

// checkNoDuplicateKeys walks the JSON token stream and refuses any object that
// carries a repeated key at any nesting level (the domains map and the domain
// objects alike). It runs before the unmarshal precisely because encoding/json
// would otherwise collapse the duplicate silently. The stdlib decoder enforces a
// max nesting depth, so no separate depth guard is needed.
func checkNoDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		// A malformed or empty document is left for the unmarshal to report.
		return nil
	}
	return checkDupValue(dec, tok)
}

// checkDupValue recursively verifies the value whose opening token is tok. For an
// object it tracks the keys seen at that level; for an array it descends into each
// element. Scalars terminate. Any read error is swallowed as nil so the richer
// json.Unmarshal error remains the one the caller surfaces.
func checkDupValue(dec *json.Decoder, tok json.Token) error {
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil // scalar
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return nil
			}
			key, ok := kt.(string)
			if !ok {
				return nil
			}
			if seen[key] {
				return fmt.Errorf("duplicate key %q (last-wins is silent — refusing)", key)
			}
			seen[key] = true
			vt, err := dec.Token()
			if err != nil {
				return nil
			}
			if err := checkDupValue(dec, vt); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil { // closing '}'
			return nil
		}
	case '[':
		for dec.More() {
			vt, err := dec.Token()
			if err != nil {
				return nil
			}
			if err := checkDupValue(dec, vt); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil { // closing ']'
			return nil
		}
	}
	return nil
}

// Merge overlays over onto base. Domain fields are per-field: a field set on the
// override wins; an absent field inherits the base (so {"state":"dormant"} on a
// default domain silences it while keeping its recall and rules). New domain
// keys are added. The kill switch is sticky (either side can enable it). Every
// domain the override names is recorded as SourceRepo — the one moment the
// origin is still known, so the renderer and the diagnostics can say so later.
func Merge(base, over RuleSet) RuleSet {
	return mergeFrom(base, over, SourceRepo)
}

// mergeFrom is Merge with the label the override's layer carries: the user
// layer merges with SourceUser and the repo layer with SourceRepo, through the
// same per-field rules.
func mergeFrom(base, over RuleSet, source string) RuleSet {
	out := cloneRuleSet(base)
	if over.SchemaVersion != 0 {
		out.SchemaVersion = over.SchemaVersion
	}
	out.Disabled = base.Disabled || over.Disabled
	if over.Disabled {
		out.killedBy = append(out.killedBy, source)
	}
	if out.Domains == nil && len(over.Domains) > 0 {
		out.Domains = make(map[string]Domain, len(over.Domains))
	}
	if out.origins == nil && len(over.Domains) > 0 {
		out.origins = make(map[string]string, len(over.Domains))
	}
	for name, od := range over.Domains {
		out.Domains[name] = mergeDomain(out.Domains[name], od)
		out.origins[name] = source
	}
	return out
}

// resolved builds the ResolvedDomain for name, carrying its recorded origin; a
// domain no override ever named is the bundled default.
func (rs RuleSet) resolved(name string) ResolvedDomain {
	src := rs.origins[name]
	if src == "" {
		src = SourceBundled
	}
	return ResolvedDomain{Name: name, Source: src, Domain: rs.Domains[name]}
}

func mergeDomain(base, over Domain) Domain {
	r := base
	if over.State != "" {
		r.State = over.State
	}
	if over.Recall != nil {
		r.Recall = append([]string(nil), over.Recall...)
	}
	if over.Aliases != nil {
		r.Aliases = append([]string(nil), over.Aliases...)
	}
	if over.Rules != nil {
		r.Rules = append([]string(nil), over.Rules...)
	}
	return r
}

// Validate checks structural invariants: schema_version == 1, every domain name
// matches [A-Z][A-Z0-9_]*, every state is active/dormant (or empty), every
// domain carries at least one rule, and no rule body is empty or
// whitespace-only. An empty rule body otherwise passes here and renders as a
// bare contentless "- " bullet in the injected block (iss-2608261550497978),
// and a domain with no rules at all — an override of {"rules": []}, or a custom
// domain declared without any — renders as a heading-only "## NAME" block,
// suppression wearing the domain's name (GHSA-22f8-qf5r-gjgq sibling). Neither
// silent shape is acceptable; {"state": "dormant"} is the way to silence a
// domain.
//
// The ruleless-domain rule is a REFUSAL here and a per-domain skip in Load. It
// refuses here because Validate is what guards the bundled defaults, where a
// heading-only domain is a build error. Load reaches Validate with the ruleless
// domains already dropped and a note naming each (see dropRulelessDomains),
// because failing a repo's whole file — every domain, on an upgrade — is not
// proportionate to one malformed entry.
func Validate(rs RuleSet) error { return validate(rs, true) }

// validate is Validate with the rules-present check optional: Load checks a
// layer before the next lands with it off, because a domain one layer declares
// without rules may take them from the next.
func validate(rs RuleSet, requireRules bool) error {
	if rs.SchemaVersion != 1 {
		return fmt.Errorf("schema_version must be 1, got %d", rs.SchemaVersion)
	}
	for name, d := range rs.Domains {
		if !domainNameRe.MatchString(name) {
			return fmt.Errorf("domain name %q must match [A-Z][A-Z0-9_]*", name)
		}
		switch d.State {
		case "", StateActive, StateDormant:
		default:
			return fmt.Errorf("domain %q: unknown state %q", name, d.State)
		}
		if requireRules && len(d.Rules) == 0 {
			return fmt.Errorf("domain %q: has no rules (it would inject a heading-only block; set \"state\": \"dormant\" to silence a domain)", name)
		}
		for i, r := range d.Rules {
			if strings.TrimSpace(r) == "" {
				return fmt.Errorf("domain %q: rule %d is empty or whitespace-only", name, i)
			}
		}
	}
	return nil
}

// Match returns the domains to inject for prompt, in deterministic (name-sorted)
// order. The top-level kill switch suppresses everything. Otherwise a domain is
// injected if a star-command names it (overriding dormant) or — when active —
// its recall keywords or aliases hit the prompt.
func (rs RuleSet) Match(prompt string) []ResolvedDomain {
	if rs.Disabled {
		return nil
	}
	stars := parseStarCommands(prompt)
	idx := indexPrompt(prompt)

	names := make([]string, 0, len(rs.Domains))
	for name := range rs.Domains {
		names = append(names, name)
	}
	sort.Strings(names)

	var out []ResolvedDomain
	for _, name := range names {
		d := rs.Domains[name]
		if stars[name] {
			out = append(out, rs.resolved(name))
			continue
		}
		if d.State == StateDormant {
			continue
		}
		if idx.hit(d) {
			out = append(out, rs.resolved(name))
		}
	}
	return out
}

// Active returns every injectable domain (state != dormant) in name-sorted
// order — the full set the diagnostic `abcd rules` render shows. The top-level
// kill switch yields nothing.
func (rs RuleSet) Active() []ResolvedDomain {
	if rs.Disabled {
		return nil
	}
	names := make([]string, 0, len(rs.Domains))
	for name := range rs.Domains {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []ResolvedDomain
	for _, name := range names {
		if rs.Domains[name].State != StateDormant {
			out = append(out, rs.resolved(name))
		}
	}
	return out
}

// Lookup returns one domain by name regardless of its state (a dormant domain is
// still inspectable); ok is false when the name is absent.
func (rs RuleSet) Lookup(name string) (ResolvedDomain, bool) {
	if _, ok := rs.Domains[name]; !ok {
		return ResolvedDomain{}, false
	}
	return rs.resolved(name), true
}

// parseStarCommands extracts the set of *<DOMAIN> names, enforcing that the star
// is at the start or preceded by whitespace and the name is followed by the end
// or whitespace (the RE2-safe form of the pinned lookahead boundary).
func parseStarCommands(prompt string) map[string]bool {
	out := map[string]bool{}
	for _, loc := range starCommandRe.FindAllStringSubmatchIndex(prompt, -1) {
		starStart, nameEnd := loc[0], loc[3]
		if starStart > 0 && !isSpace(prompt[starStart-1]) {
			continue
		}
		if nameEnd < len(prompt) && !isSpace(prompt[nameEnd]) {
			continue
		}
		out[prompt[loc[2]:loc[3]]] = true
	}
	return out
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v'
}

// promptIndex is a prompt prepared for recall matching once per Match call: a
// space-padded normalized form for word-boundary and multi-word matching, a
// second padded form with every token stemmed (so inflected multi-word phrases
// still hit their alias), plus a set of candidate stems for the single tokens so
// inflected forms (commits->commit, pushes->push, committing->commit,
// merging->merge) recall-match their keyword.
type promptIndex struct {
	padded        string          // " tok tok " for boundary/phrase matching
	stemmedPadded string          // " stem stem " for stemmed phrase matching
	stems         map[string]bool // candidate stems of the single tokens
}

// indexPrompt lowercases, collapses non-alphanumeric runs to single spaces, and
// builds both the stemmed-token set (with every candidate root per token) and the
// stemmed padded form used for multi-word phrase matching.
func indexPrompt(s string) promptIndex {
	collapsed := strings.TrimSpace(nonAlnumRe.ReplaceAllString(strings.ToLower(s), " "))
	idx := promptIndex{padded: " " + collapsed + " ", stems: map[string]bool{}}
	tokens := strings.Fields(collapsed)
	stemmed := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		for _, v := range stemVariants(tok) {
			idx.stems[v] = true
		}
		stemmed = append(stemmed, stem(tok))
	}
	idx.stemmedPadded = " " + strings.Join(stemmed, " ") + " "
	return idx
}

// hit reports whether any of a domain's recall keywords or aliases match.
func (idx promptIndex) hit(d Domain) bool {
	for _, term := range d.Recall {
		if idx.termHit(term) {
			return true
		}
	}
	for _, term := range d.Aliases {
		if idx.termHit(term) {
			return true
		}
	}
	return false
}

// termHit matches a single term. A multi-word term is a word-boundary substring
// of the padded prompt OR — with each of its words stemmed — of the stemmed
// padded prompt, so inflected phrases ("pull requests") still hit their alias
// ("pull request"). A single token matches on an exact word boundary OR when its
// stem is among the prompt tokens' candidate stems, so plural/tense variants
// recall their keyword.
func (idx promptIndex) termHit(term string) bool {
	t := strings.TrimSpace(nonAlnumRe.ReplaceAllString(strings.ToLower(term), " "))
	if t == "" {
		return false
	}
	if strings.Contains(t, " ") {
		if strings.Contains(idx.padded, " "+t+" ") {
			return true
		}
		return strings.Contains(idx.stemmedPadded, " "+stemPhrase(t)+" ")
	}
	if strings.Contains(idx.padded, " "+t+" ") {
		return true
	}
	return idx.stems[stem(t)]
}

// stemPhrase stems each whitespace-separated word of a normalized multi-word
// term, so a phrase alias can be compared against the stemmed padded prompt.
func stemPhrase(t string) string {
	words := strings.Fields(t)
	for i, w := range words {
		words[i] = stem(w)
	}
	return strings.Join(words, " ")
}

// stem strips a common English suffix to a root. Short tokens are left untouched
// so acronyms and 2–4 letter keywords (sota, pr, docs) are never over-stemmed —
// the guard that keeps stemming from matching e.g. "test" against "attestation".
func stem(w string) string {
	switch {
	case len(w) > 5 && strings.HasSuffix(w, "ing"):
		return w[:len(w)-3]
	case len(w) > 4 && strings.HasSuffix(w, "ed"):
		return w[:len(w)-2]
	case len(w) > 4 && strings.HasSuffix(w, "es"):
		// "es" plural attaches to sibilant roots (boxes->box, pushes->push);
		// elsewhere it is root+"s" (issues->issue), so only strip "es" after a
		// sibilant, otherwise drop just the trailing "s".
		root := w[:len(w)-2]
		if hasSibilantSuffix(root) {
			return root
		}
		return w[:len(w)-1]
	case len(w) > 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss"):
		return w[:len(w)-1]
	}
	return w
}

// hasSibilantSuffix reports whether a root takes an "-es" plural (s/x/z/ch/sh).
func hasSibilantSuffix(root string) bool {
	for _, suf := range []string{"s", "x", "z", "ch", "sh"} {
		if strings.HasSuffix(root, suf) {
			return true
		}
	}
	return false
}

// stemVariants returns every candidate root a prompt token may share with a base
// keyword. It is the asymmetric counterpart of stem(): a keyword stems to a
// single canonical root, while a prompt token expands to that root plus the two
// forms an "-ing"/"-ed" inflection would otherwise hide — the e-drop restore
// ("merg"->"merge", "rebas"->"rebase") and the undoubled consonant
// ("committ"->"commit"). Only the "-ing"/"-ed" cases branch, so plural handling
// and the short-token guard from stem() are unchanged, and the extra roots stay
// conservative (no bare-vowel or sub-3-char stems) to avoid over-matching.
func stemVariants(w string) []string {
	var root string
	switch {
	case len(w) > 5 && strings.HasSuffix(w, "ing"):
		root = w[:len(w)-3]
	case len(w) > 4 && strings.HasSuffix(w, "ed"):
		root = w[:len(w)-2]
	default:
		return []string{stem(w)}
	}
	out := []string{root, root + "e"}
	if u := undouble(root); u != "" {
		out = append(out, u)
	}
	return out
}

// undouble collapses a trailing doubled consonant to a single one
// ("committ"->"commit", "stopp"->"stop"), or returns "" when the root does not
// end in a doubled consonant. Such doubling is introduced when an "-ing"/"-ed"
// inflection is stripped from a verb whose final consonant was doubled. The
// undoubled root must stay at least 3 characters to avoid tiny, over-matching
// stems.
func undouble(root string) string {
	n := len(root)
	if n < 4 {
		return ""
	}
	if a, b := root[n-2], root[n-1]; a == b && isConsonant(a) {
		return root[:n-1]
	}
	return ""
}

// isConsonant reports whether b is an ASCII lowercase consonant.
func isConsonant(b byte) bool {
	if b < 'a' || b > 'z' {
		return false
	}
	switch b {
	case 'a', 'e', 'i', 'o', 'u':
		return false
	}
	return true
}

// Render is the single renderer both front doors use. It emits a header plus one
// section per domain. An empty domain list renders to zero bytes (D3: no
// model-facing tokens on a no-match).
func Render(domains []ResolvedDomain) string {
	if len(domains) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# abcd rules — %d domain(s) active\n", len(domains))
	for _, d := range domains {
		b.WriteString(renderDomain(d))
	}
	return b.String()
}

// renderDomain renders one domain's block deterministically. Signature hashes
// exactly this, so the format is the dedup unit — the provenance marker
// included, by design: a repo-sourced domain reads "## NAME (repo override)"
// and a user-scope one "## NAME (user override)" in the agent's context and in
// `abcd rules`, so whose words these are is never invisible
// (GHSA-22f8-qf5r-gjgq), and the one-time signature move on upgrade re-injects
// overridden domains once. The heading keeps its "## " line start, the split
// key host-side parsers rely on.
func renderDomain(d ResolvedDomain) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n", Label(d.Name, d.Source))
	for _, r := range d.Rules {
		body := sanitizeRuleBody(r)
		// Defence behind Validate's loud refusal: a body that sanitises to
		// nothing (empty or whitespace-only) never becomes a contentless "- "
		// bullet, even on an unvalidated render path (iss-2608261550497978).
		if body == "" {
			continue
		}
		fmt.Fprintf(&b, "- %s\n", body)
	}
	return b.String()
}

// sanitizeRuleBody makes a repo-controlled rule body safe to embed under its
// domain heading. The rendered "## NAME" lines are the injection contract
// host-side parsers rely on, and a rule body is untrusted input: control
// characters are stripped (newline and tab kept) and every continuation line
// is indented under the bullet, so a hostile rules.json cannot forge domain
// headings, forge sibling rules, or smuggle escapes into the rendered block.
func sanitizeRuleBody(r string) string {
	// CRLF collapses to LF (editor-neutral signatures); a lone CR, LINE
	// SEPARATOR and PARAGRAPH SEPARATOR become LF — they are line starts to
	// the host-side parser's multiline split (or overprint terminals), so the
	// heading-defusing indent below must apply everywhere a line starts.
	//
	// The normalisation runs BEFORE the trailing-whitespace trim: a body ending
	// in a raw CR/LS/PS would otherwise sail past a newline-only trim and gain
	// a stray terminator on the next render, breaking the fixed point.
	r = strings.ReplaceAll(r, "\r\n", "\n")
	r = strings.Map(func(c rune) rune {
		if c == '\n' {
			return c
		}
		if c == '\r' || c == '\u2028' || c == '\u2029' {
			return '\n'
		}
		return c
	}, r)
	// Trailing whitespace carries no meaning in a rule body, and the renderer
	// adds its own terminator — trimming here makes render a fixed point
	// across the pipeline boundary: rendered text written back as a rule body
	// re-renders byte-identically instead of gaining a newline per round-trip.
	r = strings.TrimRight(r, " \t\n")
	lines := strings.Split(r, "\n")
	for i, line := range lines {
		// The first line is defused by the "- " bullet prefix the renderer adds.
		// EVERY continuation line sits flush-left otherwise — Validate accepts a
		// bare newline and the normalisation above turns CR, LS and PS into one
		// — so each is indented two spaces, the markdown list-item continuation:
		// the line stays inside its bullet instead of reading as a loose
		// paragraph, a leading "- " cannot forge a sibling rule, and a leading
		// "#" cannot forge a heading. The contract defended is the line-start
		// one the host-side parser splits on ("## " flush-left); CommonMark
		// would still read a two-space "## x" as a heading, which is not the
		// boundary this block is parsed by. A line already indented keeps its
		// own deeper indent, and a blank line stays blank, so the pass is a
		// fixed point — rendered text fed back as a body re-renders unchanged.
		if i > 0 && line != "" && !strings.HasPrefix(line, "  ") {
			line = "  " + line
		}
		lines[i] = line
		// Terminal-display safety is the canonical termsafe mask: C0/C1
		// controls, bidi overrides and zero-width runes become '?', so a
		// hostile rule body cannot recolour, reorder, or hide rendered text.
		lines[i] = termsafe.Sanitize(lines[i])
	}
	return strings.Join(lines, "\n")
}

// Label is a domain name as every provenance surface prints it: bare for a
// bundled domain, "NAME (user override)" or "NAME (repo override)" for one an
// override layer named. The rendered heading and the hook's diagnostic both use
// it, so the two agree byte for byte.
func Label(name, source string) string {
	switch source {
	case SourceUser, SourceRepo:
		return name + " (" + source + " override)"
	}
	return name
}

// Signature is the per-domain dedup key: an FNV-1a hash of the rendered block,
// so identical rendered content (defaults or override) dedups and any content
// drift invalidates. FNV is sufficient — this is dedup, not security.
func Signature(d ResolvedDomain) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(renderDomain(d)))
	return fmt.Sprintf("%016x", h.Sum64())
}

func cloneRuleSet(rs RuleSet) RuleSet {
	out := RuleSet{SchemaVersion: rs.SchemaVersion, Disabled: rs.Disabled}
	out.notes = append([]string(nil), rs.notes...)
	out.killedBy = append([]string(nil), rs.killedBy...)
	if rs.origins != nil {
		out.origins = make(map[string]string, len(rs.origins))
		for name, src := range rs.origins {
			out.origins[name] = src
		}
	}
	if rs.Domains != nil {
		out.Domains = make(map[string]Domain, len(rs.Domains))
		for name, d := range rs.Domains {
			out.Domains[name] = Domain{
				State:   d.State,
				Recall:  append([]string(nil), d.Recall...),
				Aliases: append([]string(nil), d.Aliases...),
				Rules:   append([]string(nil), d.Rules...),
			}
		}
	}
	return out
}
