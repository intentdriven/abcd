package ahoy

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPluginDataDirResolvesEnvThenRootStamp pins the resolver's ladder
// (iss-2609012111168716): a hook's CLAUDE_PLUGIN_DATA wins as given; from a
// terminal the plugin root's .data-dir stamp answers, but only with an
// absolute path naming an existing directory; and every miss carries a story
// that names what was consulted, so the degradation note can repeat it.
func TestPluginDataDirResolvesEnvThenRootStamp(t *testing.T) {
	root := t.TempDir()
	data := t.TempDir()
	stamp := filepath.Join(root, dataDirStampFile)
	writeStamp := func(recorded string) {
		t.Helper()
		if err := os.WriteFile(stamp, []byte("data_dir="+recorded+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("CLAUDE_PLUGIN_DATA", "/from/hook")
	writeStamp(data)
	if got := pluginDataDir(root); got.dir != "/from/hook" || !strings.Contains(got.story, "CLAUDE_PLUGIN_DATA") {
		t.Errorf("the environment must win over the stamp: %+v", got)
	}

	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	if got := pluginDataDir(root); got.dir != data {
		t.Errorf("with no environment the stamp must answer: %+v", got)
	}

	if got := pluginDataDir(""); got.dir != "" || !strings.Contains(got.story, "no plugin root") {
		t.Errorf("no plugin root means no stamp to read, and the story must say so: %+v", got)
	}

	if err := os.Remove(stamp); err != nil {
		t.Fatal(err)
	}
	if got := pluginDataDir(root); got.dir != "" || !strings.Contains(got.story, dataDirStampFile) {
		t.Errorf("an absent stamp is a miss that names the stamp: %+v", got)
	}

	for name, recorded := range map[string]string{
		"relative":     "relative/data",
		"absent":       filepath.Join(t.TempDir(), "gone"),
		"regular file": filepath.Join(root, dataDirStampFile),
	} {
		writeStamp(recorded)
		got := pluginDataDir(root)
		if got.dir != "" {
			t.Errorf("%s: a recorded path that is not an existing absolute directory must not be followed: %+v", name, got)
		}
		if !strings.Contains(got.story, "CLAUDE_PLUGIN_DATA is unset") || !strings.Contains(got.story, dataDirStampFile) {
			t.Errorf("%s: the story must name both sources: %+v", name, got)
		}
	}

	if err := os.WriteFile(stamp, []byte("release_tag=v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := pluginDataDir(root); got.dir != "" {
		t.Errorf("a stamp with no data_dir line records nothing: %+v", got)
	}

	// The explanation for a directory that resolved but holds no cache names the
	// directory and the gap, never a bare "unavailable".
	writeStamp(data)
	if why := pluginDataDir(root).explainMissingCache(); !strings.Contains(why, "recorded checksum") || !strings.Contains(why, dataDirStampFile) {
		t.Errorf("explainMissingCache must say the resolved directory holds no verified artefact: %q", why)
	}
}

// The harness's persistent per-plugin data directory (spc-35) is always an
// absolute path, never inside a project checkout, and never world-writable.
// A CLAUDE_PLUGIN_DATA of any other shape is not that directory, so `ahoy
// install` must not treat what it finds there as a verified release artefact
// (sub-finding of GHSA-4q78-ccfv-f374): a relative value resolves against the
// checkout the verb runs in, an in-checkout value blesses committed bytes as
// the owned PATH binary, and a world-writable cache lets any local user swap
// both the artefact and its recorded hash. The design fork the parent record
// holds open (binding the cache to an attestation the env cannot supply) is
// untouched by these; they only refuse shapes the harness never produces.

// assertCacheIgnored: the owned copy was not written from the cache, no
// provenance record vouches for the untrusted artefact, and a note names the
// variable.
//
// A record does now exist — install degraded to the pinned symlink, and every
// entry abcd installs is recorded, because the hook shims run only a recorded
// one. So the assertion is on WHAT it vouches for: the entry that was actually
// written, carrying the digest of the plugin-root binary the link resolves to,
// never the untrusted cache artefact's.
func assertCacheIgnored(t *testing.T, target string, res InstallResult) {
	t.Helper()
	if fi, err := os.Lstat(target); err == nil && fi.Mode().IsRegular() {
		if got, _ := os.ReadFile(target); string(got) == string(cacheArtefact) {
			t.Fatalf("install copied the cache artefact out of an untrusted data dir into %s", target)
		}
	}
	if rec, ok := readPathEntry(); ok {
		if !sameEntry(rec.path, target) {
			t.Fatalf("install recorded %q, which is not the entry it wrote at %s; notes: %v", rec.path, target, res.Notes)
		}
		sum := sha256.Sum256(cacheArtefact)
		if rec.sha == hex.EncodeToString(sum[:]) {
			t.Fatalf("install recorded the untrusted cache artefact's digest as this machine's abcd; notes: %v", res.Notes)
		}
	}
	said := false
	for _, n := range res.Notes {
		if strings.Contains(n, "CLAUDE_PLUGIN_DATA") {
			said = true
		}
	}
	if !said {
		t.Fatalf("no note names the ignored data directory; notes: %v", res.Notes)
	}
}

func adoptableRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestInstallIgnoresRelativeDataCache(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	work := t.TempDir()
	seedDataCacheAt(t, filepath.Join(work, "data"), cacheArtefact)
	t.Chdir(work)
	t.Setenv("CLAUDE_PLUGIN_DATA", "data")

	res, err := Install(adoptableRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertCacheIgnored(t, filepath.Join(binDir, "abcd"), res)
}

func TestInstallIgnoresDataCacheInsideTheCheckout(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	repo := adoptableRepo(t)
	data := filepath.Join(repo, ".plugin-data")
	seedDataCacheAt(t, data, cacheArtefact)
	t.Setenv("CLAUDE_PLUGIN_DATA", data)

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertCacheIgnored(t, filepath.Join(binDir, "abcd"), res)

	// Detection agrees: the pinned symlink the install degraded to is not
	// offered a heal from a cache that lives inside the checkout.
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if g := gapByID(det.Gaps, "symlink.legacy"); g != nil {
		t.Fatalf("detection offers a heal from an in-checkout cache: %+v", *g)
	}
}

func TestInstallIgnoresWorldWritableDataCache(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	data := seedDataCache(t, cacheArtefact)
	if err := os.Chmod(filepath.Join(data, "cache"), 0o777); err != nil {
		t.Fatal(err)
	}

	res, err := Install(adoptableRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertCacheIgnored(t, filepath.Join(binDir, "abcd"), res)
}
