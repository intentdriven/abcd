package tools

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sigstore/sigstore-go/pkg/root"
)

// --- the release table -------------------------------------------------------

// TestEveryPinnedReleaseIsComplete keeps the table honest: every registered
// tool has a pinned version, a download for each platform the intent scopes
// (macOS and Linux on arm64 and amd64), each one a full GitHub release
// download URL of the tool's own repository at the pinned tag, with a sha256
// and an archive member; and the pinned version lies inside the tool's
// compatible range, so abcd's own copy is always one it works with.
func TestEveryPinnedReleaseIsComplete(t *testing.T) {
	for _, name := range Names() {
		rel := registry[name].Release
		if rel.Repo == "" || rel.Version == "" {
			t.Fatalf("%s: no pinned release: %+v", name, rel)
		}
		if !rel.Compatible.Contains(rel.Version) {
			t.Errorf("%s: pinned %s lies outside its own compatible range %s", name, rel.Version, rel.Compatible)
		}
		for _, p := range []string{"darwin/arm64", "darwin/amd64", "linux/arm64", "linux/amd64"} {
			a, ok := rel.Assets[p]
			if !ok {
				t.Errorf("%s: no download for %s", name, p)
				continue
			}
			prefix := "https://github.com/" + rel.Repo + "/releases/download/v" + rel.Version + "/"
			if !strings.HasPrefix(a.URL, prefix) || strings.Contains(a.URL, "latest") {
				t.Errorf("%s/%s: %q is not a pinned release download under %s", name, p, a.URL, prefix)
			}
			if !hexDigest(a.SHA256) {
				t.Errorf("%s/%s: sha256 %q is not a lowercase hex digest", name, p, a.SHA256)
			}
			if a.Member == "" || (a.Archive != TarGz && a.Archive != Zip) {
				t.Errorf("%s/%s: incomplete asset %+v", name, p, a)
			}
			if !strings.HasSuffix(a.URL, "."+string(a.Archive)) {
				t.Errorf("%s/%s: URL %q does not end in its archive kind %s", name, p, a.URL, a.Archive)
			}
		}
	}
}

// TestHowEachToolIsCheckedIsSaid is the second criterion's second half: a
// tool that publishes no signed build record (gitleaks) is said to be checked
// by checksum only; gh is checked against its pinned signer too.
func TestHowEachToolIsCheckedIsSaid(t *testing.T) {
	gl := registry["gitleaks"].Release
	if gl.Signer != nil || !strings.Contains(gl.Checks(), "checksum only") {
		t.Fatalf("gitleaks: %q", gl.Checks())
	}
	gh := registry["gh"].Release
	if gh.Signer == nil || gh.Signer.SAN() != "https://github.com/cli/cli/.github/workflows/deployment.yml@refs/heads/trunk" {
		t.Fatalf("gh signer = %+v", gh.Signer)
	}
	if strings.Contains(gh.Checks(), "checksum only") || !strings.Contains(gh.Checks(), "signed build record") {
		t.Fatalf("gh: %q", gh.Checks())
	}
}

func TestVersionRange(t *testing.T) {
	r := VersionRange{Min: "8.19.0", Below: "9.0.0"}
	for v, want := range map[string]bool{
		"8.19.0": true, "8.30.1": true, "v8.30.1": true, "gitleaks version 8.24.3": true,
		"8.18.9": false, "9.0.0": false, "10.1.0": false, "dev": false, "": false,
	} {
		if got := r.Contains(v); got != want {
			t.Errorf("%q in %s = %v, want %v", v, r, got, want)
		}
	}
	if versionIn("gh version 2.103.2 (2026-10-23)") != "2.103.2" {
		t.Error("gh's version line does not parse")
	}
}

// --- explanations ------------------------------------------------------------

