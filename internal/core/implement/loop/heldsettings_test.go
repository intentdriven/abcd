package loop

import (
	"os/exec"
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// heldSettingsTable is, for each setting git-config(1) (git 2.52) documents
// as naming a program git starts or as a mode that makes git start one, in a
// canonical spelling, whether a drive holds it across its lanes; and the
// settings left out, with why beside them (driveHeldConfig's comment).
var heldSettingsTable = map[string]bool{
	// Watched by the run's fingerprint too.
	"core.hookspath": true, "core.fsmonitor": true, "core.sshcommand": true, "core.askpass": true,
	"core.gitproxy": true, "core.pager": true, "core.editor": true,
	"credential.helper": true, "credential.https://example.invalid.helper": true,
	"remote.origin.vcs": true, "remote.origin.uploadpack": true, "remote.origin.receivepack": true,
	"url.https://example.invalid/.insteadof": true, "protocol.allow": true, "protocol.file.allow": true,
	"filter.lfs.process": true, "diff.x.command": true, "diff.x.textconv": true, "merge.x.driver": true,
	"gpg.program": true, "gpg.ssh.program": true, "gpg.ssh.defaultkeycommand": true,
	"include.path": true, "includeif.gitdir:/x/.path": true,
	// Held by a drive alone.
	"http.proxy": true, "http.https://example.invalid/.proxy": true, "http.sslverify": true,
	"core.alternaterefscommand": true, "core.attributesfile": true, "sequence.editor": true,
	"pager.log": true, "diff.external": true, "merge.renormalize": true, "merge.verifysignatures": true,
	"trailer.sign.command": true, "trailer.sign.cmd": true, "format.pretty": true, "pretty.changelog": true,
	"gpg.format": true, "gpg.x509.program": true, "commit.gpgsign": true, "tag.gpgsign": true,
	"push.gpgsign": true, "tag.forcesignannotated": true, "log.showsignature": true, "ssh.variant": true,
	"uploadpack.packobjectshook": true, "submodule.lib.update": true, "submodule.recurse": true,
	"fetch.recursesubmodules": true, "push.recursesubmodules": true, "status.submodulesummary": true,
	"lfs.customtransfer.x.path": true, "lfs.extension.x.clean": true, "lfs.standalonetransferagent": true,
	"hook.lint.command": true, "gc.auto": true, "gc.recentobjectshook": true, "maintenance.auto": true,
	"maintenance.gc.enabled": true, "init.templatedir": true, "difftool.x.cmd": true, "mergetool.x.path": true,
	"browser.x.cmd": true, "man.x.path": true, "guitool.x.cmd": true, "web.browser": true,
	"interactive.difffilter": true, "sendemail.smtpserver": true,
	// Left out: no command abcd runs reads it (alias.*), or it names no
	// program.
	"alias.st": false, "core.excludesfile": false, "fetch.fsckobjects": false, "transfer.fsckobjects": false,
	"receive.fsckobjects": false, "http.postbuffer": false, "http.https://example.invalid/.extraheader": false,
	"protocol.version": false, "core.quotepath": false, "user.signingkey": false,
}

// TestTheDriveHoldsEachSettingThatNamesAProgram: every setting of the table
// is held, or left out, as the table says.
func TestTheDriveHoldsEachSettingThatNamesAProgram(t *testing.T) {
	for key, want := range heldSettingsTable {
		if got := remoteIdentityKey(key); got != want {
			t.Errorf("remoteIdentityKey(%q) = %v, want %v", key, got, want)
		}
	}
}

// TestTheHeldPatternGitIsGivenSelectsWhatTheDriveHolds: the pattern a
// refusal's remedy hands to `git config --get-regexp` is the same set as the
// predicate the hold applies, as git itself reads the pattern: every key of
// the table set in a repository, git's selection among them is exactly the
// held ones.
func TestTheHeldPatternGitIsGivenSelectsWhatTheDriveHolds(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	r := gittest.NewRepo(t)
	var want []string
	for key, held := range heldSettingsTable {
		r.Git("config", "--add", key, "true")
		if held {
			want = append(want, key)
		}
	}
	entries, err := gitutil.RepoConfig(r.Root(), heldPattern)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		// The repository's own settings beside the table's (the test
		// repository's maintenance switches) are not the table's to judge.
		if _, ok := heldSettingsTable[e.Key]; ok && !slices.Contains(got, e.Key) {
			got = append(got, e.Key)
		}
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("git selected %q\nthe hold holds %q", got, want)
	}
}
