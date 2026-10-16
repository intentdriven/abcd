package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The committed pre-push hook's work-in-progress entry (spc-2610120941441282
// step 6, itd-2610120738075547 C7, adr-2610120937435856, brief invariant 23).
// A cloud session cannot earn a preflight receipt where it runs, and no cloud
// receipt is trusted anyway (decision 9), so its work goes to
// <wip_prefix><item-id> without one. The prefix is not written in the hook: it
// is each prefix a host connector declares, read through
// scripts/cloud/connector.sh. The push passes without a receipt only when the
// branch is the prefix and one record id, and every commit it moves passes the
// attribution gate; any other push under the prefix is gated as any branch is.
//
// The fixture is newPrePushCase's clone with the connector loader, the default
// branch reader and the attribution gate copied in, and one connector of its
// own, so no test here names a real host.

const (
	wipHost   = "testhost"
	wipPrefix = wipHost + "/abcd-wip/"
	wipItem   = "iss-2610120729087253"
	wipRef    = "refs/heads/" + wipPrefix + wipItem
	wipConf   = "scripts/cloud/connectors/" + wipHost + "/connector.conf"
	assisted  = "Assisted-by: Claude:claude-opus-5"
	// An AI vendor's identity, which the attribution gate refuses.
	wipVendorName = "Claude"
	wipVendorMail = "noreply@anthropic.com"
)

// newWIPCase is a pre-push case carrying the cloud scripts the hook reads and
// one declared connector, all on origin's default branch, so a push from the
// feature branch moves only the commits a test makes.
func newWIPCase(t *testing.T) *prePushCase {
	t.Helper()
	bin := builtAbcd.Path(t)
	c := newPrePushCase(t)
	c.env = append(c.env, "ABCD_OUTBOUND_BIN="+bin)
	top, err := c.run(".", "git", "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("not in a git checkout: %s", top)
	}
	root := strings.TrimSpace(top)
	for _, rel := range []string{"scripts/cloud/connector.sh", "scripts/cloud/default-branch.sh", "scripts/check-attribution.sh",
		"internal/core/identity/tool-identities.txt"} {
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read the committed %s: %v", rel, err)
		}
		c.write(rel, string(src), 0o755)
	}
	c.write(wipConf, "name="+wipHost+"\nwip_prefix="+wipPrefix+"\nis_remote=is-remote.sh\n", 0o644)
	c.write("scripts/cloud/connectors/"+wipHost+"/is-remote.sh", "#!/usr/bin/env bash\nexit 1\n", 0o755)
	c.git("add", "-A")
	c.git("-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", "fixture: the cloud scripts")
	c.git("-c", "core.hooksPath=/dev/null", "push", "-q", "origin", "HEAD:main")
	c.git("fetch", "-q", "origin")
	return c
}

// commitMsg commits one file with the message given.
func (c *prePushCase) commitMsg(rel, content, msg string) string {
	c.t.Helper()
	c.write(rel, content, 0o644)
	c.git("add", rel)
	c.git("commit", "-q", "-m", msg)
	return c.git("rev-parse", "HEAD")
}

// attributed is a commit the attribution gate passes: a person's identity and
// the Assisted-by trailer.
func (c *prePushCase) attributed(rel string) string {
	c.t.Helper()
	return c.commitMsg(rel, rel+"\n", "fix: "+rel+"\n\n"+assisted)
}

// preflightRan reports whether the hook ran the fixture's preflight.
func (c *prePushCase) preflightRan() bool {
	_, err := os.Stat(filepath.Join(c.dir, ".abcd", ".work.local", "preflight-ran"))
	return err == nil
}

