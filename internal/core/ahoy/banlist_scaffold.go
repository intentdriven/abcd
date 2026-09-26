package ahoy

import (
	"bytes"
	_ "embed"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The committed guard's two halves, repo-relative and slash-separated (they are
// os.Root paths as well as report text).
//
// They are COMMITTED hooks, not .git/hooks copies: .git/hooks is per-clone and
// untracked, so a guard installed there protects one working copy and no one
// else's. A clone points git at this directory once
// (`git config core.hooksPath .githooks`) and every checkout inherits the check.
//
// pre-merge-commit exists because git runs NO pre-commit hook for a merge commit.
// Without it every name the private layer bans walks into history the moment a
// branch carrying it is merged, and the guard reports nothing because it was never
// asked.
const (
	GuardHookRelPath      = ".githooks/pre-commit"
	GuardMergeHookRelPath = ".githooks/pre-merge-commit"
	guardHooksDirRelPath  = ".githooks"
)

// guardHookMarker identifies a hook abcd wrote. A repo may already have its own
// pre-commit hook, and reporting a foreign one as "the abcd guard is installed" is
// the worst of both states: nothing checks the banlist and the status board says
// something does. Presence is not identity, so identity is stamped.
//
// Recognition is a WHOLE LINE matching guardHookMarkerRe, never a substring of the
// blob: a foreign hook that merely MENTIONS the marker — in a comment, in a grep for
// it, in a copied fragment — would otherwise classify as abcd's own, and the board
// would claim coverage while the merge shim was written beside it. The version is
// matched as a group rather than literally, so a v2 template cannot reclassify every
// hook a v1 abcd scaffolded as someone else's.
const (
	guardHookMarker       = guardHookMarkerPrefix + " v1"
	guardHookMarkerPrefix = "# abcd-name-guard:"
)

// guardHookMarkerRe matches the marker as a complete line, surrounding ASCII blanks
// tolerated. `v<digits>` and nothing else: `v1-fork` is a fork's marker, not abcd's.
//
// A trailing CR is tolerated because a CRLF checkout is the exact failure the EOL pin
// exists for: a hook that arrived mangled is still OURS, and install must be able to
// heal it rather than be told for ever that someone else's file is in the way.
var guardHookMarkerRe = regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(guardHookMarkerPrefix) + `[ \t]*v[0-9]+[ \t\r]*$`)

// guardHookTemplate and guardMergeHookTemplate are the canonical scaffolded guard:
// the generalised form of the prototype this repo runs on itself, with the
// repo-specific gates dropped. They are embedded so the binary is self-contained
// (the marker block's precedent) and so scaffolding cannot depend on a plugin-root
// file that a broken install left behind.
//
//go:embed defaults/pre-commit
var guardHookTemplate []byte

//go:embed defaults/pre-merge-commit
var guardMergeHookTemplate []byte

// gitattributesRelPath and guardEOLAttribute pin the committed hooks to LF.
//
// A hook is a script git EXECUTES. A clone with core.autocrlf=true rewrites it to
// CRLF on checkout, and the shebang line then names an interpreter called
// "/usr/bin/env bash\r", which no kernel resolves: the guard stops running and the
// only symptom is a "bad interpreter" line most people read as a broken tool rather
// than as an unprotected repo. The attribute is one line, and it is the difference
// between the scaffolded guard working on every clone and working on some.
const (
	gitattributesRelPath = ".gitattributes"
	guardEOLAttribute    = guardHooksDirRelPath + "/* text eol=lf"
)

// publicFamilySeed is the docs-lint config a repo with none inherits. Two of its
// fields carry different owners, and the seed treats them differently
// (iss-2609150805167646):
//
//   - The Writing-Guide rules are abcd's own: the present_tense and spelling token
//     families and the links_resolve, harness_leak and stray_root_docs rules. They are seeded ARMED, because a lint that runs no rule
//     reports "0 findings" over any tree, a green that means nothing. A repository
//     that wants a family off removes it deliberately, a decision with a diff rather
//     than an absence nobody chose. The token entries are held to the set abcd runs
//     on itself by a parity test, so the two cannot drift.
//   - The banned names (the names/ entries `abcd banlist add --public` writes) are
//     the repository's own, and none is seeded: abcd cannot know which names a repo
//     may not publish, and a ban nobody declared would fail a build over a word the
//     repository never chose.
//
// The harness token family is withheld too, as a per-repository fit decision: it
// refuses naming a specific agent tool, which is right for abcd's published surface
// and wrong for a repository whose content is teaching those tools. The parity test
// names every deliberate omission with its reason.
//
// The punctuation/em-dash-in-list-item token is carried, at a severity the
// ADOPTER chooses: it is abcd's house style rather than a currency rule (it drew
// 419 of the 545 findings in the repository that reported the empty seed), so
// the install offers it as blocking or warning and an unattended install seeds
// the warning (ruling G1; docsLintSeed renders the choice). The embedded file
// carries the warning, so it is a loadable config as it stands.
//
// The stray_root_docs allowlist names CLAUDE and AGENTS, the two root files the
// scaffold itself may write, so the seeded gate does not refuse its own output.
//
//go:embed defaults/docs-lint.json
var publicFamilySeed string

// privateStubBody is the scaffolded private banlist: the format declaration, the
// format's documentation, and worked examples that are ALL COMMENTED OUT.
//
// Commented, because a scaffolded live entry would be a ban the user never declared;
// and because the guard must report this machine as unprotected until someone opts
// in, which is exactly what a zero-entry store makes it do (its loud NO ENTRIES
// warning). Every illustrative value is a reserved documentation value or a
// persona-derived fixture host, per examples-use-reserved-identifiers: this file is
// scaffolded into every managed repo, so an example that looked like a real host or
// address would teach the shape of the leak the layer exists to prevent.
const privateStubHead = `# abcd-banlist: keyed
# abcd private banlist — LOCAL TO THIS MACHINE, never committed.
#
# The committed pre-commit and pre-merge-commit guards (.githooks/) refuse any
# commit whose staged content matches an entry below, naming the entry's KEY alone.
# Their reach:
`

