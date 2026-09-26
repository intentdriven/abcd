package intent

// bundle.go is the bundle command and its close (itd-34, spc-2609211859391533
// scopes 1 and 2): several drafts planned as ONE shared spec, and that spec's
// close shipping every member together.
//
// A bundle is a delivery shape, not a dependency graph: its members are
// distinct user moments that only make sense delivered together, so they share
// one spec and move together through planned/ and shipped/. Two rules follow
// from "one shared spec shipped together", and both are enforced here before
// anything is written:
//
//   - a bundle cannot contain its own blocker. A member naming another member
//     in `blocked_by` says one must ship before the other, and a shared spec
//     ships them at the same moment (decision 5, adr-2609212115255771, which
//     replaced the retired same-phase rule with this check);
//   - the person planning names the bundle. The name is what every member
//     carries in its `bundle:` field and what the shared spec's close matches
//     members on, so it is a slug, and one no other record already carries.
//
// Every write happens under the intent store's mint lock, and the plan is
// all-or-nothing: each refusal fires before the spec is minted, and a failure
// after the mint puts every member back where it was and takes the spec back.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/relink"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// BundleKey is the frontmatter key a bundle-member names its bundle under.
const BundleKey = "bundle"

// BlockedByKey is the frontmatter key a record names the records it waits on
// under.
const BlockedByKey = "blocked_by"

// BundleOptions parameterises PlanBundle.
type BundleOptions struct {
	// Bundle is the bundle's name, required: the person planning names it
	// (decision 2). It becomes every member's `bundle:` value and the shared
	// spec's slug.
	Bundle string
	// ProductionMode is the disclosure the minted spec carries, as on Plan.
	ProductionMode string
	// Impact is the judgement stamped onto EVERY member, under the rules Plan
	// applies to one; empty stamps nothing.
	Impact string
}

// BundleResult reports a completed PlanBundle: the bundle's name, the ONE spec
// minted for it, and each member as Plan would report it.
type BundleResult struct {
	Bundle      string           `json:"bundle"`
	Spec        spec.Spec        `json:"spec"`
	Members     []PlanResult     `json:"members"`
	Relinked    []relink.Rewrite `json:"relinked,omitempty"`
	RelinkError string           `json:"relink_error,omitempty"`
}

// bundleMember is one member as the plan stages it: what was read under the
// lock, what will be written, and how far the write got, so a failure can put
// it back.
type bundleMember struct {
	it         Intent
	draftAbs   string
	orig       string
	blockedBy  []string
	impact     string
	stamped    int
	plannedRel string
	written    bool
	moved      bool
}

