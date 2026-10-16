package cli

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// The list (scripts/cloud/list.sh, C8), spc-2610120941441282 step 10. The
// fixture is the hand-back's (newForgeCase): a bare origin behind a github.com
// URL, abcd's verbs through a binary built from this checkout, and a fake `gh`
// serving the tracking issue's comments. Every session's state is put on the
// remote directly: its work-in-progress branch with or without a report, and
// its reservation as `abcd reserve` writes one, so the list reads exactly what
// a hosted session leaves behind.

const listScript = "scripts/cloud/list.sh"

// reportText is a report as the hand-back writes it.
func reportText(item, state, channel, did string) string {
	return "---\nitem: " + item + "\nstate: " + state + "\nchannel: " + channel +
		"\nfinished_at: 2026-10-15T10:00:00Z\n---\n\n# Cloud session report: " + item +
		"\n\n## What it did\n\n" + did + "\n\n## What failed\n\nNothing.\n\n## What it needs from the person\n\nA review.\n"
}

// pushWIP pushes a work-in-progress branch for item to origin, carrying the
// report when report is not "", with no hook in the way.
func (c *cloudCase) pushWIP(item, report string) {
	c.t.Helper()
	c.git("checkout", "-q", "-B", "wip-"+item, "origin/main")
	c.write("work-"+item+".md", item+"\n", 0o644)
	c.git("add", "-A")
	c.git("-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", "fix: "+item+"\n\n"+sessionTrailer)
	if report != "" {
		c.write(reportPath(item), report, 0o644)
		c.git("add", "-A")
		c.git("-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", "chore(cloud): report on "+item+"\n\n"+sessionTrailer)
	}
	c.git("-c", "core.hooksPath=/dev/null", "push", "-q", "origin", "HEAD:refs/heads/"+hostWIPPrefix+item)
	c.git("checkout", "-q", "session")
}

// reserveOnOrigin pushes a reservation for item, as `abcd reserve` writes
// one, that expires at expires.
func (c *cloudCase) reserveOnOrigin(item string, expires time.Time) {
	c.t.Helper()
	claimed := expires.Add(-2 * time.Hour)
	if now := time.Now(); claimed.After(now) {
		claimed = now.Add(-time.Minute)
	}
	pushTip(c.t, c.dir, item, "abcd reservation: "+item+"\n\n"+
		"Abcd-Reservation: "+item+"\nAbcd-Holder: octo\nAbcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\n"+
		"Abcd-Footprint: one.md\nAbcd-Claimed-At: "+claimed.UTC().Format(time.RFC3339)+
		"\nAbcd-Expires-At: "+expires.UTC().Format(time.RFC3339)+"\nAssisted-by: abcd:0.0.0\n")
}

// listRows runs the list and returns its rows by item: branch, state, report.
func (c *cloudCase) listRows(extra []string) (rows map[string][3]string, stdout, stderr string) {
	c.t.Helper()
	out, errOut, code := c.scriptSplit(extra, listScript)
	if code != 0 {
		c.t.Fatalf("list exited %d\n%s%s", code, out, errOut)
	}
	rows = map[string][3]string{}
	for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 4 {
			c.t.Fatalf("list line %d has %d tab-separated fields, want 4: %q\n%s", i+1, len(f), line, out)
		}
		if i == 0 {
			if line != "ITEM\tBRANCH\tSTATE\tREPORT" {
				c.t.Fatalf("list's header is %q", line)
			}
			continue
		}
		if _, dup := rows[f[0]]; dup {
			c.t.Fatalf("list shows %s twice\n%s", f[0], out)
		}
		rows[f[0]] = [3]string{f[1], f[2], f[3]}
	}
	return rows, out, errOut
}

const (
	itemHanded  = "iss-2610150000000021"
	itemStopped = "iss-2610150000000022"
	itemWorking = "itd-2610150000000023"
	itemBare    = "iss-2610150000000024"
	itemLapsed  = "iss-2610150000000025"
	itemComment = "iss-2610150000000026"
)