// privateStubTail is everything after the reach note.
const privateStubTail = `#
# That is the design, not a gap: a pattern written into public CI config is a
# published pattern. Names a repo may state out loud belong in the public family
# instead (.abcd/docs-lint.json), where the lint gates them for everyone.
#
# The first line above declares the KEYED format: every entry line is
#     KEY<space-or-tab>PATTERN
#   KEY      a stable, non-sensitive handle ([A-Za-z0-9][A-Za-z0-9._/-]*). It is
#            the only part of an entry any output ever names.
#   PATTERN  a POSIX extended regular expression, matched case-insensitively by
#            grep. Inline flag groups and Perl escapes are NOT supported there:
#            no (?i) — matching is already case-insensitive — and no \d, \w, \b.
# Blank lines and lines starting with '#' are ignored; leading and trailing ASCII
# spaces and tabs are stripped. A line that does not parse is refused by number,
# never skipped. Machine identifiers — hostnames, IP addresses, CIDR prefixes, MAC
# addresses, device names — are ordinary entries.
#
# Add entries with ` + "`abcd banlist add --private <key> <pattern>`" + `, which keeps the
# pattern out of argv and shell history when you pipe it:
#     printf %s 'PATTERN' | abcd banlist add --private lab-host -
#
# Worked examples, commented out. Uncomment nothing: replace them with your own
# values. Every value here is a reserved documentation value (RFC 5737, RFC 3849,
# RFC 2606, RFC 7042) or a persona-derived fixture host, so nothing below names a
# real machine:
#
#   lab-host      alice-laptop\.example\.com
#   lab-device    bob-desktop
#   lab-ipv4      192\.0\.2\.17
#   lab-subnet    198\.51\.100\.
#   lab-ipv6      2001:db8:
#   lab-mac       00:00:5e:00:53:
#   lab-domain    carol-server\.example\.org
#   lab-project   a-private-project-name
#
# Remove the first line and every line below becomes a whole-line pattern again.
`

// privateStubContent is the stub's bytes as scaffolding writes them.
//
// The reach paragraph is DERIVED from banlist.PrivateReachNote rather than restated:
// this header ships into every managed repo, so a maintainer reads it as the account
// of what their own guard covers. A hand-copied list drifts — and the way it drifts
// is by omission, which reads as coverage.
func privateStubContent() string {
	return privateStubHead + commentWrap(banlist.PrivateReachNote) + privateStubTail
}

// commentWrap renders text as `# `-prefixed lines inside the stub's 80-column
// header, breaking on whitespace. A backtick-quoted span CAN straddle a line break
// — the wrap knows nothing about backticks — which costs nothing here, because the
// stub is a plain-text file whose readers are people, not a markdown renderer.
func commentWrap(text string) string {
	const width = 76
	var out strings.Builder
	line := "#"
	for _, word := range strings.Fields(text) {
		if len(line)+1+len(word) > width && line != "#" {
			out.WriteString(line + "\n")
			line = "#"
		}
		line += " " + word
	}
	if line != "#" {
		out.WriteString(line + "\n")
	}
	return out.String()
}

// ---------------------------------------------------------------------------
// classification
// ---------------------------------------------------------------------------

// HookState is what occupies one of the guard's two hook paths.
type HookState string

const (
	HookAbsent    HookState = "absent"
	HookInstalled HookState = "installed"
	// HookForeign is a hook abcd did not write. It is NOT a failure of the repo —
	// a maintainer's own pre-commit hook is legitimate — but it is a state in which
	// nothing checks the banlist, so it can never be reported as installed.
	HookForeign HookState = "foreign"
	// HookUnreadable is a path that exists and cannot be read as a hook (a symlink,
	// a directory, an oversize file). Like GuardHealth's unknown state it asserts
	// nothing it did not check.
	HookUnreadable HookState = "unreadable"
)

// classifyHook reports what occupies rel, judged against the marker line that
// identifies a hook abcd wrote. Identity comes from that line, never from mere
// presence: a repo's own hook at one of these paths is legitimate, and calling it
// "abcd's" would mean nothing runs the check while the status board says something
// does.
//
// It takes the marker as a parameter because abcd scaffolds hooks for two
// unrelated conventions — the name guard and the attribution prompt — and a repo
// may adopt either without the other. One classifier, two markers; a second copy of
// this function is how the two would drift into judging containment differently.
func classifyHook(root *os.Root, rel string, marker *regexp.Regexp) HookState {
	fi, err := root.Lstat(rel)
	switch {
	case err != nil && os.IsNotExist(err):
		return HookAbsent
	case err != nil:
		return HookUnreadable
	case fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular():
		return HookUnreadable
	}
	// Read THROUGH the root, like the write path: a read resolved outside the
	// containment scope would classify a file the scaffold could never touch, which
	// is the read/write asymmetry that lets a report describe a different file from
	// the one apply acts on.
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxAhoyFileBytes)
	if err != nil {
		return HookUnreadable
	}
	if marker.Match(data) {
		return HookInstalled
	}
	return HookForeign
}

// classifyGuardHook is classifyHook against the name guard's marker — the form
// every guard call site reads better as.
func classifyGuardHook(root *os.Root, rel string) HookState {
	return classifyHook(root, rel, guardHookMarkerRe)
}

