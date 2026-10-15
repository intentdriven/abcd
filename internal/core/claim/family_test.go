package claim

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
	if !slices.Equal(got, want) {
		t.Fatalf("pushTo = %q; want %q", got, want)
	}
	reservation := []string{"push", "--porcelain", "origin", sha + ":refs/heads/abcd-reserved/iss-5"}
	if got := pushArgs(sha, "iss-5"); !slices.Equal(got, reservation) {
		t.Fatalf("pushArgs = %q; want the reservation's push %q", got, reservation)
	}
}

// announceFamily stands for a family other than the reservation's, with its
// own noun and remedy.
var announceFamily = &family{name: "announcement", keys: []string{"Abcd-Announce"}, shorten: "say less"}

// announceMessage is a message of announceFamily: its text as given.
type announceMessage string

func (m announceMessage) text() string         { return string(m) }
func (announceMessage) messageFamily() *family { return announceFamily }

// TestAnotherFamilysCommitIsRefusedInItsOwnWords: the empty-tree writer
// writes any family's message, and every refusal on the way names that
// family's message and commit, and its remedy at the cap.
func TestAnotherFamilysCommitIsRefusedInItsOwnWords(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	tree, err := emptyTree(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	who := identity{Name: "Pat Example", Email: "1234+octo@users.noreply.github.com"}
	text := "abcd announcement\n\nAbcd-Announce: hello\n"
	sha, err := writeEmptyCommit(r.Root(), who, t0, announceMessage(text), tree, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Git("log", "-1", "--format=%B", sha); got != strings.TrimSpace(text) {
		t.Fatalf("the commit's message = %q; want %q", got, text)
	}
	big := strings.Repeat("x", MaxMessageBytes+1)
	const capped = "the announcement message would be 4097 bytes, over the 4096-byte cap: say less; nothing was pushed"
	if err := checkText(r.Root(), announceFamily, big); err == nil || err.Error() != capped {
		t.Fatalf("checkText over the cap = %v; want %q", err, capped)
	}
	if _, err := writeEmptyCommit(r.Root(), who, t0, announceMessage(big), tree, ""); err == nil || err.Error() != capped {
		t.Fatalf("writeEmptyCommit over the cap = %v; want %q", err, capped)
	}
	leaky := text + "See https://claude.ai/code/session_01AbCdEfGhIjKlMnOpQrStUv for the run.\n" // abcd-lint:allow -- a fixture the outbound policy must refuse
	const outbound = "the announcement message carries harness:session_url, which public text may not carry; nothing was pushed"
	if _, err := writeEmptyCommit(r.Root(), who, t0, announceMessage(leaky), tree, ""); err == nil || err.Error() != outbound {
		t.Fatalf("writeEmptyCommit with a session URL = %v; want %q", err, outbound)
	}
	_, err = writeEmptyCommit(r.Root(), who, t0, announceMessage(text), "1111111111111111111111111111111111111111", "")
	if err == nil || !strings.HasPrefix(err.Error(), "git could not write the announcement commit: ") ||
		!strings.HasSuffix(err.Error(), "; nothing was pushed") {
		t.Fatalf("a commit git cannot write = %v; want it named as the announcement commit", err)
	}
}

// TestAnOversizedCommitOfAnotherFamilyIsUnreadable: a commit object past the
// bound is unreadable whatever family reads it, dated from the commit, and
// never reaches the family's parser.
func TestAnOversizedCommitOfAnotherFamilyIsUnreadable(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	tree := r.Git("hash-object", "-w", "-t", "tree", "/dev/null")
	sha := r.Git("commit-tree", tree, "-m", strings.Repeat("y", maxObjectBytes))
	written, err := strconv.ParseInt(r.Git("show", "-s", "--format=%ct", sha), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	parsed := false
	parse := func(string) (int, string) { parsed = true; return 7, "" }
	h, n, err := readCommit(r.Root(), sha, tree, t0, false, parse)
	if err != nil || h.Commit != sha || h.Unreadable != reasonObjectTooBig || n != 0 || parsed ||
		!h.WrittenAt.Equal(time.Unix(written, 0)) || h.DatedFromFetch {
		t.Fatalf("an oversized commit = %+v, %d, %v (parsed %v); want it unreadable, dated from the commit", h, n, err, parsed)
	}
}

// TestANamespaceAbsentOnTheRemoteIsFetchedAsEmpty: a namespace origin holds
// no branch of is no failure: the fetch stamps each namespace and tracks no
// branch of either.
func TestANamespaceAbsentOnTheRemoteIsFetchedAsEmpty(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if gap, err := fetch(p.dir, t0, "nothing was read", reservations, testNamespace); err != nil || gap != "" {
		t.Fatalf("fetch = %q, %v; want no failure", gap, err)
	}
	for _, ns := range []namespace{reservations, testNamespace} {
		if got := lastFetched(p.dir, ns); !got.Equal(t0) {
			t.Errorf("%s stamped %v; want %v", ns.noun, got, t0)
		}
		if refs := p.git("for-each-ref", "--format=%(refname)", ns.tracking); refs != "" {
			t.Errorf("%s tracks %q; want nothing", ns.noun, refs)
		}
	}
}

// TestAStampThatCannotBeWrittenKeepsTheStampsBeforeIt: when the second
// namespace's stamp cannot be written, the fetch is refused in its words,
// both namespaces' branches are as fetched, the first keeps the stamp it was
// given (it was fetched then), and the second keeps none.
func TestAStampThatCannotBeWrittenKeepsTheStampsBeforeIt(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("5678+cat@users.noreply.github.com", "")
	if _, err := q.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	w.seed.Git("push", "-q", "origin", "main:refs/heads/abcd-test/one")
	// A file where the namespace's directory would go: its stamp cannot be made.
	if err := os.WriteFile(filepath.Join(p.dir, ".abcd", ".work.local", "test"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t1 := t0.Add(time.Hour)
	_, err := fetch(p.dir, t1, "nothing was read", reservations, testNamespace)
	if !errors.Is(err, ErrRefused) || !strings.HasPrefix(err.Error(), "fetched the test branches, but could not record when: ") ||
		!strings.HasSuffix(err.Error(), "; nothing was read") {
		t.Fatalf("fetch = %v; want the second stamp's refusal", err)
	}
	if got := lastFetched(p.dir, reservations); !got.Equal(t1) {
		t.Errorf("the reservations stamped %v; want %v, the fetch that read them", got, t1)
	}
	if got := lastFetched(p.dir, testNamespace); !got.IsZero() {
		t.Errorf("the test branches stamped %v; want no stamp", got)
	}
	if p.git("for-each-ref", "--format=%(refname)", testNamespace.tracking) != testNamespace.track("one") ||
		p.git("for-each-ref", "--format=%(refname)", TrackingPrefix) != TrackingPrefix+"iss-5" {
		t.Fatalf("the fetch did not land both namespaces' branches")
	}
}
