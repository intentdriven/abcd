package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAhoyStatusStatesThePrivateBanlistReach is spc-20 AC7 on the status board.
// `abcd ahoy` reports the two-layer name guard, and a reader who sees "hook
// committed" beside a list of private keys would reasonably assume a pull request
// is covered. It is not, and it cannot be: a pattern in CI config is a published
// pattern. The status surface says so in the same words the verb does.
func TestAhoyStatusStatesThePrivateBanlistReach(t *testing.T) {
	hermeticEnv(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)

	out := string(runCLI(t, "ahoy"))
	if !strings.Contains(out, "banlist:") {
		t.Fatalf("bare ahoy does not report the name-banlist layers:\n%s", out)
	}
	if !strings.Contains(out, banlist.PrivateReachNote) {
		t.Errorf("the status board does not state the private layer's local-only reach (%q):\n%s",
			banlist.PrivateReachNote, out)
	}
	// The reach is only honest if it names what a pre-commit hook cannot see. An
	// opted-in machine is still uncovered for a rebase, a `git am`, a cherry-pick,
	// and any commit made with --no-verify.
	for _, want := range []string{"rebase", "no-verify"} {
		if !strings.Contains(out, want) {
			t.Errorf("the stated reach omits %q, so it overclaims coverage:\n%s", want, out)
		}
	}
}

// TestAhoyStatusReportsEachScaffoldedArtefact keeps the line useful as well as
// honest: a reader has to be able to tell which artefact is missing, and whether
// this machine has opted into the private layer at all.
func TestAhoyStatusReportsEachScaffoldedArtefact(t *testing.T) {
	hermeticEnv(t)
	// A REAL repository: the stub write is gated on git's own ignore verdict, and a
	// bare `.git` directory is a state the guard deliberately fails closed on.
	repo := gittest.NewRepo(t).Root()
	t.Chdir(repo)

	bare := string(runCLI(t, "ahoy"))
	if !strings.Contains(bare, "INACTIVE") {
		t.Errorf("an absent private store must read as inactive, never as silence:\n%s", bare)
	}

	if _, err := runCLIErr(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false"); err != nil {
		t.Fatalf("install: %v", err)
	}
	after := string(runCLI(t, "ahoy"))
	for _, want := range []string{"pre-commit hook committed", "pre-merge-commit hook committed", "public family", "private store"} {
		if !strings.Contains(after, want) {
			t.Errorf("the banlist line omits %q after scaffolding:\n%s", want, after)
		}
	}
	if strings.Contains(after, "INACTIVE") {
		t.Errorf("the private store is scaffolded, so the line must not report it inactive:\n%s", after)
	}
	// A committed hook is not a running hook: git only runs it once this clone's
	// hooks path points at the committed directory, and the line must not let
	// "committed" be read as "armed".
	if !strings.Contains(after, "core.hooksPath") {
		t.Errorf("the line claims the hook is in place without naming how to arm a clone:\n%s", after)
	}
}

// TestAhoyEnvelopeCarriesTheReach: the JSON envelope is a report surface too, and a
// machine consumer that reads the fields without the caveat would draw exactly the
// wrong conclusion. The reach travels with the state it qualifies.
func TestAhoyEnvelopeCarriesTheReach(t *testing.T) {
	hermeticEnv(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)

	var env struct {
		Banlist struct {
			Reach        string `json:"reach"`
			Hook         string `json:"hook"`
			MergeHook    string `json:"merge_hook"`
			PublicFamily string `json:"public_family"`
		} `json:"banlist"`
	}
	if err := json.Unmarshal(runCLI(t, "ahoy", "--dry-run"), &env); err != nil {
		t.Fatalf("dry-run envelope does not parse: %v", err)
	}
	if env.Banlist.Reach != banlist.PrivateReachNote {
		t.Errorf("envelope reach = %q; want %q", env.Banlist.Reach, banlist.PrivateReachNote)
	}
	// Each artefact's state is a NAMED state, not a boolean: "absent", "foreign" and
	// "unreadable" are three different things a consumer must be able to tell apart,
	// and a bool would collapse two of them into "installed".
	if env.Banlist.Hook != "absent" || env.Banlist.MergeHook != "absent" {
		t.Errorf("hook states = %q/%q; want absent/absent in a bare repo", env.Banlist.Hook, env.Banlist.MergeHook)
	}
	if env.Banlist.PublicFamily != "absent" {
		t.Errorf("public family = %q; want absent", env.Banlist.PublicFamily)
	}
}

