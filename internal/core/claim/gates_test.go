package claim

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/guard"
)

// The repository's gates over what a reservation puts on the remote
// (spc-2610031753359849 step 4): the attribution gate over its commits, and the
// shell-hazard registry over the one push the verb makes.

// sourceRoot is this checkout's root, where the gate scripts live.
func sourceRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// TestReservationCommitsPassTheAttributionGate is C33: a reserve, a renewal
// and a release, authored by a GitHub no-reply identity and carrying abcd's
// composed `Assisted-by: abcd:<version>` label, pass
// `scripts/check-attribution.sh commits` over their whole range.
func TestReservationCommitsPassTheAttributionGate(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("the attribution gate is a bash script")
	}
	root := sourceRoot(t)
	// The gate's session-URL half runs `abcd lint outbound`; the checker is
	// built from this checkout, as the landing's attribution test builds it.
	bin := filepath.Join(t.TempDir(), "abcd")
	build := exec.Command("go", "build", "-o", bin, "./cmd/abcd")
	build.Dir = root
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the outbound checker: %v\n%s", err, b)
	}

	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(time.Hour))
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "iss-5", Release: true}); err != nil {
		t.Fatal(err)
	}
	tip := w.remoteTip("iss-5")
	p.git("fetch", "-q", "origin", RefPrefix+"iss-5")
	if n := p.git("rev-list", "--count", "origin/main.."+tip); n != "3" {
		t.Fatalf("the reservation chain holds %s commits, want 3", n)
	}

	cmd := exec.Command("bash", filepath.Join(root, "scripts", "check-attribution.sh"), "commits", "origin/main", tip)
	cmd.Dir = p.dir
	cmd.Env = append(os.Environ(), "ABCD_OUTBOUND_BIN="+bin)
	if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "check-attribution: clean") {
		t.Fatalf("the attribution gate refused the reservation commits: %v\n%s", err, out)
	}
}

// reservationPushFixture is the registry's known_good reservation push, the
// spec's own example of the one push the verb makes.
const reservationPushFixture = "git push --porcelain origin 9f2c41d07be35a6c80e1f4b2a7d39c559f2c41d0:refs/heads/abcd-reserved/iss-2609020716570699"

// TestTheRegistryPassesTheReservationPush: the push the verb builds is the
// shape the registry's two force-push entries list as known_good, and the
// registry allows it.
func TestTheRegistryPassesTheReservationPush(t *testing.T) {
	got := "git " + strings.Join(pushArgs("9f2c41d07be35a6c80e1f4b2a7d39c559f2c41d0", "iss-2609020716570699"), " ")
	if got != reservationPushFixture {
		t.Fatalf("the verb pushes %q; the registry's fixture is %q", got, reservationPushFixture)
	}
	r := guard.Defaults()
	for _, id := range []string{"git-push-force-refspec", "git-push-force"} {
		e, ok := r.Entries[id]
		if !ok {
			t.Fatalf("the bundled registry has no %s entry", id)
		}
		if !slices.Contains(e.Fixtures.KnownGood, reservationPushFixture) {
			t.Errorf("%s lists no reservation push among its known_good fixtures", id)
		}
	}
	d, err := r.Check(got)
	if err != nil || d.Verdict != guard.VerdictAllow {
		t.Fatalf("the registry judged the reservation push %q: %+v, %v", got, d, err)
	}
}
