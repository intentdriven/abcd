package launch

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// bundleTreeFor resolves root's payload the way a ship would and returns the
// read side of it, so every resolution test runs against WHAT WOULD SHIP rather
// than against the working tree.
func bundleTreeFor(t *testing.T, root string) PayloadTree {
	t.Helper()
	bundle, err := ResolveBundle(root, nil)
	if err != nil {
		t.Fatalf("resolve the payload bundle: %v", err)
	}
	return NewBundleTree(bundle)
}

// findEntry returns the resolved entry at kind/path, or false.
func findEntry(s InstallSurface, kind SurfaceKind, path string) (SurfaceEntry, bool) {
	for _, e := range s.Entries {
		if e.Kind == kind && e.Path == path {
			return e, true
		}
	}
	return SurfaceEntry{}, false
}

// TestResolveInstallSurfaceDeclarations pins what "the declared surface" IS: the
// UNION of the manifest's explicit declarations and the auto-discovery
// conventions, because the plugin-manifest schema defines every explicit key as
// declaring entries "in addition to those in the <kind>/ directory". Each case
// fixes one rule of that union so the deep tier inherits the same list.
func TestResolveInstallSurfaceDeclarations(t *testing.T) {
	type want struct {
		kind        SurfaceKind
		path        string
		origin      SurfaceOrigin
		requirement SurfaceRequirement
	}
	cases := []struct {
		name    string
		files   map[string]string
		want    []want
		absent  []want
		wantErr bool
	}{
		{
			name: "convention discovery finds the auto-loaded surfaces",
			files: map[string]string{
				"commands/a.md":        "cmd\n",
				"commands/ns/b.md":     "namespaced cmd\n",
				"commands/README.md":   "doc\n",
				"agents/x.md":          "agent\n",
				"agents/README.md":     "doc\n",
				"agents/x/fixture.txt": "not an agent\n",
				"skills/s/SKILL.md":    "skill\n",
				"hooks/hooks.json":     `{"hooks": {}}`,
			},
			want: []want{
				{SurfaceCommand, "commands/a.md", OriginConvention, RequirePayload},
				{SurfaceCommand, "commands/ns/b.md", OriginConvention, RequirePayload},
				{SurfaceCommand, "commands/README.md", OriginConvention, RequirePayload},
				{SurfaceAgent, "agents/x.md", OriginConvention, RequirePayload},
				// iss-110: the loader globs agents/*.md, so a README in that
				// directory IS registered as an agent. The resolver reports what
				// the harness registers; filtering it here would hide the very
				// defect iss-110 tracks.
				{SurfaceAgent, "agents/README.md", OriginConvention, RequirePayload},
				{SurfaceSkill, "skills/s/SKILL.md", OriginConvention, RequirePayload},
				{SurfaceHook, "hooks/hooks.json", OriginConvention, RequirePayload},
			},
			absent: []want{{SurfaceAgent, "agents/x/fixture.txt", OriginConvention, RequirePayload}},
		},
		{
			name: "an explicit declaration supplements, never replaces, the convention",
			files: map[string]string{
				"plugin_extra":   `"commands": "./extra/one.md", "agents": ["./extra/two.md"]`,
				"commands/a.md":  "cmd\n",
				"extra/one.md":   "extra cmd\n",
				"extra/two.md":   "extra agent\n",
				"agents/keep.md": "agent\n",
			},
			want: []want{
				{SurfaceCommand, "commands/a.md", OriginConvention, RequirePayload},
				{SurfaceCommand, "extra/one.md", OriginManifest, RequirePayload},
				{SurfaceAgent, "agents/keep.md", OriginConvention, RequirePayload},
				{SurfaceAgent, "extra/two.md", OriginManifest, RequirePayload},
			},
		},
		{
			name: "a declaration naming a directory expands to the files beneath it",
			files: map[string]string{
				"plugin_extra":      `"commands": "./extra"`,
				"extra/one.md":      "cmd\n",
				"extra/deep/two.md": "cmd\n",
			},
			want: []want{
				{SurfaceCommand, "extra/one.md", OriginManifest, RequirePayload},
				{SurfaceCommand, "extra/deep/two.md", OriginManifest, RequirePayload},
			},
		},
		{
			name:  "a declaration naming nothing in the payload survives as its own entry",
			files: map[string]string{"plugin_extra": `"commands": "./commands/ghost.md"`},
			want:  []want{{SurfaceCommand, "commands/ghost.md", OriginManifest, RequirePayload}},
		},
		{
			name:  "a declaration escaping the payload survives as its own entry",
			files: map[string]string{"plugin_extra": `"agents": "../outside.md"`},
			want:  []want{{SurfaceAgent, "../outside.md", OriginManifest, RequirePayload}},
		},
		{
			name: "a hook command referencing a payload file is a required entry",
			files: map[string]string{
				"hooks/hooks.json": `{"hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "bash \"$CLAUDE_PLUGIN_ROOT/scripts/go.sh\""}]}]}}`,
				"scripts/go.sh":    "#!/bin/sh\n",
			},
			want: []want{
				{SurfaceHook, "hooks/hooks.json", OriginConvention, RequirePayload},
				{SurfaceHook, "scripts/go.sh", OriginHookCommand, RequirePayload},
			},
		},
		{
			name: "a hook command referencing the plugin executable is install-supplied",
			files: map[string]string{
				"hooks/hooks.json": `{"hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "\"${CLAUDE_PLUGIN_ROOT}/abcd\" hook session-start"}]}]}}`,
			},
			want: []want{{SurfaceHook, "abcd", OriginHookCommand, RequireInstalled}},
		},
		{
			name:    "an unparseable plugin manifest is a resolution error",
			files:   map[string]string{"plugin_broken": "yes"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeSurfaceFixture(t, root, tc.files)
			surface, err := ResolveInstallSurface(bundleTreeFor(t, root))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected a resolution error, got surface %+v", surface)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			for _, w := range tc.want {
				got, ok := findEntry(surface, w.kind, w.path)
				if !ok {
					t.Errorf("%s %s was not resolved; entries=%+v", w.kind, w.path, surface.Entries)
					continue
				}
				if got.Origin != w.origin {
					t.Errorf("%s %s: origin %q, want %q", w.kind, w.path, got.Origin, w.origin)
				}
				if got.Requirement != w.requirement {
					t.Errorf("%s %s: requirement %q, want %q", w.kind, w.path, got.Requirement, w.requirement)
				}
			}
			for _, w := range tc.absent {
				if got, ok := findEntry(surface, w.kind, w.path); ok {
					t.Errorf("%s %s must NOT be resolved, got %+v", w.kind, w.path, got)
				}
			}
		})
	}
}

