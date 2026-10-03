package ahoy

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/fsutil"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/gitleaks"
	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/tools"
)

// Enumerations for config-value validation.
var (
	visibilityChoices = []string{"private", "public"}
	// docsTargetChoices is the set a saved docs.target is READ against, so a
	// project set up through claude_md or both still reads as configured and
	// still uninstalls cleanly. docsTargetWritable is the set setup WRITES:
	// AGENTS.md is the one conventions file abcd writes (adr-2610030814023326),
	// and RetiredDocsTarget explains the difference.
	docsTargetChoices    = []string{"claude_md", "agents_md", "both", "skip"}
	docsTargetWritable   = []string{"agents_md", "skip"}
	oracleBackendChoices = []string{"host-delegated", "native", "cli", "api", "mcp"}
	// scan.deep is a boolean, but it is collected and overridden as a string
	// through the same choice-set seam as the enums above, so its vocabulary is
	// declared once here rather than restated at each of the three sites that
	// judge it (collect-missing, override, would-change).
	scanDeepChoices = []string{"true", "false"}
)

const (
	// docsTargetDefault is skip: the managed block names abcd and documents its
	// internals, so a default adoption writes it into none of the repository's
	// committed conventions files (iss-2609110944498549). A project that wants
	// it chooses a target; the repo still classifies as managed on its registry
	// entry, the other strong signal.
	docsTargetDefault    = "skip"
	oracleBackendDefault = "host-delegated"
	scanDeepDefault      = "false"
)

// Detect runs the full detection pass over cwd and returns the canonical
// envelope. Total over a normal folder: broken-plugin conditions surface as
// gaps, never as a hard error. An error is returned only for malformed input.
func Detect(cwd string) (DetectionResult, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return DetectionResult{}, err
	}

	// Step 4 (pre-computed): identity, once per pass.
	identity := deriveIdentity(abs)

	// Step 0: folder classification.
	idx, _ := loadHistoryIndex()
	kind, signals := classify(abs, identity, idx)

	// Step 1: plugin root.
	pluginRoot, pluginOK := resolvePluginRoot()
	pluginStatus := "missing"
	if pluginOK {
		pluginStatus = "resolved"
	}

	// The PATH-target install mode is a machine-scope fact independent of folder
	// kind, surfaced so status reports "dev (tip build)" honestly (never invisible).
	signals["install_mode"] = detectInstallMode(pluginRoot, pluginOK)

	// The host harness's status line is a machine-scope fact too (spc-70): one
	// read of the harness's settings, classified, so the board can say whether
	// abcd's row is wired, absent, foreign, or pointing at an abcd that is gone.
	harness := readHarnessSettings()
	signals["statusline"] = string(harness.state)

	// The citation baseline's coverage and age, when this repo has armed the
	// citation gate (spc-17). Omitted entirely otherwise, so a repo that has not
	// adopted the gate carries no line about it.
	if citations := detectCitations(abs); citations != "" {
		signals["citations"] = citations
	}

	res := DetectionResult{
		FolderKind:       kind,
		RootSHA:          identity.RootSHA,
		PluginRootStatus: pluginStatus,
		RepoIdentity:     identity,
		Signals:          signals,
		pluginRoot:       pluginRoot,
	}

	// Seeded non-nil: a folder with nothing outstanding renders "gaps": [], never
	// null, so a consumer iterating the list does not fail on the healthy case
	// (iss-2609120447487070).
	gaps := []Gap{}
	gaps = append(gaps, detectPluginRoot(pluginOK)...)
	if kind != UnmanagedFolder {
		gaps = append(gaps, detectDependencies(abs)...)
		gaps = append(gaps, detectSkeleton(abs)...)
		gaps = append(gaps, detectLocalTier(abs)...)
		gaps = append(gaps, detectIdentity(identity, idx)...)
		gaps = append(gaps, detectGitIdentity(abs)...)
		gaps = append(gaps, detectHistoryStore(identity.RootSHA)...)
		gaps = append(gaps, detectConfigIntegrity(abs)...)
		gaps = append(gaps, detectConfigValues(abs)...)
		gaps = append(gaps, detectMarkerDrift(abs)...)
		gaps = append(gaps, detectPathSymlink(abs, pluginRoot, pluginOK)...)
		gaps = append(gaps, detectStatusLine(harness)...)
		gaps = append(gaps, detectOracleRouting(abs)...)
		gaps = append(gaps, detectDrainRule(abs)...)
		// A tool's own conventions file read in place of AGENTS.md
		// (itd-2610030814013772): the owner's is named, a repeat is offered.
		gaps = append(gaps, detectToolConventionsFiles(abs)...)
		gaps = append(gaps, detectProviderAdapter(abs)...)
		gaps = append(gaps, detectHookManifest(pluginRoot, pluginOK)...)
		gaps = append(gaps, detectVersion(abs)...)
		// Guard health is computed for every managed or adoptable repo, so a
		// broken guard is visible on the status board and not only from inside a
		// session that has already stopped being protected (itd-103 AC 1).
		guardHealth := detectGuardHealth(abs, pluginRoot, pluginOK)
		res.Guard = &guardHealth
		gaps = append(gaps, detectGuardGaps(guardHealth)...)
		// The name guard's scaffolding is reported the same way and for the same
		// reason: a private layer nobody opted into on this machine is invisible from
		// inside a commit that passed, so the state has to be legible from outside.
		banlistHealth := detectBanlistHealth(abs)
		res.Banlist = &banlistHealth
		gaps = append(gaps, detectBanlistScaffold(banlistHealth)...)
		// The attribution prompt is opt-in, so this reports nothing at all for a repo
		// that never adopted it — and a hand-deleted hook for one that did.
		gaps = append(gaps, detectAttributionHook(abs)...)
		// What the repository ships, declared once for the launch verbs
		// (itd-2609150819432059).
		gaps = append(gaps, detectArtefact(abs)...)
	}

	sortGaps(gaps)
	res.Gaps = gaps
	return res, nil
}

