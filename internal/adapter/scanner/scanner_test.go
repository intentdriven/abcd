package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func writeFile(t *testing.T, root, rel, content string) string {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return abs
}

func hasKind(f []Finding, kind string) bool {
	for _, fn := range f {
		if fn.Kind == kind {
			return true
		}
	}
	return false
}

// scanLine is a small helper: scan one line with the default patterns and no
// identity.
func scanLine(line string) []Finding {
	return ScanText(line, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "f")
}

// TestSecretPatterns is a table test over one positive + one negative per §2.2
// pattern.
func TestSecretPatterns(t *testing.T) {
	r := strings.Repeat
	cases := []struct {
		kind string
		pos  string
		neg  string
	}{
		{"token:github_pat", "ghp_" + r("a", 36), "ghp_short"},
		{"token:github_server", "ghs_" + r("b", 36), "ghs_short"},
		{"token:github_oauth", "gho_" + r("c", 36), "gho_short"},
		{"token:github_user", "ghu_" + r("d", 36), "ghu_short"},
		{"token:github_refresh", "ghr_" + r("e", 36), "ghr_short"},
		{"token:github_pat_finegrained", "github_pat_" + r("A", 22) + "_" + r("B", 59), "github_pat_short"},
		{"token:pem_private_key", "-----BEGIN RSA PRIVATE KEY-----", "----- not a key -----"},
		{"token:anthropic", "sk-ant-" + r("A", 40), "sk-ant-short"},
		{"token:openai_project", "sk-proj-" + r("B", 40), "sk-proj-short"},
		{"token:openai_svcacct", "sk-svcacct-" + r("C", 40), "sk-svcacct-short"},
		{"token:aws_access_key", "AKIA" + r("A", 16), "AKIA-not-a-key"},
		{"token:slack", "xoxb-" + r("1", 12), "xoxb-x"},
		{"token:google_api", "AIza" + r("Z", 35), "AIza-too-short"},
		{"token:stripe_live", "sk_live_" + r("9", 24), "sk_live_short"},
		{"token:stripe_test", "sk_test_" + r("8", 24), "sk_test_short"},
		{"token:jwt_shaped", "eyJ" + r("a", 12) + "." + r("b", 12) + "." + r("c", 12), "eyJ-not-jwt"},
		{"rp_session_key", `"sessionKey": "real-uuid-value-here"`, `"sessionKey": ""`},
	}
	for _, c := range cases {
		t.Run(c.kind, func(t *testing.T) {
			if got := scanLine(c.pos); !hasKind(got, c.kind) {
				t.Errorf("positive %q did not flag %s: %+v", c.pos, c.kind, got)
			}
			if got := scanLine(c.neg); hasKind(got, c.kind) {
				t.Errorf("negative %q wrongly flagged %s", c.neg, c.kind)
			}
		})
	}
}

// TestGoogleAPIKeyTrailingDashDetected proves a Google API key whose 35th
// character is '-' (a valid key char) is still flagged. The fixed-length {35}
// class plus a trailing ASCII \b had no shorter match to satisfy the boundary, so
// such a key silently slipped the hard_fail secret gate.
func TestGoogleAPIKeyTrailingDashDetected(t *testing.T) {
	key := "AIza" + strings.Repeat("Z", 34) + "-" // 35th body char is '-'
	if got := scanLine("api key " + key + " done"); !hasKind(got, "token:google_api") {
		t.Errorf("Google API key ending in '-' not flagged: key=%q findings=%+v", key, got)
	}
}

// TestIdentityHomeSelfCaseInsensitive proves a case variant of the caller's own
// home path still trips the hard_fail home_path_self gate — on a case-folding
// filesystem it is the SAME directory, and the sibling email/name/github matchers
// already fold case.
func TestIdentityHomeSelfCaseInsensitive(t *testing.T) {
	id := Identity{HomePath: "/Users/Alice", HomeUser: "Alice"} // abcd-audit:allow
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	if got := ScanText("wrote /USERS/ALICE/.aws/credentials", id, pats, sev, "f"); !hasKind(got, kindHomeSelf) { // abcd-audit:allow
		t.Errorf("case-variant home_path_self not flagged: %+v", got)
	}
}

// TestRedactSealsOverlappingSecrets is the disk-redactor analogue of the
// serializer overlap test: two partially-overlapping secret spans (an sk-ant key
// running into a JWT) must not leave any raw token bytes behind, and a re-scan of
// the redacted text must find no surviving hard_fail (the store's fail-closed
// guarantee). The old substring-replace Redact left the JWT's tail verbatim.
func TestRedactSealsOverlappingSecrets(t *testing.T) {
	line := "sk-ant-" + strings.Repeat("X", 34) + "-eyJ" + "ABCDEFGHIJ.KLMNOPQRST.UVWXYZ0123456"
	findings := ScanText(line, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "f")
	redacted, _ := Redact(line, findings)
	for _, raw := range []string{"KLMNOPQRST", "UVWXYZ0123456"} {
		if strings.Contains(redacted, raw) {
			t.Errorf("raw secret substring %q survived redaction: %q", raw, redacted)
		}
	}
	rescan := ScanText(redacted, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "f")
	for _, f := range rescan {
		if f.Severity == SeverityHardFail {
			t.Errorf("hard_fail survived redaction: %+v (out=%q)", f, redacted)
		}
	}
}