// TestBanlistLineNeverClaimsAForeignHookIsTheGuard is correctness M2 at the
// surface. A repo's own pre-commit hook is legitimate; calling it "the abcd guard"
// is the worst of both states — nothing checks the banlist and the board says
// something does.
func TestBanlistLineNeverClaimsAForeignHookIsTheGuard(t *testing.T) {
	line := strings.Join(banlistHealthLines(ahoy.BanlistHealth{
		Hook:         ahoy.HookForeign,
		MergeHook:    ahoy.HookAbsent,
		PublicFamily: ahoy.PublicFamilyPresent,
	}), " ")
	if strings.Contains(line, "pre-commit hook committed") {
		t.Errorf("a foreign hook is rendered as the abcd guard: %q", line)
	}
	if !strings.Contains(line, "foreign") {
		t.Errorf("the line does not name the foreign hook: %q", line)
	}
}

// TestBanlistLineDistinguishesPublicFamilyFaults is m5/m6: "absent", "unusable",
// "unreadable" and "ignored" need four different responses, so they cannot share
// one rendering. An ignored family is the sharpest — it is present, readable, and
// enforced by nobody.
func TestBanlistLineDistinguishesPublicFamilyFaults(t *testing.T) {
	for state, want := range map[ahoy.PublicFamilyState]string{
		ahoy.PublicFamilyAbsent:     "MISSING",
		ahoy.PublicFamilyUnusable:   "unusable",
		ahoy.PublicFamilyUnreadable: "unreadable",
		ahoy.PublicFamilyIgnored:    "NOT ENFORCEABLE",
	} {
		line := strings.Join(banlistHealthLines(ahoy.BanlistHealth{PublicFamily: state}), " ")
		if !strings.Contains(line, want) {
			t.Errorf("public family %q renders as %q; want it to say %q", state, line, want)
		}
	}
}

// TestBanlistLineReportsThePrivateStoresShape is m1: a status board may say how
// many entries a store holds and whether any line is unusable — those are counts,
// not content — and it must, because a store whose lines do not parse stops every
// commit while looking populated.
func TestBanlistLineReportsThePrivateStoresShape(t *testing.T) {
	line := strings.Join(banlistHealthLines(ahoy.BanlistHealth{
		Hook: ahoy.HookInstalled, MergeHook: ahoy.HookInstalled, PublicFamily: ahoy.PublicFamilyPresent,
		PrivateStore: true, PrivateStoreIgnored: true, PrivateKeyed: true,
		PrivateEntries: 3, PrivateUnparsed: 1,
	}), " ")
	if !strings.Contains(line, "3 entries") {
		t.Errorf("the line does not report the entry count: %q", line)
	}
	if !strings.Contains(line, "1 unusable") {
		t.Errorf("the line does not report the unusable line, which stops every commit: %q", line)
	}
}

// TestBanlistLineWarnsWhenTheStoreIsNotIgnored: the layer's entire safety is that
// the store is untracked, so a store git would commit is the one state the board
// must never render quietly.
func TestBanlistLineWarnsWhenTheStoreIsNotIgnored(t *testing.T) {
	line := strings.Join(banlistHealthLines(ahoy.BanlistHealth{
		Hook: ahoy.HookInstalled, MergeHook: ahoy.HookInstalled, PublicFamily: ahoy.PublicFamilyPresent,
		PrivateStore: true, PrivateStoreIgnored: false, PrivateEntries: 2,
	}), " ")
	if !strings.Contains(line, "NOT gitignored") {
		t.Errorf("a trackable private store is not called out: %q", line)
	}
}

// TestBanlistLineNeverOffersARemedyInstallRefuses: when the pre-commit half is
// foreign, install deliberately does NOT write the merge shim — it would delegate to
// a hook abcd does not own. Telling a reader "`abcd ahoy install` writes it" sends
// them to run a command that will decline, and the second run teaches them the
// status board is wrong rather than that the state is.
func TestBanlistLineNeverOffersARemedyInstallRefuses(t *testing.T) {
	line := strings.Join(banlistHealthLines(ahoy.BanlistHealth{
		Hook:         ahoy.HookForeign,
		MergeHook:    ahoy.HookAbsent,
		PublicFamily: ahoy.PublicFamilyPresent,
	}), " ")
	if strings.Contains(line, "`abcd ahoy install` writes it") {
		t.Errorf("the line offers a remedy install refuses to perform: %q", line)
	}
	if !strings.Contains(line, "aside") {
		t.Errorf("the line does not carry the remedy the merge_hook_inert gap gives: %q", line)
	}
	// The ordinary absent case keeps its ordinary remedy.
	ordinary := strings.Join(banlistHealthLines(ahoy.BanlistHealth{
		Hook: ahoy.HookAbsent, MergeHook: ahoy.HookAbsent, PublicFamily: ahoy.PublicFamilyPresent,
	}), " ")
	if !strings.Contains(ordinary, "`abcd ahoy install` writes it") {
		t.Errorf("an absent hook install WILL write must still say so: %q", ordinary)
	}
}