// TestExplainNamesTheCheckedDownload is the first criterion as the person
// reads it: the version abcd's release fixes, the exact URL, the person's own
// bin folder, how it is checked, and never a step that needs admin rights or a
// package manager.
func TestExplainNamesTheCheckedDownload(t *testing.T) {
	e := explainFor("gitleaks", TranscriptScan, "darwin", "arm64", "/home/p/.local/bin", "/home/p")
	if !e.Known || e.Requirement != Optional || e.Download == nil {
		t.Fatalf("explanation = %+v", e)
	}
	text := strings.Join(e.Lines(), "\n")
	for _, want := range []string{
		"gitleaks", "optional", "without it:", "native secret scanner", "what abcd uses it for:",
		"install: download gitleaks 8.30.1 into ~/.local/bin/gitleaks, checked by checksum only",
		"from: https://github.com/gitleaks/gitleaks/releases/download/v8.30.1/gitleaks_8.30.1_darwin_arm64.tar.gz (no admin rights",
		"nothing is written until the download passes its check", "what the install does:",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("explanation lacks %q:\n%s", want, text)
		}
	}
	if !strings.Contains(e.ManualText(), "b40ab0ae55c505963e365f271a8d3846efbc170aa17f2607f13df610a9aeb6a5") {
		t.Errorf("the by-hand text does not carry the sha256 to check: %q", e.ManualText())
	}
	noAdmin(t, text+"\n"+e.StepText()+"\n"+e.ManualText())
}

// noAdmin fails on any text that shows a step needing admin rights or a
// package manager.
func noAdmin(t *testing.T, text string) {
	t.Helper()
	for _, bad := range []string{"sudo", "brew", "Homebrew", "chown", "administrator", "apt-get"} {
		if strings.Contains(text, bad) {
			t.Errorf("text shows %q:\n%s", bad, text)
		}
	}
}

func TestExplainIsPerCapability(t *testing.T) {
	e := explainFor("gitleaks", TranscriptScanArmed, "linux", "amd64", "/h/.local/bin", "/h")
	if e.Requirement != Required || !strings.Contains(e.WithoutIt, "enabled") {
		t.Fatalf("armed: %+v", e)
	}
	gh := explainFor("gh", GitHubSettings, "darwin", "amd64", "/h/.local/bin", "/h")
	if gh.Requirement != Required || !gh.Known || !strings.Contains(gh.Effects, "gh auth login") {
		t.Fatalf("gh: %+v", gh)
	}
	if gh.Download == nil || !strings.HasSuffix(gh.Download.URL, "gh_2.103.2_macOS_amd64.zip") || gh.Download.ChecksumOnly {
		t.Fatalf("gh download = %+v", gh.Download)
	}
}

func TestUnknownToolGetsGenericTextAndARegistryGap(t *testing.T) {
	e := explainFor("frobnicate", TranscriptScan, "darwin", "arm64", "/h/.local/bin", "/h")
	if e.Known || e.Download != nil {
		t.Fatalf("an unregistered tool carries a download: %+v", e)
	}
	if !strings.Contains(e.RegistryGap, "abcd capture") || !strings.Contains(e.RegistryGap, "frobnicate") {
		t.Fatalf("registry gap: %q", e.RegistryGap)
	}
	if text := strings.Join(e.Lines(), "\n"); !strings.Contains(text, "install: none") {
		t.Fatalf("generic explanation:\n%s", text)
	}
}

func TestUnknownPlatformHasNoDownload(t *testing.T) {
	e := explainFor("gitleaks", TranscriptScan, "plan9", "amd64", "/h/.local/bin", "/h")
	if e.Download != nil || !strings.Contains(strings.Join(e.Lines(), "\n"), "install: none") {
		t.Fatalf("plan9 borrowed a download: %+v", e.Download)
	}
}

func TestEveryRegistryEntryIsComplete(t *testing.T) {
	for _, name := range Names() {
		tool := registry[name]
		if tool.Name != name || tool.What == "" || tool.Homepage == "" || tool.Effects == "" || len(tool.Verify) == 0 || tool.Verify[0] != name {
			t.Errorf("%s: incomplete entry %+v", name, tool)
		}
		for c, u := range tool.Uses {
			if u.Capability == "" || u.Does == "" || u.WithoutIt == "" || u.OnDecline == "" {
				t.Errorf("%s/%s: incomplete use %+v", name, c, u)
			}
		}
		noAdmin(t, tool.Effects)
	}
}

func TestBinDirFollowsUvsOrder(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	for _, tc := range []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{"XDG_BIN_HOME": "/x/bin", "XDG_DATA_HOME": "/d/share"}, "/x/bin"},
		{map[string]string{"XDG_DATA_HOME": "/d/share"}, "/d/bin"},
		{map[string]string{"XDG_BIN_HOME": "relative"}, "/h/.local/bin"},
		{nil, "/h/.local/bin"},
	} {
		if got := userBinDir(env(tc.env), "/h"); got != tc.want {
			t.Errorf("userBinDir(%v) = %q, want %q", tc.env, got, tc.want)
		}
	}
}

// --- install, against a local release server ---------------------------------