func TestListShowsHandedBackStoppedAndWorking(t *testing.T) {
	c := newForgeCase(t, "")
	c.pushWIP(itemHanded, reportText(itemHanded, "handed-back", "branch", "Fixed the walk over the readings folder."))
	c.pushWIP(itemStopped, reportText(itemStopped, "stopped", "branch", "Stopped before any work: gitleaks is missing."))
	c.reserveOnOrigin(itemWorking, time.Now().Add(time.Hour))
	c.pushWIP(itemBare, "")
	c.pushWIP(itemLapsed, "")
	c.reserveOnOrigin(itemLapsed, time.Now().Add(-time.Hour))
	c.resetCalls()

	rows, out, errOut := c.listRows(nil)
	for item, want := range map[string][3]string{
		itemHanded:  {hostWIPPrefix + itemHanded, "handed-back", "Fixed the walk over the readings folder."},
		itemStopped: {hostWIPPrefix + itemStopped, "stopped", "Stopped before any work: gitleaks is missing."},
		itemWorking: {"-", "working", "-"},
		itemBare:    {hostWIPPrefix + itemBare, "unknown", "-"},
		itemLapsed:  {hostWIPPrefix + itemLapsed, "unknown", "-"},
	} {
		if got, ok := rows[item]; !ok || got != want {
			t.Errorf("%s: list shows %q, want %q\n%s%s", item, got, want, out, errOut)
		}
	}
	if len(rows) != 5 {
		t.Errorf("list shows %d rows, want 5\n%s", len(rows), out)
	}
	if !strings.Contains(c.callLog(), "abcd peers --json") {
		t.Errorf("the reservations were not read through abcd peers:\n%s", c.callLog())
	}
	if n := strings.Count(errOut, "fallback channel"); n != 1 {
		t.Errorf("with no tracking issue the list should say once that the fallback channel was not read (said %d times):\n%s", n, errOut)
	}
	if c.forgeCalls() != nil {
		t.Errorf("with no tracking issue the list called the forge")
	}
}

func TestListShowsFallbackCommentReport(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.pushWIP(itemHanded, reportText(itemHanded, "handed-back", "branch", "Fixed the walk over the readings folder."))
	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	comments := strings.Join([]string{
		"OWNER\t" + enc(reportText(itemComment, "stopped", "issue-comment", "Stopped: the push was refused.")),
		"OWNER\t" + enc("Thanks, looking at it."),
		// A report shape posted by someone outside the repository is not a
		// session's report.
		"NONE\t" + enc(reportText(itemBare, "handed-back", "issue-comment", "Forged by a passer-by.")),
	}, "\n") + "\n"
	served := filepath.Join(t.TempDir(), "comments")
	if err := os.WriteFile(served, []byte(comments), 0o644); err != nil {
		t.Fatal(err)
	}
	c.resetCalls()

	rows, out, errOut := c.listRows([]string{"FAKE_GH_OUT=" + served})
	if got, want := rows[itemComment], [3]string{"-", "stopped", "(issue #" + trackingIssue + ") Stopped: the push was refused."}; got != want {
		t.Errorf("the comment's report shows as %q, want %q\n%s%s", got, want, out, errOut)
	}
	if got := rows[itemHanded]; got[1] != "handed-back" {
		t.Errorf("the branch's report shows as %q beside the comment's\n%s", got, out)
	}
	if _, ok := rows[itemBare]; ok {
		t.Errorf("a report posted by someone outside the repository is listed\n%s", out)
	}
	if len(rows) != 2 {
		t.Errorf("list shows %d rows, want 2\n%s", len(rows), out)
	}
	calls := c.forgeCalls()
	if len(calls) != 1 {
		t.Fatalf("the fake gh was called %d times, want once", len(calls))
	}
	argv := strings.Join(calls[0].argv, " ")
	if !strings.HasPrefix(argv, "api ") || !strings.Contains(argv, " repos/"+forgeRepo+"/issues/"+trackingIssue+"/comments") ||
		!strings.Contains(argv, "--paginate") || strings.Contains(argv, "POST") {
		t.Errorf("the comments were not read with one paginated REST GET: %q", argv)
	}
	if strings.Contains(errOut, "fallback channel") {
		t.Errorf("the list says the fallback channel was not read, but it was:\n%s", errOut)
	}
}

