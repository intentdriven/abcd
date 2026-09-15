package history

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

const testRootSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

// setupStore points HOME at a temp dir (so both the store root and the probed
// identity home path resolve there) and opens the store there, returning
// (repoRoot, home). Opening is what creates it: there is no install step to
// stand in for (iss-95).
// TestCaptureAcceptsSHA256RootKey proves history verbs work for a repo in git's
// SHA-256 object format, whose root-commit SHA is 64 hex chars. The old
// `^[0-9a-f]{40}$` key rejected it, so Capture/List/Read all failed for such a
// repo — the ahoy layer derives the 64-char SHA but history refused it.
func TestCaptureAcceptsSHA256RootKey(t *testing.T) {
	sha256Root := strings.Repeat("b", 64)
	home := t.TempDir()
	t.Setenv("HOME", home)
	tdir := filepath.Join(home, ".abcd", "transcripts", sha256Root, "records")
	if err := os.MkdirAll(tdir, 0o755); err != nil {
		t.Fatal(err)
	}
	repoRoot := t.TempDir()

	res, err := Capture(repoRoot, sha256Root, "sess-sha256", []byte("assistant: hi\n"), "native")
	if err != nil {
		t.Fatalf("Capture with a SHA-256 root key failed: %v", err)
	}
	if !res.Wrote {
		t.Fatal("expected Wrote=true for a SHA-256-keyed capture")
	}
	if _, err := List(repoRoot, sha256Root); err != nil {
		t.Fatalf("List with a SHA-256 root key failed: %v", err)
	}
}

func setupStore(t *testing.T) (string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoRoot := t.TempDir()
	if _, err := Resolve(repoRoot, testRootSHA); err != nil {
		t.Fatal(err)
	}
	return repoRoot, home
}

// TestCaptureRedactsSecretsAndHomePaths is the load-bearing guarantee: a stored
// transcript must NEVER contain a live secret or the caller's absolute home
// path. It plants a ghp_-style PAT and the caller's own home path, captures,
// then asserts the on-disk record carries neither. (In production HOME is a
// /Users/<user> path, so this is exactly the "absolute /Users path redacted"
// case; TestRedactionEngineStripsUsersHomePath proves the literal /Users form
// through the same scanner engine Capture uses.)
func TestCaptureRedactsSecretsAndHomePaths(t *testing.T) {
	repoRoot, home := setupStore(t)

	pat := "ghp_" + strings.Repeat("a", 40)
	selfPath := home + "/private/journal.md"

	transcript := strings.Join([]string{
		"user: deploy with token " + pat,
		"assistant: reading " + selfPath,
		"assistant: done",
	}, "\n")

	res, err := Capture(repoRoot, testRootSHA, "sess-abc123", []byte(transcript), "native")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if !res.Wrote {
		t.Fatalf("expected Wrote=true on first capture")
	}

	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}

	// The literal sensitive values must be gone from the whole record file.
	mustBeRedacted := []struct {
		name  string
		value string
	}{
		{"github PAT", pat},
		{"self home path", home},
	}
	for _, c := range mustBeRedacted {
		if bytes.Contains(onDisk, []byte(c.value)) {
			t.Errorf("%s leaked into the stored record: %q found on disk", c.name, c.value)
		}
	}

	// The redaction actually happened and was audited.
	if res.Record.Secrets < 1 {
		t.Errorf("expected >=1 secret redaction counted, got %d", res.Record.Secrets)
	}
	if res.Record.HomePaths < 1 {
		t.Errorf("expected >=1 home-path redaction counted, got %d", res.Record.HomePaths)
	}

	// The body is still present and readable, just sanitised.
	if !bytes.Contains(onDisk, []byte("deploy with token")) {
		t.Errorf("non-secret body content was lost")
	}
	if !bytes.Contains(onDisk, []byte("~/private/journal.md")) {
		t.Errorf("self home path should collapse to ~/…, not vanish; got:\n%s", onDisk)
	}
}

