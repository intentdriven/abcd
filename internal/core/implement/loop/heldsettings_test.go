package loop

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
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
	"http.useragent": true, "http.https://example.invalid/.useragent": true, "push.pushoption": true,
	"transfer.advertisesid": true, "remote.origin.promisor": true, "remote.origin.partialclonefilter": true,
	"remote.origin.mirror": true, "remote.origin.token": true, "extensions.partialclone": true, "promisor.acceptfromserver": true,
	"promisor.checkfields": true, "promisor.quiet": true, "promisor.advertise": true, "promisor.sendfields": true,
	"push.followtags": false, "remote.origin.push": false,
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
	"receive.fsckobjects": false, "http.postbuffer": false,
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
		"useragent",
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

// remoteSettingRows puts every push.*, remote.<name>.*, fetch.*, transfer.*,
// url.* and promisor.* setting git heldListsGit documents, and
// extensions.partialClone, (by its canonical spelling, a placeholder as "x")
// in one row, each with why: held (a change stops the
// drive); pinned (the drive's own command line decides what it would); unread
// (driveUnreadSettings gives the reason); and neither a program nor trust
// nor a value sent.
var remoteSettingRows = map[string]struct{ row, why string }{
	// Where a fetch or push goes, how it reaches the remote, whom it trusts
	// and what it sends: held.
	"remote.x.url":                {"held", "where the remote is"},
	"remote.x.pushurl":            {"held", "where a push goes"},
	"remote.x.uploadpack":         {"held", "the program a fetch runs on the remote side"},
	"remote.x.receivepack":        {"held", "the program a push runs on the remote side"},
	"remote.x.proxy":              {"held", "the proxy a fetch or push goes through"},
	"remote.x.proxyauthmethod":    {"held", "what is sent to the proxy"},
	"remote.x.vcs":                {"held", "the remote helper git runs"},
	"remote.x.serveroption":       {"held", "what a fetch sends the server"},
	"remote.x.promisor":           {"held", "makes a git that reads a missing object fetch it from the remote"},
	"remote.x.partialclonefilter": {"held", "what such a fetch asks the remote for"},
	"remote.x.mirror":             {"held", "a push to the remote behaves as if --mirror were given (git-config(1))"},
	"remote.x.token":              {"held", "what a fetch compares with the token a server sends before it accepts a promisor remote the server names; git help --config does not list it, and git-config(1) documents it under promisor.checkFields"},
	"extensions.partialclone":     {"held", "names the promisor remote a git that reads a missing object fetches it from (git-config(1))"},
	"promisor.acceptfromserver":   {"held", "which promisor remotes a server names that a fetch accepts, and so where missing objects are later fetched from"},
	"promisor.checkfields":        {"held", "which fields a server sends must match before a fetch accepts a promisor remote it names"},
	"promisor.quiet":              {"held", "whether a fetch of missing objects prints; held with its section"},
	"promisor.advertise":          {"held", "whether this repository, serving a fetch, names its promisor remotes; held with its section"},
	"promisor.sendfields":         {"held", "what this repository, serving a fetch, sends with the promisor remotes it names; held with its section"},
	"url.x.insteadof":             {"held", "rewrites where a fetch or push goes"},
	"url.x.pushinsteadof":         {"held", "rewrites where a push goes"},
	"fetch.bundleuri":             {"held", "where a fetch first downloads from"},
	"push.pushoption":             {"held", "what a push sends the server (git-config(1): as if each value were given as --push-option)"},
	"push.gpgsign":                {"held", "starts the signing program"},
	"push.recursesubmodules":      {"held", "makes the push go into each submodule"},
	"transfer.advertisesid":       {"held", "makes a fetch or push send the server its session id"},
	// Decided by the drive's own command line.
	"push.followtags":           {"pinned", "the drive's pushes pass --no-follow-tags, which overrides it"},
	"push.default":              {"pinned", "decides what a push that names no refspec sends; the drive's pushes name theirs"},
	"push.autosetupremote":      {"pinned", "read by a push that names no refspec; the drive's pushes name theirs"},
	"remote.x.push":             {"pinned", "the refspecs of a push that names none; the drive's pushes name theirs"},
	"remote.pushdefault":        {"pinned", "the remote of a push that names none; the drive's pushes name origin"},
	"fetch.all":                 {"pinned", "fetches every remote when none is named; the drive's fetches name origin"},
	"remote.x.tagopt":           {"pinned", "tag following on fetch; the drive's fetches pass --no-tags, which overrides it"},
	"remote.x.fetch":            {"pinned", "the refspecs of a fetch that names none; the drive's fetches name theirs, and git uses these only to choose the remote-tracking refs here that the named ones also update"},
	"remote.x.followremotehead": {"pinned", "read by a fetch that uses the configured refspecs; the drive's fetches name theirs"},
	// Read by no git command the drive runs (driveUnreadSettings).
	"fetch.recursesubmodules": {"unread", ""},
	"transfer.bundleuri":      {"unread", ""},
	// Neither a program, nor trust, nor a value sent.
	"fetch.bundlecreationtoken":    {"neither", "a number git writes itself after a bundle download"},
	"fetch.fsck.x":                 {"neither", "a check of the objects received"},
	"fetch.fsck.skiplist":          {"neither", "a check of the objects received"},
	"fetch.fsckobjects":            {"neither", "a check of the objects received"},
	"transfer.fsckobjects":         {"neither", "a check of the objects received"},
	"fetch.negotiationalgorithm":   {"neither", "which of the repository's own commits a fetch offers as common"},
	"push.negotiate":               {"neither", "whether a push first learns which of the repository's own commits the remote has"},
	"fetch.output":                 {"neither", "how a fetch prints the refs it updated"},
	"fetch.showforcedupdates":      {"neither", "whether a fetch says which updates were forced"},
	"fetch.parallel":               {"neither", "how many fetches run at once; the drive's fetches name one remote, with submodule recursion off"},
	"fetch.prune":                  {"neither", "removes remote-tracking refs here that the remote no longer has, among those a fetch's refspecs name"},
	"fetch.prunetags":              {"neither", "removes tags here that the remote no longer has, only for a fetch that names no refspec"},
	"remote.x.prune":               {"neither", "removes remote-tracking refs here that the remote no longer has, among those a fetch's refspecs name"},
	"remote.x.prunetags":           {"neither", "removes tags here that the remote no longer has, only for a fetch that names no refspec"},
	"fetch.unpacklimit":            {"neither", "whether what is received is kept as a pack"},
	"transfer.unpacklimit":         {"neither", "whether what is received is kept as a pack"},
	"fetch.writecommitgraph":       {"neither", "whether a fetch writes a commit graph here"},
	"push.usebitmaps":              {"neither", "how the pack a push sends is built"},
	"push.useforceifincludes":      {"neither", "makes a push given a lease refuse more"},
	"remote.x.skipdefaultupdate":   {"neither", "which remotes `git remote update` and a fetch of every remote leave out"},
	"remote.x.skipfetchall":        {"neither", "which remotes a fetch of every remote leaves out"},
	"transfer.advertiseobjectinfo": {"neither", "what a serving upload-pack advertises, read on the remote's side"},
	"transfer.hiderefs":            {"neither", "which refs a serving upload-pack or receive-pack leaves out, read on the remote's side"},
	"transfer.credentialsinurl":    {"neither", "whether git warns about, or refuses, a URL in the settings that holds a password; it sends nothing"},
}