// TestRE2LookaroundPorts proves each ported lookaround predicate.
func TestRE2LookaroundPorts(t *testing.T) {
	// AWS docs example is skipped; a real AKIA key is caught.
	if hasKind(scanLine("AKIAIOSFODNN7EXAMPLE"), "token:aws_access_key") {
		t.Error("AWS docs example must NOT be flagged")
	}
	if !hasKind(scanLine("AKIA"+"1234567890ABCDEF"), "token:aws_access_key") {
		t.Error("a real AKIA key must be flagged")
	}
	// A redacted sessionKey value is not re-flagged; a real one is.
	if hasKind(scanLine(`"sessionKey": "<RP-SESSION-UUID-REDACTED>"`), "rp_session_key") {
		t.Error("redacted sessionKey must NOT be re-flagged")
	}
	if !hasKind(scanLine(`"sessionKey": "abc-123-def-456"`), "rp_session_key") {
		t.Error("a real sessionKey value must be flagged")
	}
}

// TestAWSAccessKeyPrefixFamily proves the aws_access_key gate covers the whole
// documented AWS access-key-ID prefix family, not just long-term AKIA keys
// (gh-358). A temporary STS key ID (ASIA) is shape-identical to its AKIA sibling
// and equally a hard_fail credential; the same holds for ABIA (STS service
// bearer token), ACCA (context-specific credential) and the legacy A3T access
// key. All FAKE shapes. The exclusions guard against over-broadening onto AWS
// RESOURCE identifiers (roles, users, groups, …), which are not credentials and
// appear openly in ARNs and policy documents.
func TestAWSAccessKeyPrefixFamily(t *testing.T) {
	r := strings.Repeat
	// Positives: every credential-bearing prefix, all FAKE shapes, must be
	// flagged as token:aws_access_key AND classified hard_fail.
	positives := []struct {
		name string
		key  string
	}{
		{"AKIA_longterm", "AKIA" + r("A", 16)},
		{"ASIA_sts_temporary", "ASIA" + r("B", 16)},
		{"ABIA_sts_bearer", "ABIA" + r("C", 16)},
		{"ACCA_context", "ACCA" + r("D", 16)},
		{"A3T_legacy", "A3T" + r("E", 17)}, // A3T + [0-9A-Z] + [0-9A-Z]{16}
	}
	for _, p := range positives {
		t.Run("positive/"+p.name, func(t *testing.T) {
			got := scanLine("cred = " + p.key + " end")
			if !hasKind(got, "token:aws_access_key") {
				t.Fatalf("AWS key %q (%s) not flagged: %+v", p.key, p.name, got)
			}
			for _, f := range got {
				if f.Kind == "token:aws_access_key" && f.Severity != SeverityHardFail {
					t.Errorf("AWS key %q flagged but severity %s, want hard_fail", p.key, f.Severity)
				}
			}
		})
	}
	// Negatives: must NOT be flagged. AROA/AIDA/AGPA/… are AWS resource
	// identifiers, not credentials — deliberately excluded to avoid false
	// positives on ARNs and policy docs. The dash cases break the 16-char body.
	negatives := []struct {
		name string
		s    string
	}{
		{"AROA_role_resource_id", "AROA" + r("A", 16)},
		{"AIDA_user_resource_id", "AIDA" + r("A", 16)},
		{"AGPA_group_resource_id", "AGPA" + r("A", 16)},
		{"ASIA_dash_not_a_key", "ASIA-not-a-key"},
		{"bare_AsiaBanana_word", "AsiaBananaRepublic"},
	}
	for _, n := range negatives {
		t.Run("negative/"+n.name, func(t *testing.T) {
			if got := scanLine("value = " + n.s + " end"); hasKind(got, "token:aws_access_key") {
				t.Errorf("non-credential %q wrongly flagged as AWS access key: %+v", n.s, got)
			}
		})
	}
}