// TestCaptureRedactsPercentEncodedSecrets is the gh-370 regression: a live token
// embedded in a percent-encoded URL (an OAuth/redirect/magic-link) has a %XX hex
// word-char before it, so the leading \b every bundled token pattern anchors on
// never holds and the raw scan misses it. Before the percent-decode pre-pass the
// live PAT/AKIA/JWT was written verbatim into the committed transcript. The tokens
// here are FAKE shapes (structurally valid, never real credentials).
func TestCaptureRedactsPercentEncodedSecrets(t *testing.T) {
	repoRoot, _ := setupStore(t)

	ghp := "ghp_" + "0123456789abcdefABCDEF0123456789abcd" // ghp_ + 36 alnum
	akia := "AKIA" + "ABCDEFGHIJKLMNOP"                    // AKIA + 16 [0-9A-Z]
	jwt := "eyJ" + "hbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ" + "zdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV0abcdefgh"

	transcript := strings.Join([]string{
		"user: GET /cb?redirect=%2Fauth%3Ftoken%3D" + ghp + "&s=x",
		"assistant: curl 'https://api?key%3D" + akia + "'",
		"assistant: magic link%3Ftoken%3D" + jwt + " sent",
		"assistant: done",
	}, "\n")

	res, err := Capture(repoRoot, testRootSHA, "sess-pct370", []byte(transcript), "native")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if !res.Wrote {
		t.Fatalf("expected Wrote=true on first capture")
	}
	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	for _, tok := range []struct{ name, value string }{
		{"github PAT", ghp}, {"AWS key", akia}, {"JWT", jwt},
	} {
		if bytes.Contains(onDisk, []byte(tok.value)) {
			t.Errorf("%s leaked into the stored record behind a percent-encoded delimiter:\n%s", tok.name, onDisk)
		}
	}
	if res.Record.Secrets < 3 {
		t.Errorf("expected >=3 secret redactions counted, got %d", res.Record.Secrets)
	}
}

// TestCaptureRedactsHomePathFollowedByPunctuation is the Finding-A regression:
// a home path followed by a non-boundary punctuation char (#, &, =, @) must NOT
// survive in a stored record. Before the fix the scanner's trailing-boundary
// heuristic DROPPED such spans and the store's stage-two re-scan (same detector)
// missed them too, leaving the absolute home path and the local username
// verbatim on disk. It uses a distinctive home basename so the username assertion
// cannot alias a timestamp/schema digit.
func TestCaptureRedactsHomePathFollowedByPunctuation(t *testing.T) {
	base := t.TempDir()
	user := "zzhomeuser42"
	home := filepath.Join(base, user)
	t.Setenv("HOME", home)
	tdir := filepath.Join(home, ".abcd", "transcripts", testRootSHA, "records")
	if err := os.MkdirAll(tdir, 0o755); err != nil {
		t.Fatal(err)
	}
	repoRoot := t.TempDir()

	cases := []struct {
		name string
		line string
	}{
		{"hash-after-self-home", "cd " + home + "#draft"},
		{"amp-after-self-home", "run --root=" + home + "&flag=1"},
		{"at-after-self-home", "scp " + home + "@backup"},
		{"eq-after-users-user", "see /Users/" + user + "=cfg"},
		{"dot-after-self-home", "saved under " + home + "."},
		{"file-url-self-home", "open file://" + home + "/x"},
		{"underscore-after-self-home", "notes in " + home + "_2/x"},
	}
	var lines []string
	for _, c := range cases {
		lines = append(lines, c.line)
	}
	transcript := strings.Join(lines, "\n") + "\n"

	res, err := Capture(repoRoot, testRootSHA, "sess-punct", []byte(transcript), "native")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if !res.Wrote {
		t.Fatalf("expected Wrote=true on first capture")
	}
	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}

	if bytes.Contains(onDisk, []byte(home)) {
		t.Errorf("absolute self home path survived on disk:\n%s", onDisk)
	}
	if bytes.Contains(onDisk, []byte("/Users/"+user)) {
		t.Errorf("/Users/<user> home path survived on disk:\n%s", onDisk)
	}
	if bytes.Contains(onDisk, []byte(user)) {
		t.Errorf("local username %q survived on disk:\n%s", user, onDisk)
	}
	// The counters must reflect that redaction actually occurred.
	if res.Record.HomePaths < 1 {
		t.Errorf("expected >=1 home-path redaction counted, got %d", res.Record.HomePaths)
	}
	// Non-secret body content is preserved.
	if !bytes.Contains(onDisk, []byte("draft")) || !bytes.Contains(onDisk, []byte("cfg")) {
		t.Errorf("non-secret body content was lost:\n%s", onDisk)
	}
}