// fakeExec answers each verify by the program's base name.
type fakeExec struct {
	calls [][]string
	out   map[string]string
	fail  map[string]error
}

func (f *fakeExec) run(_ context.Context, argv []string) ([]byte, error) {
	f.calls = append(f.calls, append([]string(nil), argv...))
	return []byte(f.out[filepath.Base(argv[0])]), f.fail[filepath.Base(argv[0])]
}

// rig is one test machine: a home, a repository to guard, a PATH, and a
// release server holding one archive for the tool.
type rig struct {
	t       *testing.T
	home    string
	repo    string
	pathDir string // a directory on PATH for copies installed "some other way"
	srv     *httptest.Server
	hits    int
	archive []byte
	fx      *fakeExec
	env     map[string]string
	attest  Attester
	signer  *Signer
}

func tarGz(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for _, f := range []struct {
		n string
		b []byte
	}{{"README.md", []byte("readme")}, {name, body}} {
		if err := tw.WriteHeader(&tar.Header{Name: f.n, Mode: 0o755, Size: int64(len(f.b)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write(f.b)
	}
	tw.Close()
	zw.Close()
	return buf.Bytes()
}

func zipOf(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	w.Write(body)
	zw.Close()
	return buf.Bytes()
}

func newRig(t *testing.T) *rig {
	t.Helper()
	r := &rig{t: t, home: t.TempDir(), repo: t.TempDir(), pathDir: t.TempDir()}
	r.archive = tarGz(t, "gitleaks", []byte("#!/bin/sh\necho 8.30.1\n"))
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.hits++
		w.Write(r.archive)
	}))
	t.Cleanup(r.srv.Close)
	r.fx = &fakeExec{out: map[string]string{"gitleaks": "8.30.1\n"}, fail: map[string]error{}}
	r.env = map[string]string{"PATH": r.pathDir + string(os.PathListSeparator) + filepath.Join(r.home, ".local", "bin")}
	return r
}

func (r *rig) installer() *Installer {
	sum := sha256.Sum256(r.archive)
	u, _ := url.Parse(r.srv.URL)
	f := newFetcher(nil, true, u.Host)
	return &Installer{
		LookPath: func(name string) (string, error) {
			for _, d := range filepath.SplitList(r.env["PATH"]) {
				if p := filepath.Join(d, name); exists(p) {
					return p, nil
				}
			}
			return "", errors.New("not found")
		},
		Run:    r.fx.run,
		Getenv: func(k string) string { return r.env[k] },
		GOOS:   "linux", GOARCH: "amd64",
		Guard: r.repo, Home: r.home,
		Get:    func(ctx context.Context, u string, limit int64) ([]byte, error) { return f.get(ctx, u, limit, nil) },
		Attest: r.attest,
		Release: func(name string) (Release, bool) {
			rel, ok := registryRelease(name)
			if !ok {
				return rel, false
			}
			rel.Signer = r.signer
			rel.Assets = map[string]Asset{"linux/amd64": {URL: r.srv.URL + "/gitleaks.tar.gz", SHA256: hex.EncodeToString(sum[:]), Archive: TarGz, Member: "gitleaks"}}
			return rel, true
		},
	}
}

func (r *rig) dest() string { return filepath.Join(r.home, ".local", "bin", "gitleaks") }

func yes(Explanation) Answer { return Answer{Yes: true, Why: "typed yes"} }
func no(Explanation) Answer  { return Answer{Why: "typed no"} }

// TestInstallDownloadsChecksWritesAndRecords is the first criterion: a
// missing tool is downloaded at the pinned version into the person's own
// folder, checked, written, verified, and recorded; and gitleaks' result says
// it was checked by checksum only.
func TestInstallDownloadsChecksWritesAndRecords(t *testing.T) {
	r := newRig(t)
	var asked Explanation
	res := r.installer().Install("gitleaks", TranscriptScan, func(e Explanation) Answer { asked = e; return yes(e) })
	if !res.Ready || !res.Ran || !res.Installed || !res.Verified {
		t.Fatalf("result = %+v", res)
	}
	if asked.Download == nil || asked.Plan == nil || asked.Plan.Action != InstallOwn {
		t.Fatalf("the confirmation was not handed the plan: %+v", asked)
	}
	got, err := os.ReadFile(r.dest())
	if err != nil || !strings.Contains(string(got), "8.30.1") {
		t.Fatalf("program not written: %v %q", err, got)
	}
	if fi, _ := os.Stat(r.dest()); fi.Mode().Perm()&0o111 == 0 {
		t.Fatal("the program is not executable")
	}
	if !res.ChecksumOnly || !strings.Contains(res.Summary(), "checksum only") {
		t.Fatalf("checksum-only not said: %q", res.Summary())
	}
	if !strings.Contains(res.Summary(), "~/.local/bin/gitleaks") || strings.Contains(res.Summary(), r.home) {
		t.Fatalf("summary does not name the copy in the tilde form: %q", res.Summary())
	}
	rec, err := readRecord(r.home)
	if err != nil || rec.Tools["gitleaks"].Path != r.dest() || rec.Tools["gitleaks"].Version != "8.30.1" {
		t.Fatalf("record = %+v, %v", rec, err)
	}
	if last := r.fx.calls[len(r.fx.calls)-1]; last[0] != r.dest() || last[1] != "version" {
		t.Fatalf("verify ran %v, not the written copy", last)
	}
	noAdmin(t, strings.Join(asked.Lines(), "\n")+res.Summary())
}

