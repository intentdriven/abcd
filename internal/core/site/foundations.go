package site

// Principles — the record family keyed by file name.
//
// A principle is a markdown file whose name is its handle and whose H1 is its
// title; it has no lifecycle directory to sit in, and most carry no
// frontmatter at all (a typed one carries four claim keys the site does not
// publish). The lint scan reads the family as a slug-keyed store under the
// handle prn-<stem>; the site reads it here instead — a directory listing and a
// first heading, through the same section walk every other page uses — and
// joins it to the record graph under the stem, the handle every published page
// carries, so the dashboard can count them, the chart can draw them and each
// gets a page (withoutPrincipleNodes keeps the two reads from publishing one
// record twice).
//
// The directory is DERIVED, never configured: it is `principles/` under the
// record root the lint configuration already names. A repository that keeps none
// yields none, and every page that would list them is omitted.

import (
	"os"
	"path"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// principlesDirName is the record format's own name for the store, and
// principleType the store name its records carry.
const (
	principlesDirName = "principles"
	principleType     = "principle"
)

// disciplinesLifecycle is the bucket an intent sits in when it states a rule
// that holds across the whole record rather than a change to ship.
const disciplinesLifecycle = "disciplines"

// maxPrincipleBytes bounds one principle read.
const maxPrincipleBytes = 1 << 20

// PrinciplesDir is the record's principle store, or "" where the lint
// configuration names no record root.
func PrinciplesDir(cfg lint.Config) string {
	if len(cfg.Roots) == 0 {
		return ""
	}
	return path.Join(cfg.Roots[0], principlesDirName)
}

// LoadPrinciples reads the principle store as record nodes.
//
// An absent directory is a state, not a fault: it yields no nodes and no error,
// and the foundations page is omitted rather than rendered empty.
func LoadPrinciples(repoRoot, dir string) ([]lint.RecordNode, error) {
	if dir == "" {
		return nil, nil
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	// The listing itself resolves inside the root, so a symlinked ancestor of the
	// principle store cannot redirect the crawl outside the repository (gh #487).
	d, err := root.Open(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	entries, err := d.ReadDir(-1)
	d.Close()
	if err != nil {
		return nil, err
	}
	var out []lint.RecordNode
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		// The store's own index is not one of its records.
		if strings.EqualFold(name, "README.md") {
			continue
		}
		rel := dir + "/" + name
		data, err := fsutil.ReadGuardedInRoot(root, rel, maxPrincipleBytes)
		if err != nil {
			return nil, err
		}
		// No lifecycle, no status. The store grades its records by neither, and
		// a word invented here to fill the column would be published as though
		// the record had declared it.
		id := strings.TrimSuffix(name, ".md")
		out = append(out, lint.RecordNode{
			ID: id, Type: principleType,
			Title: firstHeading(rel, string(data), id), Path: rel,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// withoutPrincipleNodes drops the principle nodes the record graph carries, and
// any edge touching one.
//
// The lint scan reads the principles store as a declared, slug-keyed record
// family (adr-2609021016270132), so the graph carries each principle under its
// record handle, prn-<stem>. The site publishes the same file under the stem
// alone, the handle its pages and links have always carried, and one record
// published twice under two handles is the duplicate this drop prevents. The
// site's own read is kept rather than the graph's because it is what every
// published page URL is keyed on; the graph's copy adds no field the site
// reads, since a principle carries none of the typed references the graph
// draws edges from.
func withoutPrincipleNodes(g lint.RecordGraph) lint.RecordGraph {
	drop := map[string]bool{}
	nodes := g.Nodes[:0:0]
	for _, n := range g.Nodes {
		if n.Type == principleType {
			drop[n.ID] = true
			continue
		}
		nodes = append(nodes, n)
	}
	if len(drop) == 0 {
		return g
	}
	keep := func(es []lint.RecordEdge) []lint.RecordEdge {
		out := es[:0:0]
		for _, e := range es {
			if !drop[e.From] && !drop[e.To] {
				out = append(out, e)
			}
		}
		return out
	}
	g.Nodes, g.Edges, g.Dangling = nodes, keep(g.Edges), keep(g.Dangling)
	return g
}

// firstHeading is a document's H1, or the fallback where it carries none.
func firstHeading(rel, text, fallback string) string {
	body, consumed := StripFrontmatter(text)
	secs, err := Sections(rel, body, consumed)
	if err != nil {
		return fallback
	}
	for _, s := range secs {
		if s.Level > 0 && s.Title != "" {
			return s.Title
		}
	}
	return fallback
}