// TestScanBundleReadsContent proves the scanner inspects file CONTENTS: a
// planted secret inside an included file is caught and counts as a hard-fail.
func TestScanBundleReadsContent(t *testing.T) {
	root := t.TempDir()
	secret := "ghp_" + strings.Repeat("z", 40)
	abs := writeFile(t, root, "commands/x.md", "# doc\ntoken = "+secret+"\n")

	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sc.ScanBundle([]BundleFile{{LogicalPath: "commands/x.md", ResolvedPath: abs}})
	if err != nil {
		t.Fatal(err)
	}
	if res.FilesScanned != 1 {
		t.Fatalf("expected 1 file scanned, got %d", res.FilesScanned)
	}
	if res.HardFails == 0 {
		t.Fatalf("planted secret not caught: %+v", res)
	}
	if !hasKind(res.Findings, "token:github_pat") {
		t.Errorf("expected github_pat finding, got %+v", res.Findings)
	}
	if res.Findings[0].File != "commands/x.md" || res.Findings[0].Line != 2 {
		t.Errorf("finding not reported under logical path/line: %+v", res.Findings[0])
	}
}

// TestScanBundleSkipsBinary proves a binary file is not scanned.
func TestScanBundleSkipsBinary(t *testing.T) {
	root := t.TempDir()
	abs := writeFile(t, root, "assets/blob.bin", "prefix\x00"+"ghp_"+strings.Repeat("a", 40))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := sc.ScanBundle([]BundleFile{{LogicalPath: "assets/blob.bin", ResolvedPath: abs}})
	if res.FilesScanned != 0 || res.HardFails != 0 {
		t.Errorf("binary file must be skipped: %+v", res)
	}
}

// TestIdentityEmailAndNoreply proves identity email detection and the noreply
// suppression (pure ScanText path).
func TestIdentityEmailAndNoreply(t *testing.T) {
	id := Identity{GitUserEmail: "person@example.com"}
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	if got := ScanText("contact person@example.com today", id, pats, sev, "f"); !hasKind(got, kindRealEmail) {
		t.Errorf("real email not flagged: %+v", got)
	}
	// The noreply form of the SAME email must be suppressed.
	id2 := Identity{GitUserEmail: "12345+person@users.noreply.github.com"}
	if got := ScanText("author 12345+person@users.noreply.github.com", id2, pats, sev, "f"); hasKind(got, kindRealEmail) {
		t.Errorf("noreply email must be suppressed: %+v", got)
	}
}

// TestIdentityNonASCIIName proves a hard_fail real_name whose edge runes are
// non-ASCII (accented/CJK/Cyrillic) is still detected — RE2's ASCII \b used to
// silently miss it, publishing the name unredacted.
func TestIdentityNonASCIIName(t *testing.T) {
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	for _, name := range []string{"Émilie Dupont", "Иван Петров", "张伟明"} {
		id := Identity{GitUserName: name}
		line := "Reviewed-by: " + name
		if got := ScanText(line, id, pats, sev, "f"); !hasKind(got, kindRealName) {
			t.Errorf("non-ASCII real_name %q not flagged: %+v", name, got)
		}
	}
}

// TestIdentityNameEqualsNoreplyLogin proves the real_name suppression survives
// an org-owned remote (iss-283): when user.name equals the GitHub login embedded
// in the caller's users.noreply.github.com address, the name is a public handle,
// not a real name — even though the remote's owner (an organisation) no longer
// matches it.
func TestIdentityNameEqualsNoreplyLogin(t *testing.T) {
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	for _, email := range []string{
		"77722411+octopat@users.noreply.github.com",
		"octopat@users.noreply.github.com",
		"77722411+OctoPat@users.noreply.github.com",
	} {
		id := Identity{GitUserName: "octopat", GitUserEmail: email, GitRemoteUsername: "some-org"}
		if got := ScanText(`"name": "octopat",`, id, pats, sev, "f"); hasKind(got, kindRealName) {
			t.Errorf("public noreply login flagged as real_name (email %q): %+v", email, got)
		}
	}
	// A user.name that does NOT match the noreply login stays a real name.
	id := Identity{GitUserName: "Octo Pat", GitUserEmail: "77722411+octopat@users.noreply.github.com", GitRemoteUsername: "some-org"}
	if got := ScanText("Reviewed-by: Octo Pat", id, pats, sev, "f"); !hasKind(got, kindRealName) {
		t.Errorf("real name unlike the noreply login not flagged: %+v", got)
	}
}

// TestIdentityEmailCaseInsensitive proves a case variant of the caller's own
// email still trips the hard_fail real_email gate.
func TestIdentityEmailCaseInsensitive(t *testing.T) {
	id := Identity{GitUserEmail: "person@example.com"}
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	if got := ScanText("contact Person@Example.COM today", id, pats, sev, "f"); !hasKind(got, kindRealEmail) {
		t.Errorf("case-variant real email not flagged: %+v", got)
	}
}

// TestIdentityHomePathOtherDetected proves a third-party home path at a realistic
// boundary (line start, after a space) is flagged — the leading RE2 \b used to
// make genericHomeRe never match in these positions.
func TestIdentityHomePathOtherDetected(t *testing.T) {
	id := Identity{HomePath: "/Users/me", HomeUser: "me"} // abcd-audit:allow
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	if got := ScanText("scp backup.tgz /Users/colleague/incoming/", id, pats, sev, "f"); !hasKind(got, kindHomeOther) { // abcd-audit:allow
		t.Errorf("third-party home path after a space not flagged: %+v", got)
	}
}