// TestAFailedCheckWritesNothingAndSaysWhy is the second criterion's first
// half, for each check: a download whose sha256 is not the pinned one, a
// signed build record that does not verify, and a download that fails.
func TestAFailedCheckWritesNothingAndSaysWhy(t *testing.T) {
	for _, tc := range []struct {
		name string
		rig  func(*rig) *Installer
		want string
	}{
		{"tampered", func(r *rig) *Installer {
			in := r.installer()
			r.archive = tarGz(t, "gitleaks", []byte("evil"))
			return in
		}, "but abcd's release fixes"},
		{"unsigned", func(r *rig) *Installer {
			r.signer = &Signer{Repo: "o/r", Workflow: "w.yml", Ref: "refs/heads/main"}
			r.attest = func(context.Context, Signer, []byte) (string, error) { return "", errors.New("no record verifies") }
			return r.installer()
		}, "signed build record does not check out: no record verifies"},
		{"unreachable", func(r *rig) *Installer {
			in := r.installer()
			r.srv.Close()
			return in
		}, "the download from"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t)
			res := tc.rig(r).Install("gitleaks", TranscriptScan, yes)
			if res.Ready || res.Installed || exists(r.dest()) {
				t.Fatalf("a failed check wrote something: %+v", res)
			}
			if !strings.Contains(res.Summary(), tc.want) || !strings.Contains(res.Summary(), "nothing was written") ||
				!strings.Contains(res.Summary(), "continuing on the native secret scanner") {
				t.Fatalf("summary = %q", res.Summary())
			}
			if rec, _ := readRecord(r.home); len(rec.Tools) != 0 {
				t.Fatalf("a failed check was recorded: %+v", rec)
			}
			noAdmin(t, res.Summary())
		})
	}
}

// TestASignedToolIsCheckedAgainstItsSigner: the provenance check is handed
// the pinned signer and the archive's digest, and the result says both checks.
func TestASignedToolIsCheckedAgainstItsSigner(t *testing.T) {
	r := newRig(t)
	r.signer = &Signer{Repo: "o/r", Workflow: ".github/workflows/w.yml", Ref: "refs/heads/main"}
	var gotSigner Signer
	var gotDigest []byte
	r.attest = func(_ context.Context, s Signer, d []byte) (string, error) {
		gotSigner, gotDigest = s, d
		return s.SAN(), nil
	}
	res := r.installer().Install("gitleaks", TranscriptScan, yes)
	sum := sha256.Sum256(r.archive)
	if !res.Ready || gotSigner != *r.signer || !bytes.Equal(gotDigest, sum[:]) {
		t.Fatalf("result %+v; attest saw %+v %x", res, gotSigner, gotDigest)
	}
	if res.ChecksumOnly || !strings.Contains(res.Summary(), "signed build record") {
		t.Fatalf("summary = %q", res.Summary())
	}
}

// TestACompatibleCopyIsUsedAndNothingInstalled is the third criterion's first
// half: a copy installed some other way whose version abcd works with is used;
// nothing is asked, downloaded or written.
func TestACompatibleCopyIsUsedAndNothingInstalled(t *testing.T) {
	r := newRig(t)
	other := filepath.Join(r.pathDir, "gitleaks")
	os.WriteFile(other, []byte("other"), 0o755)
	r.fx.out["gitleaks"] = "8.24.3\n"
	asked := false
	res := r.installer().Install("gitleaks", TranscriptScan, func(e Explanation) Answer { asked = true; return yes(e) })
	if !res.Ready || res.Ran || asked || r.hits != 0 || exists(r.dest()) {
		t.Fatalf("a compatible copy was not used: %+v asked=%v hits=%d", res, asked, r.hits)
	}
	if !strings.Contains(res.Summary(), "nothing installed") || !strings.Contains(res.Summary(), "8.24.3") {
		t.Fatalf("summary = %q", res.Summary())
	}
	if b, _ := os.ReadFile(other); string(b) != "other" {
		t.Fatal("the other copy was changed")
	}
}