// DryRun runs Detect and returns the envelope (Adopted=nil). Zero writes.
func DryRun(cwd string) (DetectionResult, error) { return Detect(cwd) }

// classify keys on the signal hierarchy (brief step 0 / itd-40).
func classify(cwd string, id RepoIdentity, idx *historyIndex) (FolderKind, map[string]any) {
	signals := map[string]any{}

	registered := indexHasRoot(idx, id.RootSHA)
	signals["index_registered"] = registered

	abcdDir := fsutil.IsRealDir(filepath.Join(cwd, ".abcd"))
	signals["abcd_dir"] = abcdDir

	markerFired := false
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		if markerFileHasBlock(filepath.Join(cwd, name)) {
			markerFired = true
			break
		}
	}
	signals["marker_block"] = markerFired

	gitRepo := gitPresent(cwd)
	signals["git_repo"] = gitRepo

	// A bare .abcd/ directory is not a managed signal on its own (iss-88): only
	// index registration or a marker block promotes a folder to managed-repo.
	strong := registered || markerFired
	switch {
	case strong:
		return ManagedRepo, signals
	case gitRepo:
		return UnmanagedRepo, signals
	default:
		return UnmanagedFolder, signals
	}
}

// gitPresent reports whether cwd is the top of a git checkout — a `.git`
// directory in a normal clone, but a `.git` regular file ("gitdir: …") in a
// linked worktree or a submodule. Both are genuine checkouts abcd can adopt, so
// the test is existence, not dir-ness; the old isDir check misread a worktree or
// submodule as an unmanaged folder and silently aborted `ahoy install` there.
// os.Stat (not Lstat) is deliberate: a dangling symlink named `.git` reads
// false, matching core.exists and the iss-72 reasoning.
func gitPresent(cwd string) bool {
	_, err := os.Stat(filepath.Join(cwd, ".git"))
	return err == nil
}

// pluginFilesMissing is the one sentence for the state in which abcd cannot find
// the folder its plugin was installed into. The plugin.root_missing gap and the
// guard-health reason both say it, because one render shows both, and a state
// worded two ways reads as two problems (iss-2609260057111298).
const pluginFilesMissing = "abcd looked for the folder its plugin was installed into and found none, so it cannot check the automatic hooks that run it"

func detectPluginRoot(ok bool) []Gap {
	if ok {
		return nil
	}
	return []Gap{{
		ID:       "plugin.root_missing",
		Category: PluginOwned,
		Scope:    "machine",
		// Plain words for the person, not the mechanism (iss-164): the two
		// environment names are named only in the fix hint, with what they are.
		Title:   "abcd's plugin files were not found on this machine",
		Detail:  pluginFilesMissing + ".",
		FixHint: "Reinstall the abcd plugin in your AI assistant; or, to point abcd at a plugin folder by hand, set the ABCD_PLUGIN_ROOT environment variable to that folder (the assistant normally supplies it as CLAUDE_PLUGIN_ROOT).",
	}}
}

// DependencyTools names the tools detectDependencies checks for, and so the
// tools `ahoy install --install-tool` can name: the front door refuses any
// other, since no gap would ever put it to the question.
var DependencyTools = []string{"gitleaks"}

// detectDependencies names each external tool a capability here would use and
// cannot find, with the tool registry's explanation rather than a bare command
// (itd-63). gitleaks is the one ahoy checks: optional over the native secret
// scanner, and REQUIRED in a repository that armed it in
// .abcd/config/gitleaks.json, whose release refuses without it. An
// armed repository that names an existing binary by path has no gap: the
// adapter judges that path itself, and a refusal there is not a missing tool.
//
// trufflehog is not offered: nothing in abcd runs it (iss-2609261447331434),
// and a gap asking a person to install a program abcd never uses is not an
// explanation anyone can act on.
func detectDependencies(cwd string) []Gap {
	capability := tools.TranscriptScan
	if cfg, err := gitleaks.LoadConfig(cwd); err == nil && cfg.Enabled {
		capability = tools.TranscriptScanArmed
		if p := strings.TrimSpace(cfg.Path); p != "" && fileExists(p) {
			return nil
		}
	}
	if onPath(cwd, "gitleaks") {
		return nil
	}
	e := tools.Explain("gitleaks", capability)
	return []Gap{{
		ID: "deps.gitleaks_missing", Category: Dependency, Scope: "machine",
		Title:      "gitleaks not on PATH",
		Detail:     string(e.Requirement) + " for " + e.CapabilityName + "; without it: " + e.WithoutIt + ".",
		FixHint:    e.StepText() + " (abcd ahoy install explains it and runs it only on your yes)",
		Required:   e.Requirement == tools.Required,
		Resolvable: true,
		Tool:       &e,
	}}
}

// onPath reports whether tool is installed where abcd would run it from: PATH
// resolves it to an absolute program outside the checkout at root. A program
// PATH resolves inside the checkout, lexically or after symlinks, is
// repository content that the installer refuses to run, so it does not count
// as installed either; the judgement is the installer's own (tools.WithinTree).
func onPath(root, tool string) bool {
	p, err := exec.LookPath(tool)
	if err != nil || !filepath.IsAbs(p) {
		return false
	}
	guard, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return false
	}
	return !tools.WithinTree(p, resolved, guard)
}