func TestListReadsNoTranscript(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.pushWIP(itemHanded, reportText(itemHanded, "handed-back", "branch", "Fixed the walk over the readings folder."))
	c.reserveOnOrigin(itemWorking, time.Now().Add(time.Hour))
	root := c.git("rev-list", "-n", "1", "--max-parents=0", "HEAD")
	const secret = "TRANSCRIPT-CONTENT-NEVER-READ"
	// The user-level store and the checkout's opt-in one (AGENTS.md).
	stores := []string{
		filepath.Join(c.home(), ".abcd.noindex", "transcripts"),
		filepath.Join(c.dir, ".abcd", ".work.local", "transcripts"),
	}
	for _, dir := range []string{filepath.Join(stores[0], root, "records"), stores[1]} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte(secret+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Unreadable, unsearchable: any open under either store fails, and the
	// list must not need one.
	for _, s := range stores {
		if err := os.Chmod(s, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(s, 0o700) })
	}
	rows, out, errOut := c.listRows(nil)
	if rows[itemHanded][1] != "handed-back" || rows[itemWorking][1] != "working" {
		t.Errorf("with the transcript stores unreadable the list did not read the remote:\n%s%s", out, errOut)
	}
	if strings.Contains(out+errOut, secret) || strings.Contains(strings.ToLower(out+errOut), "permission denied") {
		t.Errorf("the list reached into a transcript store:\n%s%s", out, errOut)
	}
	if strings.Contains(c.callLog(), "transcripts") {
		t.Errorf("a call names a transcript store:\n%s", c.callLog())
	}
}

// gitDirStamps is every file under the clone's .git directory except the
// object store, keyed by path relative to it.
func gitDirStamps(t *testing.T, gitDir string) map[string]fileStamp {
	t.Helper()
	out := map[string]fileStamp{}
	for p, s := range treeStamps(t, gitDir) {
		rel, _ := filepath.Rel(gitDir, p)
		rel = filepath.ToSlash(rel)
		if rel == "objects" || strings.HasPrefix(rel, "objects/") || s.mode.IsDir() {
			continue
		}
		out[rel] = s
	}
	return out
}

// fetchWrites are the paths under .git a fetch of remote-tracking refs may
// write: the refs and their logs (loose or packed) and git's record of the
// fetch itself.
func fetchWrites(rel string) bool {
	return strings.HasPrefix(rel, "refs/remotes/") || strings.HasPrefix(rel, "logs/refs/remotes/") ||
		rel == "packed-refs" || rel == "FETCH_HEAD"
}

