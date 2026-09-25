package launch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// renderFixture writes a version-ABSENT dev tree with a readable contract and a
// payload config that ships the manifests, and returns its root.
func renderFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": [".claude-plugin", "README.md"]}`)
	writeFile(t, root, "README.md", "readme\n")
	writeLockstepTree(t, root, "", "", "")
	return root
}

func sampleEntry() ChangelogEntry {
	return ChangelogEntry{
		Tier:      "minor",
		Reason:    "additive itd-67 shipped",
		Date:      time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC),
		SourceSHA: "a1b2c3d4e5f6",
	}
}

// TestRenderPayloadSatisfiesPublicLockstep is the core detector: the rendered
// payload carries the derived version at all three pinned locations and passes
// the SAME public checker a ship would run against it.
func TestRenderPayloadSatisfiesPublicLockstep(t *testing.T) {
	root := renderFixture(t)
	dest := filepath.Join(t.TempDir(), "payload")

	res, err := RenderPayload(PayloadRenderRequest{
		RepoRoot: root, Dest: dest, Version: "0.4.0", Entry: sampleEntry(),
	})
	if err != nil {
		t.Fatalf("render must succeed on a clean tree: %v", err)
	}
	if res.Lockstep.ExitCode != 0 || !res.Lockstep.OK {
		t.Fatalf("rendered payload must pass the public check, got %+v", res.Lockstep)
	}

	vl := filepath.Join(root, versionLocationRelPath)
	if got := CheckLockstep(TreePublic, dest, vl); !got.OK {
		t.Fatalf("independent public check over the payload must pass, got %+v", got)
	}

	plugin, err := loadJSON(filepath.Join(dest, ".claude-plugin/plugin.json"))
	if err != nil {
		t.Fatalf("read the rendered primary manifest: %v", err)
	}
	market, err := loadJSON(filepath.Join(dest, ".claude-plugin/marketplace.json"))
	if err != nil {
		t.Fatalf("read the rendered marketplace: %v", err)
	}
	cases := []struct {
		name string
		doc  any
		ptr  string
		want any
	}{
		{"primary version", plugin, "/version", "0.4.0"},
		{"marketplace version", market, secondaryVersionPointer, "0.4.0"},
		{"changelog version", market, changelogVersionPointer, "0.4.0"},
		{"changelog tier", market, "/plugins/0/changelog/tier", "minor"},
		{"changelog date", market, "/plugins/0/changelog/date", "2026-07-21"},
		{"changelog source_sha", market, "/plugins/0/changelog/source_sha", "a1b2c3d4e5f6"},
		{"preserved name", plugin, "/name", "abcd"},
	}
	for _, tc := range cases {
		got, present := resolvePointer(tc.doc, tc.ptr)
		if !present || got != tc.want {
			t.Errorf("%s: expected %v at %s, got %v (present=%v)", tc.name, tc.want, tc.ptr, got, present)
		}
	}

	// The payload is a payload, not a manifest pair: the non-manifest includes
	// must be copied too, or "renders the release payload" would be a lie.
	if _, err := os.Stat(filepath.Join(dest, "README.md")); err != nil {
		t.Errorf("payload must carry the non-manifest includes: %v", err)
	}
}

// TestRenderPayloadLeavesSourceTreeUnversioned is the adr-19 detector: rendering
// this repository's own payload must leave the committed manifests
// BYTE-IDENTICAL and the dev polarity still OK. A render that stamped the
// working tree would pass every other test here and silently break the premise.
func TestRenderPayloadLeavesSourceTreeUnversioned(t *testing.T) {
	root := repoRootForTest(t)
	manifests := []string{".claude-plugin/plugin.json", ".claude-plugin/marketplace.json"}
	before := make(map[string][]byte, len(manifests))
	for _, rel := range manifests {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		before[rel] = data
	}

	dest := filepath.Join(t.TempDir(), "payload")
	if _, err := RenderPayload(PayloadRenderRequest{
		RepoRoot: root, Dest: dest, Version: "9.9.9", Entry: sampleEntry(),
	}); err != nil {
		t.Fatalf("render this repository's payload: %v", err)
	}

	for _, rel := range manifests {
		after, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("re-read %s: %v", rel, err)
		}
		if string(after) != string(before[rel]) {
			t.Errorf("%s was mutated by the render — adr-19 requires the working tree stay version-absent", rel)
		}
	}
	if res := CheckLockstep(TreeDev, root, filepath.Join(root, versionLocationRelPath)); !res.OK {
		t.Errorf("the working tree must still satisfy the dev polarity after a render, got %+v", res)
	}
}