// PublicFamilyState classifies the public layer's config for detection.
type PublicFamilyState string

const (
	PublicFamilyPresent PublicFamilyState = "present"
	// PublicFamilyAbsent is no docs-lint config at all — scaffolding writes one.
	PublicFamilyAbsent PublicFamilyState = "absent"
	// PublicFamilyUnusable is a config that parses to no usable banned_tokens array.
	// abcd must not rewrite it: the config gates CI and a contributor owns it.
	PublicFamilyUnusable PublicFamilyState = "unusable"
	// PublicFamilyUnreadable is an I/O fault — an oversize file, a symlink, a
	// permission error. It is reported apart from "unusable" because "add a
	// banned_tokens array" is not the fix for a file nobody can open.
	PublicFamilyUnreadable PublicFamilyState = "unreadable"
	// PublicFamilyIgnored is the contradiction: a readable family in a file git
	// ignores. The layer's whole claim is that it is committed and CI-enforced, and
	// an ignored config is enforced by nobody — most sharply under
	// `visibility: public`, where the abcd fence ignores the whole `.abcd/` namespace
	// and public exposure is the risk the layer exists for.
	PublicFamilyIgnored PublicFamilyState = "ignored"
)

// classifyPublicFamily reports whether the repo's docs-lint config carries a
// banned-names family CI can actually enforce. It asks the banlist package about the
// family rather than re-reading the file, so detection and the verbs cannot disagree
// about what "the family is present" means (one canonical primitive), and it asks
// git whether the file is enforceable at all rather than inferring it from a
// configured visibility.
func classifyPublicFamily(cwd string, root *os.Root, ignored bool) PublicFamilyState {
	if _, err := fsutil.ReadGuardedInRoot(root, banlist.PublicConfigRelPath, maxAhoyFileBytes); err != nil {
		if os.IsNotExist(err) {
			return PublicFamilyAbsent
		}
		return PublicFamilyUnreadable
	}
	if _, err := banlist.ListPublic(cwd); err != nil {
		return PublicFamilyUnusable
	}
	// git's own verdict: check-ignore consults the index, so a config that is already
	// committed reads as not-ignored whatever the fence says — which is the right
	// answer, because it IS in the repo and CI sees it.
	if ignored {
		return PublicFamilyIgnored
	}
	return PublicFamilyPresent
}

// ignoreVerdict is git's answer about the two paths the banlist cares about,
// obtained in ONE subprocess. `git check-ignore` is the only authority on the
// question: a text comparison of the abcd-managed .gitignore block answers a
// different one, because a repo can carry a byte-perfect block and still track a
// path (a negation after it, a tracked tier) and a repo that never reached the
// visibility step has no block at all.
//
// answerable=false means git could not be asked. That is NOT the same as "safe":
// see storePathIsSafe.
//
// It is a WEAKER signal than it looks. gitutil.CheckIgnored fails open — git absent,
// a probe that errored, and "nothing here is ignored" all come back as an empty set —
// so answerable only reports that this directory is a work tree, never that the
// ignore probe itself succeeded. Both write gates therefore treat an empty answer as
// unanswerable rather than as "not ignored, go ahead".
type ignoreVerdict struct {
	answerable bool
	private    bool
	public     bool
}

// checkBanlistIgnores asks git once about both paths. check-ignore answers for a
// path that does not exist yet, which is what makes it usable as a placement test
// for a config abcd is about to write.
func checkBanlistIgnores(cwd string) ignoreVerdict {
	if !gitutil.InRepo(cwd) {
		return ignoreVerdict{}
	}
	set := gitutil.CheckIgnored(cwd, []string{banlist.PrivateRelPath, banlist.PublicConfigRelPath})
	if len(set) == 0 {
		// Indistinguishable from a probe that errored (CheckIgnored folds both into an
		// empty set), and inside an abcd-managed repo at least the private store's path
		// IS ignored — so an empty answer here is far likelier to be a failed probe
		// than a true "nothing is ignored". Report it as unanswerable and let the write
		// gates fail closed.
		return ignoreVerdict{}
	}
	_, priv := set[banlist.PrivateRelPath]
	_, pub := set[banlist.PublicConfigRelPath]
	return ignoreVerdict{answerable: true, private: priv, public: pub}
}

// storePathIsSafe reports whether scaffolding may create the private store here.
//
// Three states, not two. Inside a repo, git decides. Outside any repo the check is
// SKIPPED and the path is safe, exactly as banlist's own requireIgnoredStore skips
// it — a directory that is not a repository cannot commit anything, so a scaffold
// cannot demand proof no one can supply. But a directory that LOOKS like a repo and
// that git will not answer for — git missing from PATH, a corrupt .git, a version
// that fails the probe — is neither: something here can still be committed, and
// nothing can say whether this path would be. That fails closed, because the whole
// hazard is a store entering history.
func storePathIsSafe(cwd string) bool {
	if v := checkBanlistIgnores(cwd); v.answerable {
		return v.private
	}
	return !repoShaped(cwd)
}

// publicPathIsWritable is storePathIsSafe's counterpart for the docs-lint config,
// and it fails closed the same way. Writing a config into a path git ignores
// delivers a family CI never sees; writing one on an answer git did not actually
// give is the same wager with the evidence missing.
func publicPathIsWritable(cwd string) bool {
	if v := checkBanlistIgnores(cwd); v.answerable {
		return !v.public
	}
	return !repoShaped(cwd)
}

// repoShaped tells "not a repository" apart from "a repository git would not
// answer for". The walk lives in gitutil.RepoShaped so banlist's own write gate
// shares the one discriminator (banlist cannot import ahoy); this wrapper keeps
// the scaffold's tests pinning the behaviour at this boundary.
func repoShaped(cwd string) bool { return gitutil.RepoShaped(cwd) }

