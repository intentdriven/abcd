package launch

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// parityRepo is a committed payload repository tagged v0.1.0, then moved on by
// one commit that changes README.md, removes commands/b.md and adds
// commands/c.md — one path of each kind the diff reports.
func parityRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	repo := gittest.NewRepo(t)
	root := repo.Root()
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": [".claude-plugin", "commands", "README.md"]}`)
	writeLockstepTree(t, root, "", "", "")
	writeFile(t, root, "README.md", "readme at the tag\n")
	writeFile(t, root, "commands/a.md", "---\ndescription: a\n---\nA\n")
	writeFile(t, root, "commands/b.md", "---\ndescription: b\n---\nB\n")
	repo.Commit("the first release")
	repo.Git("tag", "v0.1.0")

	writeFile(t, root, "README.md", "readme after the tag\n")
	repo.Remove("commands/b.md")
	writeFile(t, root, "commands/c.md", "---\ndescription: c\n---\nC\n")
	repo.Commit("work after the release")
	return repo
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func parityEntry(rep ParityReport, path string) (ParityEntry, bool) {
	for _, e := range rep.Entries {
		if e.Path == path {
			return e, true
		}
	}
	return ParityEntry{}, false
}

func resolveOrFatal(t *testing.T, root string) Bundle {
	t.Helper()
	b, err := ResolveBundle(root, nil)
	if err != nil {
		t.Fatalf("resolve the payload: %v", err)
	}
	return b
}

// TestParityAgainstARenderAtTheTagReportsEveryChange is AC3 over the disk-only
// baseline: the previous release's payload is rendered fresh at its tag, and
// every path added, changed or removed since is listed with its digest.
func TestParityAgainstARenderAtTheTagReportsEveryChange(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()

	rep := PayloadParity(root, resolveOrFatal(t, root), ParityInput{Baseline: "v0.1.0"})
	if rep.Refused {
		t.Fatalf("a readable baseline must not refuse: %s", rep.RefusalReason)
	}
	if rep.Source != ParitySourceRenderAtTag || rep.Baseline != "v0.1.0" {
		t.Fatalf("the baseline must be a render at v0.1.0, got source %q baseline %q", rep.Source, rep.Baseline)
	}
	want := map[string]struct {
		change         ParityChange
		digest, before string
	}{
		"README.md":     {ParityChanged, sha256Hex("readme after the tag\n"), sha256Hex("readme at the tag\n")},
		"commands/b.md": {ParityRemoved, "", sha256Hex("---\ndescription: b\n---\nB\n")},
		"commands/c.md": {ParityAdded, sha256Hex("---\ndescription: c\n---\nC\n"), ""},
	}
	if len(rep.Entries) != len(want) {
		t.Fatalf("expected exactly %d changed paths, got %+v", len(want), rep.Entries)
	}
	for path, w := range want {
		e, ok := parityEntry(rep, path)
		if !ok {
			t.Errorf("%s is missing from the diff: %+v", path, rep.Entries)
			continue
		}
		if e.Change != w.change || e.Digest != w.digest || e.BaselineDigest != w.before {
			t.Errorf("%s: want %s %s<-%s, got %+v", path, w.change, w.digest, w.before, e)
		}
	}
	if rep.Added != 1 || rep.Changed != 1 || rep.Removed != 1 || rep.Unchanged == 0 {
		t.Errorf("counts must agree with the entries: %+v", rep)
	}
	// A render at a tag is temp-tree only: the source checkout gains no
	// worktree, no ref and no dirt.
	if out := repo.Git("status", "--porcelain"); out != "" {
		t.Errorf("the parity render left the source tree dirty:\n%s", out)
	}
	if out := repo.Git("worktree", "list"); strings.Count(out, "\n") != 0 {
		t.Errorf("the parity render left a worktree behind:\n%s", out)
	}
}

// TestParityWithNoPreviousReleaseIsAllAdded is AC7's first half: a first launch
// reports every payload path as added and says why.
func TestParityWithNoPreviousReleaseIsAllAdded(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()
	bundle := resolveOrFatal(t, root)

	rep := PayloadParity(root, bundle, ParityInput{})
	if rep.Refused {
		t.Fatalf("a first launch is not a refusal: %s", rep.RefusalReason)
	}
	if rep.Source != ParitySourceNone || rep.Note == "" {
		t.Errorf("a first launch must name its source as none and say why, got %q / %q", rep.Source, rep.Note)
	}
	if rep.Added != len(bundle.Included) || len(rep.Entries) != len(bundle.Included) || rep.Changed+rep.Removed != 0 {
		t.Fatalf("every one of the %d payload paths must be added, got %+v", len(bundle.Included), rep)
	}
	for _, e := range rep.Entries {
		if e.Change != ParityAdded || len(e.Digest) != 64 {
			t.Errorf("%s must be added with its digest, got %+v", e.Path, e)
		}
	}
}