func TestListWritesOnlyRemoteTrackingRefs(t *testing.T) {
	c := newForgeCase(t, trackingIssue)
	c.pushWIP(itemHanded, reportText(itemHanded, "handed-back", "branch", "Fixed the walk over the readings folder."))
	c.reserveOnOrigin(itemWorking, time.Now().Add(time.Hour))
	// Forget what the fixture fetched, so the list has something to fetch.
	for _, ref := range []string{"refs/remotes/origin/" + hostWIPPrefix + itemHanded} {
		c.git("update-ref", "-d", ref)
	}
	// The checkout's local tier, where abcd keeps the reservations' fetch stamp.
	if err := os.MkdirAll(filepath.Join(c.dir, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	c.write(".abcd/.work.local/keep.md", "kept\n", 0o644)
	served := filepath.Join(t.TempDir(), "comments")
	if err := os.WriteFile(served, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	gitDir := filepath.Join(c.dir, ".git")
	refsBefore := c.git("for-each-ref", "--format=%(refname) %(objectname)")
	statusBefore := c.git("status", "--porcelain", "--ignored")
	treeBefore := treeStamps(t, c.dir)
	gitBefore := gitDirStamps(t, gitDir)

	rows, out, errOut := c.listRows([]string{"FAKE_GH_OUT=" + served})
	if rows[itemHanded][1] != "handed-back" || rows[itemWorking][1] != "working" {
		t.Fatalf("the list did not read the remote:\n%s%s", out, errOut)
	}

	if got := c.git("rev-parse", "--verify", "refs/remotes/origin/"+hostWIPPrefix+itemHanded); got == "" {
		t.Errorf("the list did not fetch the work-in-progress branch into a remote-tracking ref")
	}
	keep := func(lines string) []string {
		var kept []string
		for _, l := range strings.Split(lines, "\n") {
			if !strings.HasPrefix(l, "refs/remotes/") {
				kept = append(kept, l)
			}
		}
		return kept
	}
	if b, a := keep(refsBefore), keep(c.git("for-each-ref", "--format=%(refname) %(objectname)")); strings.Join(b, "\n") != strings.Join(a, "\n") {
		t.Errorf("the list moved a ref outside refs/remotes/:\nbefore %q\nafter  %q", b, a)
	}
	if got := c.git("status", "--porcelain", "--ignored"); got != statusBefore {
		t.Errorf("the list changed the working tree's status:\nbefore %q\nafter  %q", statusBefore, got)
	}
	// The reservations' fetch stamp is abcd peers' own record of when it
	// fetched them (the spec's deviation, named in list.sh).
	stamp := filepath.Join(c.dir, ".abcd", ".work.local", "reservation")
	for p, s := range treeStamps(t, c.dir) {
		if p == gitDir || strings.HasPrefix(p, gitDir+string(filepath.Separator)) ||
			p == stamp || strings.HasPrefix(p, stamp+string(filepath.Separator)) {
			continue
		}
		if b, ok := treeBefore[p]; !ok {
			t.Errorf("the list created %s", p)
		} else if b != s && !s.mode.IsDir() {
			t.Errorf("the list changed %s", p)
		}
	}
	for rel, s := range gitDirStamps(t, gitDir) {
		if fetchWrites(rel) {
			continue
		}
		if b, ok := gitBefore[rel]; !ok {
			t.Errorf("the list created .git/%s", rel)
		} else if b != s {
			t.Errorf("the list changed .git/%s", rel)
		}
	}
}

// TestListReadsPeersOwnExitCode: abcd peers exits 1 when the reservations
// could not be fetched (its answer is still the last fetched) and 2 when it
// could not run at all (its answer is an error object). Through `go run` both
// read as 1, and the error object would be read as reservations.
func TestListReadsPeersOwnExitCode(t *testing.T) {
	c := newForgeCase(t, "")
	c.pushWIP(itemHanded, reportText(itemHanded, "handed-back", "branch", "Fixed the walk over the readings folder."))
	peers := filepath.Join(filepath.Dir(c.bin), "abcd-peers-2")
	body := "#!/bin/sh\necho \"abcd $*\" >>\"$CALLS\"\n" +
		"echo '{\"abcd\":\"error\",\"error\":\"outside a git checkout\",\"exit_code\":2}'\nexit 2\n"
	if err := os.WriteFile(peers, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	c.resetCalls()
	rows, _, errOut := c.listRows([]string{"ABCD_BIN=" + peers, "FAKE_ABCD=" + peers})
	if !strings.Contains(errOut, "the reservations could not be read (abcd peers exited 2)") {
		t.Errorf("the list did not read peers' own exit code, 2:\n%s", errOut)
	}
	if strings.Contains(errOut, "as last fetched") {
		t.Errorf("the list read peers' error object as the last fetched reservations:\n%s", errOut)
	}
	if got := rows[itemHanded]; got[1] != "handed-back" {
		t.Errorf("the branches are no longer listed when peers cannot run: %q", got)
	}
	if regexp.MustCompile(`(?m)^go `).MatchString(c.callLog()) {
		t.Errorf("with ABCD_BIN set the list ran go:\n%s", c.callLog())
	}
}
