package positioning

import (
	"os"
	"path/filepath"
	"testing"
)

// TestThisRepositoryHoldsItsCatalogueToTheBlock is iss-2610241606415232: the
// marketplace catalogue's plugin description is registered beside the plugin
// manifest's, so a catalogue that tells a different story from the block is
// reported as drift rather than going unseen. The registration is this
// repository's own .abcd/positioning.json, run over a fixture whose manifest
// says what the block says and whose catalogue does not.
func TestThisRepositoryHoldsItsCatalogueToTheBlock(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cfg, ok, err := LoadConfig(repo)
	if err != nil || !ok {
		t.Fatalf("LoadConfig(this repository) = ok %v, err %v", ok, err)
	}
	var catalogue *Surface
	for i, s := range cfg.Surfaces {
		for _, f := range s.Files {
			if f == ".claude-plugin/marketplace.json" {
				catalogue = &cfg.Surfaces[i]
			}
		}
	}
	if catalogue == nil {
		t.Fatal("no registered surface reads .claude-plugin/marketplace.json")
	}

	root := writeRepo(t, map[string]string{"brief.md": lockupBrief})
	b, err := ParseBlock(root, BlockLocation{File: "brief.md", Heading: "Identity (canonical)"})
	if err != nil {
		t.Fatal(err)
	}
	write := func(desc string) {
		t.Helper()
		body := `{
  "name": "abcd-marketplace",
  "plugins": [
    {
      "description": "` + desc + `",
      "name": "abcd"
    }
  ]
}
`
		path := filepath.Join(root, ".claude-plugin", "marketplace.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := openRepoRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	write(catalogue.render(b))
	if res, err := checkSurface(r, *catalogue, b); err != nil || res.Status != StatusOK {
		t.Fatalf("a catalogue rendered from the block reads %+v, err %v; want ok", res, err)
	}
	write("Host-agnostic configuration layer for development.")
	res, err := checkSurface(r, *catalogue, b)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusDrifted {
		t.Fatalf("a catalogue telling another story reads %q, want drifted", res.Status)
	}
}
