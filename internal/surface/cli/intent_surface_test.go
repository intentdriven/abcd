package cli

import (
	"encoding/json"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// iss-104 acceptance corpus for extending the unrecognized-input-never-writes
// guard (iss-29) to the intent quoted-text create path. A mistyped intent
// subverb must error with a did-you-mean and file no draft; a genuine multi-word
// draft title whose first word merely resembles a subverb still files. The guard
// is id-aware for intent's itd/spc ids (the faithful-mirror + id-aware choice).

var reIntentDraftFile = regexp.MustCompile(`^itd-\d+-.*\.md$`)

// intentDraftCount walks a repo tree and counts written intent draft files.
func intentDraftCount(t *testing.T, root string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && reIntentDraftFile.MatchString(d.Name()) {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return n
}

// TestIntentTypoSubcommandNeverWrites is the headline: `intent lnk itd-5` (a
// typo for `link` followed by an itd id) must be refused with a did-you-mean and
// must not file a draft. Before the fix it was swallowed as create text.
func TestIntentTypoSubcommandNeverWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out, err := runCLIErr(t, "intent", "lnk", "itd-5")
	if err == nil {
		t.Fatalf("expected an error for the mistyped subcommand, got success:\n%s", out)
	}
	if !strings.Contains(err.Error(), "link") {
		t.Fatalf("expected a did-you-mean pointing at %q, got: %v", "link", err)
	}
	if n := intentDraftCount(t, repo); n != 0 {
		t.Fatalf("a mistyped subcommand filed %d draft(s); it must write nothing", n)
	}
}

// TestIntentTypoLoneTokenNeverWrites covers the lone-token shape: `intent paln`
// (a typo for `plan`, no trailing arg) must be refused, not filed.
func TestIntentTypoLoneTokenNeverWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out, err := runCLIErr(t, "intent", "paln")
	if err == nil {
		t.Fatalf("expected an error for the lone mistyped subcommand, got success:\n%s", out)
	}
	if !strings.Contains(err.Error(), "plan") {
		t.Fatalf("expected a did-you-mean pointing at %q, got: %v", "plan", err)
	}
	if n := intentDraftCount(t, repo); n != 0 {
		t.Fatalf("a mistyped subcommand filed %d draft(s); it must write nothing", n)
	}
}

// TestIntentIdShapeDistinguishesTypoFromProse pins the id-aware widening (the
// maintainer's chosen behaviour): with the SAME near-miss first token, a record
// id (itd/spc) as the second token trips the guard, while a prose second token
// does not — so a regression narrowing recordIDRe back to iss-only would be
// caught here, not silently.
func TestIntentIdShapeDistinguishesTypoFromProse(t *testing.T) {
	// itd/spc second token -> shaped like a subcommand call -> refused.
	for _, id := range []string{"itd-5", "spc-2"} {
		repo := t.TempDir()
		t.Chdir(repo)
		out, err := runCLIErr(t, "intent", "lnk", id)
		if err == nil {
			t.Fatalf("intent lnk %s must be refused as a typoed subcommand, got:\n%s", id, out)
		}
		if n := intentDraftCount(t, repo); n != 0 {
			t.Fatalf("intent lnk %s filed %d draft(s); must write nothing", id, n)
		}
	}
	// Same first token, prose second token -> a genuine title -> still files.
	repo := t.TempDir()
	t.Chdir(repo)
	runCLI(t, "intent", "lnk", "widen", "the", "public", "api")
	if n := intentDraftCount(t, repo); n != 1 {
		t.Fatalf("a prose title after a verb-ish first word wrote %d draft(s), want 1", n)
	}
}

// TestIntentFreeTextTitleStillWrites guards the high-precision contract: a
// genuine multi-word draft title whose first word resembles a subverb but is
// followed by prose (not a record id) still files.
func TestIntentFreeTextTitleStillWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out := runCLI(t, "intent", "plans", "the", "release", "cadence", "for", "next", "quarter", "--json")
	var r struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("intent output not JSON: %v\n%s", err, out)
	}
	if !cliNativeIntentIDRe.MatchString(r.ID) {
		t.Fatalf("free-text draft id = %q, want a native itd id", r.ID)
	}
	if n := intentDraftCount(t, repo); n != 1 {
		t.Fatalf("free-text draft wrote %d draft(s), want 1", n)
	}
}

