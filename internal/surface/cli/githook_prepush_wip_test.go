package cli

import (
	"os"
	"os/exec"
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
)

// outboundChecker builds abcd from this checkout for the attribution gate's
// outbound half, which the fixture clone cannot `go run`. It is built before
// the fixture isolates HOME, so the build reads the machine's warm cache.
func outboundChecker(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go unavailable")
	}
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	bin := filepath.Join(t.TempDir(), "abcd")
	build := exec.Command("go", "build", "-o", bin, "./cmd/abcd")
	build.Dir = strings.TrimSpace(string(top))
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build the outbound checker: %v\n%s", err, out)
	}
	return bin
}

// newWIPCase is a pre-push case carrying the cloud scripts the hook reads and
// one declared connector, all on origin's default branch, so a push from the
// feature branch moves only the commits a test makes.
func newWIPCase(t *testing.T) *prePushCase {
	t.Helper()
	bin := outboundChecker(t)
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
			if !strings.Contains(out, "check-attribution: FAILED") {
				t.Errorf("the refusal does not show the attribution gate's finding\n%s", out)
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

// TestPrePushRefusesOnBadDeclaration: the prefixes come from the connector
// declarations through connector.sh's own validation, so a declaration that
// fails it refuses the push outright, naming the file, rather than leaving the
// hook to act on a partial list.
func TestPrePushRefusesOnBadDeclaration(t *testing.T) {
	for name, conf := range map[string]string{
		"a bare host namespace": "name=" + wipHost + "\nwip_prefix=" + wipHost + "/\nis_remote=is-remote.sh\n",
		"the default branch":    "name=" + wipHost + "\nwip_prefix=main\nis_remote=is-remote.sh\n",
		"two segments before":   "name=" + wipHost + "\nwip_prefix=a/b/abcd-wip/\nis_remote=is-remote.sh\n",
		"an unknown key":        "name=" + wipHost + "\nwip_prefix=" + wipPrefix + "\nis_remote=is-remote.sh\nexempt=all\n",
	} {
		t.Run(name, func(t *testing.T) {
			c := newWIPCase(t)
			c.attributed("one.md")
			c.write(wipConf, conf, 0o644)
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
		c.write(wipConf, "name="+wipHost+"\nwip_prefix=main\nis_remote=is-remote.sh\n", 0o644)
		c.git("add", wipConf)
		c.git("commit", "-q", "-m", "a bad declaration\n\n"+assisted)
		c.preflight(c.dir)
		out, err := c.push("origin", "feature")
		if err == nil || !strings.Contains(out, wipConf) {
			t.Fatalf("a push went ahead under a declaration that fails validation (err %v)\n%s", err, out)
		}
	})
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
