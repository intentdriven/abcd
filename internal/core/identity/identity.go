// Package identity checks that the git author and committer identities a commit
// would use in a managed repo match the identity pinned in
// .abcd/config/identity.json, and recognises a machine identity in either role.
//
// It is the single source of truth for the iss-62 managed-repo identity gate:
// `ahoy doctor` surfaces a divergence as a detection gap, and the installed
// pre-commit hook calls Check to fail closed before a mis-attributed commit can
// land — so a stray repo-local override (e.g. a sandbox "Test User") is caught
// up front rather than discovered later.
package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/provenance"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// PinRelPath is the committed identity pin, relative to the repo root.
const PinRelPath = ".abcd/config/identity.json"

// Pin is the expected commit identity, committed so every checkout enforces the
// same value regardless of local git config.
type Pin struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	// ProductionMode is the repo's DEFAULT production mode: how the text of a
	// record this repository mints was produced, unless the minting verb says
	// otherwise (itd-178). It rides the pin rather than a second config file
	// because this is already the repo's attribution seam and already has a
	// reader; a second file would be a second reader of the same question.
	//
	// Optional and omitted when empty, so a pin written before the member existed
	// round-trips byte-identically. An absent member means provenance.DefaultMode.
	// The self-contained pre-commit identity guard seds `name` and `email` out of
	// this file by name, so an added member is invisible to it.
	ProductionMode string `json:"production_mode,omitempty"`
}

// Effective is the identity git would actually stamp on a commit in the repo,
// resolved as git resolves the author: the GIT_AUTHOR_NAME/GIT_AUTHOR_EMAIL
// environment overrides first, then git config's local > global > system
// layering.
type Effective struct {
	Name  string
	Email string
}

// Status is the outcome of comparing the effective identity to the pin.
type Status int

const (
	// StatusOK: a pin exists and the effective identity matches it.
	StatusOK Status = iota
	// StatusNoPin: no identity.json — the repo has not opted into the gate.
	StatusNoPin
	// StatusMismatch: a pin exists and the effective identity differs.
	StatusMismatch
	// StatusUnset: a pin exists but git has no author identity configured.
	StatusUnset
)

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "ok"
	case StatusNoPin:
		return "no-pin"
	case StatusMismatch:
		return "mismatch"
	case StatusUnset:
		return "unset"
	default:
		return "unknown"
	}
}

// Result carries the comparison outcome and the identities for reporting.
//
// Status and Effective describe the AUTHOR, with the meaning they have always
// had. The committer is reported beside them rather than folded into Status, so
// every existing reader of the author statuses reads exactly what it did.
type Result struct {
	Status    Status
	Pin       Pin
	Effective Effective
	Reason    string

	// Committer is the committer identity git would stamp (EffectiveCommitter).
	Committer Effective
	// CommitterDiverges reports a committer that differs from the author and is
	// not the pinned identity either: a GIT_COMMITTER_* override, a committer.*
	// config key, or an author override the committer does not share. A
	// committer that is the same wrong identity as the author is not reported
	// again — the author status already says so, and one fix mends both.
	CommitterDiverges bool
	// CommitterReason says what diverges, for the person reading the gap.
	CommitterReason string

	// AuthorIsTool and CommitterIsTool report a machine identity in that role
	// (IsToolIdentity), pinned or not: the routine case, made visible.
	AuthorIsTool    bool
	CommitterIsTool bool
}

// Blocks reports whether a pre-commit hook should refuse the commit. A mismatch
// or an unset author identity blocks, and so does a committer that diverges from
// the pin; a match, or an un-pinned (opted-out) repo, does not — an absent pin
// must never break commits in a repo that has not adopted the gate.
func (r Result) Blocks() bool {
	if r.Status == StatusMismatch || r.Status == StatusUnset {
		return true
	}
	return r.Status != StatusNoPin && r.CommitterDiverges
}

