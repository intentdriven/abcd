package claim

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// world is a bare repository standing for origin, seeded with a main branch,
// and the clones that stand for the people and computers sharing it.
type world struct {
	t    *testing.T
	seed *gittest.Repo
	bare string
	mu   sync.Mutex
	now  time.Time
	// homes is the home each email's clones share when a test names none:
	// one person's copies on one machine share their abcd home, and with it
	// their holder id (holderid.go).
	homes map[string]string
}

// participant is one working copy: a clone with its own git identity, local
// tier, home (for the machine layer) and the world's clock.
type participant struct {
	w    *world
	dir  string
	home string
}

func newWorld(t *testing.T) *world {
	t.Helper()
	return newWorldFormat(t, "")
}

// newWorldFormat is newWorld in git's object format named ("sha256" for the
// longer object names; "" is git's default). The first world of a format in a
// test process is built with git and kept as that format's template; every
// later one is a copy of it (worldTemplates).
func newWorldFormat(t *testing.T, format string) *world {
	t.Helper()
	worldTemplates.mu.Lock()
	defer worldTemplates.mu.Unlock()
	if tpl, ok := worldTemplates.byFormat[format]; ok {
		return tpl.copyFor(t)
	}
	w := buildWorld(t, format)
	keepWorldTemplate(t, format, w)
	return w
}

// buildWorld seeds a world with git: a checkout holding one commit, pushed to
// main on a fresh bare origin.
func buildWorld(t *testing.T, format string) *world {
	t.Helper()
	seed := gittest.NewRepoFormat(t, format)
	seed.Write("README.md", "seed\n")
	seed.Commit("seed")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBareFormat(t, bare, format)
	seed.Git("remote", "add", "origin", bare)
	seed.Git("push", "-q", "origin", "main")
	return &world{t: t, seed: seed, bare: bare, now: t0}
}

// worldTemplates holds, per object format, the first world this test process
// built, copied before its test touched it. Every world starts from the same
// seeded history, and building one with git cost about seventeen processes,
// near a tenth of what a typical test in this package spawns; a copy costs
// one, the git that points its seed at its own origin. Each copy is still a test's own seed and origin, in its own temporary
// directories, its seed's origin naming its own bare repository. TestMain
// removes the templates when the tests end.
var worldTemplates struct {
	mu       sync.Mutex
	root     string
	byFormat map[string]worldTemplate
}

// worldTemplate is one kept world: the seed's and the origin's copies.
type worldTemplate struct {
	seed, bare string
}

// keepWorldTemplate copies w, just built, into the format's template.
func keepWorldTemplate(t *testing.T, format string, w *world) {
	t.Helper()
	if worldTemplates.root == "" {
		dir, err := os.MkdirTemp("", "claim-worlds-")
		if err != nil {
			t.Fatal(err)
		}
		worldTemplates.root = dir
	}
	dir := filepath.Join(worldTemplates.root, "format-"+format)
	tpl := worldTemplate{seed: filepath.Join(dir, "seed"), bare: filepath.Join(dir, "origin.git")}
	copyTree(t, w.seed.Root(), tpl.seed)
	copyTree(t, w.bare, tpl.bare)
	if worldTemplates.byFormat == nil {
		worldTemplates.byFormat = map[string]worldTemplate{}
	}
	worldTemplates.byFormat[format] = tpl
}

// copyFor is a new world for t copied from the template, laid out as
// buildWorld lays one out.
func (tpl worldTemplate) copyFor(t *testing.T) *world {
	t.Helper()
	seedDir := t.TempDir()
	copyTree(t, tpl.seed, seedDir)
	bare := filepath.Join(t.TempDir(), "origin.git")
	copyTree(t, tpl.bare, bare)
	// git writes the new origin itself, so a path it must quote (a '#' from a
	// subtest's generated name, a space) is quoted as buildWorld's was.
	seed := gittest.OpenRepo(t, seedDir)
	seed.Git("remote", "set-url", "origin", bare)
	return &world{t: t, seed: seed, bare: bare, now: t0}
}

// copyTree copies the directory src into dst, creating dst if need be.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatalf("copying %s: %v", src, err)
	}
}