// ---------------------------------------------------------------------------
// health
// ---------------------------------------------------------------------------

// BanlistHealth is the two-layer name guard's state in one repo: which of the
// scaffolded artefacts are there, what shape the private layer is in, and — always —
// how far it reaches (spc-20 AC7).
//
// Reach is carried in the struct rather than added by a renderer because this report
// is read on two surfaces at once. A human sees a line; a machine consumer reads the
// fields. "hook: installed" beside a populated private store reads as "covered" to
// both of them, and it is not: the guard runs on machines that opted in, for the
// commits git asks a hook about, and nowhere else. The caveat travels with the state
// it qualifies.
type BanlistHealth struct {
	// Hook and MergeHook are the committed guard's two halves. A merge commit runs
	// NO pre-commit hook, so a repo with only the first half is uncovered for exactly
	// the commit that merges someone else's branch.
	Hook      HookState `json:"hook"`
	MergeHook HookState `json:"merge_hook"`
	// HooksPathArmed reports whether THIS clone's local git config points at the
	// committed hooks directory. False is not a fault — the hooks path can be armed
	// by a user-level dispatcher this deliberately does not claim to see — so the
	// surfaces phrase it as an instruction, never as an accusation.
	HooksPathArmed bool `json:"hooks_path_armed"`
	// PublicFamily is the committed layer's state, including the case where it is
	// present and readable but git ignores the file, so CI never sees it.
	PublicFamily PublicFamilyState `json:"public_family"`
	// HookEOLPinned reports whether .gitattributes keeps the committed hooks at LF.
	// Meaningful only where abcd owns the hooks: pinning the EOL of a directory whose
	// contents are someone else's is not abcd's call.
	HookEOLPinned bool `json:"hook_eol_pinned"`
	// PublicConfigIgnored is git's verdict on the config's PATH, whether or not a
	// file is there yet. It is what makes the absent-family gap honest: writing a
	// config into a path git ignores delivers no enforcement, so abcd does not offer
	// to.
	PublicConfigIgnored bool `json:"public_config_ignored"`
	// PublicConfigWritable reports whether scaffolding may create the config here. It
	// is NOT the negation of PublicConfigIgnored: a probe git could not answer is
	// neither ignored nor safe, and it fails closed inside anything repo-shaped.
	PublicConfigWritable bool `json:"public_config_writable"`
	// PrivateStore reports whether THIS MACHINE has opted in. False is the honest
	// "inactive here" state, never silence: an absent store checks nothing, and the
	// guard says so at commit time for the same reason.
	PrivateStore bool `json:"private_store"`
	// PrivateStoreIgnored is git's verdict on the store's path. A present store git
	// would track is one `git add -A` from committing the patterns it exists to keep
	// out of history.
	PrivateStoreIgnored bool `json:"private_store_ignored"`
	// PrivateKeyed, PrivateEntries and PrivateUnparsed are the store's SHAPE, never
	// its content: a format flag and two counts, taken from the shared parser. A line
	// that does not parse stops every commit until it is fixed.
	PrivateKeyed    bool `json:"private_keyed"`
	PrivateEntries  int  `json:"private_entries"`
	PrivateUnparsed int  `json:"private_unparsed"`
	// PrivateUnreadable reports a store that exists and cannot be read for what it
	// is — a damaged format declaration, an unreadable file. The guard refuses every
	// commit in that state, so a status board must not render it as healthy.
	PrivateUnreadable bool `json:"private_unreadable"`
	// Reach is PrivateReachNote, unconditionally.
	Reach string `json:"reach"`
}

// detectBanlistHealth answers the artefact questions for one repo. It reads the
// store's SHAPE and never its content: the patterns are the secret, and a status
// board is exactly the surface that must not hold them.
func detectBanlistHealth(cwd string) BanlistHealth {
	// ONE ignore probe for both paths, and one config read: a detection pass runs on
	// every status render and on the session hooks, so each extra fork is paid on
	// every prompt.
	ign := checkBanlistIgnores(cwd)
	storeSafe, publicWritable := ign.private, !ign.public
	if !ign.answerable {
		shaped := repoShaped(cwd)
		storeSafe, publicWritable = !shaped, !shaped
	}
	h := BanlistHealth{
		PublicConfigIgnored:  ign.public,
		PublicConfigWritable: publicWritable,
		PrivateStoreIgnored:  storeSafe,
		Reach:                banlist.PrivateReachNote,
	}
	// One containment root for every read, opened where the writes are: a report
	// resolved outside it could describe a file apply can never act on.
	root, err := os.OpenRoot(cwd)
	if err != nil {
		h.Hook, h.MergeHook = HookUnreadable, HookUnreadable
		h.PublicFamily = PublicFamilyUnreadable
		h.PrivateStore, h.PrivateUnreadable = true, true
		return h
	}
	defer root.Close()
	h.Hook = classifyGuardHook(root, GuardHookRelPath)
	h.MergeHook = classifyGuardHook(root, GuardMergeHookRelPath)
	h.HooksPathArmed = hooksPathArmed(cwd)
	h.PublicFamily = classifyPublicFamily(cwd, root, ign.public)
	h.HookEOLPinned = gitattributesPinsHookEOL(cwd, root)
	sum, serr := banlist.SummarisePrivate(cwd)
	if serr != nil {
		// A store that exists and cannot be read for what it is: present, and health
		// says so rather than rendering the absence of a readable store as "inactive".
		h.PrivateStore, h.PrivateUnreadable = true, true
		return h
	}
	h.PrivateStore = sum.Present
	h.PrivateKeyed = sum.Keyed
	h.PrivateEntries = sum.Entries
	h.PrivateUnparsed = sum.Unparsed
	return h
}