// TestPrePushAdmitsWIPWithoutReceipt: a create and an update of
// <prefix><item-id>, every commit attribution-clean, pass with no receipt and
// never run the preflight.
func TestPrePushAdmitsWIPWithoutReceipt(t *testing.T) {
	c := newWIPCase(t)
	first := c.attributed("one.md")
	if out, err := c.push("origin", "HEAD:"+wipRef); err != nil {
		t.Fatalf("a create of %s was refused although it is the prefix and one record id and its commit is attribution-clean\n%s", wipRef, out)
	}
	if got := c.remoteRef(wipRef); got != first {
		t.Fatalf("after the create the remote holds %q, want %s", got, first)
	}
	second := c.attributed("two.md")
	third := c.attributed("three.md")
	if out, err := c.push("origin", "HEAD:"+wipRef); err != nil {
		t.Fatalf("an update of %s was refused although every commit it moves is attribution-clean\n%s", wipRef, out)
	}
	if got := c.remoteRef(wipRef); got != third {
		t.Fatalf("after the update the remote holds %q, want %s (over %s)", got, third, second)
	}
	if c.preflightRan() {
		t.Errorf("the hook ran the preflight for a work-in-progress push")
	}

	// Every id shape the spec names: a timestamp iss-, itd- or spc- id and an
	// ordinal itd- or iss- id.
	for _, id := range []string{"itd-2610120738075547", "spc-2610120941441282", "itd-152", "iss-7"} {
		if out, err := c.push("origin", "HEAD:refs/heads/"+wipPrefix+id); err != nil {
			t.Errorf("a create of %s%s was refused\n%s", wipPrefix, id, out)
		}
	}
}

// TestPrePushRefusesOtherHostBranchWithoutReceipt: only a declared prefix is
// exempt. Another branch in the host's namespace, a prefix that merely starts
// like the declared one, and the segment under an undeclared namespace all
// need a receipt.
func TestPrePushRefusesOtherHostBranchWithoutReceipt(t *testing.T) {
	for _, branch := range []string{
		wipHost + "/fix-x",
		wipHost + "/abcd-wipe/" + wipItem,
		"other/abcd-wip/" + wipItem,
		"abcd-wip/" + wipItem,
	} {
		t.Run(branch, func(t *testing.T) {
			c := newWIPCase(t)
			c.attributed("one.md")
			out, err := c.push("origin", "HEAD:refs/heads/"+branch)
			if err == nil || !strings.Contains(out, "no passing preflight") {
				t.Fatalf("%s passed without a receipt (err %v)\n%s", branch, err, out)
			}
			if got := c.remoteRef("refs/heads/" + branch); got != "" {
				t.Errorf("the remote moved to %s although the push was refused", got)
			}
		})
	}
}

// TestPrePushRefusesWIPWithBadRecordID: under the prefix, the branch is one
// record id and nothing else. No id, two ids and a trailing path are each
// gated as any branch is.
func TestPrePushRefusesWIPWithBadRecordID(t *testing.T) {
	for name, rest := range map[string]string{
		"no id":                 "fix-x",
		"an id too short":       "iss-261012072908725",
		"an unknown family":     "adr-2610120937435856",
		"two ids":               wipItem + "-itd-2610120738075547",
		"two ids by a slash":    wipItem + "/itd-2610120738075547",
		"a trailing path":       wipItem + "/notes",
		"a trailing word":       wipItem + "-fix",
		"an ordinal too long":   "itd-15200",
		"an ordinal spc- id":    "spc-12",
		"an upper-case family":  "ISS-2610120729087253",
		"an id after a segment": "x/" + wipItem,
	} {
		t.Run(name, func(t *testing.T) {
			c := newWIPCase(t)
			c.attributed("one.md")
			ref := "refs/heads/" + wipPrefix + rest
			out, err := c.push("origin", "HEAD:"+ref)
			if err == nil || !strings.Contains(out, "no passing preflight") {
				t.Fatalf("%s passed without a receipt (err %v)\n%s", ref, err, out)
			}
			if !strings.Contains(out, "one record id") {
				t.Errorf("the refusal does not say the branch must be the prefix and one record id\n%s", out)
			}
			if got := c.remoteRef(ref); got != "" {
				t.Errorf("the remote moved to %s although the push was refused", got)
			}
		})
	}
}