// TestParityWithAnUnreadableBaselineIsANamedRefusal is AC7's second half: a
// baseline that cannot be read refuses by name, and never reads as an empty
// diff.
func TestParityWithAnUnreadableBaselineIsANamedRefusal(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()

	rep := PayloadParity(root, resolveOrFatal(t, root), ParityInput{Baseline: "v9.9.9"})
	if !rep.Refused || !strings.Contains(rep.RefusalReason, "v9.9.9") {
		t.Fatalf("an absent baseline tag must refuse by name, got %+v", rep)
	}
	if len(rep.Entries) != 0 || rep.Added+rep.Changed+rep.Removed+rep.Unchanged != 0 {
		t.Errorf("a refused diff reports no entries, got %+v", rep)
	}
	if err := ValidateBaselineTag(root, "v9.9.9"); err == nil || !strings.Contains(err.Error(), "v9.9.9") {
		t.Errorf("a configured baseline that is not a tag here must be refused by name, got %v", err)
	}
	if err := ValidateBaselineTag(root, "main"); err == nil {
		t.Error("a configured baseline that is not a release tag must be refused")
	}
	if err := ValidateBaselineTag(root, "v0.1.0"); err != nil {
		t.Errorf("a release tag in the checkout is a valid baseline: %v", err)
	}
}

// TestParityAgainstATagThatShippedNoPayloadIsAllAdded: a previous release that
// declared no payload published none, so every path is added, not refused.
func TestParityAgainstATagThatShippedNoPayloadIsAllAdded(t *testing.T) {
	repo := gittest.NewRepo(t)
	root := repo.Root()
	writeFile(t, root, "README.md", "before any payload\n")
	repo.Commit("no payload yet")
	repo.Git("tag", "v0.1.0")
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": [".claude-plugin", "README.md"]}`)
	writeLockstepTree(t, root, "", "", "")
	repo.Commit("a payload")

	bundle := resolveOrFatal(t, root)
	rep := PayloadParity(root, bundle, ParityInput{Baseline: "v0.1.0"})
	if rep.Refused || rep.Added != len(bundle.Included) || !strings.Contains(rep.Note, "no launch payload") {
		t.Fatalf("a baseline that declared no payload must read as all-added with the reason, got %+v", rep)
	}
}

// fakeAssets is the release-asset seam the tests drive; it never touches the
// network.
type fakeAssets struct {
	assets  map[string][]byte
	err     error
	fetched []string
}

func (f *fakeAssets) FetchReleaseAsset(tag, name string) ([]byte, string, bool, error) {
	url := "https://example.com/releases/download/" + tag + "/" + name
	f.fetched = append(f.fetched, url)
	if f.err != nil {
		return nil, url, false, f.err
	}
	data, ok := f.assets[name]
	return data, url, ok, nil
}

// releaseZip packs files into a zip, as the release archive is packed.
func releaseZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestParityAgainstTheReleaseAssetVerifiesAndDiffs: the tag's published
// archive is the baseline when the operator asks for it. It is verified against
// the release's own checksums, the version stamp on the archived manifest reads
// as no change, and the catalog the archive omits by construction is named as
// not compared rather than reported as added.
func TestParityAgainstTheReleaseAssetVerifiesAndDiffs(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()
	zipBytes := releaseZip(t, map[string]string{
		".claude-plugin/plugin.json": "{\n  \"name\": \"abcd\",\n  \"version\": \"0.1.0\"\n}\n",
		"README.md":                  "readme at the tag\n",
		"commands/a.md":              "---\ndescription: a\n---\nA\n",
		"commands/b.md":              "---\ndescription: b\n---\nB\n",
	})
	name := PluginArchiveName("abcd", "0.1.0")
	sums := sha256Hex(string(zipBytes)) + "  " + name + "\n" + strings.Repeat("0", 64) + "  abcd-linux-amd64\n"
	fetch := &fakeAssets{assets: map[string][]byte{"checksums.txt": []byte(sums), name: zipBytes}}

	rep := PayloadParity(root, resolveOrFatal(t, root), ParityInput{Baseline: "v0.1.0", Fetch: fetch})
	if rep.Refused {
		t.Fatalf("a verified asset must not refuse: %s", rep.RefusalReason)
	}
	if rep.Source != ParitySourceReleaseAsset {
		t.Fatalf("the baseline must be the release asset, got %q (%s)", rep.Source, rep.Note)
	}
	if len(rep.Fetched) != 2 {
		t.Errorf("both fetches must be named in the report (loud staging), got %v", rep.Fetched)
	}
	if _, ok := parityEntry(rep, ".claude-plugin/plugin.json"); ok {
		t.Errorf("a version stamp is not a payload change: %+v", rep.Entries)
	}
	if _, ok := parityEntry(rep, marketplaceFile); ok {
		t.Errorf("the catalog is not in the archive and must not read as added: %+v", rep.Entries)
	}
	if len(rep.NotCompared) != 1 || rep.NotCompared[0] != marketplaceFile {
		t.Errorf("the catalog must be named as not compared, got %v", rep.NotCompared)
	}
	for path, change := range map[string]ParityChange{"README.md": ParityChanged, "commands/b.md": ParityRemoved, "commands/c.md": ParityAdded} {
		if e, ok := parityEntry(rep, path); !ok || e.Change != change {
			t.Errorf("%s must be %s, got %+v", path, change, e)
		}
	}
}