// PlanBundle plans several drafts as one bundle (criterion 1): it refuses a
// member naming another in `blocked_by`, naming the edge, mints ONE spec whose
// `intents:` lists every member, stamps `kind: bundle-member` and the bundle's
// name on each, stamps their scope conditions, links each `spec_id` to the
// shared spec, and moves them all drafts/ → planned/ together. Any refusal
// leaves every member where it was and nothing minted.
func PlanBundle(repoRoot string, ids []string, opts BundleOptions) (BundleResult, error) {
	name := opts.Bundle
	if strings.TrimSpace(name) == "" {
		return BundleResult{}, fmt.Errorf("intent: a bundle is named by the person planning it; --bundle <name> is required to plan several intents as one (nothing moved)")
	}
	if !slugRe.MatchString(name) {
		return BundleResult{}, fmt.Errorf("intent: bundle name %q must be kebab-case, since it becomes the shared spec's filename (nothing moved)", name)
	}
	if len(ids) < 2 {
		return BundleResult{}, fmt.Errorf("intent: a bundle has at least two members; plan one intent without --bundle (nothing moved)")
	}
	for i, id := range ids {
		if !recordid.ValidIntentID(id) {
			return BundleResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", id)
		}
		for _, prev := range ids[:i] {
			if recordid.SameID(prev, id) {
				return BundleResult{}, fmt.Errorf("intent: %s is named twice; a bundle names each member once (nothing moved)", id)
			}
		}
	}
	var (
		sp      spec.Spec
		members []*bundleMember
	)
	if err := withIntentMintLock(repoRoot, func() error {
		// Every judgement is made on what is read HERE, under the lock, and
		// before the mint, so a refusal leaves nothing minted: the corpus the
		// members and the name are judged on included. Judged on a corpus
		// loaded before the lock, a second plan taking the same name in the
		// window went unseen, and two plans named one bundle with two specs
		// (iss-2609261215159796).
		corpus, err := Load(repoRoot)
		if err != nil {
			return err
		}
		if members, err = stageBundleMembers(repoRoot, corpus, ids, name); err != nil {
			return err
		}
		for _, m := range members {
			content, err := readIntentRefusingHold(m.draftAbs, m.it.Path, m.it.ID, "plan")
			if err != nil {
				return err
			}
			if !hasAcceptanceCriteria(content) {
				return fmt.Errorf("intent: %s has no non-empty '## Acceptance Criteria' section (itd-1 discipline); refusing to plan the bundle (nothing moved)", m.it.ID)
			}
			if m.impact, err = resolvePlanImpact(m.it, content, opts.Impact); err != nil {
				return err
			}
			m.orig = content
			m.blockedBy = frontmatterList(content, BlockedByKey)
			m.plannedRel = filepath.Join(IntentsRelDir, BucketPlanned, filepath.Base(m.it.Path))
			if _, err := os.Lstat(filepath.Join(repoRoot, m.plannedRel)); err == nil {
				return fmt.Errorf("intent: refusing to overwrite existing %s (nothing moved)", m.plannedRel)
			}
		}
		if err := refuseBundleBlocker(members); err != nil {
			return err
		}
		store, err := spec.Load(repoRoot)
		if err != nil {
			return err
		}
		for _, m := range members {
			if claimer, ok := store.ByIntent(m.it.ID); ok {
				return fmt.Errorf("intent: %s is already realised by %s; a bundle mints its own shared spec, so plan %s alone or retire that spec first (nothing moved)", m.it.ID, claimer.ID, m.it.ID)
			}
		}
		probeID, err := probeMinter().Mint(specFamily)
		if err != nil {
			return err
		}
		for _, m := range members {
			if err := checkBundleFaceSize(m, name, probeID); err != nil {
				return err
			}
		}

		memberIDs := make([]string, len(members))
		for i, m := range members {
			memberIDs[i] = m.it.ID
		}
		if sp, err = spec.CreateBundle(repoRoot, memberIDs, name, opts.ProductionMode); err != nil {
			return err
		}
		if err := writeBundleMembers(repoRoot, members, name, sp.ID); err != nil {
			rollbackBundleMembers(repoRoot, members)
			if rmErr := os.Remove(filepath.Join(repoRoot, sp.Path)); rmErr != nil && !os.IsNotExist(rmErr) {
				return fmt.Errorf("%w; every member was put back, but the spec minted for the bundle, %s, could not be removed (%v)", err, sp.ID, rmErr)
			}
			return fmt.Errorf("%w; every member was put back and the shared spec taken back (nothing moved)", err)
		}
		return nil
	}); err != nil {
		return BundleResult{}, err
	}

	res := BundleResult{Bundle: name, Spec: sp}
	moves := make([]relink.Move, 0, len(members))
	for _, m := range members {
		it := m.it
		it.Kind = KindBundleMember
		it.Bundle = name
		it.SpecID = sp.ID
		it.Bucket = BucketPlanned
		moves = append(moves, relink.Move{From: it.Path, To: m.plannedRel, MovedNow: true})
		it.Path = m.plannedRel
		res.Members = append(res.Members, PlanResult{Intent: it, Spec: sp, ConditionsStamped: m.stamped, ImpactStamped: m.impact})
	}
	relinked, err := relink.Repoint(repoRoot, moves)
	res.Relinked = relinked
	if err != nil {
		res.RelinkError = err.Error()
	}
	return res, nil
}

