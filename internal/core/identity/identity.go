// Package identity checks that the git author identity a commit would use in a
// managed repo matches the identity pinned in .abcd/config/identity.json.
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

// Result carries the comparison outcome and both identities for reporting.
type Result struct {
	Status    Status
	Pin       Pin
	Effective Effective
	Reason    string
}

// Blocks reports whether a pre-commit hook should refuse the commit. A mismatch
// or an unset identity blocks; a match, or an un-pinned (opted-out) repo, does
// not — an absent pin must never break commits in a repo that has not adopted
// the gate.
func (r Result) Blocks() bool {
	return r.Status == StatusMismatch || r.Status == StatusUnset
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

// EffectiveIdentity returns the author identity git would stamp on a commit in
// root. Git gives the GIT_AUTHOR_NAME/GIT_AUTHOR_EMAIL environment variables
// HIGHER precedence than user.name/user.email config, so an agent/CI sandbox that
// exports them lands a mis-attributed commit that a config-only check would wave
// through. Each field is therefore resolved from its GIT_AUTHOR_* override first,
// falling back to `git config` (local > global > system) when the override is
// unset or blank. Unset name or email yields an empty field, not an error.
func EffectiveIdentity(root string) (Effective, error) {
	name := strings.TrimSpace(os.Getenv("GIT_AUTHOR_NAME"))
	if name == "" {
		var err error
		if name, err = gitConfig(root, "user.name"); err != nil {
			return Effective{}, err
		}
	}
	email := strings.TrimSpace(os.Getenv("GIT_AUTHOR_EMAIL"))
	if email == "" {
		var err error
		if email, err = gitConfig(root, "user.email"); err != nil {
			return Effective{}, err
		}
	}
	return Effective{Name: name, Email: email}, nil
}

// gitConfig returns the trimmed value of a git config key, or "" when the key is
// unset. Git exits 1 for an unset key; that is not an error here. Any other
// failure (git absent, not a repo) is returned.
func gitConfig(root, key string) (string, error) {
	cmd := exec.Command("git", "-C", root, "config", "--get", key)
	// Scrub repo-selection and config-injection env vars, but keep global config:
	// this reads the caller's real user.name/user.email (which live in ~/.gitconfig)
	// to enforce the commit-identity gate, so full IsolatedEnv would blind it.
	// Scrubbing still stops an inherited GIT_DIR redirecting the read at another repo
	// and an injected GIT_CONFIG_* forging the identity the gate is meant to verify.
	cmd.Env = gitutil.ScrubbedEnv()
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return "", nil // unset key
		}
		return "", fmt.Errorf("git config %s: %w", key, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Check resolves the effective identity, loads the pin, and compares them.
func Check(root string) (Result, error) {
	pin, pinned, err := LoadPin(root)
	if err != nil {
		return Result{}, err
	}
	eff, err := EffectiveIdentity(root)
	if err != nil {
		return Result{}, err
	}
	if !pinned {
		return Result{Status: StatusNoPin, Effective: eff, Reason: "no " + PinRelPath + "; repo has not adopted the identity gate"}, nil
	}
	if eff.Name == "" || eff.Email == "" {
		return Result{Status: StatusUnset, Pin: pin, Effective: eff, Reason: "git author identity is not configured (user.name/user.email)"}, nil
	}
	if eff.Name != pin.Name || eff.Email != pin.Email {
		return Result{
			Status: StatusMismatch, Pin: pin, Effective: eff,
			Reason: fmt.Sprintf("commit identity %q <%s> does not match the pin %q <%s>", eff.Name, eff.Email, pin.Name, pin.Email),
		}, nil
	}
	return Result{Status: StatusOK, Pin: pin, Effective: eff}, nil
}