// TestSurvivingCallerHomeBackstop proves the deterministic, scanner-independent
// backstop (Finding A part 2): it rewrites a "/Users/<user>" or "/home/<user>"
// segment for the caller's own username regardless of the trailing character,
// leaves a longer, different username alone, and reports only the literal
// home standing as a path — the one shape it cannot rewrite. This is the guard
// that stands even if the scanner heuristic ever regresses.
func TestSurvivingCallerHomeBackstop(t *testing.T) {
	home := "/base/zzhomeuser42"
	cases := []struct {
		name      string
		text      string
		rewritten bool // the username segment is gone from the output
		refused   bool // a finding is reported
	}{
		{"clean-redacted", "wrote ~/notes and /Users/[redacted-user]/x", false, false},
		{"literal-home-survives", "path " + home + "/x", false, true},
		{"users-user-hash", "root=/Users/zzhomeuser42#frag", true, false},                   // abcd-audit:allow
		{"home-user-amp", "root=/home/zzhomeuser42&x", true, false},                         // abcd-audit:allow
		{"users-user-eq", "cfg=/Users/zzhomeuser42=v", true, false},                         // abcd-audit:allow
		{"behind-url-host", "see https://ci.example.com/Users/zzhomeuser42/x", true, false}, // abcd-audit:allow
		{"different-longer-user", "path /Users/zzhomeuser42x/y", false, false},              // abcd-audit:allow
		{"different-user", "path /Users/someoneelse/y", false, false},                       // abcd-audit:allow
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, resid := scanner.SurvivingCallerHome(c.text, home)
			if got := len(resid) > 0; got != c.refused {
				t.Errorf("SurvivingCallerHome(%q) refused = %v, want %v", c.text, got, c.refused)
			}
			if c.rewritten {
				if strings.Contains(out, "zzhomeuser42") {
					t.Errorf("SurvivingCallerHome(%q) left the username in place: %q", c.text, out)
				}
			} else if out != c.text {
				t.Errorf("SurvivingCallerHome(%q) rewrote text it should leave alone: %q", c.text, out)
			}
		})
	}
}

// TestRedactionEngineStripsUsersHomePath proves the shared scanner engine that
// Capture drives (scanner.ScanText + scanner.Redact) strips a literal
// /Users/<user> absolute home path — the redaction Capture applies to every
// transcript before it lands on disk.
func TestRedactionEngineStripsUsersHomePath(t *testing.T) {
	id := scanner.Identity{HomePath: "/Users/alice", HomeUser: "alice"} // abcd-audit:allow
	line := "assistant: wrote /Users/alice/.aws/credentials"            // abcd-audit:allow

	findings := scanner.ScanText(line, id, scanner.DefaultPatterns(), scanner.DefaultIdentitySeverities(), "transcript")
	redacted, n := scanner.Redact(line, findings)

	if n < 1 {
		t.Fatalf("expected at least one rewrite, got %d (findings: %+v)", n, findings)
	}
	if strings.Contains(redacted, "/Users/alice") { // abcd-audit:allow
		t.Errorf("literal /Users home path survived redaction: %q", redacted)
	}
	if !strings.Contains(redacted, "~/.aws/credentials") {
		t.Errorf("home path should collapse to ~/…; got %q", redacted)
	}
}