// TestRenderPayloadRefusals pins every input the render must refuse rather than
// stamp. Each one would otherwise publish a manifest that is wrong in a way the
// lockstep check cannot catch, because the check reads the same wrong value.
func TestRenderPayloadRefusals(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(t *testing.T, root string, req *PayloadRenderRequest)
		wantErr string
	}{
		{
			name:    "empty version",
			mutate:  func(_ *testing.T, _ string, req *PayloadRenderRequest) { req.Version = "" },
			wantErr: "SemVer",
		},
		{
			name:    "leading v",
			mutate:  func(_ *testing.T, _ string, req *PayloadRenderRequest) { req.Version = "v0.4.0" },
			wantErr: "SemVer",
		},
		{
			name:    "two-component version",
			mutate:  func(_ *testing.T, _ string, req *PayloadRenderRequest) { req.Version = "0.4" },
			wantErr: "SemVer",
		},
		{
			name:    "unknown bump tier",
			mutate:  func(_ *testing.T, _ string, req *PayloadRenderRequest) { req.Entry.Tier = "huge" },
			wantErr: "tier",
		},
		{
			name: "blocked contract",
			mutate: func(t *testing.T, root string, _ *PayloadRenderRequest) {
				writeFile(t, root, versionLocationRelPath, `{"blocked": true}`)
			},
			wantErr: "blocked",
		},
		{
			name: "destination inside the repository",
			mutate: func(_ *testing.T, root string, req *PayloadRenderRequest) {
				req.Dest = filepath.Join(root, "payload")
			},
			wantErr: "inside the repository",
		},
		{
			name: "destination already populated",
			mutate: func(t *testing.T, _ string, req *PayloadRenderRequest) {
				if err := os.MkdirAll(req.Dest, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(req.Dest, "stale.txt"), []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "not empty",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := renderFixture(t)
			req := PayloadRenderRequest{
				RepoRoot: root, Dest: filepath.Join(t.TempDir(), "payload"),
				Version: "0.4.0", Entry: sampleEntry(),
			}
			tc.mutate(t, root, &req)
			_, err := RenderPayload(req)
			if err == nil {
				t.Fatalf("expected a refusal mentioning %q, got none", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected a refusal mentioning %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRenderPayloadRefusesWhenManifestsAreNotShipped proves the render fails
// closed when the payload config would not carry the manifests at all: stamping
// a version into files nobody ships is a silent no-op today and an unversioned
// release tomorrow.
func TestRenderPayloadRefusesWhenManifestsAreNotShipped(t *testing.T) {
	root := renderFixture(t)
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["README.md"]}`)
	_, err := RenderPayload(PayloadRenderRequest{
		RepoRoot: root, Dest: filepath.Join(t.TempDir(), "payload"),
		Version: "0.4.0", Entry: sampleEntry(),
	})
	if err == nil || !strings.Contains(err.Error(), "not in the payload") {
		t.Fatalf("expected a refusal naming the missing manifest, got %v", err)
	}
}

// TestRenderPayloadRefusesAnUnstampableMarketplace proves a marketplace with no
// plugins[0] refuses rather than being invented into existence: a stamp that
// created the missing entry would publish a plausible-looking artefact whose
// plugin record no harness put there.
func TestRenderPayloadRefusesAnUnstampableMarketplace(t *testing.T) {
	root := renderFixture(t)
	writeFile(t, root, ".claude-plugin/marketplace.json", `{"plugins": []}`)
	_, err := RenderPayload(PayloadRenderRequest{
		RepoRoot: root, Dest: filepath.Join(t.TempDir(), "payload"),
		Version: "0.4.0", Entry: sampleEntry(),
	})
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected a refusal naming the unwritable pointer, got %v", err)
	}
}

// TestEditManifestRefusesUncontainedRel proves the payload-write primitive itself
// contains its relative path: a ".." traversal is refused before any join/write,
// so a stamped manifest can never land outside dest (gh-488). A sibling target is
// pre-created with valid JSON so the read succeeds — without the guard,
// filepath.Join(dest, "../escape.json") then walks out of dest and
// WriteFileAtomic stamps the sibling, mutating a file the payload never owned.
func TestEditManifestRefusesUncontainedRel(t *testing.T) {
	parent := t.TempDir()
	dest := filepath.Join(parent, "payload")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	const original = `{"version":"0.0.0"}` + "\n"
	escape := filepath.Join(parent, "escape.json")
	if err := os.WriteFile(escape, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	err := stampPointer(dest, "../escape.json", "/version", "9.9.9")
	if err == nil {
		t.Errorf("stampPointer accepted an uncontained rel; want refusal")
	}
	// The sibling outside dest must be byte-for-byte untouched.
	after, readErr := os.ReadFile(escape)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != original {
		t.Fatalf("a stamp escaped dest and mutated %s: %q", escape, string(after))
	}
}

// TestRenderPayloadRefusesOnItsOwnDrift proves the backstop: if the public check
// over the rendered payload ever disagrees — the shape a newly-pinned location
// the render does not stamp would take — the render REFUSES and carries the
// failing verdict, rather than returning a payload it just proved inconsistent.
func TestRenderPayloadRefusesOnItsOwnDrift(t *testing.T) {
	root := renderFixture(t)
	restore := payloadLockstep
	t.Cleanup(func() { payloadLockstep = restore })
	payloadLockstep = func(tree LockstepTree, repoRoot, vl string) LockstepResult {
		return LockstepResult{Tree: tree, Drifts: []string{"DRIFT public /pinned/later: expected 0.4.0"}, ExitCode: 1}
	}

	res, err := RenderPayload(PayloadRenderRequest{
		RepoRoot: root, Dest: filepath.Join(t.TempDir(), "payload"),
		Version: "0.4.0", Entry: sampleEntry(),
	})
	if !errors.Is(err, ErrPayloadDrift) {
		t.Fatalf("expected ErrPayloadDrift, got %v", err)
	}
	if !strings.Contains(err.Error(), "/pinned/later") {
		t.Errorf("the refusal must name the drifting location, got %v", err)
	}
	if res.Lockstep.OK || len(res.Lockstep.Drifts) != 1 {
		t.Errorf("the failing verdict must travel with the result, got %+v", res.Lockstep)
	}
}

// TestPrecheckPayloadRefusesACaseVariantDestination is the case-folding
// detector over the payload-destination gate (iss-2608270500192615). On a
// case-folding filesystem a --payload-dir spelled ".../REPO/dist" names the same
// on-disk place as ".../repo/dist", and neither filepath.Abs nor EvalSymlinks
// case-canonicalises it — so a byte-sensitive prefix test reports the
// destination as out-of-tree and the render stages the whole payload inside the
// working tree. The gate must refuse it, in BOTH containment directions.
//
// The fold predicate is substituted rather than read off the host, so the
// refusal is proven on a case-sensitive filesystem too — and the unfolded branch
// proves the gate does not refuse two genuinely distinct directories.
func TestPrecheckPayloadRefusesACaseVariantDestination(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	restore := caseFoldsPaths
	t.Cleanup(func() { caseFoldsPaths = restore })

	caseFoldsPaths = func() bool { return true }
	dest := filepath.Join(base, "REPO", "dist")
	_, err := PrecheckPayload(root, dest, PrecheckOptions{Dirty: DirtySkip})
	if err == nil || !strings.Contains(err.Error(), "inside the repository") {
		t.Fatalf("case-variant dest %q under root %q: got %v, want a refusal naming the repository", dest, root, err)
	}
	_, err = PrecheckPayload(filepath.Join(root, "inner"), filepath.Join(base, "REPO"), PrecheckOptions{Dirty: DirtySkip})
	if err == nil || !strings.Contains(err.Error(), "contains the repository") {
		t.Fatalf("case-variant dest containing the root: got %v, want a refusal", err)
	}

	caseFoldsPaths = func() bool { return false }
	if _, err := PrecheckPayload(root, dest, PrecheckOptions{Dirty: DirtySkip}); err != nil && strings.Contains(err.Error(), "the repository") {
		t.Fatalf("on a case-sensitive filesystem %q and %q are distinct directories; got %v", dest, root, err)
	}
}