func detectSkeleton(cwd string) []Gap {
	var gaps []Gap
	if !fileExists(filepath.Join(cwd, ".abcd", "config.json")) {
		gaps = append(gaps, Gap{
			ID: "skeleton.config_missing", Category: SafeAutocreate, Scope: "repo",
			Title: ".abcd/config.json missing", Detail: "Repo requires a configuration file at .abcd/config.json.",
			FixHint: "ahoy install creates it from defaults.", Required: true, Resolvable: true,
		})
	}
	if !fileExists(filepath.Join(cwd, ".abcd", "rules.json")) {
		gaps = append(gaps, Gap{
			ID: "rules.missing", Category: SafeAutocreate, Scope: "repo",
			Title: ".abcd/rules.json missing", Detail: "Rules skeleton at .abcd/rules.json is absent.",
			FixHint: "ahoy install writes the bundled default rules.json.", Required: true, Resolvable: true,
		})
	}
	return gaps
}

func detectIdentity(id RepoIdentity, idx *historyIndex) []Gap {
	if id.RootSHA == "" {
		return nil
	}
	if idx == nil || indexHasRoot(idx, id.RootSHA) {
		return nil
	}
	gaps := []Gap{{
		ID: "identity.unregistered", Category: UserState, Scope: "repo",
		Title:   "root SHA not in history index",
		Detail:  "Root commit " + shortSHA(id.RootSHA) + " is absent from ~/.abcd/history/index.json.",
		FixHint: "ahoy install registers the repo entry.", Required: true, Resolvable: true,
	}}
	if cand := findRefoundingCandidate(idx, id); cand != nil {
		gaps = append(gaps, Gap{
			ID: "identity.refounding_candidate", Category: UserState, Scope: "repo",
			Title:   "possible re-founded repo",
			Detail:  "A sibling entry (root " + shortSHA(cand.RootCommit) + ") matches name/github. Confirm re-founding or register as new.",
			FixHint: "ahoy install asks; declining registers as new.", Required: true, Resolvable: true,
		})
	}
	return gaps
}

// The identity gap ids the detector raises and the establish step reads.
const (
	MismatchGapID     = "git_identity.mismatch"
	UnsetGapID        = "git_identity.unset"
	CommitterGapID    = "git_identity.committer"
	ToolIdentityGapID = "git_identity.tool"
)

// routineRunnerRecord is where establishing the human identity for an
// autonomous routine is owned: the runner that launches it sets the identity
// before the first commit, and this gate only detects (itd-131).
const routineRunnerRecord = "iss-2608210932052003"

// establishHint is the fix every resolvable identity gap shares: install
// proposes the human identity and asks, and nothing writes it unasked.
const establishHint = "run abcd ahoy install at a terminal: it proposes the pinned identity (else your global git identity) for this repository's user.name/user.email and writes it only if you confirm"

// detectGitIdentity compares the git author and committer identities a commit
// would use against the committed .abcd/config/identity.json pin (iss-62,
// itd-131). An author mismatch or unset identity is a required, resolvable gap;
// a committer that diverges is one too in a pinned repo and advisory in an
// un-pinned one; a machine identity in either role is required wherever it is
// found, because the human is the author of record whether or not the repo has
// pinned who that is. An un-pinned repo also gets an advisory gap (adopt the
// gate); a match yields nothing.
func detectGitIdentity(cwd string) []Gap {
	res, err := identity.Check(cwd)
	if err != nil {
		// A present-but-unreadable pin is an adopted-but-broken gate (required);
		// an error with no pin is a git/environment issue (advisory).
		_, statErr := os.Stat(filepath.Join(cwd, identity.PinRelPath))
		pinPresent := statErr == nil
		return []Gap{{
			ID: "git_identity.uncheckable", Category: ConfigChange, Scope: "repo",
			Title:      "git identity could not be checked",
			Detail:     err.Error(),
			FixHint:    "fix the pin JSON in " + identity.PinRelPath + " (both name and email), or ensure git is available",
			Required:   pinPresent,
			Resolvable: false,
		}}
	}
	var gaps []Gap
	switch res.Status {
	case identity.StatusMismatch:
		gaps = append(gaps, Gap{
			ID: MismatchGapID, Category: ConfigChange, Scope: "repo",
			Title:      "git commit identity does not match the pin",
			Detail:     res.Reason,
			FixHint:    establishHint + "; or set user.name/user.email by hand (or update the pin if the identity changed)",
			Required:   true,
			Resolvable: true,
		})
	case identity.StatusUnset:
		gaps = append(gaps, Gap{
			ID: UnsetGapID, Category: ConfigChange, Scope: "repo",
			Title:      "git author identity is not configured",
			Detail:     res.Reason,
			FixHint:    establishHint + "; or set git user.name/user.email to the pinned identity in " + identity.PinRelPath,
			Required:   true,
			Resolvable: true,
		})
	case identity.StatusNoPin:
		gaps = append(gaps, Gap{
			ID: OptionalPinGapID, Category: ConfigChange, Scope: "repo",
			Title:      "no git identity pin",
			Detail:     res.Reason,
			FixHint:    "ahoy install can pin the current git identity to " + identity.PinRelPath,
			Required:   false,
			Resolvable: true,
		})
	}
	if res.CommitterDiverges {
		gaps = append(gaps, Gap{
			ID: CommitterGapID, Category: ConfigChange, Scope: "repo",
			Title:      "git committer identity diverges",
			Detail:     res.CommitterReason,
			FixHint:    "unset any GIT_COMMITTER_NAME/GIT_COMMITTER_EMAIL override and committer.name/committer.email key; or " + establishHint,
			Required:   res.Status != identity.StatusNoPin,
			Resolvable: true,
		})
	}
	if tool := toolRoles(res); tool != "" {
		gaps = append(gaps, Gap{
			ID: ToolIdentityGapID, Category: ConfigChange, Scope: "repo",
			Title:      "commits would be made under a machine identity",
			Detail:     tool + " is a machine identity; the human is the author of record, and the attribution gate refuses a machine in these fields",
			FixHint:    establishHint + ". An autonomous routine has no one to ask: whatever launches it sets the human identity before the first commit (" + routineRunnerRecord + "), and abcd does not write it unasked",
			Required:   true,
			Resolvable: true,
		})
	}
	return gaps
}