// removeWorldTemplates deletes the kept templates; TestMain calls it once the
// tests have run.
func removeWorldTemplates() {
	worldTemplates.mu.Lock()
	defer worldTemplates.mu.Unlock()
	if worldTemplates.root != "" {
		_ = os.RemoveAll(worldTemplates.root)
	}
}

func (w *world) clock() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.now
}

func (w *world) set(at time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.now = at
}

// clone makes a participant whose user.email is email; home, when not "",
// is the home the machine layer is read under and the holder id is kept in,
// and otherwise every clone of one email shares one home, as one person's
// copies on one machine do. A test standing for another computer, or another
// account, of the same person names a home of its own.
func (w *world) clone(email, home string) *participant {
	w.t.Helper()
	dir := filepath.Join(w.t.TempDir(), "copy")
	// The clone's own config is written by the clone itself (-c), one process
	// where a `git config` apiece cost six more.
	clone := []string{"clone", "-q"}
	for _, kv := range [][2]string{{"user.name", "Pat Example"}, {"user.email", email}, {"gc.auto", "0"},
		{"gc.autodetach", "false"}, {"maintenance.auto", "false"}, {"core.fsmonitor", "false"}} {
		clone = append(clone, "-c", kv[0]+"="+kv[1])
	}
	w.seed.Git(append(clone, w.bare, dir)...)
	p := &participant{w: w, dir: dir, home: home}
	if p.home == "" {
		key := strings.ToLower(email)
		if w.homes[key] == "" {
			if w.homes == nil {
				w.homes = map[string]string{}
			}
			w.homes[key] = w.t.TempDir()
		}
		p.home = w.homes[key]
	}
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local"), 0o755); err != nil {
		w.t.Fatal(err)
	}
	// The local tier is ignored, as every managed repository's .gitignore has it.
	if err := os.WriteFile(filepath.Join(dir, ".git", "info", "exclude"), []byte(".abcd/.work.local/\n"), 0o644); err != nil {
		w.t.Fatal(err)
	}
	return p
}

func (p *participant) opts() Options { return Options{Now: p.w.clock, Home: p.home} }

func (p *participant) reserve(id string, fp ...string) (Result, error) {
	return Reserve(p.dir, Request{Record: id, Footprint: fp}, p.opts())
}

func (p *participant) do(req Request) (Result, error) { return Reserve(p.dir, req, p.opts()) }

func (p *participant) git(args ...string) string {
	p.w.t.Helper()
	return runGit(p.w.t, p.dir, args...)
}

// runGit runs git in dir under the hermetic test environment.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, filepath.Base(dir), err, out)
	}
	return strings.TrimSpace(string(out))
}

// remoteTip is the reservation branch's head on origin, "" when absent.
func (w *world) remoteTip(id string) string {
	w.t.Helper()
	out := runGit(w.t, w.bare, "for-each-ref", "--format=%(objectname)", RefPrefix+id)
	return strings.TrimSpace(out)
}

// remoteMessage is a commit's message as origin holds it.
func (w *world) remoteMessage(sha string) string {
	w.t.Helper()
	return runGit(w.t, w.bare, "log", "-1", "--format=%B", sha)
}

// parentOf is a commit's parent on origin, "" for a root commit.
func (w *world) parentOf(sha string) string {
	w.t.Helper()
	f := strings.Fields(runGit(w.t, w.bare, "rev-list", "--parents", "-n", "1", sha))
	if len(f) < 2 {
		return ""
	}
	return f[1]
}