// guardEOLAttributeRe matches abcd's own pin line, live (git ignores a line
// beginning `#`). It is the FALLBACK classifier only — what a file is pinned to is
// `git check-attr`'s answer, and this stands in when git cannot be asked at all.
// Keeping an append from duplicating is a different question, answered against the
// file's last live attribute (see lastLiveAttributeIsOurs).
var guardEOLAttributeRe = regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(guardEOLAttribute) + `[ \t\r]*$`)

// gitattributesPinsHookEOL reports whether git will keep the committed hooks at LF.
// It ASKS GIT — `check-attr eol` on the hook itself — because the question is what
// git does on checkout, and only git knows: a pattern match on abcd's own line calls
// a file pinned when a later `* text eol=crlf` overrides it, and misses a live line
// that differs in whitespace. The same authority this package already grants git for
// the ignore question.
//
// When git cannot answer — not a repository, git absent — it falls back to the
// textual check rather than asserting "unpinned": a gap that cannot be verified must
// not be reported as a fault a re-run would fix.
func gitattributesPinsHookEOL(cwd string, root *os.Root) bool {
	if out, err := gitutil.Run(cwd, "check-attr", "eol", "--", GuardHookRelPath); err == nil {
		return strings.HasSuffix(strings.TrimSpace(out), ": eol: lf")
	}
	data, err := fsutil.ReadGuardedInRoot(root, gitattributesRelPath, maxAhoyFileBytes)
	if err != nil {
		return false
	}
	return guardEOLAttributeRe.Match(data)
}

// hooksPathArmed reports whether this clone's LOCAL git config points core.hooksPath
// at the committed hooks directory. `--local` reads only .git/config, so the probe's
// own `-c core.hooksPath=…` isolation cannot answer for it. `--type=path` makes git
// expand `~/`, `~user/` and `%(prefix)/` as it does when it runs a hook, so the
// value judged is the directory git would use, never its spelling.
//
// A false answer is deliberately weak evidence: the hooks path can also be armed by
// a user-level dispatcher this never sees, so no surface may turn it into "the guard
// is not running" — only into "arm it like this".
func hooksPathArmed(cwd string) bool {
	out, err := gitutil.Run(cwd, "config", "--local", "--type=path", "--get", "core.hooksPath")
	if err != nil {
		return false
	}
	set := strings.TrimSpace(out)
	if set == "" {
		return false
	}
	// Both sides resolved before comparing: git accepts an ABSOLUTE core.hooksPath,
	// and a literal comparison against ".githooks" reads a perfectly armed clone as
	// unarmed — at which point the surface tells the user to replace a working
	// absolute path with a relative one, which is advice to downgrade their config.
	if !filepath.IsAbs(set) {
		set = filepath.Join(cwd, set)
	}
	return filepath.Clean(set) == filepath.Clean(filepath.Join(cwd, filepath.FromSlash(guardHooksDirRelPath)))
}

// ---------------------------------------------------------------------------
// detection
// ---------------------------------------------------------------------------

// detectBanlistScaffold reports the name guard's missing or occupied artefacts
// (spc-20 AC5). Each is state-keyed: the gap is the artefact's state, never a
// version stamp, so a hand-deleted hook comes back and a present one is left alone.
// It consumes the health pass rather than re-reading the repo, so a gap and the
// status line beside it can never disagree.
func detectBanlistScaffold(h BanlistHealth) []Gap {
	var gaps []Gap
	gaps = append(gaps, hookGaps("banlist.hook", GuardHookRelPath, "pre-commit", h.Hook)...)
	gaps = append(gaps, mergeHookGaps(h)...)
	if h.Hook == HookInstalled && !h.HookEOLPinned {
		gaps = append(gaps, Gap{
			ID: "banlist.hook_eol_unpinned", Category: SafeAutocreate, Scope: "repo",
			Title:      "the committed guard is not pinned to LF",
			Detail:     gitattributesRelPath + " does not carry `" + guardEOLAttribute + "`, so a clone with core.autocrlf=true rewrites the guard to CRLF and its shebang stops resolving — the hook silently stops running.",
			FixHint:    "ahoy install appends the attribute line (it never rewrites the rest of the file).",
			Required:   true,
			Resolvable: true,
		})
	}
	gaps = append(gaps, publicFamilyGaps(h)...)
	gaps = append(gaps, privateStoreGaps(h)...)
	return gaps
}

// mergeHookGaps reports the merge half, whose state is only meaningful RELATIVE to
// the half it delegates to. The shim runs whatever occupies GuardHookRelPath, so
// "the shim is present" says nothing on its own: beside a foreign hook it means
// merges run the maintainer's hook instead of the guard, and beside no hook at all
// it means merges run nothing.
func mergeHookGaps(h BanlistHealth) []Gap {
	if h.Hook == HookInstalled {
		// The ordinary case: the guard is abcd's, so the merge half is abcd's to write
		// and its own state decides.
		return hookGaps("banlist.merge_hook", GuardMergeHookRelPath, "pre-merge-commit", h.MergeHook)
	}
	if h.Hook == HookAbsent && h.MergeHook != HookInstalled {
		// Nothing to mislead anyone with, and the very next apply writes both halves.
		// Reported once, by the pre-commit half's own gap, rather than twice for one
		// cause.
		return nil
	}
	// The wording must not assert a shim that is not there: with a foreign pre-commit
	// hook and no merge half at all, "delegates to" describes a file that does not
	// exist, and a reader sent to inspect it finds nothing.
	detail := "git runs no pre-commit hook for a merge commit, and " + GuardHookRelPath + " is " +
		string(h.Hook) + ", so abcd writes no merge half beside it — merge commits are NOT checked " +
		"against the private banlist."
	if h.MergeHook == HookInstalled {
		detail = GuardMergeHookRelPath + " carries abcd's marker but delegates to " + GuardHookRelPath +
			", which is " + string(h.Hook) + " — so merge commits are NOT checked against the private " +
			"banlist, and the marker would otherwise read as coverage."
	}
	return []Gap{{
		ID: "banlist.merge_hook_inert", Category: PluginOwned, Scope: "repo",
		Title:      "merge commits are not checked against the private banlist",
		Detail:     detail,
		FixHint:    "restore the abcd guard at " + GuardHookRelPath + " (move a foreign hook aside and re-run `abcd ahoy install`); the merge half is written beside it, never beside a hook abcd does not own.",
		Required:   false,
		Resolvable: false,
	}}
}