// toolRoles words which roles hold a machine identity, or "" when none does.
func toolRoles(res identity.Result) string {
	who := func(e identity.Effective) string { return e.Name + " <" + e.Email + ">" }
	switch {
	case res.AuthorIsTool && res.CommitterIsTool && res.Effective == res.Committer:
		return "the author and committer " + who(res.Effective)
	case res.AuthorIsTool && res.CommitterIsTool:
		return "the author " + who(res.Effective) + " and the committer " + who(res.Committer)
	case res.AuthorIsTool:
		return "the author " + who(res.Effective)
	case res.CommitterIsTool:
		return "the committer " + who(res.Committer)
	}
	return ""
}

// historyHomeLinkGapID is the diagnostic for a history registry behind a
// symlinked ~/.abcd (iss-2609281129171021).
const historyHomeLinkGapID = "history.home_symlinked"

func detectHistoryStore(rootSHA string) []Gap {
	var gaps []Gap
	root, err := historyRoot()
	if errors.Is(err, fsutil.ErrHomeScopeSymlinked) {
		// A note, not an actionable gap: install refuses to create the registry
		// through the link, so a required "not bootstrapped" gap would be one it
		// reports as outstanding on every run and can never close. Only the
		// operator can, by replacing the link.
		return []Gap{{
			ID: historyHomeLinkGapID, Category: UserState, Scope: "machine",
			Title:   "history registry not kept behind a symlinked home directory",
			Detail:  "This machine's history registry is not read or written: " + err.Error() + ".",
			FixHint: "Replace the link with a real directory, then re-run `abcd ahoy install` to register this repository.",
		}}
	}
	if err != nil {
		return nil
	}
	if !isDir(root) {
		gaps = append(gaps, Gap{
			ID: "history.bootstrap_missing", Category: UserState, Scope: "machine",
			Title: "~/.abcd/history/ not bootstrapped", Detail: "The shared history store directory is absent.",
			FixHint: "ahoy install bootstraps it.", Required: true, Resolvable: true,
		})
	}
	if rootSHA == "" {
		return gaps
	}
	// There is no gap for a missing transcript corpus. The store creates itself
	// on first use — internal/core/history owns that path and bootstraps it —
	// so "absent" is the ordinary state of a repo that has not been captured
	// yet, not a gap install must close. Raising one would have this board
	// assert that transcripts will not be captured, which is false (iss-95).
	repoDir := filepath.Join(root, rootSHA)
	if !fileExists(filepath.Join(repoDir, "meta.json")) {
		gaps = append(gaps, Gap{
			ID: "history.meta_missing", Category: UserState, Scope: "repo",
			Title:   "history meta.json missing",
			Detail:  "~/.abcd/history/" + shortSHA(rootSHA) + "/meta.json is absent.",
			FixHint: "ahoy install writes the per-repo meta.json.", Required: true, Resolvable: true,
		})
	}
	return append(gaps, detectStoredCredential(rootSHA, repoDir)...)
}

// detectStoredCredential raises the one gap for a git credential left at rest in
// the user-level history store — the state a store written before
// scrubRemoteUserinfo existed is in (GHSA-qc3w-8pv5-crc3).
//
// It exists so the heal is REACHABLE. Without a gap, a repo that is otherwise
// fully installed short-circuits on the idempotency early return and never runs
// the history step, so a token sits in ~/.abcd/history for the life of the
// machine no matter how often the operator re-installs. Required and resolvable,
// because `ahoy install` closes it.
//
// The detail names the FILES only, never the value: a gap detail is rendered by
// every status board, and quoting the credentialed URL would copy the token onto
// the operator's terminal to tell them it should not be at rest.
func detectStoredCredential(rootSHA, repoDir string) []Gap {
	var where []string
	// The at-rest file, not loadHistoryIndex's scrubbed view of it.
	if idx, err := readHistoryIndexFile(); err == nil && idx != nil {
		for _, r := range idx.Repos {
			if r.Github != "" && scrubRemoteUserinfo(r.Github) != r.Github {
				where = append(where, "~/.abcd/history/index.json")
				break
			}
		}
	}
	if rootSHA != "" {
		metaPath := filepath.Join(repoDir, "meta.json")
		if g := metaGithub(metaPath); g != "" && scrubRemoteUserinfo(g) != g {
			where = append(where, "~/.abcd/history/"+shortSHA(rootSHA)+"/meta.json")
		}
	}
	if len(where) == 0 {
		return nil
	}
	return []Gap{{
		ID: credentialAtRestGapID, Category: UserState, Scope: "machine",
		Title:    "git credential at rest in the history store",
		Detail:   "A registry value carries userinfo (a token or password in a remote URL): " + strings.Join(where, ", ") + ".",
		FixHint:  "ahoy install rewrites the affected entries without the credential; revoke the token as well — it has been on disk.",
		Required: true, Resolvable: true,
	}}
}