// TestAReservationIsAParentlessEmptyTreeCommitAndLeavesTheCheckoutAlone is C1.
func TestAReservationIsAParentlessEmptyTreeCommitAndLeavesTheCheckoutAlone(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	p.git("checkout", "-q", "--detach", "HEAD")
	if err := os.WriteFile(filepath.Join(p.dir, "README.md"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.dir, "staged.txt"), []byte("staged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p.git("add", "staged.txt")
	head, index, status := p.git("rev-parse", "HEAD"), p.git("ls-files", "-s"), p.git("status", "--porcelain")

	res, err := p.reserve("iss-5", "a.go")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	tip := w.remoteTip("iss-5")
	if res.Act != ActReserved || res.Commit != tip || tip == "" {
		t.Fatalf("result = %+v, remote tip %s; want a reservation at the tip", res, tip)
	}
	if w.parentOf(tip) != "" {
		t.Fatalf("the reservation has a parent; want a parentless commit")
	}
	if tree := runGit(t, w.bare, "rev-parse", tip+"^{tree}"); tree != "4b825dc642cb6eb9a060e54bf8d69288fbee4904" {
		t.Fatalf("tree = %s; want the empty tree", tree)
	}
	if who := runGit(t, w.bare, "log", "-1", "--format=%an <%ae>|%cn <%ce>", tip); who != "Pat Example <1234+octo@users.noreply.github.com>|Pat Example <1234+octo@users.noreply.github.com>" {
		t.Fatalf("author|committer = %q; want P's git identity", who)
	}
	msg := w.remoteMessage(tip)
	for _, want := range []string{"Abcd-Reservation: iss-5\n", "Abcd-Holder: " + p.keptHolderID() + "\n", "Abcd-Footprint: a.go\n",
		"Abcd-Claimed-At: 2026-10-03T14:00:00Z\n", "Abcd-Expires-At: 2026-10-03T16:00:00Z\n", "Assisted-by: abcd:dev"} {
		if !strings.Contains(msg+"\n", want) {
			t.Fatalf("message lacks %q:\n%s", want, msg)
		}
	}
	if h, i, s := p.git("rev-parse", "HEAD"), p.git("ls-files", "-s"), p.git("status", "--porcelain"); h != head || i != index || s != status {
		t.Fatalf("the checkout moved: HEAD %s→%s, index changed %v, status %q→%q", head, h, i != index, status, s)
	}
	if local := p.git("for-each-ref", "refs/heads/abcd-reserved/"); local != "" {
		t.Fatalf("the reservation left a local branch: %s", local)
	}
}

// TestAnotherPersonsLiveReservationIsRefusedWithTheBackOffExit is C2.
func TestAnotherPersonsLiveReservationIsRefusedWithTheBackOffExit(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	tip := w.remoteTip("iss-5")
	w.set(t0.Add(time.Hour))
	_, err := q.reserve("iss-5", "b.go")
	want := "iss-5 is reserved by " + p.keptHolderID() + " until 2026-10-03T16:00:00Z; back off and take other work"
	if !errors.Is(err, ErrBackOff) || err.Error() != want {
		t.Fatalf("Q's reserve = %v; want %q", err, want)
	}
	if got := w.remoteTip("iss-5"); got != tip {
		t.Fatalf("the branch moved to %s", got)
	}
	// --take-over never moves another person's live reservation.
	if _, err := q.do(Request{Record: "iss-5", Footprint: []string{"b.go"}, TakeOver: true}); !errors.Is(err, ErrBackOff) {
		t.Fatalf("Q's take-over of P's live reservation = %v; want the back-off", err)
	}
}

// TestAStaleViewLosesTheRaceAndNamesTheWinner is C3, through both ways the
// race shows. "advertised": P's reservation lands after Q's fetch and before
// Q's push connects, so git's own client sees the moved tip in the remote's
// advertisement and rejects the push as a non-fast-forward (this is also what
// stops any pre-push hook from ever being asked: git rejects the ref before it
// runs the hook). "landed": P's reservation lands on the remote while Q's push
// is already inside receive-pack, after the advertisement, so the remote's own
// compare-and-swap refuses Q's update; a pre-receive hook in origin stands in
// for P's push arriving in that window.
func TestAStaleViewLosesTheRaceAndNamesTheWinner(t *testing.T) {
	for _, mode := range []string{"advertised", "landed"} {
		t.Run(mode, func(t *testing.T) {
			w := newWorld(t)
			p := w.clone("1234+octo@users.noreply.github.com", "")
			q := w.clone("99+quinn@users.noreply.github.com", "")
			marker := filepath.Join(t.TempDir(), "pre-receive-ran")
			if mode == "advertised" {
				// Q's fetch has run; P reserves while Q's push is on its way.
				beforePush = func(root string) {
					if root != q.dir {
						return
					}
					beforePush = nil
					if _, err := p.reserve("iss-5", "a.go"); err != nil {
						t.Errorf("P's reserve inside Q's window: %v", err)
					}
				}
				t.Cleanup(func() { beforePush = nil })
			} else {
				// P's reservation commit is made, then parked off the branch, so
				// Q's fetch finds no reservation; origin's pre-receive puts it on
				// the branch while Q's push is being received.
				res, err := p.reserve("iss-5", "a.go")
				if err != nil {
					t.Fatal(err)
				}
				runGit(t, w.bare, "update-ref", "refs/parked/iss-5", res.Commit)
				runGit(t, w.bare, "update-ref", "-d", RefPrefix+"iss-5")
				hook := "#!/bin/sh\nif [ ! -e '" + marker + "' ]; then\n  : > '" + marker + "'\n" +
					"  env -u GIT_QUARANTINE_PATH -u GIT_OBJECT_DIRECTORY -u GIT_ALTERNATE_OBJECT_DIRECTORIES " +
					"git update-ref " + RefPrefix + "iss-5 " + res.Commit + " || exit 1\nfi\nexit 0\n"
				if err := os.WriteFile(filepath.Join(w.bare, "hooks", "pre-receive"), []byte(hook), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			_, err := q.reserve("iss-5", "b.go")
			want := "iss-5 was reserved by " + w.holderIDOf("1234+octo@users.noreply.github.com") + " while this push was on its way, until 2026-10-03T16:00:00Z; back off and take other work"
			if !errors.Is(err, ErrBackOff) || err.Error() != want {
				t.Fatalf("Q's reserve = %v; want %q", err, want)
			}
			if mode == "landed" {
				if _, serr := os.Stat(marker); serr != nil {
					t.Fatalf("origin's pre-receive never ran, so the push was not refused by the remote: %v", serr)
				}
			}
			if !strings.Contains(w.remoteMessage(w.remoteTip("iss-5")), "Abcd-Holder: "+p.keptHolderID()+"\n") {
				t.Fatalf("origin's tip is not P's")
			}
			if local := q.git("for-each-ref", "refs/heads/abcd-reserved/"); local != "" {
				t.Fatalf("Q wrote a reservation locally: %s", local)
			}
		})
	}
}

// TestAFreshAgentInTheSameCopyRenews is C4.
func TestAFreshAgentInTheSameCopyRenews(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	first, err := p.reserve("iss-5", "a.go")
	if err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(30 * time.Minute))
	res, err := p.reserve("iss-5", "a.go")
	if err != nil || res.Act != ActRenewed {
		t.Fatalf("the second agent's reserve = %+v, %v; want a renewal", res, err)
	}
	tip := w.remoteTip("iss-5")
	if w.parentOf(tip) != first.Commit || !res.ExpiresAt.After(first.ExpiresAt) ||
		!strings.Contains(w.remoteMessage(tip), "Abcd-Expires-At: 2026-10-03T16:30:00Z") {
		t.Fatalf("renewal %+v at %s; want a child of %s with a later expiry", res, tip, first.Commit)
	}
}

// TestARenewalSaysWhetherTheReservationWasStillLive: a renewal of this copy's
// live reservation reports it live, one of a reservation that had lapsed
// reports it lapsed, so a drain can tell a person's own hold from one its
// push brought back.
func TestARenewalSaysWhetherTheReservationWasStillLive(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(30 * time.Minute))
	live, err := p.reserve("iss-5", "a.go")
	if err != nil || live.Act != ActRenewed || !live.RenewedLive {
		t.Fatalf("the renewal of a live reservation = %+v, %v; want it reported live", live, err)
	}
	w.set(live.ExpiresAt.Add(time.Hour))
	lapsed, err := p.reserve("iss-5", "a.go")
	if err != nil || lapsed.Act != ActRenewed || lapsed.RenewedLive {
		t.Fatalf("the renewal of a lapsed reservation = %+v, %v; want it reported lapsed", lapsed, err)
	}
}

// TestReleasingAnotherPersonsReservationPushesNothing is C5.
func TestReleasingAnotherPersonsReservationPushesNothing(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := q.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	tip := w.remoteTip("iss-5")
	_, err := p.do(Request{Record: "iss-5", Release: true})
	if !errors.Is(err, ErrRefused) || err.Error() != "iss-5 is reserved by "+q.keptHolderID()+", not by you; nothing was pushed" {
		t.Fatalf("P's release = %v; want the refusal naming Q's holder id", err)
	}
	if w.remoteTip("iss-5") != tip {
		t.Fatalf("the branch moved")
	}
}

// TestReleasePushesATombstoneAndDeletesNothing is C6, and a second release
// pushing nothing.
func TestReleasePushesATombstoneAndDeletesNothing(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	first, err := p.reserve("iss-5", "a.go")
	if err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(20 * time.Minute))
	res, err := p.do(Request{Record: "iss-5", Release: true})
	if err != nil || res.Act != ActReleased || res.Copy != CopyThis {
		t.Fatalf("release = %+v, %v", res, err)
	}
	tip := w.remoteTip("iss-5")
	msg := w.remoteMessage(tip)
	if tip == "" || w.parentOf(tip) != first.Commit || !strings.HasPrefix(msg, "abcd reservation released: iss-5") ||
		!strings.Contains(msg, "Abcd-Expires-At: 2026-10-03T14:20:00Z\nAbcd-Released-At: 2026-10-03T14:20:00Z") {
		t.Fatalf("tombstone %s (parent %s):\n%s", tip, w.parentOf(tip), msg)
	}
	again, err := p.do(Request{Record: "iss-5", Release: true})
	if err != nil || again.Act != ActAlreadyReleased || again.Copy != CopyThis || w.remoteTip("iss-5") != tip {
		t.Fatalf("a second release = %+v, %v; want already released, nothing pushed", again, err)
	}
}

// TestATombstoneIsReservedAsItsChild is C7.
func TestATombstoneIsReservedAsItsChild(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	rel, err := p.do(Request{Record: "iss-5", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	res, err := q.reserve("iss-5", "b.go")
	if err != nil || res.Act != ActReserved || w.parentOf(w.remoteTip("iss-5")) != rel.Commit {
		t.Fatalf("Q's reserve over a tombstone = %+v, %v; want a child of %s", res, err, rel.Commit)
	}
}

// TestAReservationLapsedPastTheMarginIsTakenOverNamingWhatItReplaced is C8.
func TestAReservationLapsedPastTheMarginIsTakenOverNamingWhatItReplaced(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	first, err := p.reserve("iss-5", "a.go")
	if err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(2*time.Hour + TakeoverMargin))
	res, err := q.reserve("iss-5", "b.go")
	if err != nil || res.Act != ActTakenOver || res.Replaced == nil || res.Replaced.Holder != p.keptHolderID() ||
		!res.Replaced.EffectiveExpiry.Equal(t0.Add(2*time.Hour)) || res.Replaced.Commit != first.Commit {
		t.Fatalf("Q's take-over = %+v (%+v), %v; want P's reservation to 16:00Z replaced", res, res.Replaced, err)
	}
	if w.parentOf(w.remoteTip("iss-5")) != first.Commit {
		t.Fatalf("the take-over is not a child of P's tip")
	}
}

// TestAReservationLapsedUnderTheMarginIsRefused is C9.
func TestAReservationLapsedUnderTheMarginIsRefused(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	tip := w.remoteTip("iss-5")
	w.set(t0.Add(2*time.Hour + TakeoverMargin - time.Second))
	_, err := q.reserve("iss-5", "b.go")
	want := "iss-5 is reserved by " + p.keptHolderID() + ", lapsed at 2026-10-03T16:00:00Z, under five minutes ago by this computer's clock; retry after 2026-10-03T16:05:00Z"
	if !errors.Is(err, ErrBackOff) || err.Error() != want {
		t.Fatalf("Q's reserve = %v; want %q", err, want)
	}
	if w.remoteTip("iss-5") != tip {
		t.Fatalf("the branch moved")
	}
}

// pushHostile pushes a commit with msg on the empty tree, dated at, as the
// reservation branch for id on origin, from a scratch clone.
func (w *world) pushHostile(id, msg string, at time.Time) string {
	w.t.Helper()
	dir := filepath.Join(w.t.TempDir(), "hostile")
	w.seed.Git("clone", "-q", w.bare, dir)
	path := filepath.Join(w.t.TempDir(), "msg")
	if err := os.WriteFile(path, []byte(msg), 0o600); err != nil {
		w.t.Fatal(err)
	}
	tree := runGit(w.t, dir, "hash-object", "-w", "-t", "tree", "/dev/null")
	date := "@" + strconv.FormatInt(at.Unix(), 10) + " +0000"
	cmd := exec.Command("git", "-C", dir, "-c", "user.name=Mal", "-c", "user.email=mal@example.invalid", "-c", "commit.gpgsign=false",
		"commit-tree", tree, "-F", path)
	cmd.Env = append(gittest.Env(w.t), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	out, err := cmd.Output()
	if err != nil {
		w.t.Fatal(err)
	}
	sha := strings.TrimSpace(string(out))
	runGit(w.t, dir, "push", "-q", "origin", sha+":"+RefPrefix+id)
	return sha
}

// TestAnUnreadableTipPastItsHoldIsTakenOverNamingTheCommit is C13, with the
// hold's other side refused.
func TestAnUnreadableTipPastItsHoldIsTakenOverNamingTheCommit(t *testing.T) {
	w := newWorld(t)
	q := w.clone("99+quinn@users.noreply.github.com", "")
	bad := w.pushHostile("iss-5", "abcd reservation: iss-5\n\nAbcd-Reservation: iss-5\nAbcd-Holder: octo\n"+
		"Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\nAbcd-Claimed-At: now\nAbcd-Expires-At: never\n", t0)
	w.set(t0.Add(2*time.Hour + TakeoverMargin - time.Second))
	_, err := q.reserve("iss-5", "b.go")
	want := "the reservation on iss-5 is unreadable (" + reasonTime + "); it is held until 2026-10-03T16:00:00Z and can be taken over after 2026-10-03T16:05:00Z"
	if !errors.Is(err, ErrBackOff) || err.Error() != want {
		t.Fatalf("within the hold = %v; want %q", err, want)
	}
	w.set(t0.Add(2*time.Hour + TakeoverMargin))
	res, err := q.reserve("iss-5", "b.go")
	if err != nil || res.Act != ActTakenOver || res.Replaced == nil || res.Replaced.Commit != bad || res.Replaced.Unreadable != reasonTime {
		t.Fatalf("past the hold = %+v (%+v), %v; want the unreadable %s replaced", res, res.Replaced, err, bad)
	}
	if w.parentOf(w.remoteTip("iss-5")) != bad {
		t.Fatalf("the take-over is not a child of the unreadable tip")
	}
}

// TestATerminalRecordOnTheDefaultBranchReleasesItsReservation is C15: the
// view counts the tip released, and since spc-2610161044461448 step 1 the
// record is not reserved again (iss-2610160957404827).
func TestATerminalRecordOnTheDefaultBranchReleasesItsReservation(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	first, err := p.reserve("iss-5", "a.go")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.reserve("iss-5", "b.go"); !errors.Is(err, ErrBackOff) {
		t.Fatalf("before the record is resolved: %v; want the back-off", err)
	}
	w.seed.Write(".abcd/work/issues/resolved/iss-5-the-widget.md", "---\nid: iss-5\n---\n# The widget\n")
	w.seed.Commit("resolve iss-5")
	w.seed.Git("push", "-q", "origin", "main")
	_, err = q.reserve("iss-5", "b.go")
	if !errors.Is(err, ErrBackOff) || w.remoteTip("iss-5") != first.Commit {
		t.Fatalf("after iss-5 reached resolved/ on main: %v; want the back-off, P's tip left as it stands", err)
	}
	// An unreadable tip on a finished record counts as released too, within
	// what would otherwise be its hold.
	bad := w.pushHostile("iss-6", "abcd reservation: iss-6\n\nnot a reservation at all\n", t0)
	w.seed.Write(".abcd/work/issues/wontfix/iss-6-the-gadget.md", "---\nid: iss-6\n---\n# The gadget\n")
	w.seed.Commit("wontfix iss-6")
	w.seed.Git("push", "-q", "origin", "main")
	_, err = q.reserve("iss-6", "b.go")
	if !errors.Is(err, ErrBackOff) || w.remoteTip("iss-6") != bad {
		t.Fatalf("over an unreadable tip on a wontfix record: %v; want the back-off, the tip left as it stands", err)
	}
	snap, err := View(q.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range snap.Reservations {
		if r.State != StateReleased || r.ReleasedBy != ReleasedByDefaultBranch {
			t.Fatalf("the view shows %s as %s by %q; want it released by the default branch", r.Record, r.State, r.ReleasedBy)
		}
	}
	if len(snap.Reservations) != 2 {
		t.Fatalf("the view = %+v; want both tips", snap.Reservations)
	}
}

// TestAnotherCopyOfTheSamePersonIsRefusedWithoutTakeOver is C20: a laptop that
// resolves its username through the machine config and a cloud copy that
// resolves it through a no-reply address are one person, two copies, once the
// cloud copy's home declares the laptop's holder id.
func TestAnotherCopyOfTheSamePersonIsRefusedWithoutTakeOver(t *testing.T) {
	w := newWorld(t)
	laptop := w.clone("pat@example.com", machineConfig(t, `{"reservation": {"github_username": "octo"}}`))
	cloud := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := laptop.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	declareHolderIDs(t, cloud.home, laptop.keptHolderID()+"\n")
	tip := w.remoteTip("iss-5")
	_, err := cloud.reserve("iss-5", "a.go")
	want := "you are already building iss-5 on another computer, until 2026-10-03T16:00:00Z (" + laptop.keptHolderID() + " is an id " + abcdhome.Display("holder-ids") + " " +
		"declares as yours with no login beside it, so which account pushed this reservation is not checked); --take-over is still allowed here, only because " +
		abcdhome.Display("holder-ids") + " lists " + laptop.keptHolderID() + " as yours: it replaces that reservation with yours; if the id is not yours, remove it from " +
		abcdhome.Display("holder-ids") + " and leave the reservation be"
	if !errors.Is(err, ErrBackOff) || err.Error() != want {
		t.Fatalf("the cloud copy's reserve = %v; want %q", err, want)
	}
	if w.remoteTip("iss-5") != tip {
		t.Fatalf("the branch moved")
	}
	// Nor may it release the laptop's.
	_, err = cloud.do(Request{Record: "iss-5", Release: true})
	if !errors.Is(err, ErrRefused) || err.Error() != "iss-5 is held by your copy on another computer; release it there, or take it over here first; nothing was pushed" {
		t.Fatalf("the cloud copy's release = %v", err)
	}
}

// TestTakeOverMovesTheReservationToThisCopy is C52.
func TestTakeOverMovesTheReservationToThisCopy(t *testing.T) {
	w := newWorld(t)
	laptop := w.clone("pat@example.com", machineConfig(t, `{"reservation": {"github_username": "octo"}}`))
	cloud := w.clone("1234+octo@users.noreply.github.com", "")
	first, err := laptop.reserve("iss-5", "a.go")
	if err != nil {
		t.Fatal(err)
	}
	declareHolderIDs(t, cloud.home, laptop.keptHolderID()+"\n")
	res, err := cloud.do(Request{Record: "iss-5", Footprint: []string{"a.go"}, TakeOver: true})
	if err != nil || res.Act != ActTakenOver || res.Replaced == nil || res.Replaced.Copy != CopyAnother || res.Replaced.Holder != laptop.keptHolderID() {
		t.Fatalf("take-over = %+v (%+v), %v; want the laptop's copy replaced", res, res.Replaced, err)
	}
	cloudKey, err := copyKey(cloud.dir)
	if err != nil {
		t.Fatal(err)
	}
	tip := w.remoteTip("iss-5")
	if w.parentOf(tip) != first.Commit || !strings.Contains(w.remoteMessage(tip), "Abcd-Copy: "+cloudKey) {
		t.Fatalf("the tip %s does not name the cloud copy as a child of the laptop's", tip)
	}
}

// TestARemoteNotNamedOriginIsRefusedNamingTheRemotesFound is C21.
func TestARemoteNotNamedOriginIsRefusedNamingTheRemotesFound(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	p.git("remote", "rename", "origin", "upstream")
	_, err := p.reserve("iss-5", "a.go")
	want := "reservations live on the remote named origin, and this checkout's remotes are: upstream; nothing was pushed"
	if !errors.Is(err, ErrRefused) || err.Error() != want {
		t.Fatalf("reserve = %v; want %q", err, want)
	}
	p.git("remote", "remove", "upstream")
	if _, err := p.reserve("iss-5", "a.go"); err == nil || err.Error() != "reservations live on the remote named origin, and this checkout has no remote; nothing was pushed" {
		t.Fatalf("with no remote: %v", err)
	}
	if w.remoteTip("iss-5") != "" {
		t.Fatalf("a reservation reached origin")
	}
}

// TestAMissingOriginFailsAndMakesNoReservation is C22.
func TestAMissingOriginFailsAndMakesNoReservation(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	p.git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
	_, err := p.reserve("iss-5", "a.go")
	if !errors.Is(err, ErrRefused) || !strings.HasPrefix(err.Error(), "could not read the reservations from origin: ") ||
		!strings.Contains(err.Error(), "gone.git") || !strings.HasSuffix(err.Error(), "; no reservation was made") {
		t.Fatalf("reserve = %v; want the fetch failure naming its cause", err)
	}
	if refs := p.git("for-each-ref", "refs/heads/abcd-reserved/", TrackingPrefix); refs != "" {
		t.Fatalf("a failed reservation left refs: %s", refs)
	}
	if _, err := os.Stat(filepath.Join(p.dir, filepath.FromSlash(fetchedAtRel))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a failed fetch stamped the view: %v", err)
	}
}

// TestARejectingRemoteIsQuotedAndMakesNoReservation is C23: a pre-receive hook
// refusing the prefix stands in for a missing write permission and for a
// proxy that rejects the push.
func TestARejectingRemoteIsQuotedAndMakesNoReservation(t *testing.T) {
	w := newWorld(t)
	hook := "#!/bin/sh\nwhile read old new ref; do case $ref in refs/heads/abcd-reserved/*) " +
		"echo 'the proxy refuses abcd-reserved pushes' >&2; exit 1;; esac; done\nexit 0\n"
	if err := os.WriteFile(filepath.Join(w.bare, "hooks", "pre-receive"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	p := w.clone("1234+octo@users.noreply.github.com", "")
	_, err := p.reserve("iss-5", "a.go")
	if !errors.Is(err, ErrRefused) || !strings.HasPrefix(err.Error(), "origin refused the reservation push: ") ||
		!strings.Contains(err.Error(), "the proxy refuses abcd-reserved pushes") || !strings.HasSuffix(err.Error(), "; no reservation was made") {
		t.Fatalf("reserve = %v; want the rejection quoted", err)
	}
	if w.remoteTip("iss-5") != "" || p.git("for-each-ref", "refs/heads/abcd-reserved/") != "" {
		t.Fatalf("a rejected reservation left a ref")
	}
}

// TestARejectedPushOverATakeableTipIsReportedAsItself: when the branch moved
// while the push was on its way but now holds nothing this act is barred
// from (P reserved and released in the window), the rejection is no lost
// race; git's own word for it is quoted and no reservation was made.
func TestARejectedPushOverATakeableTipIsReportedAsItself(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	beforePush = func(root string) {
		if root != q.dir {
			return
		}
		beforePush = nil
		if _, err := p.reserve("iss-5", "a.go"); err != nil {
			t.Errorf("P's reserve: %v", err)
		}
		if _, err := p.do(Request{Record: "iss-5", Release: true}); err != nil {
			t.Errorf("P's release: %v", err)
		}
	}
	t.Cleanup(func() { beforePush = nil })
	_, err := q.reserve("iss-5", "b.go")
	if !errors.Is(err, ErrRefused) || !strings.HasPrefix(err.Error(), "origin refused the reservation push: ") ||
		!strings.Contains(err.Error(), "[rejected]") || !strings.HasSuffix(err.Error(), "; no reservation was made") {
		t.Fatalf("Q's reserve = %v; want the push failure quoted as itself", err)
	}
	if !strings.Contains(w.remoteMessage(w.remoteTip("iss-5")), "abcd reservation released: iss-5") {
		t.Fatalf("origin's tip is not P's tombstone")
	}
}

// TestACopiedWorldNamesItsOwnOriginThroughAPathGitQuotes: a world copied from
// the template points its seed at its own origin even where that path holds a
// '#', which git's config must quote (a subtest's generated name carries one),
// and the seed can still reach it.
func TestACopiedWorldNamesItsOwnOriginThroughAPathGitQuotes(t *testing.T) {
	newWorld(t) // the format's template exists before the copy below
	t.Run("a#name", func(t *testing.T) {
		w := newWorld(t)
		if !strings.Contains(w.bare, "#") {
			t.Fatalf("the subtest's origin path carries no '#': %s", w.bare)
		}
		if got := strings.TrimSpace(w.seed.Git("remote", "get-url", "origin")); got != w.bare {
			t.Fatalf("the copied seed's origin is %q, want %q", got, w.bare)
		}
		w.seed.Git("fetch", "-q", "origin")
	})
}