// TestPrePushRefusesWIPFailingAttribution: a work-in-progress push is
// receipt-less only when every commit it moves passes the attribution gate. An
// AI co-author trailer, or no Assisted-by trailer, anywhere in the range gates
// the push, on a create and on an update.
func TestPrePushRefusesWIPFailingAttribution(t *testing.T) {
	for name, msg := range map[string]string{
		"an AI co-author trailer": "fix: the walk\n\nCo-Authored-By: An Assistant <assistant@example.com>\n" + assisted,
		"no Assisted-by trailer":  "fix: the walk",
	} {
		t.Run(name+" on a create", func(t *testing.T) {
			c := newWIPCase(t)
			c.commitMsg("bad.md", "bad\n", msg)
			c.attributed("good.md")
			out, err := c.push("origin", "HEAD:"+wipRef)
			if err == nil || !strings.Contains(out, "no passing preflight") {
				t.Fatalf("a create carrying a commit with %s passed without a receipt (err %v)\n%s", name, err, out)
			}
			if !strings.Contains(out, "the pushed commits fail the attribution gate") ||
				!strings.Contains(out, "run scripts/check-attribution.sh commits ") {
				t.Errorf("the refusal does not say the attribution gate refused, and how to see why\n%s", out)
			}
			if strings.Contains(out, "check-attribution: FAILED") {
				t.Errorf("the hook repeats the attribution gate's own findings, which may name an identity\n%s", out)
			}
			if got := c.remoteRef(wipRef); got != "" {
				t.Errorf("the remote moved to %s although the push was refused", got)
			}
		})
		t.Run(name+" on an update", func(t *testing.T) {
			c := newWIPCase(t)
			first := c.attributed("good.md")
			if out, err := c.push("origin", "HEAD:"+wipRef); err != nil {
				t.Fatalf("the clean create was refused\n%s", out)
			}
			c.commitMsg("bad.md", "bad\n", msg)
			out, err := c.push("origin", "HEAD:"+wipRef)
			if err == nil || !strings.Contains(out, "no passing preflight") {
				t.Fatalf("an update carrying a commit with %s passed without a receipt (err %v)\n%s", name, err, out)
			}
			if got := c.remoteRef(wipRef); got != first {
				t.Errorf("the remote holds %q, want %s untouched", got, first)
			}
		})
	}
}

// TestPrePushWIPAttributionRefusalNamesNoIdentity: the attribution gate's
// findings name the identity it refuses. On a direct `git push` the hook keeps
// them to itself and prints a fixed sentence instead, so a vendor-authored
// work-in-progress push is refused without the hook ever printing the
// vendor's name or address (wipVendorName and wipVendorMail).
func TestPrePushWIPAttributionRefusalNamesNoIdentity(t *testing.T) {
	c := newWIPCase(t)
	c.write("bad.md", "bad\n", 0o644)
	c.git("add", "bad.md")
	c.git("-c", "user.name="+wipVendorName, "-c", "user.email="+wipVendorMail, "commit", "-q", "-m", "fix: the walk\n\n"+assisted)
	out, err := c.push("origin", "HEAD:"+wipRef)
	if err == nil || !strings.Contains(out, "no passing preflight") {
		t.Fatalf("a vendor-authored work-in-progress push passed without a receipt (err %v)\n%s", err, out)
	}
	if got := c.remoteRef(wipRef); got != "" {
		t.Errorf("the remote moved to %s although the push was refused", got)
	}
	// The sentence covers every refusal of the attribution gate, the identity
	// half and the message half, and names no identity.
	if !strings.Contains(out, attributionSentence) {
		t.Errorf("the refusal does not give the fixed sentence\n%s", out)
	}
	for _, s := range []string{wipVendorName, wipVendorMail, wipVendorMail[strings.Index(wipVendorMail, "@")+1:]} {
		if strings.Contains(out, s) {
			t.Errorf("the hook printed the refused identity's %q:\n%s", s, out)
		}
	}
}

// attributionSentence is the fixed sentence wip_shape prints when the pushed
// commits fail the attribution gate: each cause the gate refuses, no identity.
const attributionSentence = "the pushed commits fail the attribution gate (a machine author or committer, " +
	"a missing or malformed Assisted-by trailer, an AI Co-Authored-By trailer, or a tool footer or session URL " +
	"in a message); run scripts/check-attribution.sh commits "

// committedPointer is the bad-declaration lines' pointer: the hook validates
// the declarations HEAD commits, and connector.sh validate reads the working
// tree, so the pointer says which tree it judged and how to compare.
const committedPointer = "scripts/cloud/connector.sh validate names it, as committed at HEAD (commit or discard local edits to compare)"

