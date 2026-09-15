// Package record is the read side of `abcd <id>`: dispatch on a record id —
// iss-N, itd-N, spc-N, adr-N — and report what the record is, its links, and
// the concrete next move for its lifecycle state (spc-26). It is a leaf
// package over the capture, intent, and spec read paths plus a thin adr
// reader; nothing imports it back, and nothing here writes or knows a
// transport.
package record

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// IDRe is the family gate: the only positional shapes the root command routes
// here. Anything else stays on the unknown-command path, byte-for-byte.
var IDRe = regexp.MustCompile(`^(iss|itd|spc|adr)-[0-9]+$`)

// adrsRelDir is where decisions live. A file is <N>-<slug>.md carrying an
// `id: adr-N` frontmatter line, in either of the store's two id vintages: the
// hand-numbered ordinals `0001`–`0058` and the minted `<yymmddHHMMSS><rrrr>`
// stamp (the 2026-09-01 ruling). One derivation reads both — recordid.ADRFileID
// for the filename, recordid.CanonADRID for the asked id and the frontmatter
// confirm — so this reader can never be the one that reports a present decision
// as absent.
const adrsRelDir = ".abcd/development/decisions/adrs"

// Description is the structured answer to "what is this, and what is my next
// move". Paths are repo-relative; Links carries the record's outbound edges
// (spec_id / intent / promoted_to / resolved_by.* / superseded_by) as present.
type Description struct {
	ID        string            `json:"id"`
	Family    string            `json:"family"` // issue | intent | spec | adr
	Title     string            `json:"title"`
	Status    string            `json:"status"` // folder/bucket, directory-as-truth
	Path      string            `json:"path"`
	Links     map[string]string `json:"links,omitempty"`
	NextMoves []string          `json:"next_moves,omitempty"`
}

// The verb paths a next move may recommend — one closed list, so the
// anti-drift test can assert every recommended verb resolves to a registered
// command in the live cobra tree (a rename breaks the test instead of
// shipping stale advice).
const (
	verbIntentPlan     = "intent plan"
	verbIntentReady    = "intent ready"
	verbSpecClose      = "spec close"
	verbCapturePromote = "capture promote"
	verbCaptureResolve = "capture resolve"
	verbCaptureWontfix = "capture wontfix"
	// verbIntentLink is never written by this package directly: it reaches
	// NextMoves through intent.Ready's spec_link remedies, which the
	// planned-not-ready branch passes through verbatim. It is pinned here so
	// the anti-drift test covers the pass-through path too.
	verbIntentLink = "intent link"
)

// RecommendedVerbPaths enumerates every abcd verb path the next-move table can
// emit — including verbs arriving via passed-through intent.Ready remedies —
// for the surface-side anti-drift test.
func RecommendedVerbPaths() []string {
	return []string{
		verbIntentPlan, verbIntentReady, verbIntentLink, verbSpecClose,
		verbCapturePromote, verbCaptureResolve, verbCaptureWontfix,
	}
}

// Describe locates id in its store (any status folder or bucket) and renders
// the read-only description. A shape-matching id found in no store is an
// error naming the stores searched; Describe never writes.
func Describe(repoRoot, id string) (Description, error) {
	m := IDRe.FindStringSubmatch(id)
	if m == nil {
		return Description{}, fmt.Errorf("record: id %q does not match ^(iss|itd|spc|adr)-[0-9]+$", id)
	}
	switch m[1] {
	case "iss":
		return describeIssue(repoRoot, id)
	case "itd":
		return describeIntent(repoRoot, id)
	case "spc":
		return describeSpec(repoRoot, id)
	default:
		return describeADR(repoRoot, id)
	}
}

// describeIssue renders an issue: status folder, trail, and the capture-side
// next moves.
func describeIssue(repoRoot, id string) (Description, error) {
	res, err := capture.List(capture.ListRequest{RepoRoot: repoRoot, State: capture.StateAll})
	if err != nil {
		return Description{}, err
	}
	for _, iss := range res.Issues {
		if iss.ID != id {
			continue
		}
		d := Description{
			ID:     id,
			Family: "issue",
			Title:  firstLine(iss.Body, iss.Slug),
			Status: string(iss.Status),
			Path:   iss.Path,
			Links:  map[string]string{},
		}
		if iss.PromotedTo != "" {
			d.Links["promoted_to"] = iss.PromotedTo
		}
		if rb := iss.ResolvedBy; rb != nil {
			if rb.Intent != "" {
				d.Links["resolved_by.intent"] = rb.Intent
			}
			if rb.Spec != "" {
				d.Links["resolved_by.spec"] = rb.Spec
			}
			if rb.Commit != "" {
				d.Links["resolved_by.commit"] = rb.Commit
			}
		}
		switch {
		case iss.Status == capture.StateOpen && iss.PromotedTo != "":
			d.NextMoves = []string{
				"promoted — see the intent it graduated into: `abcd " + iss.PromotedTo + "`",
			}
		case iss.Status == capture.StateOpen:
			d.NextMoves = []string{
				"graduate it into an intent: `abcd " + verbCapturePromote + " " + id + " --grounds \"<pursued|deferred|declined>: <text>\"`",
				"or close it: `abcd " + verbCaptureResolve + " " + id + " \"<note>\" --impact <...> --grounds \"<pursued|deferred|declined>: <text>\"` / `abcd " + verbCaptureWontfix + " " + id + " \"<reason>\"`",
			}
		default:
			d.NextMoves = []string{"none — the issue is " + string(iss.Status) + "; the trail is above"}
		}
		return d, nil
	}
	return Description{}, fmt.Errorf("record: %s not found in the issue ledger (open/, resolved/, wontfix/)", id)
}

