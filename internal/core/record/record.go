// Package record is the read side of `abcd <id>`: dispatch on a record id —
// iss-N, itd-N, spc-N, adr-N — and report what the record is, its links, and
// the concrete next move for its lifecycle state (spc-26). It is a leaf
// package over the capture, intent, and spec read paths plus a thin adr
// reader; nothing imports it back, and nothing here writes or knows a
// transport.
package record

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

// ErrSkippedRecord marks a record whose file IS in its store but that the
// store's reader skipped on read — an unknown or retired property, a malformed
// frontmatter block, a hostile leaf. It is a different answer from "not found":
// the record exists and cannot be described until its file is repaired, so the
// fault names the file and the reader's own reason (iss-2609240200426413).
var ErrSkippedRecord = errors.New("skipped on read")

// Description is the structured answer to "what is this, and what is my next
// move". Paths are repo-relative; Links carries the record's outbound edges
// (spec_id / intent / related_intents / related_issues / resolved_by.* /
// superseded_by) as present.
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
	verbIntentUnhold   = "intent unhold"
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
		verbIntentPlan, verbIntentReady, verbIntentLink, verbIntentUnhold, verbSpecClose,
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
		if len(iss.RelatedIntents) > 0 {
			d.Links["related_intents"] = strings.Join(iss.RelatedIntents, ", ")
		}
		// Promoted is the two-sided join (itd-4 AC3), not the list: an issue may
		// name an intent it was only related to.
		promotedInto, err := capture.PromotedInto(repoRoot, iss)
		if err != nil {
			return Description{}, err
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
		case iss.Status == capture.StateOpen && promotedInto != "":
			d.NextMoves = []string{
				"promoted — see the intent it graduated into: `abcd " + promotedInto + "`",
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
	// Absent from the parsed issues is not absent from the ledger: a file the
	// reader skipped is still there, and answering "not found" for it sends the
	// reader looking elsewhere for a record sitting in plain sight. The skip
	// reason is the reader's own text — the line `abcd capture list` prints
	// beside the same file — so a reason that carries a remedy (a retired
	// property names `abcd capture migrate --apply`) carries it here unchanged.
	if sk, ok := skippedIssue(res.Skipped, id); ok {
		return Description{}, fmt.Errorf("record: %s is in the issue ledger but was %w: %s: %s",
			id, ErrSkippedRecord, sk.Path, sk.Error)
	}
	return Description{}, fmt.Errorf("record: %s not found in the issue ledger (open/, resolved/, wontfix/)", id)
}

// skippedIssueNumRe reads a skipped file's claimed number the way the reader
// admitted the file to the roster: a base name beginning `iss-` and a digit
// (capture's name-claim test), with the whole leading digit run captured. The
// strict record-filename grammar is the wrong reader here — a malformed
// FILENAME is a class the roster reports, and the strict grammar drops it.
var skippedIssueNumRe = regexp.MustCompile(`^iss-([0-9]+)`)

// skippedIssue finds the skipped-roster entry whose FILENAME claims id. The
// filename is the only identity a skipped record has — its frontmatter is what
// the reader refused — so its leading digit run is compared by number: a
// zero-padded name cannot miss its id, and a sibling whose number merely
// begins with the id's digits cannot answer for it.
func skippedIssue(skipped []capture.SkipRecord, id string) (capture.SkipRecord, bool) {
	want, err := strconv.Atoi(strings.TrimPrefix(id, "iss-"))
	if err != nil {
		return capture.SkipRecord{}, false
	}
	for _, sk := range skipped {
		m := skippedIssueNumRe.FindStringSubmatch(filepath.Base(sk.Path))
		if m == nil {
			continue
		}
		if n, err := strconv.Atoi(m[1]); err == nil && n == want {
			return sk, true
		}
	}
	return capture.SkipRecord{}, false
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
	// An intent owns one or more specs (adr-2609151513118583). The scalar spec_id
	// names the spec it was planned with, so for an intent that owns more than one
	// it is a true but partial answer: list the whole set with each spec's status,
	// which is what says "half of this is delivered and the rest is open". A
	// single-spec intent gets no second line — spec_id already said it.
	// A store that cannot be read is SAID so rather than rendered as silence: an
	// unreadable store and a single-spec intent produce the same absent line, and
	// the second is a claim about the record that this page would then be making
	// without having looked.
	realising, specErr := specsRealising(repoRoot, id)
	switch {
	case specErr != nil:
		d.Links["specs"] = "(store unreadable: " + specErr.Error() + ")"
	case len(realising) > 1:
		d.Links["specs"] = strings.Join(realising, ", ")
	}
	if len(it.RelatedIssues) > 0 {
		d.Links["related_issues"] = strings.Join(it.RelatedIssues, ", ")
	}
	if sup := fields["superseded_by"].Value; sup != "" && !frontmatter.IsNull(sup) {
		d.Links["superseded_by"] = sup
	}
	if it.Held != "" {
		d.Links["held"] = it.Held
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
	// A hold is the next move, in front of whatever the bucket would otherwise
	// say: `intent plan` and `spec close` refuse a held record, so a page that
	// led with "plan it" would be telling its reader to run a verb that will
	// refuse. The row names the reason and the verb that lifts it
	// (iss-2609200830076665). What the row can vouch for is the trust boundary
	// hold.go states: the loader read a `held:` line, and a legal line typed by
	// hand reads exactly as the verb's write does; a value in a shape no verb
	// writes is reported as that, never as a reason, and sent to the rule that
	// names the line — as is a legal value on a record in a bucket no verb can
	// hold, where `intent unhold` refuses too and the only remedy is the hand
	// edit that put it there.
	if move, ok := holdMove(it, id); ok {
		d.NextMoves = append([]string{move}, d.NextMoves...)
	}
	return d, nil
}

// holdMove renders the hold row for a held record, and reports false for a
// record that carries no `held:` key at all.
func holdMove(it intent.Intent, id string) (string, bool) {
	holdable := it.Bucket == intent.BucketDrafts || it.Bucket == intent.BucketPlanned
	switch {
	case it.HeldMalformed:
		return "held, but the `" + intent.HeldKey + ":` value is in a shape no verb writes — `abcd " + verbIntentPlan +
			"` refuses it, and `abcd " + verbIntentUnhold + "` will not remove what it could not have written; repair or remove the line by hand (record-lint's record_provenance rule names it)", true
	case it.Held != "" && !holdable:
		// hold and unhold both refuse this bucket, and spec close refuses a held
		// record before it moves, so the key is here by hand; sending the reader
		// to unhold would name a verb that refuses.
		return "held — " + it.Held + "; but " + it.Bucket + "/ is a bucket no verb can hold (`abcd intent hold` and `abcd " + verbIntentUnhold +
			"` both refuse it, and `abcd " + verbSpecClose + "` refuses a held record before it moves), so the `" + intent.HeldKey +
			":` line was written by hand — remove it by hand (record-lint's record_provenance rule reports it)", true
	case it.Held != "":
		return "held — " + it.Held + "; `abcd " + verbIntentUnhold + " " + id + "` lifts the hold, and `abcd " + verbIntentPlan +
			"` and `abcd " + verbSpecClose + "` refuse until then", true
	}
	return "", false
}

// specsRealising renders every spec that names the given intent as
// "<id> (<status>)", in minting order — the set the 1:n link makes derivable
// from the spec store alone.
//
// A store that cannot be read is RETURNED as an error, not swallowed. Failing
// the whole render over a supplementary line would turn a link report into an
// outage, so the caller still renders the page — but it renders the failure
// where the line would have been. Dropping it silently was worse than either:
// the absent line is exactly what a single-spec intent renders, so a multi-spec
// intent whose store happened to be unreadable was presented, confidently, as an
// intent with one spec.
func specsRealising(repoRoot, intentID string) ([]string, error) {
	store, err := spec.Load(repoRoot)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, sp := range store.SpecsForIntent(intentID) {
		out = append(out, sp.ID+" ("+sp.Status+")")
	}
	return out, nil
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
	if members := sp.Members(); len(members) > 1 {
		return describeBundleSpec(repoRoot, store, sp, members, d), nil
	}
	if sp.Status == spec.StatusClosed {
		// A closed spec whose intent still has open specs delivered part of it: say
		// which sibling the intent is now waiting on, rather than leaving the reader
		// to wonder why the intent did not ship (adr-2609151513118583).
		if open := store.OpenSpecsForIntent(sp.Intent); len(open) > 0 {
			ids := make([]string, len(open))
			for i, s := range open {
				ids[i] = s.ID
			}
			d.NextMoves = []string{"none — closed; " + sp.Intent + " stays planned until " + strings.Join(ids, ", ") + " closes"}
			return d, nil
		}
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

// describeBundleSpec renders a bundle's shared spec through every member it
// lists, as its close reads them (iss-2609261215160156): the links name them
// all, a superseded member is passed over and named, and the move reads each
// member still in force — the open specs a closed one waits on, or the
// readiness an open one defers to.
func describeBundleSpec(repoRoot string, store spec.Store, sp spec.Spec, members []string, d Description) Description {
	d.Links["intents"] = strings.Join(members, ", ")
	var live, superseded []string
	corpus, err := intent.Load(repoRoot)
	for _, m := range members {
		if err == nil {
			if it, ok := corpus.Lookup(m); ok && it.Bucket == intent.BucketSuperseded {
				superseded = append(superseded, m)
				continue
			}
		}
		live = append(live, m)
	}
	passedOver := ""
	if len(superseded) > 0 {
		passedOver = " (" + strings.Join(superseded, ", ") + " superseded, passed over)"
	}
	if len(live) == 0 {
		d.NextMoves = []string{"none — every member of bundle " + sp.Bundle + " is superseded" + passedOver}
		return d
	}
	if sp.Status == spec.StatusClosed {
		var waits []string
		for _, m := range live {
			if open := store.OpenSpecsForIntent(m); len(open) > 0 {
				ids := make([]string, len(open))
				for i, s := range open {
					ids[i] = s.ID
				}
				waits = append(waits, m+" stays planned until "+strings.Join(ids, ", ")+" closes")
			}
		}
		if len(waits) > 0 {
			d.NextMoves = []string{"none — closed; " + strings.Join(waits, "; ") + passedOver}
			return d
		}
		d.NextMoves = []string{"none — closed; the linked intents are " + strings.Join(live, ", ") + passedOver}
		return d
	}
	var unread, notReady []string
	for _, m := range live {
		ready, err := intent.Ready(repoRoot, m)
		switch {
		case err != nil:
			unread = append(unread, "the linked intent "+m+" could not be read: "+err.Error())
		case !ready.Ready:
			notReady = append(notReady, "`abcd "+verbIntentReady+" "+m+"`")
		}
	}
	switch {
	case len(unread) > 0:
		d.NextMoves = unread
	case len(notReady) > 0:
		d.NextMoves = []string{"not ready — the gate defers to the failing checks of " + strings.Join(notReady, ", ") + passedOver}
	default:
		d.NextMoves = []string{"implement against this spec's body; when done, `abcd " + verbSpecClose + " " + sp.ID + "`" + passedOver}
	}
	return d
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