// badDeclarations are connector declarations connector.sh refuses.
var badDeclarations = map[string]string{
	"a bare host namespace": "name=" + wipHost + "\nwip_prefix=" + wipHost + "/\nis_remote=is-remote.sh\n",
	"the default branch":    "name=" + wipHost + "\nwip_prefix=main\nis_remote=is-remote.sh\n",
	"two segments before":   "name=" + wipHost + "\nwip_prefix=a/b/abcd-wip/\nis_remote=is-remote.sh\n",
	"an unknown key":        "name=" + wipHost + "\nwip_prefix=" + wipPrefix + "\nis_remote=is-remote.sh\nexempt=all\n",
}

// commitDeclaration commits conf as the test connector's declaration: the
// hook reads the declarations from the committed tree, never the working one.
func (c *prePushCase) commitDeclaration(conf string) {
	c.t.Helper()
	c.write(wipConf, conf, 0o644)
	c.git("add", wipConf)
	c.git("commit", "-q", "-m", "chore: a declaration\n\n"+assisted)
}

// TestPrePushRefusesOnBadDeclaration: the prefixes come from the committed
// connector declarations through connector.sh's own validation. A declaration
// that fails it adds no work-in-progress row, and a push to a branch holding
// the segment abcd-wip is refused, naming the file; every other push is judged
// as if no connector existed, so one with a receipt goes ahead.
func TestPrePushRefusesOnBadDeclaration(t *testing.T) {
	for name, conf := range badDeclarations {
		t.Run(name, func(t *testing.T) {
			c := newWIPCase(t)
			c.attributed("one.md")
			c.commitDeclaration(conf)
			out, err := c.push("origin", "HEAD:"+wipRef)
			if err == nil {
				t.Fatalf("a push passed the hook with a declaration that fails validation\n%s", out)
			}
			if !strings.Contains(out, wipConf) {
				t.Errorf("the refusal does not name the declaration that failed\n%s", out)
			}
			if got := c.remoteRef(wipRef); got != "" {
				t.Errorf("the remote moved to %s although the push was refused", got)
			}
		})
	}
	t.Run("an ordinary branch with a receipt", func(t *testing.T) {
		c := newWIPCase(t)
		c.attributed("one.md")
		c.commitDeclaration("name=" + wipHost + "\nwip_prefix=main\nis_remote=is-remote.sh\n")
		c.preflight(c.dir)
		out, err := c.push("origin", "feature")
		if err != nil {
			t.Fatalf("an ordinary push with a receipt was refused for a bad declaration it does not use\n%s", out)
		}
		if !strings.Contains(out, wipConf) {
			t.Errorf("the hook did not report the declaration that fails validation\n%s", out)
		}
		if !strings.Contains(out, committedPointer) {
			t.Errorf("the report does not say the check is of the declarations committed at HEAD\n%s", out)
		}
	})
	t.Run("an ordinary branch without a receipt", func(t *testing.T) {
		c := newWIPCase(t)
		c.attributed("one.md")
		c.commitDeclaration("name=" + wipHost + "\nwip_prefix=main\nis_remote=is-remote.sh\n")
		out, err := c.push("origin", "feature")
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Fatalf("an ordinary push with no receipt passed under a bad declaration (err %v)\n%s", err, out)
		}
	})
}