// describeIntent renders an intent: bucket, links, and the lifecycle next
// move — for a planned intent the read-only ready checks pick between "write
// the spec body" and "implement".
func describeIntent(repoRoot, id string) (Description, error) {
	corpus, err := intent.Load(repoRoot)
	if err != nil {
		return Description{}, err
	}
	it, ok := corpus.Lookup(id)
	if !ok {
		return Description{}, fmt.Errorf("record: %s not found in the intent store (drafts/, planned/, shipped/, disciplines/, superseded/)", id)
	}
	fields, title := readRecordHead(filepath.Join(repoRoot, it.Path), it.Slug)
	d := Description{
		ID:     id,
		Family: "intent",
		Title:  title,
		Status: it.Bucket,
		Path:   it.Path,
		Links:  map[string]string{},
	}
	if !frontmatter.IsNull(it.SpecID) && it.SpecID != "" {
		d.Links["spec_id"] = it.SpecID
	}
	if it.PromotedFrom != "" {
		d.Links["promoted_from"] = it.PromotedFrom
	}
	if sup := fields["superseded_by"].Value; sup != "" && !frontmatter.IsNull(sup) {
		d.Links["superseded_by"] = sup
	}

	switch it.Bucket {
	case intent.BucketDrafts:
		d.NextMoves = []string{
			"hold the planning interview (a human-session act), then `abcd " + verbIntentPlan + " " + id + "`",
		}
	case intent.BucketPlanned:
		ready, err := intent.Ready(repoRoot, id)
		if err != nil {
			return Description{}, err
		}
		if ready.Ready {
			d.NextMoves = []string{
				"ready — implement against the spec body; when done, `abcd " + verbSpecClose + " " + ready.SpecID + "`",
			}
		} else {
			for _, c := range ready.Checks {
				if c.OK {
					continue
				}
				move := c.Name + ": " + c.Detail
				if c.Remedy != "" {
					move += " — " + c.Remedy
				}
				d.NextMoves = append(d.NextMoves, move)
			}
			d.NextMoves = append(d.NextMoves, "re-check with `abcd "+verbIntentReady+" "+id+"`")
		}
	case intent.BucketShipped:
		d.NextMoves = []string{"none — shipped; its audit state lives in the record's Audit Notes"}
	case intent.BucketSuperseded:
		target := d.Links["superseded_by"]
		if target == "" {
			target = "its superseding record"
		}
		d.NextMoves = []string{"read the superseding record: " + target}
	default: // disciplines
		d.NextMoves = []string{"none — a discipline is read, not shipped"}
	}
	return d, nil
}

// describeSpec renders a spec: status, linked intent, and — for an open spec
// — the linked intent's readiness decides the move.
func describeSpec(repoRoot, id string) (Description, error) {
	store, err := spec.Load(repoRoot)
	if err != nil {
		return Description{}, err
	}
	sp, ok := store.Lookup(id)
	if !ok {
		return Description{}, fmt.Errorf("record: %s not found in the spec store (open/, closed/)", id)
	}
	_, title := readRecordHead(filepath.Join(repoRoot, sp.Path), sp.Slug)
	d := Description{
		ID:     id,
		Family: "spec",
		Title:  title,
		Status: sp.Status,
		Path:   sp.Path,
		Links:  map[string]string{"intent": sp.Intent},
	}
	if sp.Status == spec.StatusClosed {
		d.NextMoves = []string{"none — closed; the linked intent is " + sp.Intent}
		return d, nil
	}
	ready, err := intent.Ready(repoRoot, sp.Intent)
	if err != nil {
		// A one-sided link is the intent store's defect, not a dispatch fault:
		// report the spec and point at the intent rather than failing the render.
		d.NextMoves = []string{"the linked intent " + sp.Intent + " could not be read: " + err.Error()}
		return d, nil
	}
	if ready.Ready {
		d.NextMoves = []string{
			"implement against this spec's body; when done, `abcd " + verbSpecClose + " " + id + "`",
		}
	} else {
		d.NextMoves = []string{
			"not ready — the gate defers to the intent's failing checks: `abcd " + verbIntentReady + " " + sp.Intent + "`",
		}
	}
	return d, nil
}