// LoadPin reads .abcd/config/identity.json. It returns (pin, true, nil) when the
// pin is present and well formed, (Pin{}, false, nil) when the file is absent,
// and an error when it is malformed or missing a field — validating this
// external input rather than trusting it.
func LoadPin(root string) (Pin, bool, error) {
	path := filepath.Join(root, PinRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Pin{}, false, nil
		}
		return Pin{}, false, fmt.Errorf("reading %s: %w", PinRelPath, err)
	}
	var p Pin
	if err := json.Unmarshal(data, &p); err != nil {
		return Pin{}, false, fmt.Errorf("malformed %s: %w", PinRelPath, err)
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Email = strings.TrimSpace(p.Email)
	if p.Name == "" || p.Email == "" {
		return Pin{}, false, fmt.Errorf("%s must set both name and email", PinRelPath)
	}
	// The optional member is validated at the boundary exactly as a malformed pin
	// is: a value outside the closed set would otherwise be stamped, unread, onto
	// every record the repo mints.
	p.ProductionMode = strings.TrimSpace(p.ProductionMode)
	if p.ProductionMode != "" {
		if _, err := provenance.ParseMode(p.ProductionMode); err != nil {
			return Pin{}, false, fmt.Errorf("malformed %s: %w", PinRelPath, err)
		}
	}
	return p, true, nil
}

// DeclaredProductionMode is the repo's default production mode: the pin's
// optional member, or provenance.DefaultMode when the repo has no pin or the pin
// declares none. It is the ONE resolution of that question, so no surface
// re-derives "absent means hand-written" for itself.
func DeclaredProductionMode(root string) (provenance.Mode, error) {
	pin, pinned, err := LoadPin(root)
	if err != nil {
		return "", err
	}
	if !pinned {
		return provenance.DefaultMode, nil
	}
	return provenance.ModeOrDefault(pin.ProductionMode)
}

// WritePin writes the pin to .abcd/config/identity.json (creating the config
// directory), pretty-printed with a trailing newline. It is how a repo adopts
// the identity gate. Both fields are required.
func WritePin(root string, p Pin) error {
	p.Name = strings.TrimSpace(p.Name)
	p.Email = strings.TrimSpace(p.Email)
	if p.Name == "" || p.Email == "" {
		return fmt.Errorf("identity pin requires both name and email")
	}
	// The optional member is refused here on the same terms LoadPin refuses it,
	// so the writer can never store a pin its own reader rejects.
	p.ProductionMode = strings.TrimSpace(p.ProductionMode)
	if p.ProductionMode != "" {
		if _, err := provenance.ParseMode(p.ProductionMode); err != nil {
			return fmt.Errorf("identity pin: %w", err)
		}
	}
	// The self-contained pre-commit identity guard reads the pin with a naive
	// sed that captures the raw bytes between the JSON quotes and compares them
	// literally to `git config`, so the stored value must round-trip through that
	// sed. Two things ensure it (iss-63): the pin is marshalled WITHOUT HTML
	// escaping (below), so &, <, > — legal in a git user.name like
	// "Marks & Spencer" — are stored literally rather than escaped; and the
	// characters JSON must escape regardless (a double-quote, a backslash, or a
	// control character), which the sed can never read back, are refused here so
	// a pin can never hold one and fail-close a correct identity. This keeps the
	// hook zero-dependency rather than delegating the gate to a possibly-stale
	// binary.
	if unpinnable(p.Name) {
		return fmt.Errorf("identity pin name must not contain a double-quote, backslash, or control character (it breaks the self-contained pre-commit identity guard); adjust git config user.name")
	}
	if unpinnable(p.Email) {
		return fmt.Errorf("identity pin email must not contain a double-quote, backslash, or control character; adjust git config user.email")
	}
	// Marshal WITHOUT HTML escaping (so &, <, > survive literally) and route the
	// bytes through the canonical atomic primitive (temp + fchmod + fsync +
	// rename + parent-dir fsync): a plain in-place os.WriteFile truncates the pin
	// before rewriting it — a crash mid-write leaves a corrupt or empty
	// identity.json — and it follows a symlink at path. json.Encoder appends the
	// trailing newline.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(p); err != nil {
		return err
	}
	// Contain the write under an os.Root opened at the repo: PinRelPath joins
	// `.abcd/config/identity.json`, and a committed `.abcd` ancestor symlink would
	// otherwise land the pin outside the working tree (GHSA-xrf8-4432-gw2f). The
	// InRoot writer resolves every component through the root — creating the
	// missing config/ parent included — so a symlinked ancestor is refused rather
	// than followed. The leaf's own symlink is still replaced, not written through.
	osRoot, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer osRoot.Close()
	return fsutil.WriteFileAtomicInRoot(osRoot, PinRelPath, buf.Bytes(), 0o644)
}