// TestPrePushStrayConnectorDirDoesNotBlockOrdinaryPush: a directory under
// scripts/cloud/connectors/ with no connector.conf, or a connector script that
// lost its exec bit, fails the declarations' validation. That costs the
// work-in-progress rows and nothing else: an ordinary push with its receipt,
// and a reservation, still go ahead.
func TestPrePushStrayConnectorDirDoesNotBlockOrdinaryPush(t *testing.T) {
	for name, damage := range map[string]func(c *prePushCase){
		"a stray directory": func(c *prePushCase) {
			c.write("scripts/cloud/connectors/stray/notes.md", "notes\n", 0o644)
			c.git("add", "scripts/cloud/connectors/stray/notes.md")
		},
		"a lost exec bit": func(c *prePushCase) {
			c.git("update-index", "--chmod=-x", "scripts/cloud/connectors/"+wipHost+"/is-remote.sh")
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := newWIPCase(t)
			c.attributed("one.md")
			damage(c)
			c.git("commit", "-q", "-m", "chore: damage\n\n"+assisted)
			c.git("checkout", "-q", "--", ".")
			c.preflight(c.dir)
			out, err := c.push("origin", "feature")
			if err != nil {
				t.Fatalf("an ordinary push with its receipt was refused for a bad declaration\n%s", out)
			}
			if !strings.Contains(out, "scripts/cloud/connectors/") {
				t.Errorf("the hook did not report the declaration that fails validation\n%s", out)
			}
			empty := c.emptyTreeCommit("", "reserve")
			if out, err := c.push("origin", empty+":"+reservedRef); err != nil {
				t.Errorf("a reservation was refused for a bad declaration\n%s", out)
			}
		})
	}
}

// TestPrePushBadDeclarationStillRefusesWIPName: under a declaration that fails
// validation, a push to any branch holding the segment abcd-wip, in any case,
// is refused naming the bad declaration, even when it carries a receipt.
func TestPrePushBadDeclarationStillRefusesWIPName(t *testing.T) {
	for _, ref := range []string{wipRef, "refs/heads/other/abcd-wip/" + wipItem, "refs/heads/Other/ABCD-WIP/" + wipItem} {
		t.Run(ref, func(t *testing.T) {
			c := newWIPCase(t)
			c.attributed("one.md")
			c.commitDeclaration("name=" + wipHost + "\nwip_prefix=main\nis_remote=is-remote.sh\n")
			c.preflight(c.dir)
			out, err := c.push("origin", "HEAD:"+ref)
			if err == nil {
				t.Fatalf("a push to %s passed under a declaration that fails validation\n%s", ref, out)
			}
			if !strings.Contains(out, wipConf) || !strings.Contains(out, "abcd-wip") {
				t.Errorf("the refusal does not name the bad declaration and the segment\n%s", out)
			}
			if !strings.Contains(out, committedPointer) {
				t.Errorf("the refusal does not say the check is of the declarations committed at HEAD\n%s", out)
			}
			if got := c.remoteRef(ref); got != "" {
				t.Errorf("the remote moved to %s although the push was refused", got)
			}
		})
	}
}

// TestPrePushReadsCommittedDeclarations: an uncommitted edit to a declaration
// cannot widen the exemption, and an untracked connector directory cannot add
// one: the hook reads the declarations HEAD commits.
func TestPrePushReadsCommittedDeclarations(t *testing.T) {
	c := newWIPCase(t)
	c.attributed("one.md")
	c.write(wipConf, "name="+wipHost+"\nwip_prefix=other/abcd-wip/\nis_remote=is-remote.sh\n", 0o644)
	c.write("scripts/cloud/connectors/extra/connector.conf", "name=extra\nwip_prefix=extra/abcd-wip/\nis_remote=is-remote.sh\n", 0o644)
	c.write("scripts/cloud/connectors/extra/is-remote.sh", "#!/usr/bin/env bash\nexit 1\n", 0o755)
	for _, branch := range []string{"other/abcd-wip/" + wipItem, "extra/abcd-wip/" + wipItem} {
		out, err := c.push("origin", "HEAD:refs/heads/"+branch)
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Errorf("%s passed without a receipt on a declaration HEAD does not commit (err %v)\n%s", branch, err, out)
		}
	}
	if out, err := c.push("origin", "HEAD:"+wipRef); err != nil {
		t.Errorf("the committed prefix was refused while the working tree's declaration differs\n%s", out)
	}
}

