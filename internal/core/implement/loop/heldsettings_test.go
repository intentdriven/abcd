package loop

import (
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// heldSettingsTable is, for settings that name or start a program, or change
// where a networked git goes or whom it trusts, in a canonical spelling,
// whether a drive holds it across its lanes; driveUnreadSettings says why
// each one held false is not, and the rest name no program.
var heldSettingsTable = map[string]bool{
	// Watched by the run's fingerprint too.
	"core.hookspath": true, "core.fsmonitor": true, "core.sshcommand": true, "core.askpass": true,
	"core.gitproxy": true, "core.editor": true,
	"credential.helper": true, "credential.https://example.invalid.helper": true,
	"remote.origin.vcs": true, "remote.origin.uploadpack": true, "remote.origin.receivepack": true,
	"url.https://example.invalid/.insteadof": true, "protocol.allow": true, "protocol.file.allow": true,
	"filter.lfs.process": true, "diff.x.command": true, "diff.x.textconv": true, "merge.x.driver": true,
	"gpg.program": true, "gpg.ssh.program": true, "gpg.ssh.defaultkeycommand": true,
	"include.path": true, "includeif.gitdir:/x/.path": true, "extensions.worktreeconfig": true,
	// Held by a drive alone.
	"http.proxy": true, "http.https://example.invalid/.proxy": true, "http.sslverify": true,
	"http.sslcainfo": true, "http.https://example.invalid/.sslcainfo": true, "http.sslcapath": true,
	"http.pinnedpubkey": true, "http.sslbackend": true, "http.schannelcheckrevoke": true,
	"http.schannelusesslcainfo": true, "http.proxysslcainfo": true, "http.curloptresolve": true,
	"http.https://example.invalid/.curloptresolve": true, "http.followredirects": true, "fetch.bundleuri": true,
	"core.alternaterefscommand": true, "core.attributesfile": true,
	"diff.external": true, "merge.renormalize": true, "merge.verifysignatures": true,
	"format.pretty": true, "pretty.changelog": true,
	"gpg.format": true, "gpg.x509.program": true, "commit.gpgsign": true, "tag.gpgsign": true,
	"push.gpgsign": true, "tag.forcesignannotated": true, "log.showsignature": true, "ssh.variant": true,
	"uploadpack.packobjectshook": true, "submodule.recurse": true, "push.recursesubmodules": true,
	"status.submodulesummary": true, "lfs.customtransfer.x.path": true, "lfs.extension.x.clean": true,
	"lfs.standalonetransferagent": true, "lfs.https://example.invalid/.standalonetransferagent": true,
	"hook.lint.command": true, "gc.recentobjectshook": true, "init.templatedir": true,
	// Not held: no git command a drive runs reads it (driveUnreadSettings).
	"alias.st": false, "core.pager": false, "pager.log": false, "color.pager": false,
	"sequence.editor": false, "rebase.instructionformat": false, "interactive.difffilter": false,
	"diff.tool": false, "merge.guitool": false, "difftool.x.cmd": false, "mergetool.x.path": false,
	"guitool.x.cmd": false, "web.browser": false, "browser.x.cmd": false, "help.browser": false,
	"man.viewer": false, "man.x.path": false, "instaweb.httpd": false, "instaweb.browser": false,
	"sendemail.smtpserver": false, "sendemail.headercmd": false, "sendemail.work.tocmd": false,
	"imap.tunnel": false, "trailer.sign.command": false, "trailer.sign.cmd": false,
	"maintenance.auto": false, "maintenance.repo": false, "maintenance.gc.enabled": false,
	"gc.auto": false, "gc.autodetach": false, "fetch.recursesubmodules": false,
	"submodule.lib.update": false, "submodule.lib.url": false, "submodule.lib.active": false,
	"submodule.active": false, "submodule.fetchjobs": false, "transfer.bundleuri": false,
	"advice.ignoredhook": false,
	// Not held: names no program, and changes neither where a networked git
	// goes nor whom it trusts.
	"core.excludesfile": false, "fetch.fsckobjects": false, "transfer.fsckobjects": false,
	"receive.fsckobjects": false, "http.postbuffer": false, "http.https://example.invalid/.extraheader": false,
	"protocol.version": false, "core.quotepath": false, "user.signingkey": false,
}

// unreadSetting reports whether key is one of driveUnreadSettings.
func unreadSetting(key string) bool {
	for _, u := range driveUnreadSettings {
		if regexp.MustCompile(`^(` + u.pattern + `)$`).MatchString(key) {
			return true
		}
	}
	return false
}

// TestTheDriveHoldsWhatItsGitReadsAndNoMore: every setting of the table is
// held, or not, as the table says; one not held that names a program is
// classed unread with its reason, and no held one is.
func TestTheDriveHoldsWhatItsGitReadsAndNoMore(t *testing.T) {
	for key, want := range heldSettingsTable {
		if got := remoteIdentityKey(key); got != want {
			t.Errorf("remoteIdentityKey(%q) = %v, want %v", key, got, want)
		}
		if want && unreadSetting(key) {
			t.Errorf("%q is held and classed unread at once", key)
		}
	}
	for _, u := range driveUnreadSettings {
		if u.why == "" {
			t.Errorf("the unread setting %q gives no reason", u.pattern)
		}
	}
}

// programShaped matches a canonical setting name git documents whose last
// word is a program's (a command, a helper, a hook, an editor, a pager, a
// tool, a browser, a viewer, a tunnel), or a Git LFS path.
var programShaped = regexp.MustCompile(`(command|cmd|program|helper|hook|editor|pager|tool|browser|viewer|tunnel)$|^lfs\..*path$`)

// TestEachProgramSettingGitDocumentsIsHeldOrUnread reads the installed git's
// own list of settings (`git help --config`, git 2.52 and later list them
// all) and fails for a name whose last word is a program's that the drive
// neither holds nor classes unread with a reason (driveUnreadSettings), so a
// setting a later git documents is judged when it appears. Where git lacks
// the option it skips, saying so.
func TestEachProgramSettingGitDocumentsIsHeldOrUnread(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	out, err := gitutil.Run(t.TempDir(), "help", "--config")
	if err != nil || !strings.Contains(out, "core.editor") {
		t.Skipf("this git does not list its settings (git help --config): %v", err)
	}
	placeholder := regexp.MustCompile(`<[^>]*>`)
	checked := 0
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if name == "" || strings.ContainsAny(name, " *") {
			continue
		}
		key := strings.ToLower(placeholder.ReplaceAllString(name, "x"))
		if !programShaped.MatchString(key) {
			continue
		}
		checked++
		if !remoteIdentityKey(key) && !unreadSetting(key) {
			t.Errorf("git documents %s (%s), whose name is a program's, and the drive neither holds it nor classes it unread with a reason", name, key)
		}
	}
	if checked == 0 {
		t.Fatal("git listed no setting whose name is a program's, so this test proves nothing")
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
