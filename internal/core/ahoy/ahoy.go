// Package ahoy is abcd's install/update engine for `abcd ahoy`. It follows a
// detect -> contract -> apply architecture: a single detection pass builds an
// in-memory DetectionResult (the ahoy-state shape) and every sub-verb (install,
// dry-run, doctor, bare status) is a thin consumer of that one pass. Idempotency
// is a property of detection, never a version stamp: every check compares actual
// on-disk/registry state.
//
// The package performs I/O only under a caller-supplied cwd and the user-scope
// ~/.abcd/ store (and, on install, an owned PATH copy). It never writes to
// stdout, never calls os.Exit, and never imports a transport (cobra/MCP), so it
// is fully testable and reusable across surfaces. Interactive decisions are
// routed through the injected Prompter seam.
package ahoy

// FolderKind is the classification of the folder ahoy runs in. There is no
// workspace layer: abcd manages exactly one kind of folder, a repository.
type FolderKind string

const (
	// ManagedRepo is a git repo abcd already manages (a strong marker fired).
	ManagedRepo FolderKind = "managed-repo"
	// UnmanagedRepo is a git repo with no abcd markers yet (adoptable).
	UnmanagedRepo FolderKind = "unmanaged-repo"
	// UnmanagedFolder is not a git repo and has no abcd markers.
	UnmanagedFolder FolderKind = "unmanaged-folder"
)

// GapCategory groups gaps for the one-approval-per-category apply protocol.
type GapCategory string

const (
	// SafeAutocreate covers artefacts written without a per-item prompt once
	// the category is approved (the .abcd/ skeleton, history-store dirs).
	SafeAutocreate GapCategory = "safe-autocreate"
	// ConfigChange covers transparent-confirm changes (visibility, symlink).
	ConfigChange GapCategory = "config-change"
	// PluginOwned covers the marker block and the (verify-only) hook manifest.
	PluginOwned GapCategory = "plugin-owned"
	// Dependency covers opt-in scanners on PATH (surfaced, never auto-run).
	Dependency GapCategory = "dependency"
	// UserState covers ~/.abcd/history registry state (guided, never auto-edited).
	UserState GapCategory = "user-state"
	// StatusLine covers the host harness's status-line wiring (spc-70). Its one
	// gap is advisory and is never written under --yes: the wiring rewrites a
	// harness-wide user setting and takes element choices, so only an answered
	// prompt writes it.
	StatusLine GapCategory = "status-line"
	// OracleRouting covers accepting abcd's proposed model-tier routing table,
	// at the machine and, offered separately, at the repository
	// (itd-2609170822093401). Its gaps are advisory and never written under
	// --yes: a routing table decides which model every delegated step asks
	// for, so only an answered prompt accepts one.
	OracleRouting GapCategory = "oracle-routing"
)

// Gap is one detected discrepancy between desired and actual state.
type Gap struct {
	ID         string      `json:"id"`
	Category   GapCategory `json:"category"`
	Scope      string      `json:"scope"` // "repo" | "machine"
	Title      string      `json:"title"`
	Detail     string      `json:"detail"`
	FixHint    string      `json:"fix_hint"`
	Required   bool        `json:"required"`   // advisory gaps set false
	Resolvable bool        `json:"resolvable"` // false => diagnostic only
}

// RepoIdentity is the deterministic identity of the repo under cwd.
type RepoIdentity struct {
	Name    string `json:"name"`
	Github  string `json:"github"`
	RootSHA string `json:"root_sha"`
}

// DetectionResult is the canonical envelope. dry-run marshals exactly this
// (with Adopted=nil).
type DetectionResult struct {
	FolderKind       FolderKind     `json:"folder_kind"`
	Adopted          *bool          `json:"adopted"` // nil on detect/dry-run
	RootSHA          string         `json:"root_sha"`
	PluginRootStatus string         `json:"plugin_root_status"` // "resolved" | "missing"
	RepoIdentity     RepoIdentity   `json:"repo_identity"`
	Signals          map[string]any `json:"signals"`
	// Guard is the execution-time shell guard's health. It is reported on every
	// pass over a repo, healthy or not: a guard that fails open looks exactly like
	// a working one from inside a session, so the state has to be legible from
	// outside.
	//
	// A POINTER, omitted entirely for an unmanaged folder — like Banlist below and
	// for the same reason: its fields are definite booleans, so a never-computed
	// zero value would serialise four `false` facts (and no detail) that read as "a
	// broken guard" to a consumer that never asked about a repo, in a document that
	// simultaneously reports the plugin root resolved.
	Guard *GuardHealth `json:"guard,omitempty"`
	// Banlist is the two-layer name guard's state. Like Guard it is reported on
	// every pass over a repo — and it carries its own reach, because a private layer
	// that looks installed says nothing about what a pull request is checked against.
	//
	// A POINTER, omitted entirely for an unmanaged folder: the states it reports are
	// named strings ("absent", "foreign") and its ignore flag is warning-polarity, so
	// a zero value would serialise undeclared states and a false that reads as "the
	// store is trackable" to a consumer that never asked about a repo.
	Banlist *BanlistHealth `json:"banlist,omitempty"`
	Gaps    []Gap          `json:"gaps"`

	// pluginRoot is the resolved plugin root; not serialized.
	pluginRoot string
}

// InstallConfig is the four configuration values collected/loaded during install.
type InstallConfig struct {
	Visibility    string // private | public
	DocsTarget    string // claude_md | agents_md | both | skip
	OracleBackend string // host-delegated | native | cli | api | mcp
	ScanDeep      *bool  // nil = unset (gap not emitted)
}