// TestAnIncompatibleCopyGetsAbcdsOwnAndSaysWhichRuns is the third criterion's
// second half: an old copy is named with why it does not fit, abcd's own is
// offered, the question says which of the two will run, and the other copy is
// never changed.
func TestAnIncompatibleCopyGetsAbcdsOwnAndSaysWhichRuns(t *testing.T) {
	r := newRig(t)
	other := filepath.Join(r.pathDir, "gitleaks")
	os.WriteFile(other, []byte("old"), 0o755)
	r.fx.out["gitleaks"] = "8.18.0\n"
	in := r.installer()
	plan := in.Plan("gitleaks")
	if plan.Action != InstallOwn || plan.Existing == nil || plan.Existing.Compatible {
		t.Fatalf("plan = %+v", plan)
	}
	var asked Explanation
	// The written copy answers the verify with the pinned version.
	in.Run = func(ctx context.Context, argv []string) ([]byte, error) {
		if argv[0] == r.dest() {
			return []byte("8.30.1\n"), nil
		}
		return r.fx.run(ctx, argv)
	}
	res := in.Install("gitleaks", TranscriptScan, func(e Explanation) Answer { asked = e; return yes(e) })
	if !res.Ready || !res.Installed {
		t.Fatalf("result = %+v", res)
	}
	text := strings.Join(asked.Lines(), "\n")
	for _, want := range []string{"already installed: " + other, "8.18.0 is outside the versions abcd works with",
		"which copy runs:", "abcd runs its own copy at ~/.local/bin/gitleaks", "still runs " + other} {
		if !strings.Contains(text, want) {
			t.Errorf("question lacks %q:\n%s", want, text)
		}
	}
	if b, _ := os.ReadFile(other); string(b) != "old" {
		t.Fatal("the other copy was changed")
	}
	if !strings.Contains(res.Summary(), "in your shell, gitleaks runs "+other) {
		t.Errorf("summary does not say which copy the shell runs: %q", res.Summary())
	}
}

// TestABinFolderCopyAbcdDidNotInstallIsNeverReplaced: when the bin folder
// already holds a program of that name abcd did not install, abcd installs
// nothing and says why, rather than overwrite it.
func TestABinFolderCopyAbcdDidNotInstallIsNeverReplaced(t *testing.T) {
	r := newRig(t)
	os.MkdirAll(filepath.Dir(r.dest()), 0o755)
	os.WriteFile(r.dest(), []byte("theirs"), 0o755)
	r.fx.out["gitleaks"] = "7.0.0\n"
	res := r.installer().Install("gitleaks", TranscriptScan, yes)
	if res.Ready || res.Ran || r.hits != 0 {
		t.Fatalf("result = %+v", res)
	}
	if b, _ := os.ReadFile(r.dest()); string(b) != "theirs" || !strings.Contains(res.Why, "abcd never changes another copy") {
		t.Fatalf("copy changed or not said: %q", res.Why)
	}
}

// TestAFailedVerifyRemovesTheCopy: a written program that does not report the
// pinned version is removed again and never recorded.
func TestAFailedVerifyRemovesTheCopy(t *testing.T) {
	r := newRig(t)
	r.fx.out["gitleaks"] = "8.29.0\n"
	res := r.installer().Install("gitleaks", TranscriptScan, yes)
	if res.Ready || res.Verified || exists(r.dest()) {
		t.Fatalf("result = %+v", res)
	}
	if !strings.Contains(res.Summary(), "not the pinned 8.30.1") || !strings.Contains(res.Summary(), "removed the copy") {
		t.Fatalf("summary = %q", res.Summary())
	}
}