// publicFamilyGaps turns the public layer's state into gaps. Only a genuinely absent
// config is abcd's to write; every other fault names its own remedy, because "add a
// banned_tokens array" is the wrong instruction for a file nobody can open and for
// one git will never show CI.
func publicFamilyGaps(h BanlistHealth) []Gap {
	switch h.PublicFamily {
	case PublicFamilyAbsent:
		// Resolvable only where a written config would actually be enforced. Under
		// `visibility: public` the fence ignores the whole .abcd/ namespace, so writing
		// one there produces a file abcd would immediately report as unenforceable —
		// an offer to fix that fixes nothing.
		fix := "ahoy install writes the docs-lint config with an empty banned-names family."
		if !h.PublicConfigWritable && !h.PublicConfigIgnored {
			fix = "git could not be asked whether " + banlist.PublicConfigRelPath + " would be ignored here, " +
				"so abcd will not write a config it cannot check; make git available in this repo and re-run."
		}
		if h.PublicConfigIgnored {
			fix = "git ignores " + banlist.PublicConfigRelPath + " here, so a config written there would reach no CI run; " +
				"settle the placement question (iss-176) — commit the path explicitly, or ban the name on the private layer instead."
		}
		return []Gap{{
			ID: "banlist.public_family_missing", Category: SafeAutocreate, Scope: "repo",
			Title:      "public banned-names family absent",
			Detail:     banlist.PublicConfigRelPath + " is absent, so no banned name is gated in CI.",
			FixHint:    fix,
			Required:   true,
			Resolvable: h.PublicConfigWritable,
		}}
	case PublicFamilyUnusable:
		// Diagnostic only: the config gates CI and a contributor owns it, so abcd
		// reports the fault and never rewrites a file it cannot read (the same posture
		// stepConfigValues takes towards a malformed config.json).
		return []Gap{{
			ID: "banlist.public_family_unusable", Category: ConfigChange, Scope: "repo",
			Title:      "public banned-names family is not readable",
			Detail:     banlist.PublicConfigRelPath + " carries no usable top-level banned_tokens array, so `abcd banlist add --public` cannot write to it.",
			FixHint:    "add a top-level \"banned_tokens\": [] array to " + banlist.PublicConfigRelPath + " (`abcd banlist list --public` names the fault).",
			Required:   false,
			Resolvable: false,
		}}
	case PublicFamilyUnreadable:
		return []Gap{{
			ID: "banlist.public_family_unreadable", Category: ConfigChange, Scope: "repo",
			Title:      "public banned-names config cannot be read",
			Detail:     banlist.PublicConfigRelPath + " exists but cannot be read (not a regular file, oversize, or unreadable).",
			FixHint:    "restore " + banlist.PublicConfigRelPath + " as a regular file abcd can read.",
			Required:   false,
			Resolvable: false,
		}}
	case PublicFamilyIgnored:
		return []Gap{{
			ID: "banlist.public_family_ignored", Category: ConfigChange, Scope: "repo",
			Title:  "public banned-names family is not enforceable",
			Detail: "git ignores " + banlist.PublicConfigRelPath + ", so the family it carries never reaches CI — and the public layer's whole claim is that it is committed and enforced for everyone.",
			FixHint: "commit " + banlist.PublicConfigRelPath + " (`git add -f`), or ban the name on the private layer instead; " +
				"under `visibility: public` the abcd fence ignores the whole .abcd/ namespace, which is a placement question a maintainer must settle.",
			Required:   false,
			Resolvable: false,
		}}
	}
	return nil
}

// privateStoreGaps turns the private layer's state into gaps.
func privateStoreGaps(h BanlistHealth) []Gap {
	var gaps []Gap
	if !h.PrivateStore {
		// Resolvable ONLY when git itself reports the store's path as ignored. Writing
		// the stub into a repo that would track it is the exact hazard the layer exists
		// to prevent, and advertising a resolvable gap apply refuses to close would
		// leave the repo permanently "partial" (the markerSymlink precedent).
		fix := "ahoy install writes the documented stub into the gitignored local tier."
		if !h.PrivateStoreIgnored {
			fix = "git does not ignore " + banlist.PrivateRelPath + " — add `" + banlist.PrivateDirRelPath +
				"/` to .gitignore (ahoy install writes that fence once this repo has a configured visibility), then re-run."
		}
		gaps = append(gaps, Gap{
			ID: "banlist.private_stub_missing", Category: SafeAutocreate, Scope: "repo",
			Title:      "private banlist stub absent",
			Detail:     banlist.PrivateRelPath + " is absent, so the private layer is inactive on this machine.",
			FixHint:    fix,
			Required:   true,
			Resolvable: h.PrivateStoreIgnored,
		})
		return gaps
	}
	if h.PrivateUnreadable {
		gaps = append(gaps, Gap{
			ID: "banlist.private_store_unreadable", Category: ConfigChange, Scope: "repo",
			Title:      "private banlist cannot be read",
			Detail:     banlist.PrivateRelPath + " exists but cannot be read for what it is, so the guard refuses every commit.",
			FixHint:    "`abcd banlist list --private` names the fault; the store's content is withheld by design.",
			Required:   false,
			Resolvable: false,
		})
	}
	if !h.PrivateStoreIgnored {
		gaps = append(gaps, Gap{
			ID: "banlist.private_store_tracked", Category: ConfigChange, Scope: "repo",
			Title:      "private banlist is not gitignored",
			Detail:     "git does not ignore " + banlist.PrivateRelPath + ", so it is one `git add -A` from committing the patterns it exists to keep out of history.",
			FixHint:    "add `" + banlist.PrivateDirRelPath + "/` to .gitignore, and check the path is not already tracked.",
			Required:   false,
			Resolvable: false,
		})
	}
	return gaps
}