func detectConfigValues(cwd string) []Gap {
	var gaps []Gap
	cfg, err := readConfig(cwd)
	if err != nil {
		// A config that cannot be parsed has values that are unknown, not
		// missing: reporting them missing would arm the value collection, which
		// then rebuilds the file (GHSA-mchq-gm34-3j34). detectConfigIntegrity
		// raises the one diagnostic for this state.
		return nil
	}
	repo := subMap(cfg, "repo")
	docs := subMap(cfg, "docs")
	oracle := subMap(cfg, "oracle")
	scan := subMap(cfg, "scan")

	visibility, visOK := stringVal(repo, "visibility")
	if !visOK || !inSet(visibility, visibilityChoices) {
		gaps = append(gaps, configValueGap("config.visibility_missing", "visibility", "repo.visibility not set",
			"Visibility (private/public) controls .gitignore policy."))
	}
	if v, ok := stringVal(docs, "target"); !ok || !inSet(v, docsTargetChoices) {
		gaps = append(gaps, configValueGap("config.docs_target_missing", "docs_target", "docs.target not set",
			"Whether AGENTS.md hosts the marker block (agents_md / skip)."))
	} else if _, retired := RetiredDocsTarget(v); retired {
		gaps = append(gaps, docsTargetRetiredGap(v))
	}
	if v, ok := stringVal(oracle, "backend"); !ok || !inSet(v, oracleBackendChoices) {
		gaps = append(gaps, configValueGap("config.oracle_backend_missing", "oracle_backend", "oracle.backend not set",
			"Oracle backend (host-delegated/native/cli/api/mcp)."))
	}

	validVis := ""
	if visOK && inSet(visibility, visibilityChoices) {
		validVis = visibility
	}
	// scan.deep is conditional: private + trufflehog present.
	if validVis == "private" && onPath(cwd, "trufflehog") {
		if _, ok := boolVal(scan, "deep"); !ok {
			gaps = append(gaps, configValueGap("config.scan_deep_missing", "scan_deep", "scan.deep not set",
				"Private repo + trufflehog present — confirm deep secret scanning."))
		}
	}
	// visibility.gitignore_drift only when a valid visibility is persisted.
	if validVis == "private" || validVis == "public" {
		if gitignoreBlockDrifts(cwd, validVis) {
			gaps = append(gaps, gitignoreDriftGap(validVis))
		}
	}
	return gaps
}

func cfgGap(id, title, detail string) Gap {
	return Gap{ID: id, Category: ConfigChange, Scope: "repo", Title: title, Detail: detail,
		FixHint: "ahoy install prompts for the value.", Required: true, Resolvable: true}
}

// configValueGap is a missing config value: install asks for it, and the flag
// its question's help names answers it without the question — the one reliable
// answer in a piped run, so the fix hint says so (iss-2609120447486547).
func configValueGap(id, key, title, detail string) Gap {
	g := cfgGap(id, title, detail)
	if h, ok := HelpFor(key); ok && h.Flag != "" {
		g.FixHint = "ahoy install asks for the value; " + h.FlagHint() + "."
		if key == "oracle_backend" && !oracleBackendAsked() {
			g.FixHint = "ahoy install records " + oracleBackendDefault + ", the only reviewer abcd ships, without asking; " +
				"to choose another, pass " + h.Flag + " <value>."
		}
	}
	return g
}

// gitignoreDriftGap is abcd's .gitignore block out of step with the visibility
// already persisted. Install rewrites the block for that visibility and asks
// nothing, so its hint claims no question.
func gitignoreDriftGap(visibility string) Gap {
	g := cfgGap("visibility.gitignore_drift",
		"abcd-managed .gitignore block drifts from visibility",
		"The .gitignore abcd block does not match the canonical rules for visibility="+visibility+".")
	g.FixHint = "ahoy install rewrites the block to the rules for visibility=" + visibility + "."
	return g
}

func detectMarkerDrift(cwd string) []Gap {
	cfg, err := readConfig(cwd)
	if err != nil {
		// docs.target is unknowable, so no marker gap may arm a plant into the
		// default targets (both files) the user never chose.
		return nil
	}
	docs := subMap(cfg, "docs")
	target, _ := stringVal(docs, "target")
	// A retired target names its files only for the read side: install refuses
	// to write under it, so a drift gap would promise a write that never comes.
	// config.docs_target_retired stands in for them.
	files := writableMarkerTargets(target)
	var gaps []Gap
	for _, name := range files {
		switch classifyMarker(filepath.Join(cwd, name)) {
		case markerMissing:
			gaps = append(gaps, Gap{
				ID: "marker.missing", Category: PluginOwned, Scope: "repo",
				Title: name + " marker block missing", Detail: name + " has no <!-- BEGIN ABCD --> block.",
				FixHint: "ahoy install plants the canonical marker block.", Required: true, Resolvable: true,
			})
		case markerOutdated:
			gaps = append(gaps, Gap{
				ID: "marker.outdated", Category: PluginOwned, Scope: "repo",
				Title: name + " marker block outdated", Detail: name + " marker block differs from the template.",
				FixHint: "ahoy install rewrites it to canonical (silent overwrite).", Required: true, Resolvable: true,
			})
		case markerUnplaceable:
			gaps = append(gaps, Gap{
				ID: "marker.unplaceable", Category: PluginOwned, Scope: "repo",
				Title:   name + " marker block cannot be placed",
				Detail:  name + " has no frontmatter or live H1 and ends inside a fenced block or HTML comment nothing closes, so the block would be written where no reader sees it.",
				FixHint: "Close the open fence or comment (or add an H1 above it); ahoy install then plants the block.", Required: true, Resolvable: false,
			})
		}
	}
	return gaps
}