// remoteSettingSections are the sections remoteSettingRows covers.
var remoteSettingSections = []string{"push.", "remote.", "fetch.", "transfer.", "url.", "promisor."}

// remoteSettingKey is a documented setting's canonical name with its
// placeholder given a value: a remote's name, a URL base, a check's id.
func remoteSettingKey(key string) string {
	switch {
	case strings.HasPrefix(key, "remote.x."):
		return "remote.origin." + strings.TrimPrefix(key, "remote.x.")
	case strings.HasPrefix(key, "url.x."):
		return "url.https://example.invalid/." + strings.TrimPrefix(key, "url.x.")
	case key == "fetch.fsck.x":
		return "fetch.fsck.badtimezone"
	}
	return key
}

// TestEveryRemoteSettingGitDocumentsIsInOneRow: each setting of
// remoteSettingRows is held, or not, as its row says, with a reason; every
// push.*, remote.<name>.*, fetch.*, transfer.*, url.* and promisor.* setting
// the installed git documents is in a row (on a git newer than heldListsGit,
// one the rows do not know is logged). Watched failing with push.pushOption,
// transfer.advertiseSID and the promisor settings not held, and again with
// remote.<name>.mirror, extensions.partialClone and promisor.* not held.
func TestEveryRemoteSettingGitDocumentsIsInOneRow(t *testing.T) {
	for key, r := range remoteSettingRows {
		k := remoteSettingKey(key)
		switch r.row {
		case "held":
			if !remoteIdentityKey(k) {
				t.Errorf("%s is in the held row and the drive does not hold it", k)
			}
		case "unread":
			if !unreadSetting(k) {
				t.Errorf("%s is in the unread row and driveUnreadSettings gives no reason", k)
			}
		case "pinned", "neither":
			if r.why == "" {
				t.Errorf("%s is in the %s row with no reason", k, r.row)
			}
			if remoteIdentityKey(k) {
				t.Errorf("%s is in the %s row, and the drive holds it", k, r.row)
			}
		default:
			t.Errorf("%s is in no row of the four (%q)", k, r.row)
		}
	}
	names, newer := documentedSettings(t)
	seen := 0
	for key, name := range names {
		if !slices.ContainsFunc(remoteSettingSections, func(p string) bool { return strings.HasPrefix(key, p) }) {
			continue
		}
		seen++
		if _, ok := remoteSettingRows[key]; !ok {
			unknownSetting(t, newer, fmt.Sprintf("git documents %s, which is in no row of remoteSettingRows", name))
		}
	}
	if seen == 0 {
		t.Fatal("git listed no setting of those sections, so this test proves nothing")
	}
}