// unpinnable reports whether s holds a character the identity pin cannot safely
// carry: a double-quote or backslash (which JSON must always escape) or a
// control character — none of which the self-contained pre-commit hook's sed can
// read back to compare against `git config` (iss-63). Characters like &, <, >
// are pinnable because WritePin marshals without HTML escaping.
func unpinnable(s string) bool {
	for _, r := range s {
		if r == '"' || r == '\\' || r < 0x20 {
			return true
		}
	}
	return false
}

// Role is one of the two identities git stamps on a commit. The author is who
// wrote the change and the committer is who recorded it; the contributor graph
// reads both, so the gate resolves both.
type Role string

const (
	// RoleAuthor is the commit's author (%an/%ae).
	RoleAuthor Role = "author"
	// RoleCommitter is the commit's committer (%cn/%ce).
	RoleCommitter Role = "committer"
)

// EffectiveIdentity returns the author identity git would stamp on a commit in
// root. Each field is resolved the way git resolves it: the GIT_AUTHOR_NAME /
// GIT_AUTHOR_EMAIL environment override first (an agent or CI sandbox that
// exports one lands a mis-attributed commit a config-only check would wave
// through), then the role's own author.name / author.email config key, which git
// ranks ahead of user.* (iss-2609261454332615), then user.name / user.email.
// Config keys follow git's layering: command-line configuration (`git -c`, as a
// hook inherits it), then local, global and system. An unset name or email
// yields an empty field, not an error.
func EffectiveIdentity(root string) (Effective, error) {
	return effective(root, RoleAuthor)
}

// EffectiveCommitter returns the committer identity git would stamp on a commit
// in root, resolved exactly as EffectiveIdentity resolves the author:
// GIT_COMMITTER_NAME / GIT_COMMITTER_EMAIL first, then committer.name /
// committer.email, then user.name / user.email. An unset field is empty, never
// fabricated.
func EffectiveCommitter(root string) (Effective, error) {
	return effective(root, RoleCommitter)
}

// effective resolves one role's identity field by field, with the precedence
// git itself applies: GIT_<ROLE>_NAME / GIT_<ROLE>_EMAIL, then <role>.name /
// <role>.email, then user.name / user.email. It deliberately does NOT ask
// `git var GIT_<ROLE>_IDENT`, which fabricates an identity from the account's
// gecos field and the hostname when none is configured and exits 0 — that would
// collapse the distinct StatusUnset state the pre-commit hook blocks on.
func effective(root string, role Role) (Effective, error) {
	env := "GIT_" + strings.ToUpper(string(role)) + "_"
	name, err := resolveField(root, env+"NAME", string(role)+".name", "user.name")
	if err != nil {
		return Effective{}, err
	}
	email, err := resolveField(root, env+"EMAIL", string(role)+".email", "user.email")
	if err != nil {
		return Effective{}, err
	}
	return Effective{Name: name, Email: email}, nil
}

// resolveField returns the first non-blank of an environment override and then
// each config key in turn. A blank override is treated as unset.
func resolveField(root, envKey string, keys ...string) (string, error) {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v, nil
	}
	for _, key := range keys {
		v, err := gitConfig(root, key)
		if err != nil || v != "" {
			return v, err
		}
	}
	return "", nil
}

