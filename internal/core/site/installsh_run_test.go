package site

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The served installer, run. The surface tests above hold its text to the
// other install forms; these run the committed template in a temp HOME with
// curl and gh stubbed on PATH, so its security behaviour is proven by
// what it does rather than by what it says:
//
//   - iss-2610120617521209 (the product thinker's ruling of 2026-10-12): the
//     download is verified with `gh attestation verify` when gh is present,
//     and a failed verification installs nothing; without gh, or with a gh
//     that is not signed in, the checksum-verified binary installs with a
//     warning naming the hand check.

const (
	installedTag  = "v0.14.2"
	ghPolicyFlags = "--hostname github.com --repo intentdriven/abcd --signer-workflow intentdriven/abcd/.github/workflows/release.yml"
)

type ghBehaviour int

const (
	ghVerifies ghBehaviour = iota
	ghRefuses
	ghSignedOut
	ghAbsent
)

// installerRun is one run of the template under a sandbox.
type installerRun struct {
	home, binDir string
	out          string
	err          error
	ghLog        string
	release      []byte // the binary the stub release serves
}

// runInstaller runs the committed template with HOME, curl and gh sandboxed.
// The stub curl answers the two release downloads from a fixture directory by
// the URL's last segment; the release binary is a script that reports
// installedTag; prepare runs against the sandbox home before the install.
func runInstaller(t *testing.T, gh ghBehaviour, prepare func(home string)) installerRun {
	t.Helper()
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("the installer runs on macOS and Linux only")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	r := installerRun{home: t.TempDir()}
	r.binDir = filepath.Join(r.home, ".local", "bin")
	if prepare != nil {
		prepare(r.home)
	}

	arch := runtime.GOARCH
	asset := "abcd-" + runtime.GOOS + "-" + arch
	r.release = []byte("#!/bin/sh\n[ \"$1\" = --version ] && echo 'abcd " + installedTag + "' && echo '  install:   pinned'\nexit 0\n")
	release := t.TempDir()
	if err := os.WriteFile(filepath.Join(release, asset), r.release, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(r.release)
	manifest := hex.EncodeToString(sum[:]) + "  " + asset + "\n" + strings.Repeat("0", 64) + "  abcd-other-platform\n"
	if err := os.WriteFile(filepath.Join(release, "checksums.txt"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	stubs := t.TempDir()
	curl := "#!/bin/sh\nfor a in \"$@\"; do url=\"$a\"; done\ncp '" + release + "'/\"${url##*/}\" \"${url##*/}\"\n"
	if err := os.WriteFile(filepath.Join(stubs, "curl"), []byte(curl), 0o755); err != nil {
		t.Fatal(err)
	}
	r.ghLog = filepath.Join(t.TempDir(), "gh.log")
	if gh != ghAbsent {
		verdict := "exit 0"
		switch gh {
		case ghRefuses:
			verdict = "echo 'Error: verifying with issuer \"sigstore.dev\": no matching attestation' >&2; exit 1"
		case ghSignedOut:
			verdict = "echo 'To get started with GitHub CLI, please run:  gh auth login' >&2; exit 4"
		}
		script := "#!/bin/sh\n" +
			"[ \"$1 $2 $3\" = 'attestation verify --help' ] && exit 0\n" +
			"if [ \"$1 $2\" = 'attestation verify' ]; then\n" +
			"\tsum=$( (shasum -a 256 \"$3\" 2>/dev/null || sha256sum \"$3\") | cut -d' ' -f1)\n" +
			"\tprintf '%s\\t%s\\n' \"$sum\" \"$*\" >> '" + r.ghLog + "'\n" +
			"\t" + verdict + "\nfi\nexit 1\n"
		if err := os.WriteFile(filepath.Join(stubs, "gh"), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	script := filepath.Join(t.TempDir(), "install.sh")
	tmpl, err := os.ReadFile(filepath.Join(repoRoot(), "site-src", "install.sh.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, tmpl, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", script)
	cmd.Dir = t.TempDir()
	cmd.Env = []string{
		"HOME=" + r.home,
		"PATH=" + stubs + string(os.PathListSeparator) + installerPathWithout(t, os.Getenv("PATH"), "gh", "curl"),
	}
	out, err := cmd.CombinedOutput()
	r.out, r.err = string(out), err
	return r
}

// installerPathWithout is PATH as one directory of symlinks to every command
// on base except the named ones, so gh and curl are only ever the stubs,
// wherever the machine keeps its own (/usr/bin on a Linux runner).
func installerPathWithout(t *testing.T, base string, names ...string) string {
	t.Helper()
	drop := map[string]bool{}
	for _, n := range names {
		drop[n] = true
	}
	dir := t.TempDir()
	seen := map[string]bool{}
	for _, d := range filepath.SplitList(base) {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if drop[name] || seen[name] {
				continue
			}
			full := filepath.Join(d, name)
			fi, err := os.Stat(full)
			if err != nil || fi.IsDir() || fi.Mode()&0o111 == 0 {
				continue
			}
			seen[name] = true
			if err := os.Symlink(full, filepath.Join(dir, name)); err != nil {
				t.Fatal(err)
			}
		}
	}
	return dir
}

func (r installerRun) installed(t *testing.T) bool {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(r.binDir, "abcd"))
	return err == nil && string(got) == string(r.release)
}

func (r installerRun) ghCalls(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(r.ghLog)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(raw)), "\n")
}

func TestInstallerInstallsADownloadGhVerifies(t *testing.T) {
	r := runInstaller(t, ghVerifies, nil)
	if r.err != nil {
		t.Fatalf("a verified download installs: %v\n%s", r.err, r.out)
	}
	if !r.installed(t) {
		t.Fatalf("the verified binary must be installed:\n%s", r.out)
	}
	calls := r.ghCalls(t)
	if len(calls) != 1 {
		t.Fatalf("gh attestation verify must run once, got %q", calls)
	}
	sum := sha256.Sum256(r.release)
	if !strings.HasPrefix(calls[0], hex.EncodeToString(sum[:])+"\t") {
		t.Errorf("gh must verify the downloaded binary: %q", calls[0])
	}
	if !strings.Contains(calls[0], ghPolicyFlags) {
		t.Errorf("gh must be held to the release workflow's policy %q: %q", ghPolicyFlags, calls[0])
	}
	if strings.Contains(r.out, "WARNING") {
		t.Errorf("a verified install carries no warning:\n%s", r.out)
	}
}

func TestInstallerRefusesADownloadGhCannotVerify(t *testing.T) {
	r := runInstaller(t, ghRefuses, nil)
	if r.err == nil {
		t.Fatalf("a failed verification must refuse:\n%s", r.out)
	}
	if _, err := os.Lstat(filepath.Join(r.binDir, "abcd")); !os.IsNotExist(err) {
		t.Errorf("a failed verification must install nothing: %v", err)
	}
	for _, want := range []string{"build provenance", "no matching attestation", "Nothing is installed"} {
		if !strings.Contains(r.out, want) {
			t.Errorf("the refusal must say %q:\n%s", want, r.out)
		}
	}
}

func TestInstallerWarnsWithoutGh(t *testing.T) {
	r := runInstaller(t, ghAbsent, nil)
	if r.err != nil {
		t.Fatalf("abcd does not require gh: %v\n%s", r.err, r.out)
	}
	if !r.installed(t) {
		t.Fatalf("the checksum-verified binary must be installed:\n%s", r.out)
	}
	assertInstallerWarning(t, r.out, "gh is not installed")
}

func TestInstallerWarnsWhenGhIsNotSignedIn(t *testing.T) {
	r := runInstaller(t, ghSignedOut, nil)
	if r.err != nil {
		t.Fatalf("a gh that is not signed in counts as no gh: %v\n%s", r.err, r.out)
	}
	if !r.installed(t) {
		t.Fatalf("the checksum-verified binary must be installed:\n%s", r.out)
	}
	assertInstallerWarning(t, r.out, "not signed in")
}

func assertInstallerWarning(t *testing.T, out, why string) {
	t.Helper()
	flat := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{"WARNING", "not checked against the signed build record", why, "gh attestation verify", "--repo intentdriven/abcd --signer-workflow intentdriven/abcd/.github/workflows/release.yml"} {
		if !strings.Contains(flat, want) {
			t.Errorf("the warning must say %q:\n%s", want, out)
		}
	}
}