// detectPathSymlink reports the state of abcd's own entry on PATH. It scans
// PATH rather than inspecting one blessed location (iss-171): a working
// ~/.local/bin/abcd is an install, wherever the default happens to point, so the
// detector can no longer report "not installed" while running as that very
// binary. Every path it renders goes through displayPath, so a gap pasted into
// an issue carries no username.
func detectPathSymlink(cwd, pluginRoot string, pluginOK bool) []Gap {
	if !pluginOK {
		return nil
	}
	var gaps []Gap

	// A link of ours whose binary has gone runs nothing (the shell skips it), and
	// answers again whatever reappears at its target. It is neither
	// "installed" nor "missing" — it is its own gap.
	top, topOK := danglingPathEntry(pluginRoot)
	if topOK {
		gaps = append(gaps, danglingEntryGap(top.path, true))
	}

	target := effectiveBinTarget(pluginRoot)
	if target == "" {
		return gaps // no home directory: there is no user-scope target to report on
	}
	installed := false

	fi, err := lstat(target)
	switch {
	case err != nil && isNotExist(err):
		gaps = append(gaps, Gap{
			ID: "symlink.missing", Category: ConfigChange, Scope: "machine",
			Title: "abcd is not on PATH", Detail: "No abcd-owned entry was found on PATH, and " + displayPath(target) + " does not exist.",
			FixHint: "ahoy install writes the entry (refuses to clobber, and never escalates privileges).", Required: true, Resolvable: true,
		})
	case err != nil:
		// Present but unstattable: never claim anything about it.
	case fi.Mode()&modeSymlink == 0:
		if isDevShimFile(target) {
			// Our own track-latest dev shim (abcd ahoy install --dev) — a valid
			// install, not a foreign occupant. Surfaced via the install_mode signal.
			installed = true
			gaps = append(gaps, unrecordedEntryGap(target)...)
		} else if isOwnedCopyFile(target) {
			// The spc-35 owned copy: a regular file the data dir's path-entry
			// vouches for, byte-for-byte. The healthy default install.
			installed = true
		} else {
			occupant := foreignOccupant(fi, "a regular file")
			gaps = append(gaps, Gap{
				ID: "symlink.foreign", Category: ConfigChange, Scope: "machine",
				Title:   "non-symlink at " + displayPath(target),
				Detail:  strings.ToUpper(occupant[:1]) + occupant[1:] + " occupies the PATH entry abcd would write: " + describeForeignFile(target, fi) + ".",
				FixHint: "Resolve manually; ahoy refuses to clobber.", Required: false, Resolvable: false,
			})
		}
	default:
		dest, rerr := readlink(target)
		switch {
		case rerr != nil:
			// Unreadable link: say nothing rather than guess.
		case resolveSymlinkDest(target, dest) == resolvePath(pluginBinaryPath(pluginRoot)):
			installed = true
			gaps = append(gaps, unrecordedEntryGap(target)...)
			// A working install TODAY, and a casualty of the next plugin
			// update: the link points into a directory the harness replaces and
			// garbage-collects (spc-35). It is a gap whatever the cache holds
			// (iss-2609100506263330): silence on a cold cache reported a pin
			// the next update breaks as a clean install. With a verified cache
			// artefact install heals it to the owned copy; without one install
			// leaves the working pin where it stands, and the hint names the
			// command that provides the verified copy first.
			// A pin whose plugin binary is already gone is not working today:
			// the symlink.dangling gap above carries it.
			if linkIsDangling(target) {
				break
			}
			fix := "ahoy install replaces it with an abcd-owned copy of the verified release binary, which survives updates."
			if !ownedCopySourceReady(cwd, pluginRoot) {
				fix = "No verified release binary is available to replace it with yet. " + coldCacheRemedy
			}
			gaps = append(gaps, Gap{
				ID: "symlink.legacy", Category: ConfigChange, Scope: "machine",
				Title:    "PATH entry is a symlink into the plugin root",
				Detail:   displayPath(target) + " points into the harness-owned plugin directory, which every plugin update replaces and later deletes — the entry will dangle after the next update.",
				FixHint:  fix,
				Required: true, Resolvable: true,
			})
		case supersededSiblingDest(target, dest, pluginRoot):
			// Ours, pinned into a vintage the harness has moved past but not yet
			// deleted (iss-2609161805447092): a working install that answers an
			// older release. Its remedy is the one every owned shape gets, never
			// the foreign gap's "resolve manually".
			gaps = append(gaps, Gap{
				ID: "symlink.superseded", Category: ConfigChange, Scope: "machine",
				Title:    "PATH entry points into a superseded plugin vintage",
				Detail:   displayPath(target) + " -> " + displayPath(dest) + " is abcd's own pin, but the plugin has moved on to " + displayPath(pluginRoot) + ", so `abcd` answers an older release than the plugin holds.",
				FixHint:  "ahoy install replaces it with the current release.",
				Required: true, Resolvable: true,
			})
		case strandedSiblingDest(target, dest, pluginRoot) || recordedDanglingLink(target):
			// Ours, stranded by a plugin update (iss-345) or named by the
			// provenance record (iss-2609100506263330): a foreign-worded gap
			// would tell the user to hand-resolve a link abcd itself wrote. The
			// symlink.dangling gap above carries it when the entry is on PATH;
			// an entry off PATH (an explicit --bin-dir, a bin dir not yet on
			// PATH) is not in that scan, so it is named here, once.
			if !topOK || !sameEntry(top.path, target) {
				gaps = append(gaps, danglingEntryGap(target, true))
			}
		case linkIsDangling(target):
			// A link abcd cannot prove it wrote, that resolves to NOTHING
			// (iss-2609100506256636). Refusing to clobber a foreign entry is
			// right — it is somebody's working install — but this one is
			// nobody's: it runs nothing, and it answers whatever reappears at
			// its target ahead of a healthy abcd. Reporting it as foreign made the state
			// unreachable from inside the tool, because that gap is
			// `resolvable: false` and there is no --force and no uninstall path
			// for an entry abcd does not own, so `ahoy install` could never
			// finish. The discriminator is DANGLINGNESS, not provenance: the
			// live-link case below still refuses, and clearing this one removes
			// the link itself (clearDanglingEntry) and never writes through it.
			gaps = append(gaps, danglingEntryGap(target, false))
		default:
			gaps = append(gaps, Gap{
				ID: "symlink.foreign", Category: ConfigChange, Scope: "machine",
				Title:   "foreign symlink at " + displayPath(target),
				Detail:  displayPath(target) + " -> " + displayPath(dest) + " (expected " + displayPath(pluginBinaryPath(pluginRoot)) + ").",
				FixHint: "Resolve manually; ahoy refuses to clobber.", Required: false, Resolvable: false,
			})
		}
	}

	gaps = append(gaps, detectBinDirOnPath(filepath.Dir(target), installed)...)
	gaps = append(gaps, detectShadowedEntry(pluginRoot, target)...)
	return gaps
}

