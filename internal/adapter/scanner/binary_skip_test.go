package scanner

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// GHSA-9wv7-88w3-f77m (iss-2608291807454357): a filename-keyed skip must never
// be an unverified allow. A file on the binary skip list still enters the
// payload with its bytes, so its bytes are scanned by the byte rules and the
// file is reported under ScannedBinary or ContentUnverified — never waved
// through by its name.

// fakeToken builds a secret-SHAPED value at runtime (iss-2608281752471145: no
// verbatim secret-shaped literal ever enters source).
func fakeToken() string { return "ghp_" + strings.Repeat("q", 40) }

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// scanOne scans a single bundle file under logical with the given scanner.
func scanOne(t *testing.T, sc *Scanner, logical, abs string) ScanResult {
	t.Helper()
	res, err := sc.ScanBundle([]BundleFile{{LogicalPath: logical, ResolvedPath: abs}})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestSkipListedBinaryIsSecretScanned plants a token in notes.png — no image
// data at all, the bytes are the secret — and requires a hard-fail finding and
// nothing in Unscanned. The .png is reported as ContentUnverified (a PNG can
// hide bytes in a compressed chunk) but its plaintext is still scanned.
func TestSkipListedBinaryIsSecretScanned(t *testing.T) {
	root := t.TempDir()
	abs := writeFile(t, root, "docs/assets/notes.png", "token="+fakeToken()+"\n")
	clean := writeFile(t, root, "docs/README.md", "clean documentation\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := sc.ScanBundle([]BundleFile{
		{LogicalPath: "docs/README.md", ResolvedPath: clean},
		{LogicalPath: "docs/assets/notes.png", ResolvedPath: abs},
	})
	if res.HardFails == 0 {
		t.Fatalf("a secret inside a skip-listed .png must hard-fail: %+v", res)
	}
	if !contains(res.ContentUnverified, "docs/assets/notes.png") {
		t.Errorf("the .png must be reported as ContentUnverified: %+v", res)
	}
	if contains(res.Unscanned, "docs/assets/notes.png") {
		t.Errorf("a readable .png is byte-scanned, not an Unscanned gap: %+v", res)
	}
	if res.Unavailable {
		t.Errorf("a bundle with a fully scanned text file is not zero coverage: %+v", res)
	}
	if res.FilesScanned != 1 {
		t.Errorf("FilesScanned counts full-rule-set coverage only, want 1 got %d", res.FilesScanned)
	}
}

// TestSkipListedBinaryPEMKeyIsCaught covers the other secret class the skip
// exempted: a private-key block dropped into a .pdf.
func TestSkipListedBinaryPEMKeyIsCaught(t *testing.T) {
	root := t.TempDir()
	body := "%PDF-1.4\n-----BEGIN " + "RSA PRIVATE KEY-----\nMIIE\n"
	abs := writeFile(t, root, "docs/paper.pdf", body)
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "docs/paper.pdf", abs)
	if !hasKind(res.Findings, "token:pem_private_key") || res.HardFails == 0 {
		t.Fatalf("a private key inside a skip-listed .pdf must hard-fail: %+v", res)
	}
}