// writeSurfaceFixture writes a shippable plugin payload: a launch-payload config
// that includes everything the case names, a plugin.json (optionally carrying
// extra declaration keys via the "plugin_extra" pseudo-file) and a marketplace
// listing that sources the payload root.
func writeSurfaceFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	extra := files["plugin_extra"]
	broken := files["plugin_broken"]
	includes := map[string]struct{}{".claude-plugin": {}}
	for rel, content := range files {
		if rel == "plugin_extra" || rel == "plugin_broken" {
			continue
		}
		writeFile(t, root, rel, content)
		includes[firstSegment(rel)] = struct{}{}
	}
	list := ""
	for inc := range includes {
		if list != "" {
			list += ", "
		}
		list += `"` + inc + `"`
	}
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": [`+list+`]}`)
	plugin := `{"name": "abcd"`
	if extra != "" {
		plugin += ", " + extra
	}
	plugin += "}"
	if broken != "" {
		plugin = "{not json"
	}
	writeFile(t, root, ".claude-plugin/plugin.json", plugin)
	writeFile(t, root, ".claude-plugin/marketplace.json",
		`{"name": "abcd-marketplace", "plugins": [{"name": "abcd", "source": "./"}]}`)
}

// TestResolveInstallSurfaceCommittedPayload runs the resolver over THIS
// repository's real payload, so the seam is pinned against the surface that
// actually ships rather than only against fixtures.
func TestResolveInstallSurfaceCommittedPayload(t *testing.T) {
	root := repoRootForTest(t)
	surface, err := ResolveInstallSurface(bundleTreeFor(t, root))
	if err != nil {
		t.Fatalf("resolve the committed payload's surface: %v", err)
	}
	if surface.PluginName != "abcd" {
		t.Errorf("plugin name %q, want abcd", surface.PluginName)
	}
	if len(surface.Marketplace) != 1 {
		t.Fatalf("expected exactly one marketplace plugin entry, got %+v", surface.Marketplace)
	}
	mp := surface.Marketplace[0]
	if mp.Name != "abcd" || mp.SourceKind != SourceLocal || mp.Root != "" {
		t.Errorf("adr-28 says the single repo is its own marketplace: got %+v", mp)
	}
	for _, want := range []struct {
		kind SurfaceKind
		path string
	}{
		{SurfaceCommand, "commands/launch.md"},
		{SurfaceAgent, "agents/ruthless-reviewer.md"},
		{SurfaceHook, "hooks/hooks.json"},
	} {
		if _, ok := findEntry(surface, want.kind, want.path); !ok {
			t.Errorf("the committed payload must declare %s %s", want.kind, want.path)
		}
	}
	// The hooks manifest invokes the plugin executable, which the install
	// supplies and the payload never carries.
	bin, ok := findEntry(surface, SurfaceHook, "abcd")
	if !ok || bin.Requirement != RequireInstalled {
		t.Errorf("the plugin executable must resolve as install-supplied, got %+v (ok=%v)", bin, ok)
	}
}