// TestParityReleaseAssetChecksumMismatchFailsClosed: bytes the release's own
// manifest does not vouch for are never diffed against, and never replaced by
// a quieter baseline.
func TestParityReleaseAssetChecksumMismatchFailsClosed(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()
	name := PluginArchiveName("abcd", "0.1.0")
	zipBytes := releaseZip(t, map[string]string{"README.md": "x\n"})
	fetch := &fakeAssets{assets: map[string][]byte{
		"checksums.txt": []byte(strings.Repeat("a", 64) + "  " + name + "\n"),
		name:            zipBytes,
	}}
	rep := PayloadParity(root, resolveOrFatal(t, root), ParityInput{Baseline: "v0.1.0", Fetch: fetch})
	if !rep.Refused || !strings.Contains(rep.RefusalReason, "checksum") {
		t.Fatalf("a digest mismatch must refuse, got %+v", rep)
	}
	if len(rep.Entries) != 0 {
		t.Errorf("a refused diff reports no entries, got %+v", rep.Entries)
	}
}

// TestParityReleaseAssetAbsentFallsBackToARenderAtTheTag: a release that
// publishes no verifiable archive is measured by a fresh render at its tag, and
// the report says which baseline it used and why.
func TestParityReleaseAssetAbsentFallsBackToARenderAtTheTag(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()
	for name, fetch := range map[string]*fakeAssets{
		"no checksums manifest":         {assets: map[string][]byte{}},
		"checksums without the archive": {assets: map[string][]byte{"checksums.txt": []byte(strings.Repeat("0", 64) + "  abcd-linux-amd64\n")}},
	} {
		t.Run(name, func(t *testing.T) {
			rep := PayloadParity(root, resolveOrFatal(t, root), ParityInput{Baseline: "v0.1.0", Fetch: fetch})
			if rep.Refused || rep.Source != ParitySourceRenderAtTag || rep.Note == "" {
				t.Fatalf("an absent asset must fall back to a render at the tag and say so, got %+v", rep)
			}
			if rep.Added != 1 || rep.Changed != 1 || rep.Removed != 1 {
				t.Errorf("the fallback diff must be the render's, got %+v", rep)
			}
		})
	}
}

// TestParityReleaseAssetNamedButNotServedRefuses: a manifest that vouches for
// an archive the release does not serve is an unreadable baseline, and a fetch
// that fails in transport is one too.
func TestParityReleaseAssetNamedButNotServedRefuses(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()
	name := PluginArchiveName("abcd", "0.1.0")
	cases := map[string]*fakeAssets{
		"named, not served": {assets: map[string][]byte{"checksums.txt": []byte(strings.Repeat("0", 64) + "  " + name + "\n")}},
		"transport failure": {err: errors.New("connection refused")},
	}
	for label, fetch := range cases {
		t.Run(label, func(t *testing.T) {
			rep := PayloadParity(root, resolveOrFatal(t, root), ParityInput{Baseline: "v0.1.0", Fetch: fetch})
			if !rep.Refused || !strings.Contains(rep.RefusalReason, "v0.1.0") {
				t.Fatalf("must refuse by name, got %+v", rep)
			}
		})
	}
}

// TestParityBaselineErrorFromTheFrontDoorRefuses: a baseline the caller could
// not resolve is a named refusal, not a first launch.
func TestParityBaselineErrorFromTheFrontDoorRefuses(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()
	rep := PayloadParity(root, resolveOrFatal(t, root), ParityInput{BaselineError: "the release tags could not be listed"})
	if !rep.Refused || !strings.Contains(rep.RefusalReason, "could not be listed") || len(rep.Entries) != 0 {
		t.Fatalf("an unresolved baseline must refuse, got %+v", rep)
	}
}