// describeADR probes decisions/adrs/ for the numbered file carrying the id and
// renders it read-only. Decisions are read, never acted on.
func describeADR(repoRoot, id string) (Description, error) {
	// The number the caller asked for is the identity; render the canonical
	// spelling of it regardless of how the caller or the file spelled it.
	canonical := recordid.CanonADRID(id)
	if canonical == "" {
		return Description{}, fmt.Errorf("record: malformed adr id %q", id)
	}
	dir := filepath.Join(repoRoot, adrsRelDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Description{}, fmt.Errorf("record: %s not found — no decision record store at %s", canonical, adrsRelDir)
	}
	for _, e := range entries {
		// Regular files only: a symlinked directory entry in a hostile clone
		// must never reach the read (readRecordHead refuses it again with
		// O_NOFOLLOW — two independent layers).
		if !e.Type().IsRegular() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		// Route by the id the filename claims, padding- and width-agnostic.
		if recordid.ADRFileID(e.Name()) != canonical {
			continue
		}
		rel := filepath.Join(adrsRelDir, e.Name())
		fields, title := readRecordHead(filepath.Join(dir, e.Name()), strings.TrimSuffix(e.Name(), ".md"))
		// The filename routes; the frontmatter id confirms. Both go through the
		// canonicaliser rather than a byte compare: a quoted, zero-padded or
		// case-shifted id is the same handle its record-lint and citation-resolver
		// siblings accept, so the dispatch must not be the one reader that reports
		// a present record as absent. A refused/empty head yields an empty value
		// that canonicalises to "" and matches no handle, preserving the
		// hostile-leaf guard.
		got := strings.Trim(strings.TrimSpace(fields["id"].Value), `"'`)
		if recordid.CanonADRID(got) != canonical {
			continue
		}
		d := Description{
			ID:     canonical,
			Family: "adr",
			Title:  title,
			Status: fields["status"].Value,
			Path:   rel,
			Links:  map[string]string{},
		}
		if sup := fields["superseded_by"].Value; sup != "" && !frontmatter.IsNull(sup) {
			d.Links["superseded_by"] = sup
		}
		d.NextMoves = []string{"none — decisions are read"}
		return d, nil
	}
	return Description{}, fmt.Errorf("record: %s not found in %s", canonical, adrsRelDir)
}

// maxRecordHeadBytes bounds the head read (trust boundary, mirroring the
// stores' own caps).
const maxRecordHeadBytes = 256 * 1024

// readRecordHead reads a record file leniently: its frontmatter fields and its
// first H1 as the title (fallback when the file is refused or has no H1).
// Best-effort in what it renders, but never lenient at the trust boundary: it
// opens with O_NOFOLLOW|O_NONBLOCK (a symlinked or FIFO/device leaf in a
// hostile clone is refused, never followed or blocked on), validates the same
// descriptor (regular file, size cap BEFORE the read), and bounds the read —
// the guarded-read idiom of intent.readRepoFile and spec.readRepoFile, which
// vet the intent/spec paths before this function ever sees them; the adr path
// has no store of its own, so this is its only guard.
func readRecordHead(absPath, fallbackTitle string) (map[string]frontmatter.Field, string) {
	none := map[string]frontmatter.Field{}
	f, err := os.OpenFile(absPath, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return none, fallbackTitle
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > maxRecordHeadBytes {
		return none, fallbackTitle
	}
	data, err := io.ReadAll(io.LimitReader(f, maxRecordHeadBytes+1))
	if err != nil || len(data) > maxRecordHeadBytes {
		return none, fallbackTitle
	}
	lines := strings.Split(string(data), "\n")
	fields := frontmatter.Fields(lines)
	for _, ln := range lines {
		if strings.HasPrefix(ln, "# ") {
			return fields, strings.TrimSpace(strings.TrimPrefix(ln, "# "))
		}
	}
	return fields, fallbackTitle
}

// firstLine returns the first non-blank line of body, whitespace-collapsed,
// or fallback.
func firstLine(body, fallback string) string {
	for _, ln := range strings.Split(body, "\n") {
		if f := strings.Fields(ln); len(f) > 0 {
			return strings.Join(f, " ")
		}
	}
	return fallback
}