// danglingEntryGap is the ONE wording for a PATH entry whose target is gone, in
// both the shapes that reach it: one abcd can prove it wrote (an owned link, or
// the sibling a plugin update stranded), and one it cannot. The id, category and
// title are the same because the condition and the remedy are the same — the
// link resolves to nothing, so clearing it destroys nothing and `ahoy install`
// writes a fresh entry in its place. Only the two sentences that would otherwise
// assert provenance differ: abcd never claims to have written a link it cannot
// prove it wrote, and never points at `ahoy uninstall`, which removes only what
// abcd owns.
func danglingEntryGap(path string, owned bool) Gap {
	detail := displayPath(path) + " points at a target that does not exist. It runs nothing — the shell skips it — but whatever reappears at that target would answer `abcd` first."
	fix := "ahoy install replaces it with a verified copy of the release binary once one is available: a link that resolves to nothing is nobody's working install."
	if owned {
		detail = displayPath(path) + " is an abcd-owned entry whose target no longer exists. It runs nothing — the shell skips it — but whatever reappears at that target would answer `abcd` first."
		fix = "ahoy install replaces it with a verified copy of the release binary, and names the command to run first when none is available; remove it with `ahoy uninstall` if abcd is gone."
	}
	return Gap{
		ID: "symlink.dangling", Category: ConfigChange, Scope: "machine",
		Title:    "PATH entry points at a binary that is gone",
		Detail:   detail,
		FixHint:  fix,
		Required: true, Resolvable: true,
	}
}

// unrecordedEntryGap reports an entry abcd owns that ~/.abcd/path-entry does
// not name. It is the one state where the board and the hooks disagree in
// silence: the entry is a working install by every filesystem test detection
// makes, so it reports installed, while every hook shim refuses it at the
// ownership rung and degrades — the rules loader inactive, the shell guard
// UNGUARDED, the transcript uncaptured — with nothing on either surface
// connecting the two.
//
// It is a gap and not merely a note because install is gated on gaps: a run
// with zero actionable gaps returns already_up_to_date without building an
// apply context, so without this the remedy every surface advertises would
// write nothing on exactly the machines that need it.
//
// The owned copy can never raise it: that shape is CLASSIFIED by the record,
// so an unrecorded one reads foreign and has its own gap already.
func unrecordedEntryGap(target string) []Gap {
	if pathEntryNames(target) {
		return nil
	}
	return []Gap{{
		ID: "symlink.unrecorded", Category: ConfigChange, Scope: "machine",
		Title: "PATH entry is not recorded as this machine's abcd",
		Detail: displayPath(target) + " is abcd's own entry, but ~/.abcd/path-entry does not record it. " +
			"The plugin's hooks read that record before they will run an abcd off PATH, so they ignore this install: " +
			"no rules loader, no shell guard, and no transcript capture.",
		FixHint:  "ahoy install writes the record naming this entry — with --dev if the entry is the track-latest shim, which a plain install replaces with a pinned one.",
		Required: true, Resolvable: true,
	}}
}

// detectShadowedEntry reports an `abcd` that precedes abcd's own entry on PATH.
// Without it the most common pre-iss-171 machine reports a clean, pinned install
// forever while a stale copy keeps executing: the old one-liner COPIED the binary
// to /usr/local/bin, and a copy is a regular file, so it classifies foreign, is
// never adopted, and the new entry lands behind it. abcd cannot resolve this —
// removing a binary it does not own is exactly what it refuses to do — so the gap
// is required and diagnostic, naming the occupant and both remedies.
func detectShadowedEntry(pluginRoot, target string) []Gap {
	e, ok := shadowingEntry(pluginRoot, target)
	if !ok {
		return nil
	}
	return []Gap{{
		ID: "symlink.shadowed", Category: ConfigChange, Scope: "machine",
		Title:    "another abcd on PATH answers first",
		Detail:   shadowMessage(e, target),
		FixHint:  "Remove or rename " + displayPath(e.path) + ", or install ahead of it with `abcd ahoy install --bin-dir <dir>`.",
		Required: true, Resolvable: false,
	}}
}

// detectBinDirOnPath reports an install directory that is not on PATH. It is a
// gap in its own right: the entry exists and abcd still cannot be run by name,
// which otherwise reads as a broken install. The remedy is printed, never
// applied — abcd states the one-line export and leaves the user's shell profile
// alone (script-first), so the gap is required but NOT resolvable.
func detectBinDirOnPath(dir string, installed bool) []Gap {
	if dir == "" || dirOnPath(dir) {
		return nil
	}
	// Silent while there is nothing there yet AND no install: the missing-entry
	// gap already says what to do, and the install run itself emits the same
	// wording as a note for the directory it actually writes (which is the only
	// place that knows about an explicit --bin-dir).
	if !installed {
		return nil
	}
	return []Gap{{
		ID: "path.bin_dir_not_on_path", Category: ConfigChange, Scope: "machine",
		Title:    displayPath(dir) + " is not on PATH",
		Detail:   "abcd is installed at " + displayPath(filepath.Join(dir, binName)) + ". " + pathReachMessage(dir),
		FixHint:  "Add it to your shell profile: " + exportPathLine(dir),
		Required: true, Resolvable: false,
	}}
}

