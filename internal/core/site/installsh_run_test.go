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

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The served installer, run. The surface tests above hold its text to the
// other install forms; these run the committed template in a temp HOME with
// curl and gh stubbed on PATH, so its two security behaviours are proven by
// what it does rather than by what it says:
//
//   - iss-2610120617521209 (the product thinker's ruling of 2026-10-12): the
//     download is verified with `gh attestation verify` when gh is present,
//     and a failed verification installs nothing; without gh, or with a gh
//     that is not signed in, the checksum-verified binary installs with a
//     warning naming the hand check.
//   - iss-2610120601505029: a verified install records the newest-release mark
//     the bootstrap and abcd update keep, with the same atomic owner-only
//     write, and only ever raises it.

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

// runInstaller runs the installer as abcd site build renders it from the
// committed template (the script abcdev.app serves at /install.sh), with HOME,
// curl and gh sandboxed.
// The stub curl answers the two release downloads from a fixture directory by
// the URL's last segment; the release binary is a script that reports
// installedTag; prepare runs against the sandbox home before the install.
func runInstaller(t *testing.T, gh ghBehaviour, prepare func(home string)) installerRun {
	t.Helper()
	script := filepath.Join(t.TempDir(), "install.sh")
	tmpl, err := os.ReadFile(filepath.Join(repoRoot(), "site-src", "install.sh.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, renderInstallScript(tmpl, fixtureStamp), 0o755); err != nil {
		t.Fatal(err)
	}
	return runInstallForm(t, []string{"sh", script}, gh, prepare)
}

// runInstallForm runs argv, one form of the install command, in the sandbox
// runInstaller describes. The stub release serves the binary under every
// released OS name, so a per-OS one-liner runs on either OS.
func runInstallForm(t *testing.T, argv []string, gh ghBehaviour, prepare func(home string)) installerRun {
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

	// The stub knows only --version, as abcd does: anything else is an unknown
	// command, exit 2, so a form that ran `abcd version` would fail here as it
	// fails for a reader (iss-2610120704269048).
	r.release = []byte("#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'abcd " + installedTag + "'; echo '  install:   pinned'; exit 0; fi\n" +
		"echo \"Error: unknown command \\\"$1\\\" for \\\"abcd\\\"\" >&2\nexit 2\n")
	release := t.TempDir()
	sum := sha256.Sum256(r.release)
	manifest := ""
	for _, goos := range []string{"darwin", "linux"} {
		asset := "abcd-" + goos + "-" + runtime.GOARCH
		if err := os.WriteFile(filepath.Join(release, asset), r.release, 0o644); err != nil {
			t.Fatal(err)
		}
		manifest += hex.EncodeToString(sum[:]) + "  " + asset + "\n"
	}
	manifest += strings.Repeat("0", 64) + "  abcd-other-platform\n"
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

	cmd := exec.Command(argv[0], argv[1:]...)
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

func (r installerRun) mark(t *testing.T) (string, bool) {
	t.Helper()
	raw, err := os.ReadFile(abcdhome.Path(r.home, "release-mark"))
	if err != nil {
		return "", false
	}
	return string(raw), true
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
	if _, ok := r.mark(t); ok {
		t.Error("a refused install records no release mark")
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

// TestInstallerEndsWithTheInstalledVersion: the install's last line is the
// version the installed binary reports through --version, the one way abcd
// prints it, so an install that succeeded ends in success rather than in an
// unknown-command error (iss-2610120704269048).
func TestInstallerEndsWithTheInstalledVersion(t *testing.T) {
	r := runInstaller(t, ghVerifies, nil)
	if r.err != nil {
		t.Fatalf("a verified install must exit 0: %v\n%s", r.err, r.out)
	}
	lines := strings.Split(strings.TrimSpace(r.out), "\n")
	if last := lines[len(lines)-1]; strings.TrimSpace(last) != "install:   pinned" || !strings.Contains(r.out, "abcd "+installedTag+"\n") {
		t.Errorf("the install must end with the installed binary's --version report:\n%s", r.out)
	}
	if strings.Contains(r.out, "unknown command") {
		t.Errorf("the install ran a command abcd does not have:\n%s", r.out)
	}
}

// --- the newest-release mark (iss-2610120601505029) ---

func TestInstallerRecordsTheReleaseMark(t *testing.T) {
	for _, gh := range []ghBehaviour{ghVerifies, ghAbsent} {
		r := runInstaller(t, gh, nil)
		if r.err != nil {
			t.Fatalf("install failed: %v\n%s", r.err, r.out)
		}
		got, ok := r.mark(t)
		if !ok || got != "release_tag="+installedTag+"\n" {
			t.Fatalf("a fresh install must record the installed release as the mark; got %q (%v)\n%s", got, ok, r.out)
		}
		fi, err := os.Stat(abcdhome.Path(r.home, "release-mark"))
		if err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("the mark must be mode 0600: %v %v", fi.Mode(), err)
		}
		if di, err := os.Stat(abcdhome.Path(r.home)); err != nil || di.Mode().Perm() != 0o700 {
			t.Errorf("the home it lives in must be mode 0700: %v %v", di.Mode(), err)
		}
		left, _ := filepath.Glob(abcdhome.Path(r.home, ".release-mark.*"))
		if len(left) != 0 {
			t.Errorf("the atomic write must leave no temp file: %v", left)
		}
	}
}

func TestInstallerOnlyRaisesTheMark(t *testing.T) {
	for _, tc := range []struct{ had, want string }{
		{"v0.13.4", installedTag}, // older: raised
		{"v0.15.0", "v0.15.0"},    // newer: kept
	} {
		r := runInstaller(t, ghVerifies, func(home string) {
			if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(abcdhome.Path(home, "release-mark"), []byte("release_tag="+tc.had+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		})
		if r.err != nil {
			t.Fatalf("install failed: %v\n%s", r.err, r.out)
		}
		if got, _ := r.mark(t); got != "release_tag="+tc.want+"\n" {
			t.Errorf("with a mark of %s the install must leave %s; got %q", tc.had, tc.want, got)
		}
	}
}

// TestInstallerWritesNoMarkThroughASymlinkedHome: abcd refuses a
// ~/.abcd.noindex that is a symlink rather than following it, and the
// installer writes nothing through one: the binary installs, and the line
// says the mark was not recorded and why.
func TestInstallerWritesNoMarkThroughASymlinkedHome(t *testing.T) {
	elsewhere := t.TempDir()
	r := runInstaller(t, ghVerifies, func(home string) {
		if err := os.Symlink(elsewhere, abcdhome.Path(home)); err != nil {
			t.Fatal(err)
		}
	})
	if r.err != nil {
		t.Fatalf("install failed: %v\n%s", r.err, r.out)
	}
	if _, err := os.Stat(filepath.Join(elsewhere, "release-mark")); !os.IsNotExist(err) {
		t.Errorf("the mark was written through the symlink: %v", err)
	}
	if !strings.Contains(r.out, "symlink") {
		t.Errorf("the run must say the mark was not recorded because of the symlink:\n%s", r.out)
	}
}