// TestCaptureIdempotentOnSourceSHA proves a re-capture of identical source is a
// no-op: no second file, mtime preserved, Wrote=false, same path returned.
func TestCaptureIdempotentOnSourceSHA(t *testing.T) {
	repoRoot, _ := setupStore(t)
	raw := []byte("user: hello\nassistant: hi\n")

	first, err := Capture(repoRoot, testRootSHA, "sess-idem", raw, "native")
	if err != nil {
		t.Fatalf("first capture: %v", err)
	}
	if !first.Wrote {
		t.Fatalf("first capture should write")
	}
	fi1, err := os.Stat(first.Record.Path)
	if err != nil {
		t.Fatal(err)
	}

	second, err := Capture(repoRoot, testRootSHA, "sess-idem", raw, "native")
	if err != nil {
		t.Fatalf("second capture: %v", err)
	}
	if second.Wrote {
		t.Errorf("identical source should be an idempotent no-op")
	}
	if second.Record.Path != first.Record.Path {
		t.Errorf("idempotent capture should return the existing record path")
	}
	fi2, err := os.Stat(first.Record.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !fi1.ModTime().Equal(fi2.ModTime()) {
		t.Errorf("mtime changed on idempotent no-op: %v -> %v", fi1.ModTime(), fi2.ModTime())
	}

	recs, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Errorf("expected exactly 1 record after idempotent re-capture, got %d", len(recs))
	}
}

// TestCaptureIdenticalSourceDistinctSessionsWritesBoth proves the idempotency
// no-op is scoped to the session: a second, distinct session that produces
// byte-identical source must get its own record, not be silently attributed to
// the first session's record.
func TestCaptureIdenticalSourceDistinctSessionsWritesBoth(t *testing.T) {
	repoRoot, _ := setupStore(t)
	raw := []byte("user: hello\nassistant: hi\n")

	first, err := Capture(repoRoot, testRootSHA, "sess-a", raw, "native")
	if err != nil {
		t.Fatalf("first capture: %v", err)
	}
	if !first.Wrote {
		t.Fatalf("first capture should write")
	}

	second, err := Capture(repoRoot, testRootSHA, "sess-b", raw, "native")
	if err != nil {
		t.Fatalf("second capture: %v", err)
	}
	if !second.Wrote {
		t.Fatalf("a distinct session with identical source must write its own record, not no-op")
	}
	if second.Record.SessionID != "sess-b" {
		t.Fatalf("second record attributed to %q, want sess-b", second.Record.SessionID)
	}
	if second.Record.Path == first.Record.Path {
		t.Fatalf("distinct sessions must not share a record path")
	}

	recs, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records for two distinct sessions, got %d", len(recs))
	}
	if _, _, err := Read(repoRoot, testRootSHA, "sess-b"); err != nil {
		t.Fatalf("second session must be retrievable after capture: %v", err)
	}
}

// TestListAndRead exercises the read side: metadata newest-first, and a body
// fetch that returns the sanitised content without frontmatter.
func TestListAndRead(t *testing.T) {
	repoRoot, _ := setupStore(t)

	if _, err := Capture(repoRoot, testRootSHA, "sess-one", []byte("first session\n"), "native"); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(repoRoot, testRootSHA, "sess-two", []byte("second session\n"), "native"); err != nil {
		t.Fatal(err)
	}

	recs, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}
	// Newest first: sess-two was captured last.
	if recs[0].SessionID != "sess-two" {
		t.Errorf("expected newest (sess-two) first, got %q", recs[0].SessionID)
	}

	rec, body, err := Read(repoRoot, testRootSHA, "sess-one")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if rec.SessionID != "sess-one" {
		t.Errorf("Read returned wrong record: %q", rec.SessionID)
	}
	if strings.Contains(string(body), "---") {
		t.Errorf("Read body should not include frontmatter fence")
	}
	if !strings.Contains(string(body), "first session") {
		t.Errorf("Read body missing content: %q", body)
	}
}