// detectInstallMode reports the current PATH-target install mode: "dev (tip
// build)" when the track-latest shim occupies it, "pinned" when our owned symlink
// does, and "" when the target is absent, foreign, or the plugin root is
// unresolved (nothing to attribute a mode to). A mode that another `abcd` earlier
// on PATH shadows is reported as shadowed rather than as a healthy install: the
// entry is correct and it is still not what runs.
func detectInstallMode(pluginRoot string, pluginOK bool) string {
	if !pluginOK {
		return ""
	}
	target := effectiveBinTarget(pluginRoot)
	if target == "" {
		return ""
	}
	// A link whose binary is gone is not an install mode; it is the dangling gap.
	if present, err := fsutil.Exists(target); err == nil && !present {
		return ""
	}
	mode := ""
	switch classifyBinTarget(target, pluginRoot) {
	case binTargetDevShim:
		mode = "dev (tip build)"
	case binTargetOwnedSymlink, binTargetOwnedCopy:
		mode = "pinned"
	default:
		return ""
	}
	if _, shadowed := shadowingEntry(pluginRoot, target); shadowed {
		return mode + " (shadowed on PATH)"
	}
	return mode
}

func detectHookManifest(pluginRoot string, pluginOK bool) []Gap {
	if !pluginOK {
		return nil
	}
	reason := verifyHookManifest(pluginRoot)
	if reason == "" {
		return nil
	}
	return []Gap{{
		ID: "hooks.manifest_missing", Category: PluginOwned, Scope: "machine",
		Title: "hooks/hooks.json missing or malformed", Detail: reason,
		FixHint: "Broken plugin install — reinstall.", Required: false, Resolvable: false,
	}}
}

// detectConfigIntegrity raises the one diagnostic for a config.json that is
// present but cannot be parsed (a merge-conflict marker is the usual cause).
// It is Required so it shows on the status board, and NOT Resolvable, so it is
// excluded from Remaining and never arms a step: the file is the user's data
// and repairing it is theirs to do (GHSA-mchq-gm34-3j34). Every other reader of
// the config returns no gap on the same error, so this is the only line the
// state produces.
func detectConfigIntegrity(cwd string) []Gap {
	if _, err := readConfig(cwd); err != nil {
		return []Gap{{
			ID: malformedConfigGapID, Category: ConfigChange, Scope: "repo",
			Title:    ".abcd/config.json could not be parsed",
			Detail:   ".abcd/config.json is present but could not be parsed (" + errText(err) + "); its values are unknown and ahoy install will not rewrite it.",
			FixHint:  "Repair the file by hand — a merge-conflict marker is the usual cause — and re-run ahoy install.",
			Required: true, Resolvable: false,
		}}
	}
	return nil
}

func detectVersion(cwd string) []Gap {
	cfg, err := readConfig(cwd)
	if err != nil {
		// Never install_meta.missing: that gap arms stepVersionStamp, which would
		// republish the unparseable file as a meta-only one.
		return nil
	}
	meta := subMap(cfg, "meta")
	setupVersion, hasVersion := stringVal(meta, "setup_version")
	setupDate, hasDate := stringVal(meta, "setup_date")
	if !hasVersion || !hasDate || setupVersion == "" || setupDate == "" {
		return []Gap{{
			ID: "install_meta.missing", Category: SafeAutocreate, Scope: "repo",
			Title: "setup metadata absent", Detail: "setup_version / setup_date not recorded; first install.",
			FixHint: "ahoy install stamps the meta block.", Required: true, Resolvable: true,
		}}
	}
	// Neither side a dev build (iss-2608241115259170): the gap is required, and
	// against a dev stamp or a dev binary it could never settle — every
	// release install would re-stamp and every dev install undo it.
	if current := pluginVersion(); !isDevOrUnknown(current) && !isDevOrUnknown(setupVersion) && setupVersion != current {
		return []Gap{{
			ID: "version.upgrade", Category: SafeAutocreate, Scope: "repo",
			Title:   "plugin upgrade " + setupVersion + " -> " + current,
			Detail:  "Recorded setup_version differs from the current plugin version.",
			FixHint: "ahoy install re-stamps setup_version + setup_date.", Required: true, Resolvable: true,
		}}
	}
	return nil
}

// ---------------------------------------------------------------------------
// small helpers
// ---------------------------------------------------------------------------

func sortGaps(gaps []Gap) {
	sort.SliceStable(gaps, func(i, j int) bool {
		if gaps[i].Category != gaps[j].Category {
			return gaps[i].Category < gaps[j].Category
		}
		if gaps[i].ID != gaps[j].ID {
			return gaps[i].ID < gaps[j].ID
		}
		return gaps[i].Scope < gaps[j].Scope
	})
}

func shortSHA(s string) string {
	if len(s) > 12 {
		return s[:12] + "…"
	}
	return s
}

func stringVal(m map[string]any, key string) (string, bool) {
	v, ok := m[key].(string)
	return v, ok
}

func boolVal(m map[string]any, key string) (bool, bool) {
	v, ok := m[key].(bool)
	return v, ok
}

func inSet(v string, set []string) bool {
	for _, s := range set {
		if v == s {
			return true
		}
	}
	return false
}