// stageBundleMembers looks every member up in corpus and makes the judgements
// the corpus decides: each is a plannable draft that names no other bundle,
// and the bundle's name is one no record outside the bundle already carries.
// PlanBundle calls it under the store lock, on a corpus loaded there.
func stageBundleMembers(repoRoot string, corpus Corpus, ids []string, name string) ([]*bundleMember, error) {
	members := make([]*bundleMember, 0, len(ids))
	for _, id := range ids {
		it, ok := corpus.Lookup(id)
		if !ok {
			return nil, fmt.Errorf("intent: %s not found in any bucket", id)
		}
		if it.Bucket != BucketDrafts {
			return nil, fmt.Errorf("intent: %s is in %s, not drafts; a bundle is planned from drafts, and a planned record joins one through `abcd intent reclassify --kind bundle-member` (nothing moved)", it.ID, it.Bucket)
		}
		if err := refuseIfHeld(it, "plan"); err != nil {
			return nil, err
		}
		if !slugRe.MatchString(it.Slug) {
			return nil, fmt.Errorf("intent: %s has slug %q which must be kebab-case", it.ID, it.Slug)
		}
		if !frontmatter.IsNull(it.SpecID) {
			return nil, fmt.Errorf("intent: %s is a draft with spec_id %q already set (half-planned); refusing to plan", it.ID, it.SpecID)
		}
		if !frontmatter.IsNull(it.Kind) && it.Kind != KindStandalone && it.Kind != KindBundleMember {
			return nil, fmt.Errorf("intent: %s has kind %q; only a standalone or bundle-member draft is planned as a bundle (nothing moved)", it.ID, it.Kind)
		}
		if it.Bundle != "" && it.Bundle != name {
			return nil, fmt.Errorf("intent: %s already names bundle %q, not %q (nothing moved)", it.ID, it.Bundle, name)
		}
		members = append(members, &bundleMember{it: it, draftAbs: filepath.Join(repoRoot, it.Path)})
	}
	// A name another record already carries names ANOTHER bundle: planning into
	// it would merge two bundles' members under one name with two specs.
	for _, other := range corpus.Intents {
		if other.Bundle != name {
			continue
		}
		inBundle := false
		for _, m := range members {
			if m.it.ID == other.ID {
				inBundle = true
			}
		}
		if !inBundle {
			return nil, fmt.Errorf("intent: bundle name %q is already carried by %s (%s); name this bundle something else (nothing moved)", name, other.ID, other.Path)
		}
	}
	return members, nil
}

// refuseBundleBlocker refuses a bundle one of whose members names another in
// `blocked_by`, naming the edge: a shared spec ships its members at one moment,
// so a member that must wait for another cannot share its spec.
func refuseBundleBlocker(members []*bundleMember) error {
	for _, a := range members {
		for _, blocker := range a.blockedBy {
			for _, b := range members {
				if b != a && recordid.SameID(blocker, b.it.ID) {
					return fmt.Errorf("intent: %s names %s in blocked_by; a bundle cannot contain its own blocker, since its members ship together — plan %s first, or drop the edge (nothing moved)",
						a.it.ID, b.it.ID, b.it.ID)
				}
			}
		}
	}
	return nil
}

// bundleFaceFields is the frontmatter a member's first write carries: the
// kind, the bundle's name, and the impact when the run stamps one.
func bundleFaceFields(name, impact string) map[string]string {
	fields := map[string]string{"kind": KindBundleMember, BundleKey: name}
	if impact != "" {
		fields["impact"] = impact
	}
	return fields
}