// hookGaps turns one hook's state into gaps. An absent hook is abcd's to write; a
// foreign one is the maintainer's and is reported, never replaced — mirroring
// GuardHealth's posture, where wiring abcd does not own is a diagnostic rather than
// something apply silently takes over.
//
// The two differ in Required deliberately. Required means "install has work to do
// here": true for the absent hook, which the next apply writes, and false for a
// foreign one, which no apply will ever close — a required gap nothing can resolve
// is a repo permanently reported as incomplete for a state its maintainer chose.
func hookGaps(idPrefix, rel, name string, state HookState) []Gap {
	switch state {
	case HookAbsent:
		return []Gap{{
			ID: idPrefix + "_missing", Category: SafeAutocreate, Scope: "repo",
			Title:  "private name guard's " + name + " hook not committed",
			Detail: rel + " is absent, so no " + name + " on any clone is checked against a private banlist.",
			FixHint: "ahoy install writes the guard hook; point git at it with " +
				"`git config core.hooksPath " + guardHooksDirRelPath + "`.",
			Required: true, Resolvable: true,
		}}
	case HookForeign:
		return []Gap{{
			ID: idPrefix + "_foreign", Category: PluginOwned, Scope: "repo",
			Title:      "a foreign " + name + " hook occupies " + rel,
			Detail:     rel + " carries no `" + guardHookMarker + "` line, so abcd did not write it and the private banlist is not checked on this path.",
			FixHint:    "chain the abcd guard from it by hand, or move it aside and re-run `abcd ahoy install`.",
			Required:   false,
			Resolvable: false,
		}}
	case HookUnreadable:
		return []Gap{{
			ID: idPrefix + "_unreadable", Category: PluginOwned, Scope: "repo",
			Title:      rel + " cannot be read as a hook",
			Detail:     rel + " exists but is a symlink, a directory, or otherwise unreadable, so abcd can neither identify nor replace it.",
			FixHint:    "restore " + rel + " as a regular file, or remove it and re-run `abcd ahoy install`.",
			Required:   false,
			Resolvable: false,
		}}
	}
	return nil
}

// ---------------------------------------------------------------------------
// apply
// ---------------------------------------------------------------------------

// stepBanlist scaffolds the name guard: the two committed hooks, the public family,
// and the gitignored private stub (spc-20 AC5). Every write is create-if-absent — a
// hook, a CI-gating config, and above all a populated private store are the
// maintainer's, and re-seeding one would delete work abcd cannot see.
//
// Every write is CONTAINED: it resolves through an os.Root opened at the repo, so a
// symlink committed at `.githooks` or at the local tier cannot land a 0755 hook or a
// 0600 stub outside the repo while every surface reports the in-repo path.
func (a *applyCtx) stepBanlist() {
	if !a.approved[SafeAutocreate] {
		return
	}
	root, err := os.OpenRoot(a.cwd)
	if err != nil {
		return
	}
	defer root.Close()

	// Is the guard the merge shim would delegate to abcd's own? Asked HERE, against
	// disk, rather than read from the detection snapshot: the steps before this one
	// have run, and a snapshot is by then a claim about a repo as it used to be.
	guardOwned := classifyGuardHook(root, GuardHookRelPath) == HookInstalled
	if a.has("banlist.hook_missing") {
		if a.createContained(root, GuardHookRelPath, guardHookTemplate, 0o755, 0o755) {
			guardOwned = true
		}
	}
	// The merge half is written ONLY beside abcd's own pre-commit guard. The shim
	// delegates to whatever occupies that path, so writing it beside a foreign hook
	// would do two wrong things at once: abcd's marker would land in the shim and the
	// board would read "merge hook committed" while merges stayed unchecked, and the
	// maintainer's hook would silently start running on merge commits — git never ran
	// it there, and apply would be taking over wiring the foreign-hook gap explicitly
	// disclaims owning. The state is reported instead (banlist.merge_hook_inert).
	// Keyed on the merge half's own STATE, not on its gap: on a fresh repo the gap is
	// deliberately not raised (the pre-commit half's gap covers both), and apply must
	// still write the shim it just earned the right to write.
	if guardOwned && classifyGuardHook(root, GuardMergeHookRelPath) == HookAbsent {
		a.createContained(root, GuardMergeHookRelPath, guardMergeHookTemplate, 0o755, 0o755)
	}
	// Never write a config abcd would immediately declare unenforceable — and never on
	// an answer git did not actually give. An unanswerable probe withholds the write
	// for this run exactly as it withholds the stub's: a config written into a path
	// nobody could check is the same wager on both halves.
	//
	// The em-dash house-style question is asked HERE, where the seed is about to be
	// written, and nowhere else: a repository that already has a config keeps its
	// own severity, and a question whose answer would go nowhere is not asked.
	if a.has("banlist.public_family_missing") && publicPathIsWritable(a.cwd) {
		sev, note := a.emDashSeverity()
		if a.createContained(root, banlist.PublicConfigRelPath, docsLintSeed(sev), 0o644, 0o755) && note != "" {
			a.notes = append(a.notes, note)
		}
	}
	// Re-asked HERE, after stepVisibility has written the fence, and answered by git
	// rather than by a comparison of .gitignore text: what matters is whether git
	// would track this path now, on disk.
	// The EOL pin belongs with the hooks and only with them: keyed on guardOwned, not
	// on the gap, for the same reason the shim is (a fresh repo raises no gap for a
	// hook that does not exist yet).
	if guardOwned {
		a.pinHookEOL(root)
	}
	if a.has("banlist.private_stub_missing") && storePathIsSafe(a.cwd) {
		// 0700/0600: the directory holding private patterns is no more readable than
		// the patterns are, matching the store the banlist verbs write.
		a.createContained(root, banlist.PrivateRelPath, []byte(privateStubContent()), 0o600, 0o700)
	}
}