// TestInstallRefusals: CI, a no, no confirmation, an unknown tool, and a bin
// folder inside the repository each write nothing and say so.
func TestInstallRefusals(t *testing.T) {
	for _, env := range []map[string]string{{"CI": "true"}, {"GITHUB_ACTIONS": "true"}, {"CI": "false"}, {"CI": "0"}} {
		r := newRig(t)
		for k, v := range env {
			r.env[k] = v
		}
		asked := false
		res := r.installer().Install("gitleaks", TranscriptScan, func(e Explanation) Answer { asked = true; return yes(e) })
		if res.Ran || asked || !res.Declined || !strings.Contains(res.Summary(), "CI") {
			t.Fatalf("%v: %+v", env, res)
		}
	}
	r := newRig(t)
	if res := r.installer().Install("gitleaks", TranscriptScan, no); res.Ran || !res.Declined ||
		res.Summary() != "gitleaks not installed (typed no); continuing on the native secret scanner" {
		t.Fatalf("no: %q", res.Summary())
	}
	if res := r.installer().Install("gitleaks", TranscriptScan, nil); res.Ran || !res.Declined {
		t.Fatalf("nil confirm: %+v", res)
	}
	if res := r.installer().Install("frobnicate", TranscriptScan, yes); res.Ran || !strings.Contains(res.Why, "gitleaks") {
		t.Fatalf("unknown: %+v", res)
	}
	in := r.installer()
	in.Guard = r.home // a repository rooted at the home holds ~/.local/bin
	if res := in.Install("gitleaks", TranscriptScan, yes); res.Ran || !strings.Contains(res.Why, "inside the repository") {
		t.Fatalf("bin folder in the repository: %+v", res)
	}
	if r.hits != 0 {
		t.Fatalf("a refusal downloaded: %d", r.hits)
	}
}

// --- running and removing abcd's copies --------------------------------------

func withHome(t *testing.T, home string, path func(string) (string, error)) {
	t.Helper()
	oh, op := userHome, pathLookup
	userHome = func() (string, error) { return home, nil }
	pathLookup = path
	t.Cleanup(func() { userHome, pathLookup = oh, op })
}

// TestResolvePrefersAbcdsCopyWhateverPathSays is the fourth criterion's first
// half: abcd runs its recorded copy even when another comes first on PATH,
// and falls back to PATH once the copy is no longer the bytes it wrote.
func TestResolvePrefersAbcdsCopyWhateverPathSays(t *testing.T) {
	r := newRig(t)
	if res := r.installer().Install("gitleaks", TranscriptScan, yes); !res.Ready {
		t.Fatal(res.Summary())
	}
	first := filepath.Join(r.pathDir, "gitleaks")
	withHome(t, r.home, func(string) (string, error) { return first, nil })
	if got, _ := Resolve("gitleaks"); got != r.dest() {
		t.Fatalf("Resolve = %q, want abcd's copy %q", got, r.dest())
	}
	os.WriteFile(r.dest(), []byte("changed by someone"), 0o755)
	if got, _ := Resolve("gitleaks"); got != first {
		t.Fatalf("a changed copy still resolves as abcd's: %q", got)
	}
	if got, _ := Resolve("gh"); got != first {
		t.Fatalf("an unrecorded tool does not fall back to PATH: %q", got)
	}
}

// TestUninstallRemovesOnlyAbcdsIntactCopies is the fourth criterion's second
// half: the copies abcd installed go, a recorded copy changed since stays, a
// copy installed some other way is never reached, and the record goes.
func TestUninstallRemovesOnlyAbcdsIntactCopies(t *testing.T) {
	r := newRig(t)
	if res := r.installer().Install("gitleaks", TranscriptScan, yes); !res.Ready {
		t.Fatal(res.Summary())
	}
	other := filepath.Join(r.pathDir, "gh")
	os.WriteFile(other, []byte("theirs"), 0o755)
	// A second recorded copy that someone changed since.
	changed := filepath.Join(r.home, ".local", "bin", "gh")
	os.WriteFile(changed, []byte("edited"), 0o755)
	rec, _ := readRecord(r.home)
	rec.Tools["gh"] = Installed{Path: changed, SHA256: strings.Repeat("a", 64), Version: "2.103.2"}
	if err := writeRecord(r.home, rec); err != nil {
		t.Fatal(err)
	}
	out, err := Uninstall(r.home)
	if err != nil {
		t.Fatal(err)
	}
	if exists(r.dest()) || !exists(changed) || !exists(other) {
		t.Fatalf("removed the wrong files: %+v", out)
	}
	b, _ := json.Marshal(out)
	if !strings.Contains(string(b), "removed abcd's copy") || !strings.Contains(string(b), "changed since abcd installed it") {
		t.Fatalf("removals = %s", b)
	}
	if exists(filepath.Join(r.home, ".abcd.noindex", "tools.json")) {
		t.Fatal("the record outlived the copies")
	}
}