// TestListSkipsSymlinkedRecord is iss-383: the store's read path must not
// follow a planted symlink — the write path already refuses symlinks at every
// parent (ensureRealDir, on every resolve) and the leaf (WriteFileAtomic), so a
// *.md symlink in records/ is never store-authored and reads out-of-store bytes.
func TestListSkipsSymlinkedRecord(t *testing.T) {
	repoRoot, home := setupStore(t)
	tdir := filepath.Join(home, ".abcd", "transcripts", testRootSHA, "records")
	if _, err := Capture(repoRoot, testRootSHA, "sess-real", []byte("real one\n"), "native"); err != nil {
		t.Fatal(err)
	}
	recs, err := List(repoRoot, testRootSHA)
	if err != nil || len(recs) != 1 {
		t.Fatalf("want 1 real record, got %d (err %v)", len(recs), err)
	}
	// Move the real record out of the store and symlink it back in.
	outside := filepath.Join(home, "outside.md")
	if err := os.Rename(recs[0].Path, outside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(tdir, filepath.Base(recs[0].Path))); err != nil {
		t.Fatal(err)
	}
	recs, err = List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatalf("List over a store holding a symlink errored: %v", err)
	}
	if len(recs) != 0 {
		t.Errorf("List followed a planted symlink out of the store: %+v", recs)
	}
}

// TestListAbsentCorpusIsCleanEmpty distinguishes an un-populated store (no
// records, no error) from a malformed one, mirroring history_store's LoadResult.
func TestListAbsentCorpusIsCleanEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoRoot := t.TempDir()
	// Nothing captured yet: the store resolves (and creates itself) empty.
	recs, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatalf("absent corpus should be clean-empty, got error: %v", err)
	}
	if len(recs) != 0 {
		t.Errorf("expected 0 records for absent corpus, got %d", len(recs))
	}
}

// TestCaptureRejectsBadInput validates the external-input boundary.
func TestCaptureRejectsBadInput(t *testing.T) {
	repoRoot, _ := setupStore(t)
	cases := []struct {
		name      string
		rootSHA   string
		sessionID string
		kind      string
	}{
		{"short sha", "abc", "sess", "native"},
		{"uppercase sha", strings.ToUpper(testRootSHA), "sess", "native"},
		{"empty session", testRootSHA, "", "native"},
		{"path-traversal session", testRootSHA, "../evil", "native"},
		{"bad kind", testRootSHA, "sess", "bogus"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Capture(repoRoot, c.rootSHA, c.sessionID, []byte("x\n"), c.kind); err == nil {
				t.Errorf("expected rejection for %s", c.name)
			}
		})
	}
}

// TestCaptureRedactsNetworkIdentifiers is the iss-125 guarantee at the store's
// own boundary: a transcript carrying a LAN hostname, a private address and a
// device name must not reach disk with any of them intact. Specimens are
// assembled at runtime so no non-reserved identifier is committed to this tree.
func TestCaptureRedactsNetworkIdentifiers(t *testing.T) {
	repoRoot, _ := setupStore(t)

	lanHost := strings.Join([]string{"printer", "local"}, ".")
	fritzHost := strings.Join([]string{"nas", "fritz", "box"}, ".")
	addr := strings.Join([]string{"100", "64", "3", "9"}, ".")
	device := strings.Join([]string{"zeta", "laptop"}, "-")

	transcript := strings.Join([]string{
		"user: ssh " + lanHost + " then " + fritzHost,
		"assistant: peer is " + addr,
		"assistant: synced from " + device,
	}, "\n")

	res, err := Capture(repoRoot, testRootSHA, "sess-net001", []byte(transcript), "native")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	for _, v := range []string{lanHost, fritzHost, addr, device} {
		if bytes.Contains(onDisk, []byte(v)) {
			t.Errorf("network identifier %q leaked into the stored record", v)
		}
	}
	if !bytes.Contains(onDisk, []byte("synced from")) {
		t.Errorf("non-identifier body content was lost")
	}
}