// checkBundleFaceSize refuses a member whose planned form would not fit under
// the cap its reader enforces, judged with a probe id of the mint's width, as
// checkDraftFaceSize does for Plan.
func checkBundleFaceSize(m *bundleMember, name, probeID string) error {
	stamped, _, err := stampScopeConditions(m.orig, probeMinter())
	if err != nil {
		return err
	}
	fields := bundleFaceFields(name, m.impact)
	fields["spec_id"] = probeID
	final, err := setFrontmatterFields(stamped, fields)
	if err != nil {
		return err
	}
	if len(final) > maxIntentFileBytes {
		return fmt.Errorf("intent: planning %s would produce %d bytes, past the %d-byte cap its own reader enforces; refusing before any write", m.it.Path, len(final), maxIntentFileBytes)
	}
	return nil
}

// writeBundleMembers stamps, moves and links each member in turn, in the order
// Plan uses for one: the kind and bundle while still a draft, the move, then the
// spec_id, so each intermediate record is one the lint accepts. It records how
// far each member got, for rollbackBundleMembers.
func writeBundleMembers(repoRoot string, members []*bundleMember, name, specID string) error {
	for _, m := range members {
		stamped, n, err := stampScopeConditions(m.orig, recordid.Minter{})
		if err != nil {
			return err
		}
		face, err := setFrontmatterFields(stamped, bundleFaceFields(name, m.impact))
		if err != nil {
			return err
		}
		m.written = true
		if err := writeIntentFile(m.draftAbs, m.it.Path, face); err != nil {
			return err
		}
		if _, err := moveIntentToBucket(repoRoot, m.it.Path, BucketPlanned); err != nil {
			return err
		}
		m.moved = true
		linked, err := setFrontmatterFields(face, map[string]string{"spec_id": specID})
		if err != nil {
			return err
		}
		if err := writeIntentFile(filepath.Join(repoRoot, m.plannedRel), m.plannedRel, linked); err != nil {
			return err
		}
		m.stamped = n
	}
	return nil
}

// rollbackBundleMembers puts every member a failed plan touched back in
// drafts/ with the bytes it had, in reverse order. It is best-effort by nature
// — it runs because a write already failed — so each step is attempted whatever
// the one before it did.
func rollbackBundleMembers(repoRoot string, members []*bundleMember) {
	for i := len(members) - 1; i >= 0; i-- {
		m := members[i]
		if m.moved {
			_ = os.Rename(filepath.Join(repoRoot, m.plannedRel), m.draftAbs)
		}
		if m.written {
			_ = writeIntentFile(m.draftAbs, m.it.Path, m.orig)
		}
	}
}