// TestScannedBinaryIsPlainByteScan: a skip-listed name on the plaintext
// allow-list (.gitignore, a skip FILENAME) is byte-scanned and reported as
// ScannedBinary; every other skip-listed format defaults to ContentUnverified.
func TestScannedBinaryIsPlainByteScan(t *testing.T) {
	root := t.TempDir()
	abs := writeFile(t, root, "sub/.gitignore", "# token="+fakeToken()+"\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "sub/.gitignore", abs)
	if !contains(res.ScannedBinary, "sub/.gitignore") || contains(res.ContentUnverified, "sub/.gitignore") {
		t.Errorf("a plaintext skip-listed name is ScannedBinary: %+v", res)
	}
	if res.HardFails == 0 {
		t.Errorf("its plaintext secret must still be caught: %+v", res)
	}
}

// TestUnlistedSkipFormatsDefaultToContentUnverified: the classification is
// closed on the PLAINTEXT side, not the compressed side. A database, an
// executable, a bytecode file or a config-added skip extension (.jar reaches
// the byte branch once a repo lists it) can all carry compressed payloads, so
// none of them may be labelled content-verified by default.
func TestUnlistedSkipFormatsDefaultToContentUnverified(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json", `{"skip_extensions":[".jar"]}`)
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if bad, why := sc.Unavailable(); bad {
		t.Fatalf("override must load: %s", why)
	}
	var files []BundleFile
	for _, name := range []string{"a.sqlite", "a.wav", "a.exe", "a.pyc", "a.so", "a.ico", "a.jar"} {
		files = append(files, BundleFile{LogicalPath: name, ResolvedPath: writeFile(t, root, name, "\x00opaque\n")})
	}
	res, _ := sc.ScanBundle(files)
	for _, f := range files {
		if !contains(res.ContentUnverified, f.LogicalPath) || contains(res.ScannedBinary, f.LogicalPath) {
			t.Errorf("%s must default to ContentUnverified: binary=%v unverified=%v", f.LogicalPath, res.ScannedBinary, res.ContentUnverified)
		}
	}
	if contains(res.Unscanned, "a.jar") {
		t.Errorf("a config-added skip extension takes the byte branch, not Unscanned: %+v", res)
	}
}

// pngBytes encodes a small valid RGBA image with the standard library.
func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 16), uint8(y * 16), uint8(x ^ y), 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestValidPNGWithoutSecretIsClean: a genuine image yields zero findings —
// decoding its IDAT and text chunks must not manufacture one — and is reported
// as ContentDecoded, never as ScannedBinary: the byte scan alone covers only
// its plaintext regions, and the decoder is what reads the rest
// (iss-2608291832160371).
func TestValidPNGWithoutSecretIsClean(t *testing.T) {
	root := t.TempDir()
	abs := writeFile(t, root, "docs/assets/img/logo.png", string(pngBytes(t)))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "docs/assets/img/logo.png", abs)
	if len(res.Findings) != 0 {
		t.Fatalf("a real image must not trip any rule: %+v", res.Findings)
	}
	if !contains(res.ContentDecoded, "docs/assets/img/logo.png") || contains(res.ScannedBinary, "docs/assets/img/logo.png") {
		t.Errorf("a decoded image is ContentDecoded, never ScannedBinary: %+v", res)
	}
}

// synthIdentity is a fabricated caller identity for the binary-branch tests —
// never the real account (iss-2608291444328326 shows why the literal login is
// the wrong fixture).
func synthIdentity() Identity {
	return Identity{
		GitUserName:       "Zed Q Eight",
		GitUserEmail:      "zq8@example.test",
		GitRemoteUsername: "zq8handle",
		HomePath:          "/Users/zq8home",
		HomeUser:          "zq8home",
	}
}

// TestBinaryScanKeepsLongIdentityRulesDropsShortOnes pins the rule for the
// skip-listed branch: the LONG-literal identity rules (home_path_self,
// real_email, a multi-word or long real_name) run on raw bytes because their
// chance collision is negligible and a home path, an address or an /Author
// stamp in image or PDF metadata is the same leak it is in prose; the
// SHORT/GENERIC ones (local_username, github_username, home_path_other) do not,
// because on binary bytes they are noise.
func TestBinaryScanKeepsLongIdentityRulesDropsShortOnes(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	sc.identity = synthIdentity()
	id := sc.identity
	body := "\x89PNG\r\n\x1a\ntEXt " + id.HomePath + "/deck " + id.GitUserEmail + " " +
		id.GitUserName + " " + id.GitRemoteUsername + " " + id.HomeUser + " /home/someone/x\n"
	abs := writeFile(t, root, "a.png", body)
	res := scanOne(t, sc, "a.png", abs)
	for _, kind := range []string{kindHomeSelf, kindRealEmail, kindRealName} {
		if !hasKind(res.Findings, kind) {
			t.Errorf("long-literal identity rule %s must fire on binary bytes: %+v", kind, res.Findings)
		}
	}
	for _, kind := range []string{kindLocalUser, kindGithubUser, kindHomeOther} {
		if hasKind(res.Findings, kind) {
			t.Errorf("short/generic identity rule %s must not fire on binary bytes: %+v", kind, res.Findings)
		}
	}
}