// F7: the stage-two write gate keyed on hard_fail alone, so a hostname — warn by
// design, because the hostname patterns are shape heuristics — that survived
// Stage-1 redaction was written to disk silently. Any surviving identity or
// network span must refuse the write, whatever its severity.
func TestStageTwoGateBlocksSurvivingWarnIdentifier(t *testing.T) {
	lanHost := strings.Join([]string{"printer", "local"}, ".")
	findings := scanner.ScanText("ssh "+lanHost, scanner.Identity{},
		scanner.DefaultPatterns(), scanner.DefaultIdentitySeverities(), "transcript")

	var hostname *scanner.Finding
	for i := range findings {
		if findings[i].Kind == "net:lan_hostname" {
			hostname = &findings[i]
		}
	}
	if hostname == nil {
		t.Fatalf("fixture must produce a LAN hostname finding: %+v", findings)
	}
	if hostname.Severity != scanner.SeverityWarn {
		t.Fatalf("fixture must be warn-severity to exercise the gate, got %q", hostname.Severity)
	}
	if len(scanner.BlockingResidual(findings)) == 0 {
		t.Errorf("a surviving warn-severity hostname must refuse the write: %+v", findings)
	}
	// A clean rescan still writes.
	if got := scanner.BlockingResidual(nil); len(got) != 0 {
		t.Errorf("a clean rescan must not block: %+v", got)
	}
}

// G4: the refusal the caller reads must describe the gate that fired. It named
// a hard_fail severity while blocking on ANY surviving identity or network span,
// so a reader chasing a warn-severity hostname was sent looking for a severity
// no finding carried.
func TestResidualErrorNamesNoSeverityItDoesNotGateOn(t *testing.T) {
	err := &RedactionResidualError{Residual: []scanner.Finding{
		{Kind: "net:lan_hostname", Severity: scanner.SeverityWarn},
	}}
	msg := err.Error()
	if strings.Contains(msg, string(scanner.SeverityHardFail)) {
		t.Errorf("refusal names a severity the gate does not key on: %s", msg)
	}
	if !strings.Contains(msg, "net:lan_hostname") || !strings.Contains(msg, "refusing to write") {
		t.Errorf("refusal must name the surviving kind and refuse the write: %s", msg)
	}
}

// storeEntropySpecimen assembles a 40-character alphanumeric value of genuinely
// high Shannon entropy — a deterministic shuffle of the alnum alphabet taken
// without replacement, so all 40 characters differ and the per-character entropy
// is log2(40) ≈ 5.32 bits. Deterministic across runs and machines (a hand-rolled
// LCG over a fixed seed, no dependence on math/rand's stream), and computed at
// run time, so no credential-shaped literal enters this repo's tree or history.
//
// It matters because the other unanchored specimens REPEAT: FAKEfake00×4,
// deadbeef×4 and the base64 run measure 3.12, 2.16 and 2.75 bits per character,
// all below the ~3.5-bit floor an entropy detector conventionally uses. Without
// this specimen, an entropy threshold could land on the transcript path while
// every assertion here still passed.
//
// CROSS-REFERENCE: internal/adapter/scanner/transcript_coverage_test.go carries
// the same generator (as entropySpecimen) and the same specimen shapes at the
// PATTERN-SET boundary, plus the entropy self-check that keeps the value honest.
// Separate packages, so the set is duplicated rather than shared — change one
// and change the other, or the ledger entry citing both stops describing the
// same residue.
func storeEntropySpecimen() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	a := []byte(alphabet)
	var s uint64 = 0x5EED17
	for i := len(a) - 1; i > 0; i-- {
		s = s*6364136223846793005 + 1442695040888963407
		j := int(s % uint64(i+1))
		a[i], a[j] = a[j], a[i]
	}
	return string(a[:40])
}