// --- the transport -----------------------------------------------------------

func TestFetcherRefusesWhatTheDownloadMayNotReach(t *testing.T) {
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("x")) }))
	defer elsewhere.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, elsewhere.URL+"/x", http.StatusFound)
	}))
	defer redirect.Close()
	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write(make([]byte, 2048)) }))
	defer big.Close()
	host := func(s *httptest.Server) string { u, _ := url.Parse(s.URL); return u.Host }
	ctx := context.Background()

	if _, err := newFetcher(nil, true, host(redirect)).get(ctx, redirect.URL, 1<<20, nil); err == nil || !strings.Contains(err.Error(), "not a host") {
		t.Errorf("a redirect to another host was followed: %v", err)
	}
	if _, err := newFetcher(nil, false, host(big)).get(ctx, big.URL, 1<<20, nil); err == nil || !strings.Contains(err.Error(), "not https") {
		t.Errorf("plain http was fetched: %v", err)
	}
	if _, err := newFetcher(nil, true, host(big)).get(ctx, big.URL, 1024, nil); err == nil || !strings.Contains(err.Error(), "bound") {
		t.Errorf("an oversized answer was read: %v", err)
	}
	if _, err := newFetcher(nil, true, "github.com").get(ctx, big.URL, 1<<20, nil); err == nil {
		t.Error("a host outside the list was fetched")
	}
	for _, h := range releaseHosts {
		if h != "github.com" && !strings.HasSuffix(h, ".githubusercontent.com") {
			t.Errorf("release host %q is not GitHub's", h)
		}
	}
}

func TestExtractReadsOnlyTheNamedRegularFile(t *testing.T) {
	if b, err := extractMember(zipOf(t, "d/bin/gh", []byte("gh")), Zip, "d/bin/gh"); err != nil || string(b) != "gh" {
		t.Fatalf("zip: %q %v", b, err)
	}
	if _, err := extractMember(tarGz(t, "other", []byte("x")), TarGz, "gitleaks"); err == nil {
		t.Fatal("a missing member read as present")
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	tw.WriteHeader(&tar.Header{Name: "gitleaks", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"})
	tw.Close()
	zw.Close()
	if _, err := extractMember(buf.Bytes(), TarGz, "gitleaks"); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("a link member was read: %v", err)
	}
}

// --- the provenance check, on recorded real bundles --------------------------

// The fixtures under testdata/attest are recorded, not synthesised
// (2026-10-24): GitHub's attestations API answer for the sha256 of gh
// v2.103.2's gh_2.103.2_macOS_arm64.zip, whose entries carry no inline bundle,
// only a bundle_url into GitHub's attestation store (rewritten here to
// BUNDLE_BASE, served by the test); the two snappy-compressed bundles that
// store served; and the Sigstore public-good trust root, as recorded for
// internal/core/update's fixtures.
const ghMacArm64 = "5416000f5d17580a187339e0c4068a01ca5c87f7d4fe6ca64933696f967bc418"

func attestRig(t *testing.T) *githubAttester {
	t.Helper()
	dir := filepath.Join("testdata", "attest")
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case strings.HasPrefix(req.URL.Path, "/repos/cli/cli/attestations/sha256:"+ghMacArm64):
			b, _ := os.ReadFile(filepath.Join(dir, "gh-v2.103.2-macOS-arm64.attestations.json"))
			w.Write(bytes.ReplaceAll(b, []byte("BUNDLE_BASE"), []byte(base+"/store")))
		case strings.HasPrefix(req.URL.Path, "/store/"):
			b, err := os.ReadFile(filepath.Join(dir, filepath.Base(req.URL.Path)))
			if err != nil {
				http.NotFound(w, req)
				return
			}
			w.Write(b)
		default:
			http.NotFound(w, req)
		}
	}))
	t.Cleanup(srv.Close)
	base = srv.URL
	u, _ := url.Parse(srv.URL)
	tr, err := os.ReadFile(filepath.Join(dir, "trusted_root.json"))
	if err != nil {
		t.Fatal(err)
	}
	return &githubAttester{
		f: newFetcher(nil, true, u.Host), apiBase: srv.URL + "/repos/",
		bundleOK: func(b *url.URL) bool { return b.Host == u.Host },
		trusted: func() (root.TrustedMaterial, error) {
			r, err := root.NewTrustedRootFromJSON(tr)
			if err != nil {
				return nil, err
			}
			return r, nil
		},
	}
}