// TestTheLandingPushSendsNoTagInADriveOnly: with push.followTags on, a push
// sends each annotated tag missing from origin that points into the history
// it pushes. The landing push (landPush) of a drive passes --no-follow-tags,
// so a tag a lane made on its branch is not sent with it, and the branch is;
// outside a drive (a person's `abcd build` or `abcd implement step`) the tag
// is sent, as before. Watched failing with landPushArgs passing no such
// option: the drive's push sent the tag.
func TestTheLandingPushSendsNoTagInADriveOnly(t *testing.T) {
	for _, drive := range []bool{true, false} {
		f := newLandFixture(t, queueRuleset("MERGE"))
		f.validatedOnce(t)
		for range 2 { // the landing's records are committed, which moves the head
			f.step(t)
		}
		l := currentLane(t, f.repo, f.runID)
		preflighted(t, l, l.HeadSHA)
		f.repo.Git("config", "push.followTags", "true")
		f.repo.Git("tag", "-a", "-m", "a lane's tag", "v9.9.9", l.HeadSHA)
		clear := func() {}
		if drive {
			clear = claim.DrivePushes()
		}
		_, err := advance(f.repo.Root(), f.runID, f.stages, Options{ReceiptOwn: func(string) bool { return true }})
		clear()
		if err != nil {
			t.Fatalf("drive %v: the landing: %v", drive, err)
		}
		if got := f.remoteBranch(t, l.Branch); got != l.HeadSHA {
			t.Fatalf("drive %v: origin's branch is %q, want the lane's head %s", drive, got, l.HeadSHA)
		}
		tags := f.repo.Git("ls-remote", "--tags", Remote)
		if drive && tags != "" {
			t.Errorf("a drive's landing push sent the lane's tag: %q", tags)
		}
		if !drive && !strings.Contains(tags, "refs/tags/v9.9.9") {
			t.Errorf("a landing push outside a drive follows push.followTags as before: origin holds %q", tags)
		}
	}
}

// TestADriveMarksItsPushes: RunTerminal marks its process as running a drive
// (claim.DrivePushes) for as long as it runs, so its landing push
// (landPushArgs) and the reservation code's pushes pass --no-follow-tags, and
// clears the mark as it returns. Watched failing with RunTerminal's `defer
// claim.DrivePushes()()` deleted: the landing ran unmarked.
func TestADriveMarksItsPushes(t *testing.T) {
	if claim.DriveMarked() {
		t.Fatal("the process is marked before any drive ran")
	}
	w := newTerminalWorld(t)
	o := oneLane(t, w)
	landed := 0
	for i := range w.stages {
		if w.stages[i].Name != StageLand {
			continue
		}
		run := w.stages[i].Run
		w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
			landed++
			if !claim.DriveMarked() {
				t.Error("the drive's landing ran with the process unmarked")
			}
			if !slices.Contains(landPushArgs(driveBase, l.Branch), "--no-follow-tags") {
				t.Error("the drive's landing push passes no --no-follow-tags")
			}
			return run(c, l)
		}
	}
	o.Stages = w.stages
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
	}
	if landed == 0 {
		t.Fatalf("the drive reached no landing, so this test proves nothing:\n%s", strings.Join(w.lines, "\n"))
	}
	if claim.DriveMarked() {
		t.Error("the mark is left after the drive returned")
	}
	if slices.Contains(landPushArgs(driveBase, "build/lane-1"), "--no-follow-tags") {
		t.Error("a landing push outside a drive passes --no-follow-tags")
	}
}