// TestRealNameOnBytesByLiteralShape: real_name is kept on bytes when the
// literal cannot collide by chance — 8+ characters or more than one word — and
// dropped for a short single token.
func TestRealNameOnBytesByLiteralShape(t *testing.T) {
	cases := []struct {
		name string
		keep bool
	}{
		{"Zed Q Eight", true}, // multi-word
		{"Zedquinta", true},   // long single token
		{"Zedqx", false},      // short single token: noise on bytes
	}
	for _, c := range cases {
		root := t.TempDir()
		sc, err := New(root)
		if err != nil {
			t.Fatal(err)
		}
		sc.identity = Identity{GitUserName: c.name}
		abs := writeFile(t, root, "deck.pdf", "%PDF-1.4\n/Author ("+c.name+")\n")
		res := scanOne(t, sc, "deck.pdf", abs)
		if got := hasKind(res.Findings, kindRealName); got != c.keep {
			t.Errorf("real_name %q on bytes: fired=%v want %v (%+v)", c.name, got, c.keep, res.Findings)
		}
	}
}

// TestBinaryHomePathHardFailsLikeText: renaming deck.md to deck.pdf must not
// turn a release-blocking home path in plaintext metadata into a pass.
func TestBinaryHomePathHardFailsLikeText(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	sc.identity = synthIdentity()
	body := "%PDF-1.4\n/Creator (" + sc.identity.HomePath + "/deck.key)\n"
	md := writeFile(t, root, "deck.md", body)
	pdf := writeFile(t, root, "deck.pdf", body)
	text := scanOne(t, sc, "deck.md", md)
	bin := scanOne(t, sc, "deck.pdf", pdf)
	if text.HardFails == 0 {
		t.Fatalf("control: the home path must hard-fail as text: %+v", text)
	}
	if bin.HardFails != text.HardFails {
		t.Errorf("deck.pdf hardfails=%d, deck.md hardfails=%d — the extension must not change the verdict", bin.HardFails, text.HardFails)
	}
}

// TestBinaryRealNameEqualToHandleMatchesText: when user.name equals the
// public GitHub handle, the text scan suppresses real_name (iss-283) and
// reports the handle as github_username. The byte scan must reach the same
// verdict, so deck.pdf cannot hard-fail on a name deck.md only warns on.
func TestBinaryRealNameEqualToHandleMatchesText(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	sc.identity = Identity{GitUserName: "zq8handle", GitRemoteUsername: "zq8handle"}
	body := "%PDF-1.4\n/Author (zq8handle)\n"
	md := writeFile(t, root, "deck.md", body)
	pdf := writeFile(t, root, "deck.pdf", body)
	text := scanOne(t, sc, "deck.md", md)
	bin := scanOne(t, sc, "deck.pdf", pdf)
	if hasKind(text.Findings, kindRealName) {
		t.Fatalf("control: text suppresses real_name when it equals the handle: %+v", text.Findings)
	}
	if hasKind(bin.Findings, kindRealName) || bin.HardFails != text.HardFails {
		t.Errorf("deck.pdf hardfails=%d, deck.md hardfails=%d — the handle rule must hold on bytes: %+v", bin.HardFails, text.HardFails, bin.Findings)
	}
}

// TestSessionURLInBinaryHardFailsLikeText: the harness-leak class is ONE
// definition reaching launch (AGENTS.md); a live session URL in image or PDF
// metadata is a long literal with no chance collision, so it hard-fails on
// bytes exactly as it does in prose.
func TestSessionURLInBinaryHardFailsLikeText(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	body := "%PDF-1.4\n/Subject (https://agent-host.dev/code/session_" + synthSessionID(t, 41) + ")\n"
	md := writeFile(t, root, "deck.md", body)
	pdf := writeFile(t, root, "deck.pdf", body)
	text := scanOne(t, sc, "deck.md", md)
	bin := scanOne(t, sc, "deck.pdf", pdf)
	if !hasKind(text.Findings, kindHarnessSessionURL) || text.HardFails == 0 {
		t.Fatalf("control: the session URL must hard-fail as text: %+v", text)
	}
	if !hasKind(bin.Findings, kindHarnessSessionURL) || bin.HardFails != text.HardFails {
		t.Errorf("deck.pdf hardfails=%d, deck.md hardfails=%d — a session URL must be caught on bytes: %+v", bin.HardFails, text.HardFails, bin.Findings)
	}
}

