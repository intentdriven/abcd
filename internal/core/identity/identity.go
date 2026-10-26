// Package identity checks that the git author and committer identities a commit
// would use match the person's OWN identity record, kept on their own machine
// under the user-level home (record.go), and recognises a machine identity in
// either role. Nothing in a repository names who may commit to it
// (itd-2610240918547728, adr-2610241022541063); a repository that still
// stores the identity pin itd-131 introduced is migrated (migrate.go).
//
// It is the single source of truth for the managed-repo identity gate:
// `ahoy doctor` surfaces a divergence as a detection gap, and `abcd ahoy
// --identity` exits non-zero on one, so a stray repo-local override (e.g. a
// sandbox "Test User") or a harness's machine identity is caught before the
// commit rather than discovered later.
package identity

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// Effective is the identity git would actually stamp on a commit in the repo,
// resolved as git resolves the author: the GIT_AUTHOR_NAME/GIT_AUTHOR_EMAIL
// environment overrides first, then git config's local > global > system
// layering.
type Effective struct {
	Name  string
	Email string
}

// Status is the outcome of comparing the effective identity to the person's
// own record.
type Status int

const (
	// StatusOK: a record applies and the effective identity matches it.
	StatusOK Status = iota
	// StatusNoRecord: no record on this machine applies to the repo. Nothing
	// to compare against; a machine identity is still refused (Blocks).
	StatusNoRecord
	// StatusMismatch: a record applies and the effective identity differs.
	StatusMismatch
	// StatusUnset: git has no author identity configured.
	StatusUnset
)

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "ok"
	case StatusNoRecord:
		return "no-record"
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
// Status and Effective describe the AUTHOR. The committer is reported beside
// them rather than folded into Status.
type Result struct {
	Status Status
	// Record is the person's own record, when one applies (Recorded).
	Record   Record
	Recorded bool
	// RecordAt names the record that applies, or would: the repository's own
	// record when one exists, else the machine-wide default.
	RecordAt  RecordRef
	Effective Effective
	Reason    string

	// Committer is the committer identity git would stamp (effectiveCommitter).
	Committer Effective
	// CommitterDiverges reports a committer that differs from the author and is
	// not the recorded identity either: a GIT_COMMITTER_* override, a
	// committer.* config key, or an author override the committer does not
	// share. A committer that is the same wrong identity as the author is not
	// reported again — the author status already says so, and one fix mends
	// both.
	CommitterDiverges bool
	// CommitterReason says what diverges, for the person reading the gap.
	CommitterReason string

	// AuthorIsTool and CommitterIsTool report a machine identity in that role
	// (IsToolIdentity), recorded or not.
	AuthorIsTool    bool
	CommitterIsTool bool
}

// Blocks reports whether a pre-commit hook should refuse the commit, on the
// shell guard's terms: a mismatch with the person's own record, an unset
// author, or a machine identity in either role blocks wherever it is found;
// a committer that diverges blocks when a record applies. With no record, a
// human identity — whatever launched the run supplied — commits.
func (r Result) Blocks() bool {
	if r.Status == StatusMismatch || r.Status == StatusUnset {
		return true
	}
	if r.AuthorIsTool || r.CommitterIsTool {
		return true
	}
	return r.Recorded && r.CommitterDiverges
}

// unpinnable reports whether s holds a character the identity record cannot
// safely carry: a double-quote or backslash (which JSON must always escape) or
// a control character — none of which the self-contained pre-commit hook's sed
// can read back to compare against `git config` (iss-63). Characters like &,
// <, > are fine because records are marshalled without HTML escaping.
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

// effectiveCommitter returns the committer identity git would stamp on a commit
// in root, resolved exactly as EffectiveIdentity resolves the author:
// GIT_COMMITTER_NAME / GIT_COMMITTER_EMAIL first, then committer.name /
// committer.email, then user.name / user.email. An unset field is empty, never
// fabricated.
func effectiveCommitter(root string) (Effective, error) {
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

// Check resolves the effective author and committer, loads the person's own
// record for the repository at root (from the home directory os.UserHomeDir
// names), and compares them.
func Check(root string) (Result, error) {
	return checkWithHome(root, userHome())
}

// checkWithHome is Check for the person whose home directory is home.
func checkWithHome(root, home string) (Result, error) {
	rootSHA, err := gitutil.RootCommit(root)
	if err != nil {
		return Result{}, err
	}
	rec, ref, recorded, err := loadRecord(home, rootSHA)
	if err != nil {
		return Result{}, err
	}
	if !recorded {
		ref = MachineRecordRef()
	}
	eff, err := EffectiveIdentity(root)
	if err != nil {
		return Result{}, err
	}
	committer, err := effectiveCommitter(root)
	if err != nil {
		return Result{}, err
	}
	res := authorResult(rec, recorded, ref, eff)
	res.Committer = committer
	res.CommitterDiverges, res.CommitterReason = committerDivergence(rec, recorded, eff, committer)
	res.AuthorIsTool = eff != (Effective{}) && IsToolIdentity(RoleAuthor, eff.Name, eff.Email)
	res.CommitterIsTool = committer != (Effective{}) && IsToolIdentity(RoleCommitter, committer.Name, committer.Email)
	return res, nil
}

// authorResult is the author half of Check.
func authorResult(rec Record, recorded bool, ref RecordRef, eff Effective) Result {
	res := Result{Record: rec, Recorded: recorded, RecordAt: ref, Effective: eff}
	switch {
	case eff.Name == "" || eff.Email == "":
		res.Status = StatusUnset
		res.Reason = "git author identity is not configured (user.name/user.email)"
	case !recorded:
		res.Status = StatusNoRecord
		res.Reason = "you have not recorded who you commit as on this machine (" + ref.Display + ")"
	case eff.Name != rec.Name || eff.Email != rec.Email:
		res.Status = StatusMismatch
		res.Reason = fmt.Sprintf("commit identity %q <%s> is not the one you recorded, %q <%s> (%s)", eff.Name, eff.Email, rec.Name, rec.Email, ref.Display)
	default:
		res.Status = StatusOK
	}
	return res
}

// committerDivergence is the committer half: a committer that differs from the
// author, unless it is the recorded identity (then the author is the one that
// is wrong, and the author status says so).
func committerDivergence(rec Record, recorded bool, author, committer Effective) (bool, string) {
	if committer == author {
		return false, ""
	}
	if recorded && committer.Name == rec.Name && committer.Email == rec.Email {
		return false, ""
	}
	if committer.Name == "" || committer.Email == "" {
		return true, fmt.Sprintf("git committer identity is not configured, while the author is %q <%s>", author.Name, author.Email)
	}
	if recorded {
		return true, fmt.Sprintf("committer %q <%s> is not the identity you recorded, %q <%s>", committer.Name, committer.Email, rec.Name, rec.Email)
	}
	return true, fmt.Sprintf("committer %q <%s> differs from the author %q <%s>", committer.Name, committer.Email, author.Name, author.Email)
}
