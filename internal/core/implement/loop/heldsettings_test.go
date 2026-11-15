package loop

import (
	"fmt"
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
	"http.sslversion": true, "http.sslcipherlist": true, "http.sslcert": true, "http.sslkey": true,
	"http.sslcertpasswordprotected": true, "http.proxysslcert": true, "http.proxysslkey": true,
	"http.https://example.invalid/.proxysslkey": true, "http.cookiefile": true, "http.savecookies": true,
	"http.extraheader": true, "http.https://example.invalid/.extraheader": true, "http.delegation": true,
	"http.proxyauthmethod": true, "http.emptyauth": true, "http.https://example.invalid/.emptyauth": true,
	"remote.origin.proxyauthmethod": true, "remote.upstream.serveroption": true,
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
	"receive.fsckobjects": false, "http.postbuffer": false, "http.https://example.invalid/.useragent": false,
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

// heldListsGit is the git release the held and unread lists, and the rows of
// httpSettingRows, were built on: git 2.52.
const heldListsGit = "2.52"

// documentedSettings is the installed git's own list of its settings (`git
// help --config`), each by its canonical spelling with a placeholder as "x",
// and whether that git is newer than heldListsGit, which the test logs.
// Where git cannot list them the test skips, saying so.
func documentedSettings(t *testing.T) (names map[string]string, newer bool) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	out, err := gitutil.Run(t.TempDir(), "help", "--config")
	if err != nil || !strings.Contains(out, "core.editor") {
		t.Skipf("this git does not list its settings (git help --config): %v", err)
	}
	newer, version := gitutil.GitAtLeast(2, 53)
	placeholder := regexp.MustCompile(`<[^>]*>`)
	names = map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if name == "" || strings.ContainsAny(name, " *") {
			continue
		}
		names[strings.ToLower(placeholder.ReplaceAllString(name, "x"))] = name
	}
	if newer {
		t.Logf("the installed git %s is newer than git %s, which the lists were built on: a setting it documents that the lists do not know is logged, and the test passes", version, heldListsGit)
	}
	return names, newer
}

// unknownSetting fails the test for a documented setting the lists do not
// know (why), or, on a git newer than heldListsGit, logs it.
func unknownSetting(t *testing.T, newer bool, why string) {
	t.Helper()
	if newer {
		t.Logf("not failed, since this git is newer than git %s: %s", heldListsGit, why)
		return
	}
	t.Error(why)
}

// TestEachProgramSettingGitDocumentsIsHeldOrUnread reads the installed git's
// own list of settings (documentedSettings) and, for each name whose last
// word is a program's (programShaped), fails where the drive neither holds it
// nor classes it unread with a reason (driveUnreadSettings). On git
// heldListsGit or older such a name fails; on a newer git it is logged and
// the test passes, saying so. A setting whose name is not program-shaped
// (core.fsmonitor, core.hooksPath, commit.gpgSign, the http.* keys) is not
// judged here: heldSettingsTable and TestEveryHTTPSettingGitDocumentsIsInOneRow
// judge those by name.
func TestEachProgramSettingGitDocumentsIsHeldOrUnread(t *testing.T) {
	names, newer := documentedSettings(t)
	checked := 0
	for key, name := range names {
		if !programShaped.MatchString(key) {
			continue
		}
		checked++
		if !remoteIdentityKey(key) && !unreadSetting(key) {
			unknownSetting(t, newer, fmt.Sprintf("git documents %s (%s), whose name is a program's, and the drive neither holds it nor classes it unread with a reason", name, key))
		}
	}
	if checked == 0 {
		t.Fatal("git listed no setting whose name is a program's, so this test proves nothing")
	}
}

// httpSettingRows puts every http.* setting git heldListsGit documents in
// one of three rows: held (whom an https fetch or push of the drive trusts,
// where it connects, or what it sends there); unread (no git of the drive
// reads it: none, since the transport every https fetch and push runs reads
// them all); and neither a program nor trust, each with why.
var httpSettingRows = struct {
	held, unread []string
	neither      map[string]string
}{
	held: []string{
		// Trust.
		"sslverify", "sslversion", "sslcipherlist", "sslcainfo", "sslcapath", "pinnedpubkey",
		"sslbackend", "schannelcheckrevoke", "schannelusesslcainfo", "ssltry", "proxysslcainfo",
		// Destination.
		"proxy", "curloptresolve", "followredirects",
		// What is sent.
		"sslcert", "sslcerttype", "sslkey", "sslkeytype", "sslcertpasswordprotected",
		"proxysslcert", "proxysslkey", "proxysslcertpasswordprotected", "proxyauthmethod",
		"cookiefile", "savecookies", "extraheader", "delegation", "emptyauth", "proactiveauth",
	},
	neither: map[string]string{
		"postbuffer":        "a buffer's size",
		"maxrequests":       "how many requests run at once",
		"minsessions":       "how many connections are kept",
		"keepalivecount":    "how a connection is kept alive",
		"keepaliveidle":     "how a connection is kept alive",
		"keepaliveinterval": "how a connection is kept alive",
		"lowspeedlimit":     "when a slow transfer is given up",
		"lowspeedtime":      "when a slow transfer is given up",
		"noepsv":            "how an ftp connection is made, not to whom",
		"version":           "which HTTP version is spoken, not to whom",
		"useragent":         "the name git gives itself",
	},
}

// TestEveryHTTPSettingGitDocumentsIsInOneRow: each http.* setting of
// httpSettingRows is held, plain and per URL, or not, as its row says, and
// every http.* setting the installed git documents is in exactly one row (on
// a git newer than heldListsGit, one the rows do not know is logged).
func TestEveryHTTPSettingGitDocumentsIsInOneRow(t *testing.T) {
	rows := map[string]string{}
	for _, k := range httpSettingRows.held {
		rows[k] = "held"
		for _, key := range []string{"http." + k, "http.https://example.invalid/." + k} {
			if !remoteIdentityKey(key) {
				t.Errorf("%s is in the held row and the drive does not hold it", key)
			}
		}
	}
	for _, k := range httpSettingRows.unread {
		if _, dup := rows[k]; dup {
			t.Errorf("http.%s is in two rows", k)
		}
		rows[k] = "unread"
		if !unreadSetting("http." + k) {
			t.Errorf("http.%s is in the unread row and driveUnreadSettings gives no reason", k)
		}
	}
	for k, why := range httpSettingRows.neither {
		if why == "" {
			t.Errorf("http.%s is in the third row with no reason", k)
		}
		if _, dup := rows[k]; dup {
			t.Errorf("http.%s is in two rows", k)
		}
		rows[k] = "neither"
		for _, key := range []string{"http." + k, "http.https://example.invalid/." + k} {
			if remoteIdentityKey(key) {
				t.Errorf("%s is in the row of settings that are neither a program nor trust, and the drive holds it", key)
			}
		}
	}
	names, newer := documentedSettings(t)
	seen := 0
	for key, name := range names {
		k, ok := strings.CutPrefix(key, "http.")
		if !ok || strings.HasPrefix(k, "x.") {
			continue
		}
		seen++
		if _, ok := rows[k]; !ok {
			unknownSetting(t, newer, fmt.Sprintf("git documents %s, which is in no row of httpSettingRows", name))
		}
	}
	if seen == 0 {
		t.Fatal("git listed no http.* setting, so this test proves nothing")
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
