package claim

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestTheCollectorKeepsWhatLiesOutsideItsKeys: the one trailer collector reads
// only the keys it is given, without regard to case, and keeps every other line
// of the final paragraph and the paragraphs before it, so a family whose
// validator admits nothing outside its trailers can refuse on them.
func TestTheCollectorKeepsWhatLiesOutsideItsKeys(t *testing.T) {
	text := "abcd reservation: iss-5\r\n\r\na free paragraph\n\nAbcd-Reservation: iss-5\nfree text\nX-Other: y\nabcd-holder:  octo \n\n"
	got := collectTrailers(text, reservationFamily.keys)
	if !slices.Equal(got.vals[keyReservation], []string{"iss-5"}) || !slices.Equal(got.vals[keyHolder], []string{"octo"}) {
		t.Fatalf("vals = %v; want the two known keys under their canonical spelling", got.vals)
	}
	if !slices.Equal(got.other, []string{"free text", "X-Other: y"}) {
		t.Fatalf("other = %q; want the two lines no known key names", got.other)
	}
	if got.lead != "abcd reservation: iss-5\n\na free paragraph" {
		t.Fatalf("lead = %q; want the subject and the paragraph before the trailers", got.lead)
	}
	if one := collectTrailers("Abcd-Holder: octo", reservationFamily.keys); one.lead != "" || len(one.other) != 0 {
		t.Fatalf("a one-paragraph message = %+v; want no lead and no other", one)
	}
}

// TestParseTipReadsThroughTheParserItIsGiven: the commit's bounds and empty
// tree are every family's, and the message is the given parser's to read; an
// object outside the bounds never reaches the parser.
func TestParseTipReadsThroughTheParserItIsGiven(t *testing.T) {
	const empty = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	obj := "tree " + empty + "\nparent 1111111111111111111111111111111111111111\n" +
		"committer Pat <p@example.com> 1791900000 +0000\n\nsubject\n\nKey: value\n"
	var seen []string
	parse := func(text string) (int, string) {
		seen = append(seen, text)
		return len(text), ""
	}
	h, n := parseTip("abc", []byte(obj), empty, parse)
	if h.Commit != "abc" || !h.Parent || !h.WrittenAt.Equal(time.Unix(1791900000, 0)) || h.Unreadable != "" ||
		n != len("subject\n\nKey: value\n") {
		t.Fatalf("parseTip = %+v, %d; want the head and the parser's answer", h, n)
	}
	refuse := func(string) (int, string) { return 7, "the parser's reason" }
	if h, n := parseTip("abc", []byte(obj), empty, refuse); h.Unreadable != "the parser's reason" || n != 7 {
		t.Fatalf("a parser's refusal = %+v, %d; want its reason carried", h, n)
	}
	seen = nil
	other := strings.Replace(obj, empty, "1111111111111111111111111111111111111111", 1)
	if h, n := parseTip("abc", []byte(other), empty, parse); h.Unreadable != reasonNotEmptyTree || n != 0 || seen != nil {
		t.Fatalf("a non-empty tree = %+v, %d, parser saw %q; want unreadable before the parser", h, n, seen)
	}
}

// testNamespace is a second family of branches, standing for the next family
// the fetch is given.
var testNamespace = namespace{
	remote: "refs/heads/abcd-test/", tracking: "refs/remotes/origin/abcd-test/",
	stateDir: tierRelDir + "/test/nested", noun: "the test branches",
}

// TestOneFetchReadsEveryNamespaceAndStampsEach: the namespaces given are read
// in one command beside the default branch, and each keeps its own stamp.
func TestOneFetchReadsEveryNamespaceAndStampsEach(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("5678+cat@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	w.seed.Git("push", "-q", "origin", "main:refs/heads/abcd-test/one")
	log := traced(t)
	if gap, err := fetch(q.dir, t0, "nothing was read", reservations, testNamespace); err != nil || gap != "" {
		t.Fatalf("fetch = %q, %v", gap, err)
	}
	var fetches [][]string
	for _, c := range log.cmds {
		if c[0] == "fetch" {
			fetches = append(fetches, c)
		}
	}
	want := []string{"fetch", "--prune", "--no-tags", "origin", reservations.refspec(), testNamespace.refspec(),
		"+refs/heads/main:refs/remotes/origin/main"}
	if len(fetches) != 1 || !slices.Equal(fetches[0], want) {
		t.Fatalf("fetches = %q; want the one command %q", fetches, want)
	}
	if q.git("for-each-ref", "--format=%(refname)", "refs/remotes/origin/abcd-test/") != "refs/remotes/origin/abcd-test/one" ||
		q.git("for-each-ref", "--format=%(refname)", TrackingPrefix) != TrackingPrefix+"iss-5" {
		t.Fatalf("the fetch did not read both namespaces")
	}
	for _, ns := range []namespace{reservations, testNamespace} {
		if got := lastFetched(q.dir, ns); !got.Equal(t0) {
			t.Errorf("%s stamped %v; want %v", ns.noun, got, t0)
		}
	}
	if reservations.stampRel() != fetchedAtRel {
		t.Fatalf("the reservations' stamp moved to %s", reservations.stampRel())
	}
}

// TestAFailedFetchOfAnotherNamespaceNamesItsOwnWords: the reservations'
// refusal keeps its words, and every other fetch names what it read and what
// was not done; neither stamps.
func TestAFailedFetchOfAnotherNamespaceNamesItsOwnWords(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	p.git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
	_, err := fetch(p.dir, t0, "nothing was read", reservations, testNamespace)
	var fe *FetchError
	if !errors.As(err, &fe) || !errors.Is(err, ErrRefused) ||
		!strings.HasPrefix(err.Error(), "could not read the reservations and the test branches from origin: ") ||
		!strings.HasSuffix(err.Error(), "; nothing was read") || !strings.Contains(fe.Reason, "gone.git") {
		t.Fatalf("fetch = %v; want the failure in the fetch's own words", err)
	}
	if _, err := os.Stat(filepath.Join(p.dir, filepath.FromSlash(testNamespace.stateDir))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a failed fetch made the namespace's state: %v", err)
	}
	if _, err := fetch(p.dir, t0, "nothing was read"); !errors.Is(err, ErrRefused) {
		t.Fatalf("a fetch of no namespace = %v; want a refusal", err)
	}
}

// TestThePushIsToTheFullRefItIsGiven: every family pushes one plain
// fast-forward to the full ref its namespace forms.
func TestThePushIsToTheFullRefItIsGiven(t *testing.T) {
	const sha = "9f2c41d07be35a6c80e1f4b2a7d39c559f2c41d0"
	got := pushTo(sha, testNamespace.ref("octo"))
	want := []string{"push", "--porcelain", "origin", sha + ":refs/heads/abcd-test/octo"}
	if !slices.Equal(got, want) || !slices.Equal(pushArgs(sha, "iss-5"), pushTo(sha, refName("iss-5"))) {
		t.Fatalf("pushTo = %q; want %q, and the reservation's push its special case", got, want)
	}
}