// TestGenericHomePathInBinaryIsNotAFinding: home_path_other runs outside the
// identity guards, so it must be dropped explicitly on the binary branch.
func TestGenericHomePathInBinaryIsNotAFinding(t *testing.T) {
	root := t.TempDir()
	abs := writeFile(t, root, "b.png", "\x89PNG\r\n\x1a\n/home/runner/work/repo/x\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	// Pin a synthetic identity: on a hosted Linux runner HOME is
	// /home/runner, and the third-party path would be the caller's own.
	sc.identity = synthIdentity()
	res := scanOne(t, sc, "b.png", abs)
	if len(res.Findings) != 0 {
		t.Fatalf("a third-party path in binary bytes is noise, not a finding: %+v", res.Findings)
	}
}

// TestRepoRaisedSeverityIsHonouredOnBytes: a per-repo pii.json that raises a
// dropped kind above its built-in default has made a judgement the byte scan
// must honour — the drop is a default, not a ceiling on the repo's own policy.
func TestRepoRaisedSeverityIsHonouredOnBytes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/pii.json", `{"identity_severities":{"home_path_other":"hard_fail"}}`)
	abs := writeFile(t, root, "b.png", "\x89PNG\r\n\x1a\n/home/runner/work/repo/x\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if bad, why := sc.Unavailable(); bad {
		t.Fatalf("override must load: %s", why)
	}
	sc.identity = synthIdentity() // not the runner's own /home/runner
	res := scanOne(t, sc, "b.png", abs)
	if !hasKind(res.Findings, kindHomeOther) || res.HardFails != 1 {
		t.Fatalf("a repo-raised home_path_other must hard-fail on bytes like it does on text: %+v", res)
	}
}

// TestEveryIdentityKindIsClassifiedForBytes: the byte-scan policy is an
// explicit table, so a new identity kind cannot land on either side silently —
// it fails here until it is classified.
func TestEveryIdentityKindIsClassifiedForBytes(t *testing.T) {
	for _, kind := range identityKinds() {
		if _, ok := byteScanPolicy(kind); !ok {
			t.Errorf("identity kind %s has no explicit byte-scan policy", kind)
		}
	}
	if _, ok := byteScanPolicy("no_such_kind"); ok {
		t.Errorf("an unknown kind must not report as classified")
	}
	for _, kind := range []string{kindLocalUser, kindGithubUser, kindHomeOther} {
		if p, _ := byteScanPolicy(kind); p != bytePolicyDrop {
			t.Errorf("%s must be dropped on bytes, got %v", kind, p)
		}
	}
	for _, kind := range []string{kindHomeSelf, kindRealEmail} {
		if p, _ := byteScanPolicy(kind); p != bytePolicyKeep {
			t.Errorf("%s must be kept on bytes, got %v", kind, p)
		}
	}
	if p, _ := byteScanPolicy(kindRealName); p != bytePolicyKeepLongLiteral {
		t.Errorf("real_name must be kept by literal shape, got %v", p)
	}
}

// TestCompressedIsNotContentVerified: a compressed or container format is
// byte-scanned (an uncompressed tar's plaintext still yields its secret) but
// reported as ContentUnverified, never as ScannedBinary — a deflate stream, a
// JPEG entropy stream or an mp4 box are equally invisible to a byte scan, and
// the report must not claim otherwise (iss-2608291832160371).
func TestCompressedIsNotContentVerified(t *testing.T) {
	root := t.TempDir()
	plainTar := writeFile(t, root, "docs/plain.tar", ".env\x00\x00TOKEN="+fakeToken()+"\n")
	gz := writeFile(t, root, "docs/pack.tgz", "\x1f\x8b\x08\x00opaque deflate bytes\n")
	jpg := writeFile(t, root, "docs/photo.jpg", "\xff\xd8\xff\xe0 entropy\n")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := sc.ScanBundle([]BundleFile{
		{LogicalPath: "docs/plain.tar", ResolvedPath: plainTar},
		{LogicalPath: "docs/pack.tgz", ResolvedPath: gz},
		{LogicalPath: "docs/photo.jpg", ResolvedPath: jpg},
	})
	for _, p := range []string{"docs/plain.tar", "docs/pack.tgz", "docs/photo.jpg"} {
		if !contains(res.ContentUnverified, p) {
			t.Errorf("%s must be reported as ContentUnverified: %+v", p, res)
		}
		if contains(res.ScannedBinary, p) {
			t.Errorf("%s must not claim ScannedBinary: %+v", p, res)
		}
	}
	if res.HardFails == 0 {
		t.Errorf("an uncompressed tar's plaintext secret is still caught by the byte scan: %+v", res)
	}
}