// InstallOptions encodes the non-interactive prompt-protocol flags.
type InstallOptions struct {
	Adopt              *bool                // --adopt / --refuse-adopt / nil
	Yes                bool                 // approve every resolvable category
	ApprovedCategories map[GapCategory]bool // nil => interactive; explicit => partial subset
	ValueOverrides     map[string]string    // visibility/docs_target/oracle_backend/scan_deep
	// Dev requests the track-latest dogfood mode: the PATH entry becomes a shim
	// that rebuilds abcd from the source tip on every call and execs the fresh
	// binary (failing loudly on a broken build), instead of a symlink to the
	// pinned built binary. --dev.
	Dev bool
	// BinDir is an explicit directory for the PATH entry (--bin-dir), the only
	// way to reach a system-wide location. Empty means the default single-user
	// location (~/.local/bin), or an existing owned install adopted in place.
	// abcd NEVER escalates privileges: a BinDir it cannot write to is a loud
	// error, never a silent skip and never a re-run under sudo.
	BinDir string
	// Attribution opts this repo into the committed prepare-commit-msg prompt that
	// asks every commit to declare whether a tool assisted it. --attribution.
	//
	// Opt-IN, never a default: the hook stamps a convention onto every commit
	// message, which is an opinion a repo adopts rather than one abcd may assume.
	// The choice is PERSISTED (`attribution.hook` in config.json), so a later plain
	// install keeps the hook instead of making the maintainer re-pass the flag to
	// keep something they already chose.
	Attribution bool
	// AllowStaleBinary overrides the itd-111 staleness refusal: an install run
	// through a binary that is stale against its source tip, or whose vintage
	// cannot be determined, otherwise refuses before any write. The override is
	// the documented escape when a rebuild is not an option. --allow-stale-binary.
	AllowStaleBinary bool
}

// InstallResult is the outcome of Install.
type InstallResult struct {
	Status             string   `json:"status"` // already_up_to_date | clean | partial | aborted | refused
	Writes             []string `json:"writes"`
	Changes            []string `json:"changes,omitempty"`   // value overwrites an explicit override forced ("visibility: private -> public")
	Remaining          []string `json:"remaining"`           // required+resolvable gap ids left
	DeclinedCategories []string `json:"declined_categories"` // sorted category wire values
	// Notes carries an apply step's loud refusal — a thing abcd deliberately did
	// not do, and why. A refusal that only shows up as a still-open gap reads as a
	// silent failure, so the reason travels with the result.
	Notes []string `json:"notes,omitempty"`
	// OptionalSkipped names the optional gaps a --yes run deliberately did not
	// apply: the advisory git-identity pin, the status-line offer and the two
	// model-tier routing offers (which accept a table only on an answer). --yes
	// approves every resolvable CATEGORY, but it never writes the pin, because
	// the pin captures whatever git identity happens to be configured and an
	// unattended run would canonicalise a sandbox or agent identity; and it never
	// wires the status line, because that rewrites a harness-wide user setting
	// and takes element choices only a prompt can carry. The exclusion is
	// reported rather than assumed: a run that says "already up to date" while
	// leaving optional work on the table has to say so (iss-166).
	OptionalSkipped []string `json:"optional_skipped,omitempty"`
}

// ApplyResult is the outcome of one apply step.
type ApplyResult struct {
	Category     GapCategory
	GapsResolved []string
	GapsSkipped  []string
	Notes        []string
}

// MarkerReceipt records the per-target outcome of the uninstall marker removal.
type MarkerReceipt struct {
	Removed []string `json:"removed"` // relative filenames a block was stripped from
	Skipped []string `json:"skipped"` // relative filenames left untouched
}

// SymlinkReceipt records the outcome of the uninstall symlink removal.
type SymlinkReceipt struct {
	Target  string `json:"target"`
	Removed bool   `json:"removed"`
	Note    string `json:"note"`
}

// StatusLineReceipt records the outcome of the uninstall status-line restore:
// whether the harness's status line was handed back to the command recorded
// before abcd took the row, and — restored or not — why. The note is rendered
// for a human and carries no home path.
type StatusLineReceipt struct {
	Restored bool   `json:"restored"`
	Note     string `json:"note"`
}

// UninstallReceipt is the outcome of Uninstall.
type UninstallReceipt struct {
	Marker     MarkerReceipt     `json:"marker"`
	Symlink    SymlinkReceipt    `json:"symlink"`
	StatusLine StatusLineReceipt `json:"status_line"`
}

// DoctorReport is the outcome of Doctor: the detection envelope plus read-only
// cross-machine reconciliation gaps.
type DoctorReport struct {
	Detection DetectionResult `json:"detection"`
	AuditGaps []Gap           `json:"audit_gaps"`
}

// Prompter is the interactive seam. The CLI supplies interactive impls;
// non-interactive modes supply refusing impls that auto-decline rather than
// block on stdin.
type Prompter interface {
	// Confirm asks a yes/no question and returns the answer.
	Confirm(question string) bool
	// Prompt asks the user to pick one of choices, defaulting to def.
	Prompt(key string, choices []string, def string) string
}

// RefusingPrompter auto-declines every confirm and returns the default for
// every prompt. It is the safe default for non-interactive callers.
type RefusingPrompter struct{}

// Confirm always declines.
func (RefusingPrompter) Confirm(string) bool { return false }

// Prompt always returns the supplied default.
func (RefusingPrompter) Prompt(_ string, _ []string, def string) string { return def }