// TestPrePushWIPStillRefusesForcePush: the entry sits after the
// non-fast-forward guard, so a forced push replacing a work-in-progress tip is
// refused even when the replacing commit is attribution-clean.
func TestPrePushWIPStillRefusesForcePush(t *testing.T) {
	c := newWIPCase(t)
	first := c.attributed("one.md")
	if out, err := c.push("origin", "HEAD:"+wipRef); err != nil {
		t.Fatalf("the create was refused\n%s", out)
	}
	c.git("reset", "-q", "--hard", "origin/main")
	c.attributed("rival.md")
	out, err := c.push("origin", "+HEAD:"+wipRef)
	if err == nil || !strings.Contains(out, "non-fast-forward") {
		t.Fatalf("a forced non-fast-forward to a work-in-progress branch passed the hook (err %v)\n%s", err, out)
	}
	if got := c.remoteRef(wipRef); got != first {
		t.Errorf("the remote holds %q, want %s untouched", got, first)
	}
}

// TestPrePushDefaultBranchStillRefused: the default-branch refusal comes
// before every entry, so an attribution-clean push to it is refused with a
// work-in-progress prefix declared.
func TestPrePushDefaultBranchStillRefused(t *testing.T) {
	c := newWIPCase(t)
	c.attributed("one.md")
	for _, ref := range []string{"HEAD:main", "HEAD:refs/heads/main"} {
		out, err := c.push("origin", ref)
		if err == nil || !strings.Contains(out, "protected branch") {
			t.Fatalf("a push of %s was not refused as a push to the protected branch (err %v)\n%s", ref, err, out)
		}
	}
}

// TestPrePushRefusesDeletingAWIPBranch: a deletion under the prefix is
// refused, naming the act that ends one: the person's collect. The hosted
// proxy refuses deletions anyway, and a spent branch is swept later from an
// ordinary machine, never deleted by a push through this hook.
func TestPrePushRefusesDeletingAWIPBranch(t *testing.T) {
	c := newWIPCase(t)
	first := c.attributed("one.md")
	if out, err := c.push("origin", "HEAD:"+wipRef); err != nil {
		t.Fatalf("the create was refused\n%s", out)
	}
	out, err := c.push("origin", ":"+wipRef)
	if err == nil {
		t.Fatalf("a deletion of a work-in-progress branch passed the hook\n%s", out)
	}
	if !strings.Contains(out, "scripts/cloud/collect.sh "+wipItem) {
		t.Errorf("the refusal does not name the collect that ends a work-in-progress branch\n%s", out)
	}
	if got := c.remoteRef(wipRef); got != first {
		t.Errorf("the remote holds %q, want %s untouched", got, first)
	}
}

// TestPrePushWIPRefDoesNotLaunderAReceipt: a commit is "already on a remote",
// and so needs no receipt, only when a remote-tracking ref OUTSIDE every
// receipt-less prefix holds it. A work-in-progress push needs no receipt and
// leaves such a ref behind, so pushing a commit there first and then to an
// ordinary branch must still be refused for want of a receipt.
func TestPrePushWIPRefDoesNotLaunderAReceipt(t *testing.T) {
	c := newWIPCase(t)
	commit := c.attributed("one.md")
	if out, err := c.push("origin", "HEAD:"+wipRef); err != nil {
		t.Fatalf("the work-in-progress create was refused\n%s", out)
	}
	if got := c.git("for-each-ref", "--format=%(refname)", "--contains", commit, "refs/remotes"); got == "" {
		t.Fatalf("the work-in-progress push left no remote-tracking ref holding %s; the scenario needs one", commit)
	}
	out, err := c.push("origin", "HEAD:refs/heads/feat/x")
	if err == nil || !strings.Contains(out, "no passing preflight") {
		t.Fatalf("a commit first pushed under the work-in-progress prefix then reached feat/x with no receipt (err %v)\n%s", err, out)
	}
	if got := c.remoteRef("refs/heads/feat/x"); got != "" {
		t.Errorf("the remote moved feat/x to %s although the push was refused", got)
	}

	// The same holds for a reservation-shaped commit: the reservation prefix
	// launders nothing either.
	empty := c.emptyTreeCommit("", "reserve")
	if out, err := c.push("origin", empty+":refs/heads/abcd-reserved/"+wipItem); err != nil {
		t.Fatalf("the reservation push was refused\n%s", out)
	}
	if out, err := c.push("origin", empty+":refs/heads/feat/y"); err == nil || !strings.Contains(out, "no passing preflight") {
		t.Errorf("a commit reached feat/y with no receipt because a reservation ref holds it (err %v)\n%s", err, out)
	}
}