// TestIntentAuditRenameCleanBreak (spc-28): the audit spelling is live and the
// old review spelling is an unknown sub-command — a clean break, no alias.
func TestIntentAuditRenameCleanBreak(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	if _, err := runCLIErr(t, "intent", "review", "itd-1"); err == nil ||
		!strings.Contains(err.Error(), "unknown") {
		t.Fatalf("intent review must be an unknown sub-command after the rename, got: %v", err)
	}
	// The retired TWO-word spelling refuses too — flagless `intent review
	// ingest` used to exit 2 asking for --verdict-json; it must never be
	// swallowed as free text and filed (review regression).
	if _, err := runCLIErr(t, "intent", "review", "ingest"); err == nil ||
		!strings.Contains(err.Error(), "audit") {
		t.Fatalf("intent review ingest must refuse naming the successor, got: %v", err)
	}
	if _, err := runCLIErr(t, "intent", "review", "ingest", "v.json"); err == nil {
		t.Fatalf("intent review ingest <arg> must refuse, got success")
	}
	// Genuine free text whose first word is 'review' still files.
	out := runCLI(t, "intent", "review the loader's retry budget before shipping", "--json")
	if !strings.Contains(string(out), "itd-") {
		t.Fatalf("free text starting with 'review' must still file:\n%s", out)
	}
	// audit resolves as a registered sub-command (it fails on the missing
	// intent, not as unknown).
	if _, err := runCLIErr(t, "intent", "audit", "itd-1"); err == nil ||
		strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("intent audit must be registered, got: %v", err)
	}
}

// iss-2608221328552172 acceptance corpus. The typo guard above only catches a
// NEAR-MISS of a real sub-verb; a lone token that resembles nothing —
// `abcd intent nosuchthing` — fell straight through it into the create path and
// minted a durable draft at exit 0. A one-word positional is ambiguous with a
// sub-verb call by construction, so the create path takes prose only.

// TestIntentLoneWordNeverWrites is the headline: a single bare word that is not
// a sub-verb and is not near one must be refused, and must file no draft.
func TestIntentLoneWordNeverWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out, err := runCLIErr(t, "intent", "nosuchthing")
	if err == nil {
		t.Fatalf("expected an error for the lone unknown token, got success:\n%s", out)
	}
	if n := intentDraftCount(t, repo); n != 0 {
		t.Fatalf("a lone unknown token filed %d draft(s); it must write nothing", n)
	}
}

// TestIntentLoneRecordIDNeverWrites covers the other lone-token shape a user
// actually types: `abcd intent itd-5` (a forgotten sub-verb) must not become a
// draft whose title is a record id.
func TestIntentLoneRecordIDNeverWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out, err := runCLIErr(t, "intent", "itd-5")
	if err == nil {
		t.Fatalf("expected an error for the lone record id, got success:\n%s", out)
	}
	if n := intentDraftCount(t, repo); n != 0 {
		t.Fatalf("a lone record id filed %d draft(s); it must write nothing", n)
	}
}

// TestIntentQuotedProseStillWritesAsOneArg pins the other half: the canonical
// create path passes ONE argument carrying whitespace (the shell has eaten the
// quotes), and it must still file. A fix that refused every single-argument
// invocation would break the documented create path, and this catches it.
func TestIntentQuotedProseStillWritesAsOneArg(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	runCLI(t, "intent", "widen the public api")
	if n := intentDraftCount(t, repo); n != 1 {
		t.Fatalf("quoted prose wrote %d draft(s), want 1", n)
	}
}

// TestIntentEmptyTextNeverWrites is the intent half of the empty-positional case.
func TestIntentEmptyTextNeverWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out, err := runCLIErr(t, "intent", "")
	if err == nil {
		t.Fatalf("expected an error for an empty draft title, got success:\n%s", out)
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected the error to name the title as empty, got: %v", err)
	}
	if n := intentDraftCount(t, repo); n != 0 {
		t.Fatalf("an empty draft title filed %d draft(s); it must write nothing", n)
	}
}