// TestIdentityHomePath proves home_path_self detection with the boundary
// predicate.
func TestIdentityHomePath(t *testing.T) {
	id := Identity{HomePath: "/Users/someone", HomeUser: "someone"} // abcd-audit:allow
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	got := ScanText("see /Users/someone/notes.txt for details", id, pats, sev, "f") // abcd-audit:allow
	if !hasKind(got, kindHomeSelf) {
		t.Errorf("home_path_self not flagged: %+v", got)
	}
}

// TestSeverityFloorHeld proves a config override cannot lower a bundled pattern
// below its built-in floor, but may raise an identity kind.
func TestSeverityFloorHeld(t *testing.T) {
	root := t.TempDir()
	// Try to downgrade github_pat hard_fail → warn, and raise github_username
	// warn → hard_fail.
	cfg := `{
	  "patterns": { "github_pat": { "regex": "\\bghp_[A-Za-z0-9]{36,}\\b", "severity": "warn" } },
	  "identity_severities": { "github_username": "hard_fail", "real_email": "warn" }
	}`
	writeFile(t, root, ".abcd/config/pii.json", cfg)
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	var ghp Pattern
	for _, p := range sc.patterns {
		if p.Name == "github_pat" {
			ghp = p
		}
	}
	if ghp.Severity != SeverityHardFail {
		t.Errorf("github_pat downgrade to warn must be clamped to hard_fail, got %s", ghp.Severity)
	}
	if sc.identSev[kindGithubUser] != SeverityHardFail {
		t.Errorf("github_username raise to hard_fail must be honoured, got %s", sc.identSev[kindGithubUser])
	}
	if sc.identSev[kindRealEmail] != SeverityHardFail {
		t.Errorf("real_email downgrade to warn must be clamped to hard_fail, got %s", sc.identSev[kindRealEmail])
	}
}

// TestBundledRegexNotReplaceable proves a per-repo config cannot neuter a bundled
// detector by swapping its regex for a never-match one (the floor-bypass): the
// bundled regex is immutable, so a real token is still caught at hard_fail.
func TestBundledRegexNotReplaceable(t *testing.T) {
	root := t.TempDir()
	cfg := `{ "patterns": { "anthropic_key": { "regex": "\\bZZZNEVERMATCH\\b" } } }`
	writeFile(t, root, ".abcd/config/pii.json", cfg)
	secret := "sk-ant-" + strings.Repeat("A", 40)
	abs := writeFile(t, root, "commands/x.md", "key = "+secret+"\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sc.ScanBundle([]BundleFile{{LogicalPath: "commands/x.md", ResolvedPath: abs}})
	if err != nil {
		t.Fatal(err)
	}
	if !hasKind(res.Findings, "token:anthropic") || res.HardFails == 0 {
		t.Errorf("bundled anthropic detector must survive a regex-override attempt: %+v", res)
	}
}

// TestNewPatternMergeIsAllOrNothing proves an invalid NEW-name override regex
// fails the whole merge closed (scanner unavailable) rather than half-applying a
// preceding valid override.
func TestNewPatternMergeIsAllOrNothing(t *testing.T) {
	root := t.TempDir()
	// "aaa" sorts before "zzz"; the invalid "zzz" must void the valid "aaa".
	cfg := `{ "patterns": {
	  "aaa_custom": { "regex": "\\bAAA[0-9]{6}\\b" },
	  "zzz_custom": { "regex": "(" }
	} }`
	writeFile(t, root, ".abcd/config/pii.json", cfg)
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if unavail, _ := sc.Unavailable(); !unavail {
		t.Fatal("an invalid override regex must mark the scanner unavailable")
	}
	for _, p := range sc.patterns {
		if p.Name == "aaa_custom" {
			t.Errorf("the valid earlier override must not be half-applied: %+v", p)
		}
	}
}

// TestScannerFailClosed proves an unreadable per-repo config marks the scanner
// unavailable (fail-closed) and ScanBundle surfaces it.
func TestScannerFailClosed(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json", "{ this is not valid json ")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := sc.ScanBundle(nil)
	if !res.Unavailable || res.UnavailableReason == "" {
		t.Errorf("malformed config must mark scanner unavailable: %+v", res)
	}
}

// TestBlankSkipEntriesRejected (Finding 1a) proves a blank/slash-only skip
// fragment or an empty skip-extension entry — each a substring/suffix of every
// path — is rejected rather than silently zeroing the scan's coverage: the
// planted secret is still scanned and caught.
func TestBlankSkipEntriesRejected(t *testing.T) {
	cases := []struct {
		name    string
		cfg     string
		logical string
		file    string
	}{
		{"empty_fragment", `{"skip_path_fragments": [""]}`, "commands/x.md", "commands/x.md"},
		{"slash_fragment", `{"skip_path_fragments": ["/"]}`, "commands/x.md", "commands/x.md"},
		// An empty extension entry would skip every extensionless file.
		{"empty_extension", `{"skip_extensions": [""]}`, "commands/LICENSE", "commands/LICENSE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, ".abcd/config/pii.json", tc.cfg)
			secret := "ghp_" + strings.Repeat("z", 40)
			abs := writeFile(t, root, tc.file, "token = "+secret+"\n")
			sc, err := New(root)
			if err != nil {
				t.Fatal(err)
			}
			res, _ := sc.ScanBundle([]BundleFile{{LogicalPath: tc.logical, ResolvedPath: abs}})
			if res.FilesScanned != 1 {
				t.Fatalf("blank skip entry must not zero coverage: %+v", res)
			}
			if res.HardFails == 0 {
				t.Fatalf("secret must still be caught with a rejected skip entry: %+v", res)
			}
			if res.Unavailable {
				t.Errorf("coverage was preserved, must not be unavailable: %+v", res)
			}
		})
	}
}

// TestWhitespaceFragmentDropped (Finding 1a) proves a whitespace-only skip
// fragment is dropped from the merged skip list rather than carried (it would
// never match a real path but must not persist as config, per the finding).
func TestWhitespaceFragmentDropped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json", `{"skip_path_fragments": ["   "]}`)
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, frag := range sc.skipFragments {
		if strings.TrimSpace(frag) == "" {
			t.Errorf("whitespace-only skip fragment must be dropped, found %q in %v", frag, sc.skipFragments)
		}
	}
}