// TestPayloadTreeImplementationsResolveIdentically is the drop-in-upgrade proof:
// resolution over the RESOLVED BUNDLE (what the light tier reads) and over a
// MATERIALISED payload directory (what itd-66's deep tier reads) must produce
// the identical entry list. If these ever diverge, the two tiers disagree about
// what the surface is — which is precisely the divergence this seam exists to
// prevent.
func TestPayloadTreeImplementationsResolveIdentically(t *testing.T) {
	root := repoRootForTest(t)
	dest := filepath.Join(t.TempDir(), "payload")
	if _, err := RenderPayload(PayloadRenderRequest{
		RepoRoot: root, Dest: dest, Version: "9.9.9",
		Entry: ChangelogEntry{Tier: "patch", Reason: "seam parity", Date: time.Now(), SourceSHA: "deadbeef"},
	}); err != nil {
		t.Fatalf("render the payload: %v", err)
	}

	fromBundle, err := ResolveInstallSurface(bundleTreeFor(t, root))
	if err != nil {
		t.Fatalf("resolve from the bundle: %v", err)
	}
	fromDir, err := ResolveInstallSurface(NewDirTree(dest))
	if err != nil {
		t.Fatalf("resolve from the rendered directory: %v", err)
	}
	if !reflect.DeepEqual(fromBundle.Entries, fromDir.Entries) {
		t.Errorf("bundle and directory resolution disagree:\n bundle=%+v\n dir=%+v", fromBundle.Entries, fromDir.Entries)
	}
	if !reflect.DeepEqual(fromBundle.Marketplace, fromDir.Marketplace) {
		t.Errorf("marketplace resolution disagrees:\n bundle=%+v\n dir=%+v", fromBundle.Marketplace, fromDir.Marketplace)
	}
}

// TestResolveInstallSurfaceOnAPayloadThatIsNotAPlugin is iss-2609100506255436.
//
// Where the plugin manifest LIVES is fixed by the harness's discovery rule, and
// pluginManifestFile's own comment says so. Whether the artefact HAS one is a
// different fact, and adr-19's version-location contract is the precedent eight
// lines up: abcd already chose to let a repo declare a release-shaped fact
// rather than assume it. Resolution read the manifest unconditionally and
// readManifest failed on a file that could not be READ, so a repo whose artefact
// is a binary, an application bundle or a library failed resolution before
// anything else ran — an absent plugin reported as a broken payload.
//
// A payload that declares no plugin now resolves to a surface carrying no plugin
// name and no marketplace listing, while everything the conventions do declare
// still resolves. A manifest that is PRESENT and unparseable is untouched by
// this: that payload really is broken, and it still refuses.
func TestResolveInstallSurfaceOnAPayloadThatIsNotAPlugin(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "commands/thing.md", "# thing\n")

	surface, err := ResolveInstallSurface(NewDirTree(root))
	if err != nil {
		t.Fatalf("a payload that is not a plugin must resolve, not refuse: %v", err)
	}
	if surface.PluginName != "" {
		t.Errorf("PluginName = %q, want empty — nothing declared one", surface.PluginName)
	}
	if len(surface.Marketplace) != 0 {
		t.Errorf("Marketplace = %+v, want none", surface.Marketplace)
	}
	if _, ok := findEntry(surface, SurfaceCommand, "commands/thing.md"); !ok {
		t.Errorf("the convention surface was lost with the manifest: %+v", surface.Entries)
	}

	// The negative control: present and unparseable is still a broken payload.
	broken := t.TempDir()
	writeFile(t, broken, pluginManifestFile, "{")
	if _, err := ResolveInstallSurface(NewDirTree(broken)); err == nil {
		t.Errorf("a plugin manifest that does not parse must still refuse")
	}
}