// TestZeroCoverageReasonCountsByteScannedFiles: the sentinel must not call a
// byte-scanned file "unscannable".
func TestZeroCoverageReasonCountsByteScannedFiles(t *testing.T) {
	root := t.TempDir()
	ico := writeFile(t, root, "a.ico", "\x00\x00\x01\x00\n")
	pngf := writeFile(t, root, "b.png", string(pngBytes(t)))
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := sc.ScanBundle([]BundleFile{
		{LogicalPath: "a.ico", ResolvedPath: ico},
		{LogicalPath: "b.png", ResolvedPath: pngf},
	})
	if !res.Unavailable {
		t.Fatalf("an all-binary bundle is zero full-rule-set coverage: %+v", res)
	}
	want := "2 of 2 bundle files byte-scanned only"
	if !strings.Contains(res.UnavailableReason, want) {
		t.Errorf("reason %q must say %q", res.UnavailableReason, want)
	}
	if strings.Contains(res.UnavailableReason, "unscannable") {
		t.Errorf("reason %q must not call byte-scanned files unscannable", res.UnavailableReason)
	}
}

// TestOversizedBinaryIsLoudRefusal: a skip-listed file past the byte cap is
// not read (memory bound) and lands in Unscanned — the fail-closed gap the
// launch gate refuses — never a quiet skip, and the entry says WHY so a
// 4.1 MiB asset is distinguishable from an I/O error.
func TestOversizedBinaryIsLoudRefusal(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "big.zip")
	f, err := os.Create(abs)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxBinaryScanBytes + 1); err != nil { // sparse: no real bytes written
		t.Fatal(err)
	}
	f.Close()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res := scanOne(t, sc, "big.zip", abs)
	if !contains(res.Unscanned, "big.zip") {
		t.Fatalf("an oversized skip-listed file must be an Unscanned refusal: %+v", res)
	}
	if contains(res.ScannedBinary, "big.zip") || contains(res.ContentUnverified, "big.zip") {
		t.Errorf("an unread file must not claim to be scanned: %+v", res)
	}
	if why := res.UnscannedWhy["big.zip"]; !strings.Contains(why, "4 MiB") {
		t.Errorf("the Unscanned entry must say it is over the cap, got %q", why)
	}
}

// TestUnscannedWhyDistinguishesShapes: a symlinked leaf and a leading-NUL text
// file each carry their own reason.
func TestUnscannedWhyDistinguishesShapes(t *testing.T) {
	root := t.TempDir()
	target := writeFile(t, root, "real.png", "x")
	link := filepath.Join(root, "link.png")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks unavailable")
	}
	nul := writeFile(t, root, "smuggle.md", "\x00abc")
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := sc.ScanBundle([]BundleFile{
		{LogicalPath: "link.png", ResolvedPath: link},
		{LogicalPath: "smuggle.md", ResolvedPath: nul},
	})
	if why := res.UnscannedWhy["link.png"]; !strings.Contains(why, "symlink") {
		t.Errorf("symlinked leaf must say so, got %q", why)
	}
	if why := res.UnscannedWhy["smuggle.md"]; !strings.Contains(why, "binary") {
		t.Errorf("a NUL text file must say it read as binary, got %q", why)
	}
}

// TestBinaryHomeFollowedByAnAlnumByteStillHardFails: a raw blob has no path
// syntax on either side of a literal, so the trailing half of the home anchor
// is waived on bytes exactly as the leading half is. A packed record whose
// next field starts with an alphanumeric byte must not hide the caller's home.
func TestBinaryHomeFollowedByAnAlnumByteStillHardFails(t *testing.T) {
	root := t.TempDir()
	sc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	sc.identity = synthIdentity()
	body := "\x89PNG\r\n\x1a\n\x00\x00\x00\x0dtEXtCreator\x00" + sc.identity.HomePath + "Q\x00\x00"
	png := writeFile(t, root, "bare.png", body)
	bin := scanOne(t, sc, "bare.png", png)
	if bin.HardFails == 0 || !hasKind(bin.Findings, kindHomeSelf) {
		t.Errorf("the caller's home followed by an alnum byte in a blob was not reported: %+v", bin)
	}
}