// frontmatterList reads a list-valued frontmatter key in either spelling the
// record uses: the inline flow sequence (`key: [a, b]`) and the block sequence
// (`key:` followed by `- a` lines). An absent or null key yields nothing.
func frontmatterList(content, key string) []string {
	lines := strings.Split(content, "\n")
	f, ok := frontmatter.Fields(lines)[key]
	if !ok {
		return nil
	}
	if v := strings.TrimSpace(f.Value); v != "" {
		if frontmatter.IsNull(v) {
			return nil
		}
		return frontmatter.StringList(v)
	}
	var out []string
	for i := f.Line; i < len(lines); i++ {
		trimmed := strings.TrimSpace(strings.TrimRight(lines[i], "\r"))
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		item, isItem := strings.CutPrefix(trimmed, "- ")
		if !isItem {
			break
		}
		if v := strings.Trim(strings.TrimSpace(frontmatter.StripComment(item)), `"'`); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// BundleMemberClose is one member of a bundle as its shared spec's close left
// it: where it was and is, whether this call moved it, and the fidelity-review
// receipt its ship parked — review runs per member, against the one delivery.
type BundleMemberClose struct {
	Intent         Intent `json:"intent"`
	Moved          bool   `json:"moved"`
	From           string `json:"from"`
	To             string `json:"to"`
	ReceiptID      string `json:"receipt_id,omitempty"`
	ReceiptStatus  string `json:"receipt_status,omitempty"`
	AuditEmitError string `json:"audit_emit_error,omitempty"`
}

// reconcileBundle is Reconcile for a bundle's shared spec (criterion 2): the
// close ships every member whose `bundle:` matches the spec's, together, each
// under the impact rule one intent's close applies. A member superseded out of
// the bundle is passed over and named; every other refusal — a member not
// planned, a one-sided link, a hold, an impact the rule refuses, a member still
// held open by another spec — fires before anything moves, and a failure part
// way through the moves puts every member back, so the members move together or
// not at all. The spec closes last, as on one intent's close, so a failure at a
// move leaves it open and the close retries cleanly.
func reconcileBundle(repoRoot string, store spec.Store, sp spec.Spec, impact string, remainder RemainderRequest) (ReconcileResult, error) {
	if remainder.Slug != "" {
		return ReconcileResult{}, fmt.Errorf("intent: spec %s is bundle %s's shared spec, and --remainder mints a follow-on for ONE intent; a bundle ships its members together, so nothing was minted. Close it whole, or plan the remaining work as a new intent", sp.ID, sp.Bundle)
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return ReconcileResult{}, err
	}
	// The list names every member, the first being `intent:`; a hand-written
	// list that omits it still has it considered, and a repeat once.
	names := append([]string{sp.Intent}, sp.Intents...)
	var (
		members []BundleMemberClose
		skipped []string
		seen    []string
	)
	for _, name := range names {
		if !recordid.ValidIntentID(name) {
			return ReconcileResult{}, fmt.Errorf("intent: spec %s lists %q, which is not a well-formed intent id; refusing to reconcile", sp.ID, name)
		}
		dup := false
		for _, s := range seen {
			dup = dup || recordid.SameID(s, name)
		}
		if dup {
			continue
		}
		seen = append(seen, name)
		it, ok := corpus.Lookup(name)
		if !ok {
			return ReconcileResult{}, fmt.Errorf("intent: %s (listed by spec %s) not found in any bucket; refusing to reconcile", name, sp.ID)
		}
		switch it.Bucket {
		case BucketSuperseded:
			skipped = append(skipped, it.ID)
			continue
		case BucketPlanned, BucketShipped:
		default:
			return ReconcileResult{}, fmt.Errorf("intent: %s is in %s (listed by bundle spec %s); expected planned or shipped — refusing to reconcile", it.ID, it.Bucket, sp.ID)
		}
		if it.Bundle != sp.Bundle {
			skipped = append(skipped, it.ID)
			continue
		}
		claimers := store.SpecsForIntent(it.ID)
		backLinked := false
		for _, c := range claimers {
			backLinked = backLinked || spec.SameNum(it.SpecID, c.ID)
		}
		if !backLinked {
			return ReconcileResult{}, fmt.Errorf("intent: %s spec_id is %q but no spec realising it carries that id (bundle spec %s lists it; bidirectional link disagrees); refusing to reconcile", it.ID, it.SpecID, sp.ID)
		}
		if it.Bucket == BucketPlanned {
			if err := refuseIfHeld(it, "spec close"); err != nil {
				return ReconcileResult{}, err
			}
			if open := otherOpenSpecs(claimers, sp.ID); len(open) > 0 {
				return ReconcileResult{}, fmt.Errorf("intent: bundle member %s is still realised by %s, which is open; a bundle's members ship together, so close %s first (it ships nothing while this spec is open) — nothing moved",
					it.ID, strings.Join(specIDs(open), ", "), strings.Join(specIDs(open), ", "))
			}
		}
		members = append(members, BundleMemberClose{Intent: it, From: it.Bucket, To: it.Bucket})
	}
	if len(members) == 0 {
		return ReconcileResult{}, fmt.Errorf("intent: no member of bundle %s (spec %s) is planned or shipped under that bundle; refusing to reconcile", sp.Bundle, sp.ID)
	}

	// The impact rule per member, ahead of every write.
	stamps := make([]string, len(members))
	for i, m := range members {
		if m.Intent.Bucket != BucketPlanned {
			continue
		}
		if stamps[i], err = resolveShipImpact(repoRoot, m.Intent, impact); err != nil {
			return ReconcileResult{}, err
		}
	}

	type undo struct {
		abs, rel, orig, dstRel string
		moved                  bool
	}
	var done []undo
	if err := withIntentMintLock(repoRoot, func() error {
		for i := range members {
			m := &members[i]
			if m.Intent.Bucket != BucketPlanned {
				continue
			}
			abs := filepath.Join(repoRoot, m.Intent.Path)
			content, err := readIntentRefusingHold(abs, m.Intent.Path, m.Intent.ID, "spec close")
			if err != nil {
				return err
			}
			// Recorded before the first write, so a failure at either write or at
			// the move is undone from the bytes read here.
			done = append(done, undo{abs: abs, rel: m.Intent.Path, orig: content})
			if stamps[i] != "" {
				updated, err := setFrontmatterFields(content, map[string]string{"impact": stamps[i]})
				if err != nil {
					return err
				}
				if err := writeIntentFile(abs, m.Intent.Path, updated); err != nil {
					return err
				}
			}
			dst, err := moveIntentToBucket(repoRoot, m.Intent.Path, BucketShipped)
			if err != nil {
				return err
			}
			done[len(done)-1].dstRel, done[len(done)-1].moved = dst, true
		}
		return nil
	}); err != nil {
		for i := len(done) - 1; i >= 0; i-- {
			u := done[i]
			if u.moved {
				_ = os.Rename(filepath.Join(repoRoot, u.dstRel), u.abs)
			}
			_ = writeIntentFile(u.abs, u.rel, u.orig)
		}
		return ReconcileResult{}, fmt.Errorf("%w; every member of bundle %s was put back, and spec %s stays open", err, sp.Bundle, sp.ID)
	}
	for i := range members {
		m := &members[i]
		for _, u := range done {
			if u.moved && u.rel == m.Intent.Path {
				m.Intent.Bucket = BucketShipped
				m.Intent.Path = u.dstRel
				m.Moved = true
				m.To = BucketShipped
			}
		}
	}

	specMovedNow := sp.Status == spec.StatusOpen
	res := ReconcileResult{Spec: sp, Skipped: skipped}
	if specMovedNow {
		closed, err := spec.Close(repoRoot, sp.ID)
		if err != nil {
			return ReconcileResult{}, err
		}
		res.Spec = closed
	}
	var moves []relink.Move
	if res.Spec.Status == spec.StatusClosed {
		moves = append(moves, relink.Move{
			From:     filepath.Join(spec.SpecsRelDir, spec.StatusOpen, filepath.Base(res.Spec.Path)),
			To:       res.Spec.Path,
			MovedNow: specMovedNow,
		})
	}
	var emitErrs []string
	for i := range members {
		m := &members[i]
		if m.Intent.Bucket != BucketShipped {
			continue
		}
		moves = append(moves, relink.Move{
			From:     filepath.Join(IntentsRelDir, BucketPlanned, filepath.Base(m.Intent.Path)),
			To:       m.Intent.Path,
			MovedNow: m.Moved,
		})
		emit, err := emitAuditForIntent(repoRoot, m.Intent)
		if err != nil {
			m.AuditEmitError = err.Error()
			emitErrs = append(emitErrs, m.Intent.ID+": "+err.Error())
		} else {
			m.ReceiptID, m.ReceiptStatus = emit.ReceiptID, emit.Status
		}
	}
	res.Relinked, err = relink.Repoint(repoRoot, moves)
	if err != nil {
		res.RelinkError = err.Error()
	}
	res.Members = members
	first := members[0]
	res.Intent, res.From, res.To = first.Intent, first.From, first.To
	res.ReceiptID, res.ReceiptStatus = first.ReceiptID, first.ReceiptStatus
	for _, m := range members {
		res.IntentMoved = res.IntentMoved || m.Moved
	}
	res.AuditEmitError = strings.Join(emitErrs, "; ")
	return res, nil
}