// TestZeroCoverageSentinel (Finding 1b) proves a bundle with files but zero
// scanned — here every file skipped by a valid, non-empty extension skip — is
// marked Unavailable so the launch path fails closed instead of publishing an
// unscanned bundle.
func TestZeroCoverageSentinel(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json", `{"skip_extensions": [".md"]}`)
	abs := writeFile(t, root, "commands/x.md", "clean content\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := sc.ScanBundle([]BundleFile{{LogicalPath: "commands/x.md", ResolvedPath: abs}})
	if res.FilesScanned != 0 {
		t.Fatalf("expected all files skipped, got %+v", res)
	}
	if !res.Unavailable || res.UnavailableReason == "" {
		t.Errorf("zero coverage with files present must be marked unavailable: %+v", res)
	}
}

// TestSerializedFindingRedactsSecret (Finding 2) proves a planted PAT is still
// counted as a hard-fail with its file+line preserved, but the raw token never
// appears in the serialized (JSON) scan result.
func TestSerializedFindingRedactsSecret(t *testing.T) {
	root := t.TempDir()
	secret := "ghp_" + strings.Repeat("Q", 40)
	abs := writeFile(t, root, "commands/x.md", "token = "+secret+"\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := sc.ScanBundle([]BundleFile{{LogicalPath: "commands/x.md", ResolvedPath: abs}})
	if res.HardFails == 0 {
		t.Fatalf("planted PAT must still be caught: %+v", res)
	}
	blob, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	js := string(blob)
	if strings.Contains(js, secret) {
		t.Errorf("serialized scan result must not contain the raw token: %s", js)
	}
	if !strings.Contains(js, `"file":"commands/x.md"`) || !strings.Contains(js, `"line":1`) {
		t.Errorf("file+line locator must survive redaction: %s", js)
	}
}

// TestSerializedFindingRedactsStraddlingSecret (Finding 2, straddle hole) plants
// a ghp_ PAT on a >200-byte line so the token crosses the snippet byte cap. The
// finding must still count as a hard-fail with its file/line/column intact, yet
// the serialized JSON must carry NO raw run of the token — not the whole token,
// and not any >=6-char window of it (the prefix a truncate-then-replace snippet
// would have leaked at the cut).
func TestSerializedFindingRedactsStraddlingSecret(t *testing.T) {
	token := "ghp_" + strings.Repeat("A", 40) // 44 chars; matches the github_pat pattern
	pad := strings.Repeat("x", 185)
	line := pad + " " + token // token starts at byte 186; byte 200 falls inside it
	if len(pad)+1+14 <= 200 && len(pad)+1 >= 200 {
		t.Fatal("test setup: token must straddle the 200-byte cap")
	}

	findings := ScanText(line, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "commands/x.md")

	var pat *Finding
	for i := range findings {
		if findings[i].Kind == "token:github_pat" {
			pat = &findings[i]
			break
		}
	}
	if pat == nil {
		t.Fatalf("straddling PAT must still be caught: %+v", findings)
	}
	if pat.Severity != SeverityHardFail {
		t.Errorf("PAT must remain a hard-fail: %+v", pat)
	}
	wantCol := strings.Index(line, token) + 1
	if pat.Line != 1 || pat.Column != wantCol {
		t.Errorf("locator must be intact: got line=%d column=%d want line=1 column=%d", pat.Line, pat.Column, wantCol)
	}

	blob, err := json.Marshal(findings)
	if err != nil {
		t.Fatal(err)
	}
	js := string(blob)
	if strings.Contains(js, token) {
		t.Errorf("serialized result must not contain the raw token: %s", js)
	}
	// No >=6-char window (including any prefix) of the raw token may survive.
	for i := 0; i+6 <= len(token); i++ {
		if w := token[i : i+6]; strings.Contains(js, w) {
			t.Errorf("serialized result leaks a raw token window %q: %s", w, js)
		}
	}
	if !strings.Contains(js, `"file":"commands/x.md"`) || !strings.Contains(js, `"line":1`) {
		t.Errorf("file+line locator must survive redaction: %s", js)
	}
}

// TestSerializedShortIdentityFingerprinted proves a SHORT identity match (an
// email) is masked to a non-reversible fingerprint in the JSON: neither the raw
// value nor a leading fragment of it survives. A first-3 + last-2 window would
// expose most of such a short value, so short matches are fully starred.
func TestSerializedShortIdentityFingerprinted(t *testing.T) {
	email := "me@x.co" // 7 runes — short enough that head+tail would reveal it
	id := Identity{GitUserEmail: email}
	findings := ScanText("contact "+email+" now", id, DefaultPatterns(), DefaultIdentitySeverities(), "f")
	if !hasKind(findings, kindRealEmail) {
		t.Fatalf("real email must be flagged: %+v", findings)
	}
	blob, err := json.Marshal(findings)
	if err != nil {
		t.Fatal(err)
	}
	js := string(blob)
	if strings.Contains(js, email) {
		t.Errorf("serialized result must not contain the raw email: %s", js)
	}
	if strings.Contains(js, "me@") {
		t.Errorf("serialized result leaks a reversible fragment of the email: %s", js)
	}
	if !strings.Contains(js, `"matched":"`+strings.Repeat("*", len([]rune(email)))+`"`) {
		t.Errorf("short match must be fully starred (non-reversible): %s", js)
	}
}

// TestUnscannedBinaryRecorded (Finding 3) proves a leading-NUL file, classified
// binary and not scanned, is recorded in Unscanned rather than silently dropped.
func TestUnscannedBinaryRecorded(t *testing.T) {
	root := t.TempDir()
	abs := writeFile(t, root, "assets/blob.bin", "\x00ghp_"+strings.Repeat("a", 40))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := sc.ScanBundle([]BundleFile{{LogicalPath: "assets/blob.bin", ResolvedPath: abs}})
	if res.FilesScanned != 0 {
		t.Fatalf("leading-NUL file must not be scanned as text: %+v", res)
	}
	found := false
	for _, p := range res.Unscanned {
		if p == "assets/blob.bin" {
			found = true
		}
	}
	if !found {
		t.Errorf("unscannable binary file must be recorded, not silently dropped: %+v", res.Unscanned)
	}
}

// TestSerializedFindingRedactsSiblingSecret (iss-65 C14/C17, the BLOCK) plants
// TWO distinct secrets on one line. Each finding stores the whole line; the
// serialized snippet of finding A must not leak finding B's raw token, and vice
// versa — masking only a finding's own token leaves every sibling secret verbatim.
func TestSerializedFindingRedactsSiblingSecret(t *testing.T) {
	a := "ghp_" + strings.Repeat("A", 40)
	b := "ghp_" + strings.Repeat("B", 40)
	line := "gh=" + a + " other=" + b // both on one line, minified-env style

	findings := ScanText(line, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "commands/x.md")
	if len(findings) < 2 {
		t.Fatalf("both planted PATs must be caught: %+v", findings)
	}
	blob, err := json.Marshal(findings)
	if err != nil {
		t.Fatal(err)
	}
	js := string(blob)
	for _, tok := range []string{a, b} {
		if strings.Contains(js, tok) {
			t.Errorf("serialized result leaks a sibling secret verbatim %q: %s", tok, js)
		}
		// No >=6-char window of either raw token may survive in any snippet.
		for i := 0; i+6 <= len(tok); i++ {
			if w := tok[i : i+6]; strings.Contains(js, w) {
				t.Errorf("serialized result leaks a raw token window %q: %s", w, js)
			}
		}
	}
}

// TestSerializedFindingRedactsOverlappingSecret (iss-65, the overlap variant of
// the BLOCK) plants two secrets whose matches PARTIALLY OVERLAP on one line: a
// greedy sk-ant- key that runs into a following JWT, both detected. Substring
// masking (longest-first) destroys the shorter match's substring and leaves its
// non-overlapping tail raw; only byte-span masking closes it. No >=6-char raw
// window of either token may survive in the serialized snippet.
func TestSerializedFindingRedactsOverlappingSecret(t *testing.T) {
	key := "sk-ant-" + strings.Repeat("X", 34)
	jwt := "eyJ" + "ABCDEFGHIJ.KLMNOPQRST.UVWXYZ0123456"
	line := key + "-" + jwt // the `-` lets the key run into the JWT head; both fire

	findings := ScanText(line, Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "commands/x.md")
	if len(findings) < 2 {
		t.Fatalf("both an anthropic key and a JWT must be caught on the overlapping line: %+v", findings)
	}
	blob, err := json.Marshal(findings)
	if err != nil {
		t.Fatal(err)
	}
	js := string(blob)
	// The JWT payload/signature segments are the sensitive part — assert no raw
	// >=6-char window of either token survives anywhere in the serialized result.
	for _, tok := range []string{key, jwt} {
		for i := 0; i+6 <= len(tok); i++ {
			if w := tok[i : i+6]; strings.Contains(js, w) {
				t.Errorf("serialized snippet leaks a raw token window %q from an overlapping match: %s", w, js)
			}
		}
	}
}

// TestIsTextMultibyteRuneAtSniffBoundary (iss-65 C15) proves a valid UTF-8 file
// whose multibyte rune straddles the 8192-byte sniff cap is NOT misclassified as
// binary. The cut lands mid-rune, so a naive utf8.Valid(chunk[:8192]) fails.
func TestIsTextMultibyteRuneAtSniffBoundary(t *testing.T) {
	// '€' is 3 bytes (E2 82 AC) starting at index 8190, so the 8192 cut splits it.
	data := []byte(strings.Repeat("a", 8190) + "€" + strings.Repeat("z", 100))
	if !isText(data) {
		t.Fatal("a valid UTF-8 file with a rune straddling the sniff cap must read as text, not binary")
	}
	// A genuinely invalid encoding (a lone continuation byte early on) is still binary.
	bad := append([]byte("head "), 0x82)
	bad = append(bad, []byte(" tail")...)
	if isText(bad) {
		t.Error("genuinely invalid UTF-8 must still read as binary")
	}
}

// TestUnscannedUnreadableRecorded (iss-65 C18) proves a bundle file that cannot
// be read is surfaced in Unscanned, with the same visibility as a binary-skipped
// file, rather than silently dropped.
func TestUnscannedUnreadableRecorded(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	// ResolvedPath points at a file that does not exist → os.ReadFile errors.
	res, _ := sc.ScanBundle([]BundleFile{{LogicalPath: "commands/gone.md", ResolvedPath: filepath.Join(root, "nope", "gone.md")}})
	found := false
	for _, p := range res.Unscanned {
		if p == "commands/gone.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("an unreadable bundle file must be recorded in Unscanned, not silently dropped: %+v", res.Unscanned)
	}
}

// TestIdentityLocalUsernameSystemPathSuppressed proves the iss-31 fix: a machine
// username that collides with a system directory (here "dev" vs "/dev/null") is
// not flagged when it is the top segment of an absolute system path, while a
// genuine username occurrence is still caught.
func TestIdentityLocalUsernameSystemPathSuppressed(t *testing.T) {
	id := Identity{HomeUser: "dev"}
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()

	// System path — must NOT flag (the /dev/null false positive).
	if got := ScanText(`run something 2>/dev/null || true`, id, pats, sev, "scripts/x.sh"); hasKind(got, kindLocalUser) {
		t.Errorf("system path /dev/null wrongly flagged as local username: %+v", got)
	}
	// Genuine leaks are still flagged (no false negatives from the suppression).
	if got := ScanText(`backup written to /home/dev/data`, id, pats, sev, "f"); !hasKind(got, kindLocalUser) { // abcd-audit:allow
		t.Errorf("nested username /home/dev not flagged (false negative): %+v", got) // abcd-audit:allow
	}
	if got := ScanText(`last commit authored by dev`, id, pats, sev, "f"); !hasKind(got, kindLocalUser) {
		t.Errorf("bare username not flagged (false negative): %+v", got)
	}
}

// TestIdentityLocalUsernameCaseInsensitive proves the hard_fail local_username
// matcher folds case like its home-path sibling: a case variant of the caller's
// own login (the natural prose spelling in a transcript) must still be flagged so
// redaction fires, while the iss-31 system-directory suppression keeps holding
// under a mixed-case spelling.
func TestIdentityLocalUsernameCaseInsensitive(t *testing.T) {
	id := Identity{HomeUser: "alice"} // abcd-audit:allow
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()

	for _, spelling := range []string{"ALICE", "Alice"} { // abcd-audit:allow
		line := "last commit authored by " + spelling
		if got := ScanText(line, id, pats, sev, "f"); !hasKind(got, kindLocalUser) {
			t.Errorf("case variant %q of the login not flagged (redaction would leak it): %+v", spelling, got)
		}
	}
	// iss-31 stays closed under folding: a mixed-case system path is still not a
	// username leak.
	sysID := Identity{HomeUser: "dev"}
	if got := ScanText(`run something 2>/DEV/null || true`, sysID, pats, sev, "scripts/x.sh"); hasKind(got, kindLocalUser) {
		t.Errorf("mixed-case system path /DEV/null wrongly flagged as local username: %+v", got)
	}
}

// TestIdentityLocalUsernameDottedIdentifierSuppressed proves
// iss-2609100505142469: a reverse-DNS identifier whose leading component happens
// to equal the local account name is a namespace, not a home directory, and must
// survive a capture intact. The damage is unrecoverable — the placeholder does
// not say which word it replaced — so the class is made impossible rather than
// merely reported.
//
// It is deliberately NOT the ordinary-dictionary-word case
// (iss-2609061504302157): a bare word in prose is still the caller's login and
// still a hard_fail, and the closing assertions hold that line.
func TestIdentityLocalUsernameDottedIdentifierSuppressed(t *testing.T) {
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()

	// Every leading component a reverse-DNS identifier ordinarily begins with is
	// also a plausible Unix login; the collision is the whole finding.
	for _, prefix := range []string{"com", "io", "app", "net", "org", "me", "sh", "dev"} {
		id := Identity{HomeUser: prefix}
		line := "the crash is in the bundle " + prefix + ".acme.app on launch"
		if got := ScanText(line, id, pats, sev, "f"); hasKind(got, kindLocalUser) {
			t.Errorf("reverse-DNS identifier %q.acme.app wrongly flagged as the local username: %+v", prefix, got)
		}
	}
	// A middle component is bounded by dots on both sides — a namespace by any
	// reading — and a Go module path is the same class.
	mid := Identity{HomeUser: "acme"}
	if got := ScanText("the package is com.acme.tool.Main", mid, pats, sev, "f"); hasKind(got, kindLocalUser) {
		t.Errorf("dotted namespace component wrongly flagged as the local username: %+v", got)
	}
	modID := Identity{HomeUser: "dev"}
	if got := ScanText("import dev.example.com/pkg/thing", modID, pats, sev, "f"); hasKind(got, kindLocalUser) {
		t.Errorf("module path host wrongly flagged as the local username: %+v", got)
	}

	// No false negatives. The suppression covers a whole component of a
	// three-part dotted run and nothing else.
	keep := Identity{HomeUser: "dev"}
	for _, line := range []string{
		"last commit authored by dev",       // bare prose mention
		"backup written to /home/dev/data",  // abcd-audit:allow
		"the file is dev.log",               // two components: a filename, not a namespace
		"the package is my-dev-tool.a.b",    // not a whole component
		"mail to dev.smith@example.com now", // an address, not a namespace
	} {
		if got := ScanText(line, keep, pats, sev, "f"); !hasKind(got, kindLocalUser) {
			t.Errorf("username not flagged in %q (false negative): %+v", line, got)
		}
	}
}

// TestSerializedShortMultibyteIdentityFullyStarred guards B15: a short non-ASCII
// identity value that is under 16 RUNES but at or over 16 BYTES must be fully
// starred in the sealed snippet, consistent with maskSecret's rune-based policy.
// A second finding on the same line forces the sealLine/fingerprintSpan path;
// pre-fix, fingerprintSpan compared BYTE length and leaked the email's head/tail
// (and could split a multi-byte rune) into the serialized snippet.
func TestSerializedShortMultibyteIdentityFullyStarred(t *testing.T) {
	email := "müller@täst.de" // 14 runes, 16 bytes — short by rune, long by byte
	if utf8.RuneCountInString(email) >= 16 || len(email) < 16 {
		t.Fatalf("test fixture invariant broke: %d runes, %d bytes", utf8.RuneCountInString(email), len(email))
	}
	id := Identity{GitUserEmail: email, HomePath: "/Users/someone", HomeUser: "someone"} // abcd-audit:allow
	line := "email " + email + " path /Users/someone/notes.txt"                          // abcd-audit:allow
	findings := ScanText(line, id, DefaultPatterns(), DefaultIdentitySeverities(), "f")
	if !hasKind(findings, kindRealEmail) || len(findings) < 2 {
		t.Fatalf("need the email plus a second finding on one line to seal the snippet: %+v", findings)
	}
	blob, err := json.Marshal(findings)
	if err != nil {
		t.Fatal(err)
	}
	js := string(blob)
	// The head fingerprint that leaked pre-fix ("mü") must not survive.
	if strings.Contains(js, "mü") {
		t.Errorf("sealed snippet leaks the head of a short multi-byte identity value: %s", js)
	}
	// No raw fragment of the email may survive anywhere.
	if strings.Contains(js, email) {
		t.Errorf("serialized result contains the raw email: %s", js)
	}
	// The masked snippet must remain valid UTF-8 (no rune split into 0xFFFD).
	if strings.Contains(js, "�") {
		t.Errorf("sealed snippet split a multi-byte rune into invalid UTF-8: %s", js)
	}
}