// storeEntropySpecimenGolden is the exact value storeEntropySpecimen returns, and
// is asserted so the two duplicated generators cannot silently diverge: the
// scanner package pins the identical value as entropySpecimenGolden, and either
// copy drifting now fails its OWN side loudly, with a message naming the other,
// rather than leaving the two pins describing different residues. It also pins
// determinism, which the entropy floor cannot — a changed stream would still
// measure ~5.32 bits per character.
//
// It is the one credential-SHAPED literal admitted here, on measurement rather
// than assumption: a bare 40-character alphanumeric run with no key name and no
// `=`/`:` delimiter beside it does not trip gitleaks' generic-api-key rule
// (keyword + delimiter + entropy >= 3.5), verified against the pinned 8.24.3
// default rules over this exact declaration. That clean scan hangs on the
// keyword-free name: renaming this constant to embed a keyword (a
// ...Key/...Token/...Secret suffix) or moving the value beside a delimited key
// name would trip the rule and red the CI secret scan. It is a shuffle of the
// alphabet above; it is not, and never was, a credential.
const storeEntropySpecimenGolden = "fXbLtohSxAn9d1jv75E0eDkB6JHT8OzNWcVCQgsU"

// shannonEntropy returns the per-character Shannon entropy of s in bits.
func shannonEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	counts := map[rune]int{}
	n := 0
	for _, r := range s {
		counts[r]++
		n++
	}
	h := 0.0
	for _, c := range counts {
		p := float64(c) / float64(n)
		h -= p * math.Log2(p)
	}
	return h
}