// iss-2609091647589392: the FAR-miss half of the same hole. The typo guard
// decided first whether the arguments were shaped like a subcommand call — a
// token followed by a record id satisfies that shape — and then refused only
// when some registered sub-verb lay within an edit distance of two. A word
// nothing like `plan`, `ready`, `link` or `audit` therefore passed a guard that
// had already concluded the input was a subcommand call, and was filed as the
// title of a draft in the durable record tier.

// TestIntentFarMissSubcommandNeverWrites is the headline: `intent nosuchverb itd-5`
// is shaped like a subcommand call and `nosuchverb` is no sub-verb of any distance,
// so it must be refused at exit 2 with the registered sub-verbs named, and must
// file no draft.
func TestIntentFarMissSubcommandNeverWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out, err := runCLIErr(t, "intent", "nosuchverb", "itd-5")
	if err == nil {
		t.Fatalf("expected an error for the far-missed subcommand, got success:\n%s", out)
	}
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("far-missed subcommand exited %d, want 2 (usage): %v", code, err)
	}
	if !strings.Contains(err.Error(), "nosuchverb") {
		t.Fatalf("expected the refusal to quote the unknown token, got: %v", err)
	}
	for _, sub := range []string{"plan", "ready", "link", "audit"} {
		if !strings.Contains(err.Error(), sub) {
			t.Fatalf("expected the refusal to list the sub-verb %q, got: %v", sub, err)
		}
	}
	if n := intentDraftCount(t, repo); n != 0 {
		t.Fatalf("a far-missed subcommand filed %d draft(s); it must write nothing", n)
	}
}

// TestIntentNearMissKeepsDidYouMean pins the half that must NOT change: a
// near-miss still names the one sub-verb it is near, rather than degrading into
// the far-miss listing.
func TestIntentNearMissKeepsDidYouMean(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	_, err := runCLIErr(t, "intent", "lnk", "itd-5")
	if err == nil {
		t.Fatalf("expected an error for the near-missed subcommand")
	}
	if !strings.Contains(err.Error(), "did you mean") || !strings.Contains(err.Error(), `"link"`) {
		t.Fatalf("expected a did-you-mean naming link, got: %v", err)
	}
}

// TestIntentFarMissProseStillWrites is the precision half: the same unknown
// first token followed by PROSE rather than a record id is a genuine title and
// still files. The far-miss refusal keys on the subcommand shape, not on the
// first word alone.
func TestIntentFarMissProseStillWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	runCLI(t, "intent", "nosuchverb", "the", "release", "gate", "before", "cutting")
	if n := intentDraftCount(t, repo); n != 1 {
		t.Fatalf("prose after an unknown first word wrote %d draft(s), want 1", n)
	}
}

// TestIntentRetiredSubverbSurvivesFarMissBranch: the retired spelling is
// refused naming its SUCCESSOR, not by the far-miss listing — the far-miss
// branch runs after the retired lookup and must not swallow it.
func TestIntentRetiredSubverbSurvivesFarMissBranch(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	_, err := runCLIErr(t, "intent", "review", "itd-1")
	if err == nil {
		t.Fatalf("expected an error for the retired sub-verb")
	}
	if !strings.Contains(err.Error(), `"audit"`) {
		t.Fatalf("expected the retired spelling to name its successor audit, got: %v", err)
	}
}

// TestIntentQuotedProseBeforeRecordIDStillWrites pins the precision boundary of
// the far-miss branch: a sub-verb spelling is ONE whitespace-free token, so a
// quoted multi-word first argument is prose even when a record id follows it,
// and it still files. Without that half the far-miss refusal would reach a
// legitimate title a user happened to end with an id.
func TestIntentQuotedProseBeforeRecordIDStillWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	runCLI(t, "intent", "widen the public api", "itd-5")
	if n := intentDraftCount(t, repo); n != 1 {
		t.Fatalf("quoted prose before a record id wrote %d draft(s), want 1", n)
	}
}