// pinHookEOL appends the hooks' EOL attribute to .gitattributes, creating the file
// when it is absent. It is an APPEND, not a rewrite: .gitattributes is the
// maintainer's, may carry merge drivers and filters abcd knows nothing about, and a
// file it cannot read is left alone rather than replaced with a file it can.
func (a *applyCtx) pinHookEOL(root *os.Root) {
	if gitattributesPinsHookEOL(a.cwd, root) {
		return
	}
	data, err := fsutil.ReadGuardedInRoot(root, gitattributesRelPath, maxAhoyFileBytes)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		data = nil
	default:
		return
	}
	// git says the hooks are not pinned, but the canonical line may already be there
	// and simply overridden by a later one. Appending it again is the fix — ours then
	// wins as the last match — and this keeps a SECOND re-run from appending a third.
	if bytes.Equal(data, appendEOLPin(data)) {
		return
	}
	if err := fsutil.WriteFileAtomicPreserveModeInRoot(root, gitattributesRelPath, appendEOLPin(data)); err != nil {
		return
	}
	a.note(filepath.Join(a.cwd, gitattributesRelPath))
}

// appendEOLPin returns data with the canonical attribute appended, or data unchanged
// when the canonical line is already its LAST live attribute (appending a duplicate
// after itself changes nothing but the diff).
func appendEOLPin(data []byte) []byte {
	if lastLiveAttributeIsOurs(data) {
		return data
	}
	var buf bytes.Buffer
	buf.Write(data)
	if len(data) > 0 && !bytes.HasSuffix(data, []byte("\n")) {
		buf.WriteString("\n")
	}
	if len(data) > 0 {
		buf.WriteString("\n")
	}
	buf.WriteString("# The committed name guard is a script git EXECUTES. A checkout with\n")
	buf.WriteString("# core.autocrlf=true would rewrite it to CRLF and its shebang would stop\n")
	buf.WriteString("# resolving, so the guard would silently stop running.\n")
	buf.WriteString(guardEOLAttribute + "\n")
	return buf.Bytes()
}

// lastLiveAttributeIsOurs reports whether the file's final non-blank, non-comment
// line is exactly abcd's pin — the only case where appending it again would be pure
// noise.
func lastLiveAttributeIsOurs(data []byte) bool {
	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimRight(strings.TrimSpace(lines[i]), "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line == guardEOLAttribute
	}
	return false
}

// createContained writes data at rel inside root when nothing is there, noting the
// write. rel is slash-separated and repo-relative.
//
// It uses CreateExclusiveIn, not WriteFileAtomic: the exclusive create IS both
// guarantees at once — the file cannot already exist, and every component is resolved
// through the Root, so a symlinked ancestor is refused rather than followed. An
// atomic rename would clobber whatever was there and would depend on the caller
// having resolved the directory safely beforehand, which is precisely the
// check-then-write window a scaffold must not have.
//
// Any failure — the file already exists (the ordinary idempotent no-op), an escaping
// symlink, a permission fault — leaves the artefact unwritten and unnoted. Detection
// reports it on the next pass rather than this run claiming a file it did not create.
func (a *applyCtx) createContained(root *os.Root, rel string, data []byte, perm, dirPerm os.FileMode) bool {
	if dir := path.Dir(rel); dir != "." {
		// MkdirAll on the PARENT, and it may create more than one level: every artefact
		// here sits at most two deep under the repo root (.githooks/, .abcd/.work.local/),
		// both of which are abcd's own namespaces, so there is no third party's directory
		// for it to bring into being.
		//
		// The mode is pinned ONLY on a directory this call created. MkdirAll applies the
		// process umask to the mode it is given, so a umask of 077 would leave the hooks
		// directory unsearchable — but a directory the maintainer already made is theirs,
		// and re-applying dirPerm to it WIDENS a deliberately tighter mode (0700 -> 0755).
		// A pin that loosens permissions nobody asked it to touch is the inverse of the
		// bug it exists for.
		_, statErr := root.Lstat(dir)
		created := statErr != nil
		if err := root.MkdirAll(dir, dirPerm); err != nil {
			return false
		}
		if created {
			if err := root.Chmod(dir, dirPerm); err != nil {
				return false
			}
		}
	}
	if err := fsutil.CreateExclusiveIn(root, rel, data, perm); err != nil {
		return false
	}
	// A umask-stripped exec bit is not cosmetic here: it turns the merge shim's
	// fail-closed branch into a permanent block on every merge commit.
	if err := root.Chmod(rel, perm); err != nil {
		return false
	}
	a.note(filepath.Join(a.cwd, filepath.FromSlash(rel)))
	return true
}