func TestGhsRecordedBuildRecordVerifiesAgainstItsPinnedSigner(t *testing.T) {
	g := attestRig(t)
	d, _ := hex.DecodeString(ghMacArm64)
	signer, err := g.verify(context.Background(), *registry["gh"].Release.Signer, d)
	if err != nil {
		t.Fatal(err)
	}
	if signer != "https://github.com/cli/cli/.github/workflows/deployment.yml@refs/heads/trunk" {
		t.Fatalf("signer = %q", signer)
	}
}

func TestABuildRecordFromAnotherSignerOrForOtherBytesFails(t *testing.T) {
	g := attestRig(t)
	d, _ := hex.DecodeString(ghMacArm64)
	wrong := Signer{Repo: "cli/cli", Workflow: ".github/workflows/release.yml", Ref: "refs/heads/trunk"}
	if _, err := g.verify(context.Background(), wrong, d); err == nil {
		t.Fatal("a record signed by another workflow verified")
	}
	other := append([]byte(nil), d...)
	other[0] ^= 0xff
	if _, err := g.verify(context.Background(), *registry["gh"].Release.Signer, other); err == nil {
		t.Fatal("a digest with no record verified")
	}
	g.bundleOK = func(*url.URL) bool { return false }
	if _, err := g.verify(context.Background(), *registry["gh"].Release.Signer, d); err == nil || !strings.Contains(err.Error(), "outside GitHub's attestation store") {
		t.Fatalf("a bundle_url outside the store was read: %v", err)
	}
}

func TestDecodeSnappyRefusesMalformedInput(t *testing.T) {
	for _, in := range [][]byte{nil, {0x05, 0x10, 'a'}, {0x03, 0x01, 0x00}, {0xff, 0xff, 0xff, 0xff, 0x0f}} {
		if _, err := decodeSnappy(in, 1<<20); err == nil {
			t.Errorf("decoded %x", in)
		}
	}
	// A literal "ab" then a copy of length 4 at offset 2: "ababab".
	got, err := decodeSnappy([]byte{0x06, 0x04, 'a', 'b', 0x01, 0x02}, 64)
	if err != nil || string(got) != "ababab" {
		t.Fatalf("got %q, %v", got, err)
	}
}

// --- the production runner ---------------------------------------------------

func TestRunArgvExecutesWithoutAShell(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "argv")
	script := filepath.Join(dir, "echoargs")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > "+out+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := runArgv(context.Background(), []string{script, "a;b", "$(x)"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(out); string(got) != "a;b\n$(x)\n" {
		t.Fatalf("argv reached the program as %q", got)
	}
}

func TestRunArgvKillsItsOwnGroupOnTimeout(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "hang")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30 &\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := runArgv(ctx, []string{script}); err == nil || !strings.Contains(err.Error(), "did not finish") {
		t.Fatalf("err = %v", err)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Fatalf("runArgv waited %s", d)
	}
}

func TestMissingErrorKeepsTheCauseAndCarriesTheExplanation(t *testing.T) {
	cause := errors.New("gitleaks configured but not found")
	err := Missing(cause, "gitleaks", TranscriptScanArmed)
	if !errors.Is(err, cause) || strings.Split(err.Error(), "\n")[0] != cause.Error() {
		t.Fatalf("error = %q", err.Error())
	}
}

// TestInstallSanitisesTheProgramsOutput: a program's output is untrusted text
// that reaches the terminal and --json through the result, so no escape, C1
// control or bidi override survives into Output or the Summary.
func TestInstallSanitisesTheProgramsOutput(t *testing.T) {
	const hostile = "\x1b[2J8.30.1‮\u009b31m\x07"
	for _, tc := range []struct {
		name string
		fail error
	}{{"verified", nil}, {"failed verify", errors.New("exit status 2")}} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t)
			r.fx.out["gitleaks"] = hostile + "\nsecond\n"
			if tc.fail != nil {
				r.fx.fail["gitleaks"] = tc.fail
			}
			res := r.installer().Install("gitleaks", TranscriptScan, yes)
			if !strings.Contains(res.Output, "8.30.1") {
				t.Fatalf("output lost its text: %q", res.Output)
			}
			for _, s := range []string{res.Output, res.Summary()} {
				if strings.ContainsAny(s, "\x1b‮\u009b\x07") {
					t.Errorf("a control reached the result: %q", s)
				}
			}
		})
	}
}