// TestParityUnanchoredBaselineRefusesUnlessTheAssetAnswers: a baseline the
// front door named from CHANGELOG.md because this checkout cannot read it from
// its own tags (a tagless or shallow clone) is never a first launch and never
// a render at a tag the checkout does not hold. Without --fetch-baseline it
// refuses, naming the release and both remedies; with it, the verified release
// asset is the baseline, and an asset that does not answer refuses too
// (iss-2609251902439938).
func TestParityUnanchoredBaselineRefusesUnlessTheAssetAnswers(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()
	repo.Git("tag", "-d", "v0.1.0")
	const why = "CHANGELOG.md dates release 0.1.0, and this checkout holds no release tag"

	rep := PayloadParity(root, resolveOrFatal(t, root), ParityInput{Baseline: "v0.1.0", Unanchored: why})
	if !rep.Refused || rep.Baseline != "v0.1.0" || len(rep.Entries) != 0 || rep.Source == ParitySourceNone {
		t.Fatalf("an unanchored baseline without a fetch must refuse against the named release, got %+v", rep)
	}
	for _, want := range []string{"v0.1.0", why, "git fetch --tags", "--fetch-baseline"} {
		if !strings.Contains(rep.RefusalReason, want) {
			t.Errorf("the refusal must name %q, got %q", want, rep.RefusalReason)
		}
	}

	zipBytes := releaseZip(t, map[string]string{
		".claude-plugin/plugin.json": "{\n  \"name\": \"abcd\",\n  \"version\": \"0.1.0\"\n}\n",
		"README.md":                  "readme at the tag\n",
		"commands/a.md":              "---\ndescription: a\n---\nA\n",
		"commands/b.md":              "---\ndescription: b\n---\nB\n",
	})
	name := PluginArchiveName("abcd", "0.1.0")
	served := &fakeAssets{assets: map[string][]byte{
		"checksums.txt": []byte(sha256Hex(string(zipBytes)) + "  " + name + "\n"),
		name:            zipBytes,
	}}
	rep = PayloadParity(root, resolveOrFatal(t, root), ParityInput{Baseline: "v0.1.0", Unanchored: why, Fetch: served})
	if rep.Refused || rep.Source != ParitySourceReleaseAsset || rep.Added != 1 || rep.Changed != 1 || rep.Removed != 1 {
		t.Fatalf("a verified asset is the unanchored baseline, got %+v", rep)
	}

	rep = PayloadParity(root, resolveOrFatal(t, root), ParityInput{Baseline: "v0.1.0", Unanchored: why, Fetch: &fakeAssets{assets: map[string][]byte{}}})
	if !rep.Refused || rep.Source == ParitySourceRenderAtTag || !strings.Contains(rep.RefusalReason, why) {
		t.Fatalf("an asset that does not answer must refuse, never render a tag this checkout lacks, got %+v", rep)
	}
}

// TestDryRunCarriesTheParityDiff: the preview reports the diff, and a refused
// one is on its would-refuse list.
func TestDryRunCarriesTheParityDiff(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()
	rep, err := DryRun(DryRunRequest{RepoRoot: root, Version: "0.2.0", ExistingTags: []Semver{}, Parity: &ParityInput{Baseline: "v0.1.0"}})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Parity == nil || rep.Parity.Added != 1 || rep.Parity.Changed != 1 || rep.Parity.Removed != 1 {
		t.Fatalf("the preview must carry the diff, got %+v", rep.Parity)
	}
	if !gateRan(rep.Gates, "payload-parity") {
		t.Errorf("the preview must report a payload-parity row, got %+v", rep.Gates)
	}

	rep, err = DryRun(DryRunRequest{RepoRoot: root, Version: "0.2.0", ExistingTags: []Semver{}, Parity: &ParityInput{Baseline: "v9.9.9"}})
	if err != nil {
		t.Fatal(err)
	}
	if !containsSubstring(rep.WouldRefuseOn, "v9.9.9") {
		t.Errorf("an unreadable baseline must be on the would-refuse list, got %v", rep.WouldRefuseOn)
	}
	md := rep.PreflightReport(time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)).Markdown()
	if !strings.Contains(md, "Payload parity") {
		t.Errorf("the pre-flight report must carry the parity section:\n%s", md)
	}
}

// TestParityDigestsTheBytesThatShip: the current side is read from the
// resolved bundle, so an untracked payload file is compared too.
func TestParityDigestsTheBytesThatShip(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()
	if err := os.WriteFile(filepath.Join(root, "commands", "d.md"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep := PayloadParity(root, resolveOrFatal(t, root), ParityInput{Baseline: "v0.1.0"})
	if e, ok := parityEntry(rep, "commands/d.md"); !ok || e.Digest != sha256Hex("new\n") {
		t.Errorf("an untracked payload file ships and must be diffed, got %+v", rep.Entries)
	}
}

func gateRan(gates []GateSummary, name string) bool {
	for _, g := range gates {
		if g.Name == name && g.Status == "ran" {
			return true
		}
	}
	return false
}

func containsSubstring(lines []string, sub string) bool {
	for _, l := range lines {
		if strings.Contains(l, sub) {
			return true
		}
	}
	return false
}