// TestCaptureStoresUnanchoredEntropyVerbatim is the iss-96 residue pinned at the
// boundary the ledger entry actually describes: an automatically captured
// transcript reaching disk. A value with no recognisable prefix — a bare
// secret-key shape, a bare passphrase, a prefix-less token, and a genuinely
// high-entropy 40-character run — is not detected by the token patterns, so
// Capture writes each one through unchanged on the lines BELOW the anchored
// token, which is itself masked to a fingerprint on the transcript's first line.
//
// The same capture carries the CONTRA case: a flagged network identifier on the
// last line, which A2's set does reach, and which must therefore be REDACTED in
// the stored record. Without it the test would show only what survives, and the
// class iss-96 no longer covers would rest on prose alone.
//
// The anchored control is asserted twice over: absent in raw form AND present as
// its mask. Absence alone is satisfied by a store that dropped the line, or wrote
// nothing at all — only the fingerprint proves the word "masked".
//
// It asserts CURRENT behaviour, not desired behaviour, and its reach is the
// PATTERN SET: when an entropy or key-name detector lands in DefaultPatterns the
// assertion inverts, and that failure is the signal to re-point the pin (and the
// ledger entry that cites it). It cannot see an opt-in external-scanner adapter
// run over this path — that route never consults DefaultPatterns, so this pin
// would stay green while coverage had in fact grown, and closing iss-96 on the
// pins alone would be wrong. The counterpart unit pin is
// TestTranscriptPathMissesUnanchoredEntropy in internal/adapter/scanner.
//
// Every specimen is synthetic and self-evidently fake; the passphrase, the
// high-entropy run and the network identifier are all assembled at run time, so
// no credential-shaped or non-reserved literal sits in this repo's history.
func TestCaptureStoresUnanchoredEntropyVerbatim(t *testing.T) {
	repoRoot, _ := setupStore(t)

	anchored := "ghp_" + strings.Repeat("F", 36)
	// On the Capture path the stored fingerprint is written by the scanner's
	// sealLine -> fingerprintSpan (documented there as the byte-level mirror of
	// maskSecret), not by maskSecret itself: it keeps the first three and last two
	// runes and stars the middle, so the stored fingerprint begins "ghp" followed
	// by asterisks. Only the stable head is asserted: the exact star count is the
	// token's length, which this test does not exist to pin.
	maskHead := "ghp" + strings.Repeat("*", 8)
	secretKeyShape := strings.Repeat("FAKEfake00", 4) // 40 chars, no prefix
	passphrase := strings.Join([]string{"correct", "horse", "battery", "staple", "9x"}, "-")
	hexToken := strings.Repeat("deadbeef", 4)
	highEntropy := storeEntropySpecimen()
	// The duplicated generator is guarded here too: a drifting copy that quietly
	// produced a repetitive value would restore the silent-pass gap it closes.
	if h := shannonEntropy(highEntropy); h < 4.5 {
		t.Fatalf("high-entropy specimen measures %.2f bits/char, below the 4.5 floor — it can no longer alarm an entropy detector", h)
	}
	// Entropy alone cannot see the two copies diverging: a changed stream would
	// still measure ~5.32 bits. The golden value can, and it pins determinism at
	// the same time. Never printed — a diagnostic here names the case, not the
	// value.
	if highEntropy != storeEntropySpecimenGolden {
		t.Fatalf("high-entropy specimen no longer equals storeEntropySpecimenGolden — the generator's stream has changed; internal/adapter/scanner pins the same value as entropySpecimenGolden, so update BOTH copies or the two pins stop describing the same residue")
	}
	// A private quad: flagged by the network set (allowlist inversion over the
	// reserved documentation ranges), assembled from its octets so no
	// non-reserved identifier is a literal here.
	lanAddr := strings.Join([]string{"10", "0", "0", "7"}, ".")

	transcript := strings.Join([]string{
		"user: here is the token " + anchored,
		"assistant: AWS_SECRET_ACCESS_KEY=" + secretKeyShape,
		"assistant: password: " + passphrase,
		"assistant: api_key = " + hexToken,
		"assistant: session_token " + highEntropy,
		"assistant: ssh " + lanAddr,
	}, "\n")

	res, err := Capture(repoRoot, testRootSHA, "sess-entropy1", []byte(transcript), "native")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if !res.Wrote {
		t.Fatal("expected Wrote=true: an idempotent no-op would leave every assertion below reading a record this call did not write")
	}
	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	if bytes.Contains(onDisk, []byte(anchored)) {
		t.Errorf("anchored control: the token must not survive in the stored record")
	}
	if !bytes.Contains(onDisk, []byte(maskHead)) {
		t.Errorf("anchored control: the token must be MASKED, not merely absent — no %q fingerprint in the stored record", maskHead)
	}
	// The class A2 DID close, asserted at the store rather than described: the
	// identifier is gone and its placeholder is there, so the line survived and
	// the value did not.
	if bytes.Contains(onDisk, []byte(lanAddr)) {
		t.Errorf("network control: the flagged address must not survive in the stored record")
	}
	if !bytes.Contains(onDisk, []byte("[redacted-address]")) {
		t.Errorf("network control: the address must be REDACTED, not merely absent — no [redacted-address] placeholder in the stored record")
	}
	unanchored := []struct {
		name     string
		specimen string
	}{
		{"bare 40-char secret-key shape", secretKeyShape},
		{"bare passphrase after a password: key name", passphrase},
		{"prefix-less hex token after an api_key = key name", hexToken},
		{"high-entropy 40-char token", highEntropy},
	}
	for _, c := range unanchored {
		if !bytes.Contains(onDisk, []byte(c.specimen)) {
			// Names the case, never the specimen: a diagnostic must not echo a
			// credential-shaped value into CI output.
			t.Errorf("%s: no longer stored verbatim — coverage has changed; re-point this pin and iss-96", c.name)
		}
	}
}

// TestCaptureRejectsBadRootSHAMessageNamesBothWidths pins the C3 fix: the
// diagnostic for an invalid rootSHA names both accepted widths (40 and 64), so it
// cannot drift from rootSHARe, which accepts SHA-256's 64 as well as SHA-1's 40.
func TestCaptureRejectsBadRootSHAMessageNamesBothWidths(t *testing.T) {
	_, err := Capture(t.TempDir(), "not-a-sha", "sess-x", []byte("assistant: hi\n"), "native")
	if err == nil {
		t.Fatal("expected an invalid rootSHA to be rejected")
	}
	if !strings.Contains(err.Error(), "64") || !strings.Contains(err.Error(), "40") {
		t.Errorf("rootSHA error must name both 40- and 64-char widths, got %q", err.Error())
	}
	if _, err := List(t.TempDir(), "also-not-a-sha"); err == nil ||
		!strings.Contains(err.Error(), "64") || !strings.Contains(err.Error(), "40") {
		t.Errorf("List rootSHA error must name both widths, got %v", err)
	}
}