// gitConfig returns the trimmed value of a git config key, or "" when the key is
// unset. Git exits 1 for an unset key; that is not an error here. Any other
// failure (git absent, not a repo) is returned.
func gitConfig(root, key string) (string, error) {
	cmd := exec.Command("git", "-C", root, "config", "--get", key)
	cmd.Env = commitConfigEnv()
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return "", nil // unset key
		}
		return "", fmt.Errorf("git config %s: %w", key, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// commitConfigEnv is the environment the identity read runs git under: the
// configuration git will commit with, and nothing that points it elsewhere.
//
// It starts from gitutil.ScrubbedEnv, which keeps the caller's global config
// (where user.name/user.email usually live, so full IsolatedEnv would blind the
// gate) and drops the repo-selection variables, so an inherited GIT_DIR cannot
// redirect the read at another repository. It also drops the legacy GIT_CONFIG
// file variable, which only `git config` reads and `git commit` ignores.
//
// This reader is the one exception to that scrub: it puts back
// GIT_CONFIG_PARAMETERS and the GIT_CONFIG_COUNT/GIT_CONFIG_KEY_n/
// GIT_CONFIG_VALUE_n form (iss-2609261614306830). The caller that matters is
// git itself. `git -c committer.name=X commit` hands its hooks exactly these
// variables, and git commits with them, so they are the commit's real
// configuration, not an injection. Scrubbing them made `abcd ahoy --identity`
// in a hook report the configured identity as ok while git stamped X. The
// scrub's other reason, that an injected value could forge the identity the
// gate verifies, does not hold for this reader: it already honours
// GIT_AUTHOR_*/GIT_COMMITTER_*, which any process able to set these variables
// can set as well, and a `git config --get` read executes nothing a parameter
// names. Every other ScrubbedEnv caller keeps the scrub. The redaction probe
// keeps it for the identity it resolves, because there a displacing value
// hides the real identity instead of reporting it; it reads these variables
// only for extra identities to redact (iss-2609261614450166).
func commitConfigEnv() []string {
	return append(gitutil.ScrubbedEnv(), gitutil.CommandLineConfig()...)
}

// Check resolves the effective author and committer, loads the pin, and
// compares them.
func Check(root string) (Result, error) {
	pin, pinned, err := LoadPin(root)
	if err != nil {
		return Result{}, err
	}
	eff, err := EffectiveIdentity(root)
	if err != nil {
		return Result{}, err
	}
	committer, err := EffectiveCommitter(root)
	if err != nil {
		return Result{}, err
	}
	res := authorResult(pin, pinned, eff)
	res.Committer = committer
	res.CommitterDiverges, res.CommitterReason = committerDivergence(pin, pinned, eff, committer)
	res.AuthorIsTool = eff != (Effective{}) && IsToolIdentity(RoleAuthor, eff.Name, eff.Email)
	res.CommitterIsTool = committer != (Effective{}) && IsToolIdentity(RoleCommitter, committer.Name, committer.Email)
	return res, nil
}

// authorResult is the author half of Check, unchanged in meaning.
func authorResult(pin Pin, pinned bool, eff Effective) Result {
	if !pinned {
		return Result{Status: StatusNoPin, Effective: eff, Reason: "no " + PinRelPath + "; repo has not adopted the identity gate"}
	}
	if eff.Name == "" || eff.Email == "" {
		return Result{Status: StatusUnset, Pin: pin, Effective: eff, Reason: "git author identity is not configured (user.name/user.email)"}
	}
	if eff.Name != pin.Name || eff.Email != pin.Email {
		return Result{
			Status: StatusMismatch, Pin: pin, Effective: eff,
			Reason: fmt.Sprintf("commit identity %q <%s> does not match the pin %q <%s>", eff.Name, eff.Email, pin.Name, pin.Email),
		}
	}
	return Result{Status: StatusOK, Pin: pin, Effective: eff}
}

// committerDivergence is the committer half: a committer that differs from the
// author, unless it is the pinned identity (then the author is the one that is
// wrong, and the author status says so).
func committerDivergence(pin Pin, pinned bool, author, committer Effective) (bool, string) {
	if committer == author {
		return false, ""
	}
	if pinned && committer.Name == pin.Name && committer.Email == pin.Email {
		return false, ""
	}
	if committer.Name == "" || committer.Email == "" {
		return true, fmt.Sprintf("git committer identity is not configured, while the author is %q <%s>", author.Name, author.Email)
	}
	if pinned {
		return true, fmt.Sprintf("committer %q <%s> does not match the pin %q <%s>", committer.Name, committer.Email, pin.Name, pin.Email)
	}
	return true, fmt.Sprintf("committer %q <%s> differs from the author %q <%s>", committer.Name, committer.Email, author.Name, author.Email)
}
