package intent

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/relink"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Load discovers intent files across every lifecycle bucket, parses their
// frontmatter, and returns the in-memory Corpus. A missing intents/ directory
// (or a missing individual bucket) yields no records for it (soft, mirroring
// spec.Load). A present-but-malformed intent file — one whose frontmatter lacks
// a well-formed id — is a hard, loud error.
func Load(repoRoot string) (Corpus, error) {
	var c Corpus
	for _, bucket := range Buckets {
		intents, err := loadBucket(repoRoot, bucket)
		if err != nil {
			return Corpus{}, err
		}
		c.Intents = append(c.Intents, intents...)
	}
	return c, nil
}

// loadBucket reads one bucket directory. A missing directory is soft (nil, nil).
func loadBucket(repoRoot, bucket string) ([]Intent, error) {
	dir := filepath.Join(repoRoot, IntentsRelDir, bucket)
	relDir := filepath.Join(IntentsRelDir, bucket)
	di, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("intent: stat %s: %w", relDir, err)
	}
	if di.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("intent: %s is a symlink (refusing to follow)", relDir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("intent: reading %s: %w", relDir, err)
	}
	var intents []Intent
	for _, e := range entries {
		if e.IsDir() || !intentFileRe.MatchString(e.Name()) {
			continue
		}
		rel := filepath.Join(relDir, e.Name())
		data, err := readRepoFile(filepath.Join(dir, e.Name()), rel)
		if err != nil {
			return nil, err
		}
		it, err := parseIntent(rel, string(data), bucket)
		if err != nil {
			return nil, err
		}
		intents = append(intents, it)
	}
	return intents, nil
}

// parseIntent builds an Intent from a file's content and validates it. A file
// whose frontmatter lacks a well-formed id is malformed and rejected.
func parseIntent(relPath, content, bucket string) (Intent, error) {
	fields := frontmatter.Fields(strings.Split(content, "\n"))
	it := Intent{
		ID:     fields["id"].Value,
		Slug:   fields["slug"].Value,
		Kind:   fields["kind"].Value,
		SpecID: fields["spec_id"].Value,
		Bucket: bucket,
		Path:   relPath,
	}
	if b := fields[BundleKey].Value; !frontmatter.IsNull(b) {
		it.Bundle = b
	}
	if f, ok := fields[RelatedIssuesKey]; ok && !frontmatter.IsNull(f.Value) {
		it.RelatedIssues = frontmatter.StringList(f.Value)
	}
	// The hold is read leniently: a `held:` key in a shape the verb never
	// writes marks the record MALFORMED rather than failing the corpus, because
	// one hand edit fail-closing every intent verb for everyone who pulls it is
	// the outage iss-2608270500198764 already taught. What that means for the
	// trust boundary is spelled out on Hold in hold.go.
	if f, ok := fields[HeldKey]; ok {
		if reason, ok := frontmatter.ScalarString(f.Value); ok {
			it.Held = reason
		} else {
			it.HeldMalformed = true
		}
	}
	if err := Validate(it); err != nil {
		return Intent{}, fmt.Errorf("intent: malformed %s: %w", relPath, err)
	}
	return it, nil
}

// Plan is the load-bearing verb. For a draft intent carrying a non-empty
// `## Acceptance Criteria` section, it mints a native spec, writes the intent's
// derived side of the bidirectional link (spec_id + a default kind), and moves
// the intent drafts/ → planned/.
//
// It is fail-closed: every intermediate on-disk state satisfies the
// intent_lifecycle record-lint rule, so a failure at any step leaves a
// consistent, lint-valid record rather than a half-written link. The order —
// create spec, set kind while still a draft, move to planned, then write
// spec_id — is chosen so that (kind=standalone, spec_id=null) is the only
// transient frontmatter, and that shape is valid in BOTH drafts and planned.
//
// opts carries the two optional judgements the verb takes: the production mode
// the MINTED SPEC carries, and the impact the INTENT record is stamped with
// (see PlanOptions).
func Plan(repoRoot, intentID string, opts PlanOptions) (PlanResult, error) {
	if !recordid.ValidIntentID(intentID) {
		return PlanResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return PlanResult{}, err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return PlanResult{}, fmt.Errorf("intent: %s not found in any bucket", intentID)
	}
	if it.Bucket != BucketPlanned && it.Bucket != BucketDrafts {
		return PlanResult{}, fmt.Errorf("intent: %s is in %s, not drafts; only a draft can be planned", intentID, it.Bucket)
	}
	// A hold stops Plan before anything moves, on BOTH buckets it acts on: the
	// draft's plan and the planned record's identity-only re-run are the same
	// verb, and a hold that stopped one and not the other would be prose again
	// (iss-2609200830076665). The refusal names the reason and the verb that
	// lifts it, so the way past is a deliberate act and not a re-run.
	if err := refuseIfHeld(it, "plan"); err != nil {
		return PlanResult{}, err
	}
	// A record already in planned/ takes the identity step alone. Conditions get
	// written after planning — the elicitation is a human conversation, not a
	// one-shot — and Plan is the only writer of a marker there is, so refusing the
	// re-run would leave such a record permanently unable to satisfy the gate that
	// demands the marker (iss-2608300210588874).
	if it.Bucket == BucketPlanned {
		// A planned record with no spec — planned before the spec seam existed —
		// is given one in place: the draft's mint and link, on the same criteria
		// bar, with no bucket move (iss-2609211738504433).
		if frontmatter.IsNull(it.SpecID) {
			return linkPlannedSpec(repoRoot, it, opts)
		}
		return stampPlanned(repoRoot, it, opts.Impact)
	}
	if !slugRe.MatchString(it.Slug) {
		return PlanResult{}, fmt.Errorf("intent: %s has slug %q which must be kebab-case", intentID, it.Slug)
	}

	draftRel := it.Path
	draftAbs := filepath.Join(repoRoot, draftRel)
	// Everything from the read to the last write is ONE critical section under
	// the store's advisory lock, as stampPlanned's stamp is
	// (iss-2608300235388164): the hold is judged on the bytes read HERE, and
	// those bytes are what the stamp, the kind write and the move are made
	// from. Judged outside the lock, a hold landing after the read — through
	// `intent hold`, which takes this lock, re-reads, writes atomically and
	// exits 0 — was erased by a rewrite from the stale bytes, and the record
	// reached planned/ with both verbs reporting success. The refusal above,
	// on the corpus, is the early one; this is the one that binds.
	//
	// The spec mint stays inside: spec.Create takes only the spec store's own
	// lock, nothing takes that lock and then this one (the spec package cannot
	// import this one), and keeping it inside is what keeps "a refusal here
	// leaves nothing minted" true — a spec left behind by a refused plan is
	// retry-safe through store.ByIntent, but a mint that never happened needs
	// no retry.
	var (
		sp                spec.Spec
		kind              string
		plannedRel        string
		conditionsStamped int
		impactStamp       string
	)
	if err := withIntentMintLock(repoRoot, func() error {
		content, err := readIntentRefusingHold(draftAbs, draftRel, intentID, "plan")
		if err != nil {
			return err
		}
		// The record's fields are the ones these bytes carry, not the corpus's:
		// the kind this write keeps and the spec_id it refuses on are judged on
		// what is rewritten. From the corpus, a kind a reclassify wrote in the
		// window was overwritten, leaving kind: standalone beside the bundle
		// that reclassify named (iss-2609261218301461).
		if it, err = parseIntent(draftRel, content, BucketDrafts); err != nil {
			return fmt.Errorf("intent: malformed %s: %w", draftRel, err)
		}
		if !hasAcceptanceCriteria(content) {
			return fmt.Errorf("intent: %s has no non-empty '## Acceptance Criteria' section (itd-1 discipline); refusing to plan", intentID)
		}
		// A draft that already carries a non-null spec_id is half-planned (and
		// lint-invalid): refuse rather than mint a second spec for it.
		if !frontmatter.IsNull(it.SpecID) {
			return fmt.Errorf("intent: %s is a draft with spec_id %q already set (half-planned); refusing to plan", intentID, it.SpecID)
		}
		// The impact judgement is settled BEFORE the first write and before the
		// spec is minted, so a refused value — out of vocabulary, `internal`, or a
		// disagreement with what the record already says — leaves the draft
		// byte-identical with no spec pointing at it.
		impactStamp, err = resolvePlanImpact(it, content, opts.Impact)
		if err != nil {
			return err
		}

		// 1. Reuse the spec already realising this intent, or mint one. Reusing makes
		// Plan retry-safe: a re-run after a failed drafts->planned rename completes the
		// operation instead of duplicating the spec. Both branches write the reciprocal
		// intent: itd-N side (Create writes it; a reused spec already carries it).
		store, err := spec.Load(repoRoot)
		if err != nil {
			return err
		}
		var ok bool
		sp, ok = store.ByIntent(intentID)

		// 1a. The draft face grows the record THREE times — the identity stamp, the
		// kind rewrite, the spec_id rewrite — and the largest of them is knowable
		// before any of them is written. Checking per write let a record pass the kind
		// write, MOVE to planned, and only then fail the spec_id write, leaving a
		// half-planned record neither Plan nor Link repairs; and it left a freshly
		// minted spec behind with nothing pointing at it (iss-2608300335369473). The
		// check runs first, before the spec is minted. A reused spec is judged by its
		// own id; a spec still to be minted is judged by a probe id of the mint's
		// width — every native id is the same width (adr-45), so the probe measures
		// exactly what the mint will write, and no second judgement is owed once the
		// real id is known. The check runs before the first write, so a refusal
		// leaves no half-planned record.
		specID := sp.ID
		if !ok {
			if specID, err = probeMinter().Mint(specFamily); err != nil {
				return err
			}
		}
		if err := checkDraftFaceSize(content, it, specID, impactStamp, draftRel); err != nil {
			return err
		}

		if !ok {
			sp, err = spec.Create(repoRoot, intentID, it.Slug, opts.ProductionMode)
			if err != nil {
				return err
			}
		}

		// 2. Stamp an identity onto every unmarked scope-condition bullet, set the
		// binding kind (default standalone), and write the impact judgement when one
		// was supplied — all while still in drafts. Plan is the write-capable verb of
		// the lifecycle, so it is where the identities are minted: the readiness gate
		// reports a missing marker but never writes one, a reporter that writes being
		// a reporter whose output depends on who ran it. The impact rides the same
		// write as the kind: a draft carrying (kind=standalone, spec_id=null) and an
		// impact is exactly what the create path can seed, so it stays lint-valid,
		// and a failure here leaves a consistent record (the spec exists but the
		// intent is unlinked).
		stampedContent, n, err := stampScopeConditions(content, recordid.Minter{})
		if err != nil {
			return err
		}
		conditionsStamped = n
		kind = it.Kind
		if frontmatter.IsNull(kind) {
			kind = KindStandalone
		}
		withKind, err := setFrontmatterFields(stampedContent, draftFaceFields(kind, impactStamp))
		if err != nil {
			return err
		}
		if err := writeIntentFile(draftAbs, draftRel, withKind); err != nil {
			return err
		}

		// 3. Move drafts/ → planned/ via the shared, trust-guarded move. The moved
		// file's (kind=standalone, spec_id=null) shape is a valid planned intent, so a
		// rename failure leaves a consistent state either side.
		plannedRel, err = moveIntentToBucket(repoRoot, draftRel, BucketPlanned)
		if err != nil {
			return err
		}
		plannedAbs := filepath.Join(repoRoot, plannedRel)

		// 4. Write the derived link (spec_id) now that the file is in planned. A
		// planned intent with spec_id=null is still lint-valid, so a failure here is
		// consistent too.
		withSpec, err := setFrontmatterFields(withKind, map[string]string{"spec_id": sp.ID})
		if err != nil {
			return err
		}
		return writeIntentFile(plannedAbs, plannedRel, withSpec)
	}); err != nil {
		return PlanResult{}, err
	}

	it.Kind = kind
	it.SpecID = sp.ID
	it.Bucket = BucketPlanned
	it.Path = plannedRel
	res := PlanResult{Intent: it, Spec: sp, ConditionsStamped: conditionsStamped, ImpactStamped: impactStamp}
	// Repoint every link that named the draft's path, as a close does for the
	// records it moves (iss-2609250846525896). Reported, not raised: the record
	// is planned and the plan stands.
	res.Relinked, err = relink.Repoint(repoRoot, []relink.Move{{From: draftRel, To: plannedRel, MovedNow: true}})
	if err != nil {
		res.RelinkError = err.Error()
	}
	return res, nil
}

// draftFaceFields is the frontmatter rewrite the draft face makes in one
// write: the binding kind, plus the impact judgement when the run carries one
// to stamp. One function so the size probe and the real write cannot disagree
// about what that write contains.
func draftFaceFields(kind, impact string) map[string]string {
	fields := map[string]string{"kind": kind}
	if impact != "" {
		fields["impact"] = impact
	}
	return fields
}

// checkDraftFaceSize refuses a draft whose planned form would not fit under the
// cap its own reader enforces, BEFORE the first write and before the bucket
// move. It reproduces the three growth steps in order and judges the largest;
// impact is the judgement the kind write will carry alongside it, or empty.
func checkDraftFaceSize(content string, it Intent, specID, impact, rel string) error {
	// The probe's entropy has to advance: a constant source hands the second
	// bullet the id the first already used, the redraw loop exhausts, and the
	// whole judgement is lost behind a mint error on every record with more than
	// one condition (iss-2608300352403199).
	stamped, _, err := stampScopeConditions(content, probeMinter())
	if err != nil {
		// Including a structural refusal: reporting it here, before the spec is
		// minted, is strictly better than letting the real stamp reach it later.
		return err
	}
	kind := it.Kind
	if frontmatter.IsNull(kind) {
		kind = KindStandalone
	}
	withKind, err := setFrontmatterFields(stamped, draftFaceFields(kind, impact))
	if err != nil {
		return err
	}
	withSpec, err := setFrontmatterFields(withKind, map[string]string{"spec_id": specID})
	if err != nil {
		return err
	}
	largest := len(stamped)
	for _, n := range []int{len(withKind), len(withSpec)} {
		if n > largest {
			largest = n
		}
	}
	if largest > maxIntentFileBytes {
		return fmt.Errorf("intent: planning %s would produce %d bytes, past the %d-byte cap its own reader enforces; refusing before any write", rel, largest, maxIntentFileBytes)
	}
	return nil
}

// specFamily is the spec store's id prefix, named here so the size probe mints
// an id of exactly the width spec.Create will write.
const specFamily = "spc"

// probeMinter is the size probe's mint: a fixed clock and an advancing
// counter for entropy, so every id it draws is distinct and sixteen digits
// wide. Nothing it produces is ever stored. The source is per-call, so two
// concurrent plans never share a counter.
func probeMinter() recordid.Minter {
	return recordid.Minter{
		Now:     func() time.Time { return time.Unix(0, 0).UTC() },
		Entropy: &probeEntropy{},
	}
}

// probeSuffixSpan keeps every probe draw below the mint's rejection band, so no
// draw is discarded and the counter advances one id per bullet. Suffixes repeat
// after 10,000 bullets — a record that long fails the size cap many times over.
const probeSuffixSpan = 50000

// probeEntropy is the size probe's entropy: an advancing counter, never
// crypto/rand. It exists so the probe's ids are distinct and sixteen digits
// wide; nothing it produces is ever stored.
type probeEntropy struct{ n uint16 }

func (e *probeEntropy) Read(p []byte) (int, error) {
	var b [2]byte
	for i := 0; i < len(p); i += 2 {
		e.n = (e.n + 1) % probeSuffixSpan
		binary.BigEndian.PutUint16(b[:], e.n)
		copy(p[i:], b[:])
	}
	return len(p), nil
}

// stampPlanned is Plan's idempotent second face: it mints an identity for every
// unmarked scope-condition bullet of an already-planned record, writes the
// impact judgement when the run carries one the record lacks, and writes the
// result back, touching nothing else — no spec, no other frontmatter, no bucket
// move. An already-marked bullet is left byte-identical, so re-running after an
// edit stamps only what is new. The impact is taken here under the same rules
// as on a draft: plan is the verb that runs when the judgement is made, so it
// takes the flag, and "stamp the judgement" is exactly what a planned record
// without one needs before its close, which refuses without it.
//
// A run with nothing to stamp is a refusal, not a quiet success: the caller
// asked for work to be done, and a verb that exits 0 having done none of it
// teaches its user that the command is a no-op. An impact stamped counts as
// work; an impact that merely agrees with the record does not, and the refusal
// then names the recorded judgement so it cannot read as "not stamped".
func stampPlanned(repoRoot string, it Intent, impact string) (PlanResult, error) {
	rel := it.Path
	abs := filepath.Join(repoRoot, rel)
	var stampedCount int
	var impactStamp string
	// The read, the mint and the write are one critical section under the store's
	// existing advisory lock: two sessions stamping the same record would
	// otherwise each write the file they read, and the later write would drop the
	// earlier one's identities (iss-2608300235388164).
	err := withIntentMintLock(repoRoot, func() error {
		data, err := readRepoFile(abs, rel)
		if err != nil {
			return err
		}
		content := string(data)
		// The judgement is settled before the identity mint, so a refused value
		// leaves the record byte-identical: no bullet is marked on a run that
		// then reports a refusal.
		stamp, err := resolvePlanImpact(it, content, impact)
		if err != nil {
			return err
		}
		stamped, n, err := stampScopeConditions(content, recordid.Minter{})
		if err != nil {
			return err
		}
		if n == 0 && stamp == "" {
			if impact != "" {
				return fmt.Errorf("intent: %s is already planned, carries no unmarked scope condition and already records impact %q; nothing to stamp", it.ID, impact)
			}
			return fmt.Errorf("intent: %s is already planned and carries no unmarked scope condition; nothing to stamp", it.ID)
		}
		if stamp != "" {
			if stamped, err = setFrontmatterFields(stamped, map[string]string{"impact": stamp}); err != nil {
				return err
			}
		}
		if err := writeIntentFile(abs, rel, stamped); err != nil {
			return err
		}
		stampedCount = n
		impactStamp = stamp
		return nil
	})
	if err != nil {
		return PlanResult{}, err
	}
	res := PlanResult{Intent: it, ConditionsStamped: stampedCount, StampOnly: true, ImpactStamped: impactStamp}
	// The stamp mints no spec, but the intent has one, and an empty spec object in
	// the result reads as "this intent has no spec" to anything consuming it. The
	// lookup is lenient: a broken link is the readiness gate's finding to report,
	// not a reason to fail a write that already succeeded.
	if store, lerr := spec.Load(repoRoot); lerr == nil {
		if sp, ok := store.Lookup(it.SpecID); ok {
			res.Spec = sp
		}
	}
	return res, nil
}

// linkPlannedSpec mints (or reuses) the spec for a record already in planned/
// whose spec_id is null, and writes the link in place. It is the draft face of
// Plan without the move: the Acceptance Criteria bar, the impact judgement,
// the scope-condition stamp and the size check all apply, in the same order,
// under the same lock. A refusal before the mint leaves the record
// byte-identical with no spec minted; a refusal after it (the intent write is
// the last step, and it can fail) removes the spec this run minted, so the
// store is left as it was too. A spec that already names the intent (a
// one-sided link) is reused rather than duplicated, which also repairs that
// link, and is never removed. The one write sets spec_id together with the
// kind (defaulted only when null), the impact and the stamped identities, so
// there is no intermediate record to lint.
func linkPlannedSpec(repoRoot string, it Intent, opts PlanOptions) (PlanResult, error) {
	if !slugRe.MatchString(it.Slug) {
		return PlanResult{}, fmt.Errorf("intent: %s has slug %q which must be kebab-case", it.ID, it.Slug)
	}
	rel := it.Path
	abs := filepath.Join(repoRoot, rel)
	var (
		sp                spec.Spec
		kind              string
		conditionsStamped int
		impactStamp       string
	)
	if err := withIntentMintLock(repoRoot, func() error {
		content, err := readIntentRefusingHold(abs, rel, it.ID, "plan")
		if err != nil {
			return err
		}
		if !hasAcceptanceCriteria(content) {
			return fmt.Errorf("intent: %s is planned with no spec, and has no non-empty '## Acceptance Criteria' section (itd-1 discipline) to mint one from; refusing to plan", it.ID)
		}
		impactStamp, err = resolvePlanImpact(it, content, opts.Impact)
		if err != nil {
			return err
		}
		store, err := spec.Load(repoRoot)
		if err != nil {
			return err
		}
		var reused bool
		sp, reused = store.ByIntent(it.ID)
		specID := sp.ID
		if !reused {
			if specID, err = probeMinter().Mint(specFamily); err != nil {
				return err
			}
		}
		if err := checkDraftFaceSize(content, it, specID, impactStamp, rel); err != nil {
			return err
		}
		if !reused {
			if sp, err = spec.Create(repoRoot, it.ID, it.Slug, opts.ProductionMode); err != nil {
				return err
			}
		}
		err = func() error {
			stamped, n, err := stampScopeConditions(content, recordid.Minter{})
			if err != nil {
				return err
			}
			conditionsStamped = n
			kind = it.Kind
			if frontmatter.IsNull(kind) {
				kind = KindStandalone
			}
			fields := draftFaceFields(kind, impactStamp)
			fields["spec_id"] = sp.ID
			linked, err := setFrontmatterFields(stamped, fields)
			if err != nil {
				return err
			}
			return writeIntentFile(abs, rel, linked)
		}()
		if err != nil && !reused {
			// The spec this run minted is taken back with the refusal, so the
			// spec store is as it was (iss-2609260221563975). Nothing else can
			// name it: it was written under this lock a moment ago, and the
			// intent write that would have linked it is what failed. Were the
			// removal itself to fail, the spec is still one a retry reuses
			// through ByIntent, and the refusal says so.
			if rmErr := os.Remove(filepath.Join(repoRoot, sp.Path)); rmErr != nil && !os.IsNotExist(rmErr) {
				return fmt.Errorf("%w; the spec minted for it, %s, could not be removed (%v) and a retry reuses it", err, sp.ID, rmErr)
			}
		}
		return err
	}); err != nil {
		return PlanResult{}, err
	}
	it.Kind = kind
	it.SpecID = sp.ID
	return PlanResult{Intent: it, Spec: sp, ConditionsStamped: conditionsStamped,
		LinkedInPlace: true, ImpactStamped: impactStamp}, nil
}

// Link retroactively writes the derived spec_id link on an existing planned
// intent for an existing spec. It validates both ids, that the intent is in
// planned/, and that the spec exists AND already declares this intent (the
// reciprocal intent: itd-N side); a spec that realises a different intent is a
// mismatch and fails closed rather than forging a one-sided link.
func Link(repoRoot, intentID, specID string) (LinkResult, error) {
	if !recordid.ValidIntentID(intentID) {
		return LinkResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
	}
	if !recordid.ValidSpecID(specID) {
		return LinkResult{}, fmt.Errorf("intent: spec id %q must match ^spc-[0-9]+$", specID)
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return LinkResult{}, err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return LinkResult{}, fmt.Errorf("intent: %s not found in any bucket", intentID)
	}
	if it.Bucket != BucketPlanned {
		return LinkResult{}, fmt.Errorf("intent: %s is in %s, not planned; only a planned intent can be linked", intentID, it.Bucket)
	}
	store, err := spec.Load(repoRoot)
	if err != nil {
		return LinkResult{}, err
	}
	sp, ok := store.Lookup(specID)
	if !ok {
		return LinkResult{}, fmt.Errorf("intent: spec %s not found", specID)
	}
	// Canonical, not literal: a back-link written `itd-007` names itd-7 and is
	// lint-green, so refusing it here would make a spec unlinkable for a spelling
	// difference the gate accepts (recordid.SameID).
	if !recordid.SameID(sp.Intent, intentID) {
		return LinkResult{}, fmt.Errorf("intent: spec %s realises %s, not %s (mismatch); refusing to link", specID, sp.Intent, intentID)
	}

	rel := it.Path
	abs := filepath.Join(repoRoot, rel)
	data, err := readRepoFile(abs, rel)
	if err != nil {
		return LinkResult{}, err
	}
	updated, err := setFrontmatterFields(string(data), map[string]string{"spec_id": specID})
	if err != nil {
		return LinkResult{}, err
	}
	if err := writeIntentFile(abs, rel, updated); err != nil {
		return LinkResult{}, err
	}

	it.SpecID = specID
	return LinkResult{Intent: it, Spec: sp}, nil
}

// RelatedIssuesKey is the frontmatter key of the intent half of the promote
// join (itd-4 AC3): the ledger records the intent graduated from.
const RelatedIssuesKey = "related_issues"

// RetiredRelatedIssuesKey is the key the promote back-edge was written under
// before itd-4 AC3 renamed it. Nothing writes it and the reader does not read
// it: a record still carrying it is migrated by `abcd capture migrate --apply`,
// and a write that meets it refuses rather than leaving both spellings of one
// join on a record.
const RetiredRelatedIssuesKey = "promoted_from"

// ErrRetiredField reports a record that still carries a retired back-link key,
// so a write would leave both spellings of one join side by side.
var ErrRetiredField = fmt.Errorf("intent: the record carries a retired back-link field")

// AddRelatedIssue appends source to an existing intent's `related_issues`, in
// any bucket. It is the intent half of link mode: `capture promote <iss-N|rdi-N>
// --intent <itd-N>` stamps the source's `related_intents` and this writes the
// edge pointing back, so the join reads from both ends.
//
// It writes that one key and NOTHING else. It never reads or rewrites `origin`
// or `production_mode`, which is what "the origin is unchanged" rests on: an
// origin is stamped at mint and never rewritten, so a draft filed from quoted
// text and linked to a reading item stays researcher-authored and says so.
//
// A list already naming source is a no-op that leaves the record byte-identical.
// A list naming OTHER records keeps them, in order, and appends source: an intent
// occasioned by several items is promoted from the first and joined to the rest
// (itd-2609020625400169, first scope condition), so nothing is overwritten.
func AddRelatedIssue(repoRoot, intentID, source string) (Intent, error) {
	if !recordid.ValidIntentID(intentID) {
		return Intent{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
	}
	if !relatedIssueRe.MatchString(source) {
		return Intent{}, fmt.Errorf("intent: related issue %q must match ^(iss|rdi)-[0-9]+$", source)
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return Intent{}, err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return Intent{}, fmt.Errorf("intent: %s not found in any bucket", intentID)
	}
	rel := it.Path
	abs := filepath.Join(repoRoot, rel)
	data, err := readRepoFile(abs, rel)
	if err != nil {
		return Intent{}, err
	}
	if _, retired := frontmatter.Fields(strings.Split(string(data), "\n"))[RetiredRelatedIssuesKey]; retired {
		return Intent{}, fmt.Errorf("%w: %s carries `%s`, renamed to `%s`; run `abcd capture migrate --apply` first, nothing written",
			ErrRetiredField, intentID, RetiredRelatedIssuesKey, RelatedIssuesKey)
	}
	for _, have := range it.RelatedIssues {
		if have == source {
			return it, nil // already joined; the write would change no byte
		}
	}
	list := append(append([]string{}, it.RelatedIssues...), source)
	updated, err := setFrontmatterFields(string(data), map[string]string{RelatedIssuesKey: "[" + strings.Join(list, ", ") + "]"})
	if err != nil {
		return Intent{}, err
	}
	if err := writeIntentFile(abs, rel, updated); err != nil {
		return Intent{}, err
	}
	it.RelatedIssues = list
	return it, nil
}

// Reconcile is the deterministic half of `abcd spec close`: it closes a spec
// and, when that close was the intent's last open spec, ships the intent — so
// one command marks the spec done AND moves the intent exactly when the
// capability is whole.
//
// An intent owns one or more specs (adr-2609151513118583, invariant 17). A spec
// that delivers only part of an intent is closed on its own terms while another
// spec still names the intent, and the intent stays in planned/; the close after
// which no open spec names it is the one that ships it. More than one spec
// naming one intent is the normal state, not an ambiguity — the question that
// decides the move is "does this intent have an open spec left?".
//
// Ordering is intent-first, spec-last on the close that ships, so a partial
// failure is recoverable by re-running: the intent moves planned/ → shipped/
// before spec.Close runs, so a failure at the move leaves the spec OPEN
// (retry-safe), never a closed spec with a still-planned intent. A remainder is
// minted before either, so a failure there moves nothing at all. It is
// idempotent: an already-shipped intent is not re-moved, and a re-run on an
// already-closed spec is a clean no-op/complete rather than an error.
//
// It fails closed with NO partial move when: the spec has no/empty intent link;
// the named intent does not exist; the intent's spec_id names no spec that
// realises it (bidirectional drift); the intent is in an unexpected bucket (e.g.
// still in drafts — it was never planned); an impact is supplied at a close that
// ships nothing; a remainder is asked for on an already-shipped intent or an
// already-closed spec (neither has a delivery boundary left to split); or the
// intent would enter shipped/ without the impact judgement that bucket requires
// (see resolveShipImpact). Every id is validated against
// the ^spc-/^itd- regexes before any path is built. The intent's `## Audit
// Notes` are left untouched (the fidelity audit is a later phase; the intent
// ships with them empty).
//
// impact is the judgement `abcd spec close --impact` carries: empty means "the
// record already carries its own", and a value is stamped onto a record that has
// none. It is never a silent override — see resolveShipImpact — and it is
// accepted only at the close that ships, because that is the only close that
// writes it.
//
// remainder asks this close to mint the follow-on spec for what the closing spec
// did not deliver (see RemainderRequest); the zero value asks for none.
func Reconcile(repoRoot, specID, impact string, remainder RemainderRequest) (ReconcileResult, error) {
	if !recordid.ValidSpecID(specID) {
		return ReconcileResult{}, fmt.Errorf("intent: spec id %q must match ^spc-[0-9]+$", specID)
	}
	store, err := spec.Load(repoRoot)
	if err != nil {
		return ReconcileResult{}, err
	}
	sp, ok := store.Lookup(specID)
	if !ok {
		return ReconcileResult{}, fmt.Errorf("intent: spec %s not found", specID)
	}
	// A bundle's shared spec ships every member together (itd-34).
	if sp.Bundle != "" && len(sp.Intents) > 0 {
		return reconcileBundle(repoRoot, store, sp, impact, remainder)
	}

	// Resolve the linked intent from the spec's intent: field, validated before it
	// is ever used to build a path.
	intentID := sp.Intent
	if !recordid.ValidIntentID(intentID) {
		return ReconcileResult{}, fmt.Errorf("intent: spec %s has no well-formed intent link (got %q); refusing to reconcile", specID, intentID)
	}

	corpus, err := Load(repoRoot)
	if err != nil {
		return ReconcileResult{}, err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return ReconcileResult{}, fmt.Errorf("intent: %s (linked by spec %s) not found in any bucket; refusing to reconcile", intentID, specID)
	}
	// Bidirectional agreement, 1:n-aware: the intent's spec_id must name one of
	// the specs that realise it. Under the 1:1 rule this was equality with THIS
	// spec; under 1:n the second spec of an intent legitimately closes while the
	// intent's spec_id still names the first, so the check is membership. A null
	// spec_id, or one naming a spec that does not realise this intent, is still
	// drift (a one-sided link) and still fails closed.
	// The comparison is canonical (spec.SameNum), not literal: record-lint matches
	// a spec_id on its NUMBER, so a slug-suffixed or zero-padded value is
	// lint-green and this verb must not refuse what the lint accepts.
	claimers := store.SpecsForIntent(intentID)
	backLinked := false
	for _, c := range claimers {
		if spec.SameNum(it.SpecID, c.ID) {
			backLinked = true
			break
		}
	}
	if !backLinked {
		return ReconcileResult{}, fmt.Errorf("intent: %s spec_id is %q but no spec realising it carries that id (spec %s claims it; bidirectional link disagrees); refusing to reconcile",
			intentID, it.SpecID, specID)
	}
	// Bucket guard runs BEFORE any move, so an unexpected bucket (drafts,
	// disciplines, superseded) yields no partial move.
	switch it.Bucket {
	case BucketPlanned, BucketShipped:
		// planned → advance; shipped → idempotent (already advanced).
	default:
		return ReconcileResult{}, fmt.Errorf("intent: %s is in %s (linked by spec %s); expected planned or shipped — refusing to reconcile", intentID, it.Bucket, specID)
	}
	// A hold blocks EVERY lifecycle move until `intent unhold`, and the close is
	// one: a held planned record is refused here, ahead of the remainder mint,
	// the impact stamp, the move and the close, naming the reason and the lift
	// exactly as Plan does (iss-2609200830076665). This is the early refusal, on
	// the corpus; the one that binds is re-run under the store lock on the bytes
	// the move is made from, immediately before it. The key is never stripped by
	// this verb. A shipped record is past every move a hold stops and no verb
	// can put a hold on one — hold refuses the bucket, and this refusal is what
	// keeps a held record out of shipped/ — so a `held:` there is
	// record_provenance's finding, not this verb's, and the idempotent re-run
	// is left to complete.
	if it.Bucket == BucketPlanned {
		if err := refuseIfHeld(it, "spec close"); err != nil {
			return ReconcileResult{}, err
		}
	}

	// --remainder mints a follow-on spec for what THIS close did not deliver, so
	// it is meaningful only at a close that is actually happening against an
	// intent that can still receive work. Two shapes are refused here, before any
	// mint, rather than acted on:
	//
	//   - a SHIPPED intent. The capability is already announced and its fidelity
	//     audit already owed; attaching a fresh OPEN spec to it produces exactly
	//     the shipped-intent-with-an-open-spec state invariant 17 forbids, which
	//     the surface could only render as a contradiction ("stays shipped —
	//     still open"). The remainder of a shipped intent is a NEW intent.
	//   - an already-CLOSED spec. The close is complete, so there is no delivery
	//     boundary left to split. Without this the flag mints another spec on
	//     every invocation of a command that is otherwise a clean no-op — the
	//     re-run of a finished close silently grows the ledger.
	//
	// Both refuse before the mint for the same reason the impact refusal does: a
	// failure that has already written a record is not a refusal.
	if remainder.Slug != "" {
		if it.Bucket == BucketShipped {
			// it.ID, not the back-link spelling: the message names the record as
			// the tree holds it, so a padded link does not read as a second intent.
			return ReconcileResult{}, fmt.Errorf("intent: %s is already shipped, and --remainder would attach an OPEN spec to it — a shipped intent has no open spec left (adr-2609151513118583); nothing was minted. Plan a new intent for the remaining work",
				it.ID)
		}
		if sp.Status != spec.StatusOpen {
			return ReconcileResult{}, fmt.Errorf("intent: spec %s is already %s, so this close splits no delivery boundary; --remainder would mint another spec on every re-run; nothing was minted. Mint the follow-on deliberately if one is still wanted",
				specID, sp.Status)
		}
	}

	// The remainder carries the closing spec's steps not marked landed
	// (itd-2609212103565953, criterion 3), so the section is read before the
	// mint — and a section that is not a numbered list refuses here, with
	// nothing written, because the copy would otherwise be a guess
	// (unrecognized-input-never-writes). A close without a remainder never
	// reads it: the section is the build's, not the close's.
	var carried []spec.Step
	if remainder.Slug != "" {
		listed, err := spec.ReadSteps(repoRoot, sp)
		if err != nil {
			return ReconcileResult{}, fmt.Errorf("intent: %v; --remainder carries the steps not marked landed, and this section cannot be read as steps; nothing was minted. Fix what it names, then re-run the close", err)
		}
		carried = spec.Unlanded(listed)
		// Numbered as the remainder lists them, so the result and the file agree.
		for i := range carried {
			carried[i].Number = i + 1
		}
	}

	// Does any OTHER spec still hold this intent open? Asked before the mint, so
	// the impact refusal below can fire before anything is written, and asked
	// again after it (the remainder counts too).
	held := otherOpenSpecs(claimers, specID)

	// An impact supplied at a close that ships nothing is refused, not ignored:
	// the flag's only effect is to stamp the judgement shipped/ requires, so
	// accepting it here would report a write that never happened — and stamping it
	// early would pre-decide the derived version of a release this close does not
	// reach. The judgement belongs at the close that ships (adr-2609151513118583).
	if strings.TrimSpace(impact) != "" {
		if len(held) > 0 {
			return ReconcileResult{}, fmt.Errorf("intent: --impact is the judgement %s carries into shipped/, and this close ships nothing — %s is still open on %s; re-run without --impact, and supply it at the close that ships",
				intentID, strings.Join(specIDs(held), ", "), intentID)
		}
		if remainder.Slug != "" {
			return ReconcileResult{}, fmt.Errorf("intent: --impact is the judgement %s carries into shipped/, and a remainder spec leaves it planned; re-run without --impact, and supply it at the close that ships",
				intentID)
		}
	}

	// Mint the remainder FIRST, before any move: a failure here leaves the spec
	// open and the intent planned, and nothing at all has been written.
	//
	// The mint is IDEMPOTENT, because a failure at any LATER step of this
	// operation leaves the remainder already on disk. Re-running the same command
	// — which is the documented recovery, and the only one the operator has — then
	// minted a second remainder for the same work, and a third on the next
	// attempt: the ledger grew a spec per retry while the state the retry was
	// trying to reach never arrived. So an OPEN spec that already realises this
	// intent under the requested slug IS the remainder, and is reused. It is
	// already counted in `held` (it is an open spec naming the intent and is not
	// the closing spec), so it is not appended a second time.
	var minted spec.Spec
	mintedHere := false
	if remainder.Slug != "" {
		if existing, ok := openRemainderWithSlug(claimers, remainder.Slug, specID); ok {
			minted = existing
		} else {
			minted, err = spec.CreateWithSteps(repoRoot, intentID, remainder.Slug, remainder.ProductionMode, carried)
			if err != nil {
				return ReconcileResult{}, err
			}
			mintedHere = true
			held = append(held, minted)
		}
	}

	// Impact gate, ahead of every write: shipped/ is the one bucket
	// intent_impact_valid requires an impact in, and this is the one verb that
	// moves a record there. Resolving it here — not after the move — is what
	// keeps abcd from producing, out of its own verbs alone, a record its own
	// record-lint refuses (iss-126).
	stamp := ""
	if it.Bucket == BucketPlanned && len(held) == 0 {
		if stamp, err = resolveShipImpact(repoRoot, it, impact); err != nil {
			return ReconcileResult{}, err
		}
	}

	res := ReconcileResult{Spec: sp, Intent: it, From: it.Bucket, To: it.Bucket, Remainder: minted, RemainderMinted: mintedHere, OpenSpecs: specIDs(held)}
	if mintedHere {
		res.RemainderSteps = carried
	}
	// 1. Advance the intent planned/ → shipped/ FIRST — but only when this close
	// leaves no open spec naming it. Its (kind, spec_id) are already set (Plan
	// wrote them), and its impact is either already recorded or stamped just
	// below, so the shipped record is lint-valid. If this fails, the spec stays
	// open — the whole operation retries cleanly.
	if it.Bucket == BucketPlanned && len(held) == 0 {
		// The read, the hold check, the stamp and the move are ONE critical
		// section under the store's advisory lock, and the hold is judged on the
		// bytes read HERE — the bytes the stamp writes from and the rename moves.
		// The refusal on the corpus above is the early one; a hold landing after
		// that load, through `intent hold` (which takes this lock, re-reads and
		// writes atomically), was otherwise shipped intact under it: nothing
		// re-read the record between the corpus and the rename. Refused here,
		// nothing has been written by this close — the remainder mint cannot have
		// run, because a remainder leaves the record planned and this branch is
		// never entered.
		abs := filepath.Join(repoRoot, it.Path)
		var dstRel string
		if err := withIntentMintLock(repoRoot, func() error {
			content, err := readIntentRefusingHold(abs, it.Path, it.ID, "spec close")
			if err != nil {
				return err
			}
			// The stamp is written while the record is still in planned/, where a
			// valid impact is equally lint-legal, so a failure at the move leaves a
			// consistent record and the retry finds the judgement already recorded.
			if stamp != "" {
				updated, err := setFrontmatterFields(content, map[string]string{"impact": stamp})
				if err != nil {
					return err
				}
				if err := writeIntentFile(abs, it.Path, updated); err != nil {
					return err
				}
			}
			dstRel, err = moveIntentToBucket(repoRoot, it.Path, BucketShipped)
			return err
		}); err != nil {
			return ReconcileResult{}, err
		}
		it.Bucket = BucketShipped
		it.Path = dstRel
		res.Intent = it
		res.IntentMoved = true
		res.To = BucketShipped
	}

	// 2. Close the spec, but only if still open — a re-run on an already-closed
	// spec is a clean completion, not the "already closed" error spec.Close raises.
	specMovedNow := sp.Status == spec.StatusOpen
	if specMovedNow {
		closed, err := spec.Close(repoRoot, specID)
		if err != nil {
			return ReconcileResult{}, err
		}
		res.Spec = closed
	}

	// 2b. Repoint every link that named either record's old path. The close is
	// the one place that knows both paths, so a close that reports success
	// hands on a tree record-lint accepts rather than a links_resolve refusal
	// the next command meets (iss-2609091732329046). The moves are derived from
	// the records' current buckets, not from what THIS invocation moved, so a
	// re-run after a failure here completes the repoint of every other file's
	// links. Only a record THIS invocation moved has its own links re-read from
	// the folder it left: a record an earlier run moved may have been edited
	// where it is now. A failure is reported, not raised: the records have moved
	// and the close stands.
	res.Relinked, err = relink.Repoint(repoRoot, closeMoves(res, specMovedNow))
	if err != nil {
		res.RelinkError = err.Error()
	}

	// 3. Emit the fidelity-review OWED stub + ephemeral request over the shipped
	// intent. The review is REPORT-ONLY: a failure here is captured, not raised —
	// the intent has already shipped and must not be un-shipped by a review-emit
	// error. The surface prints AuditEmitError loudly.
	if res.Intent.Bucket == BucketShipped {
		emit, err := emitAuditForIntent(repoRoot, res.Intent)
		if err != nil {
			res.AuditEmitError = err.Error()
		} else {
			res.ReceiptID = emit.ReceiptID
			res.ReceiptStatus = emit.Status
		}
	}
	return res, nil
}

// closeMoves names the renames a close stands for, derived from where the two
// records are now: a closed spec left open/, a shipped intent left planned/.
// Deriving rather than recording what this call moved is what makes the repoint
// idempotent — relink.Repoint ignores a move the tree does not show. Each move
// is marked MovedNow only when this call made it (specMovedNow: the spec was
// open at entry; res.IntentMoved), so a re-run never re-reads an already-moved
// record's own links from the folder it left.
func closeMoves(res ReconcileResult, specMovedNow bool) []relink.Move {
	var moves []relink.Move
	if res.Spec.Status == spec.StatusClosed {
		moves = append(moves, relink.Move{
			From:     filepath.Join(spec.SpecsRelDir, spec.StatusOpen, filepath.Base(res.Spec.Path)),
			To:       res.Spec.Path,
			MovedNow: specMovedNow,
		})
	}
	if res.Intent.Bucket == BucketShipped {
		moves = append(moves, relink.Move{
			From:     filepath.Join(IntentsRelDir, BucketPlanned, filepath.Base(res.Intent.Path)),
			To:       res.Intent.Path,
			MovedNow: res.IntentMoved,
		})
	}
	return moves
}

// otherOpenSpecs narrows a set of specs realising one intent to the OPEN ones
// that are not the spec being closed — the specs that, after this close, still
// hold the intent in planned/. It is the single question the 1:n lifecycle asks
// (adr-2609151513118583), and the answer is derived from the specs' own
// back-links, never from a list on the intent.
//
// The exclusion is canonical (spec.SameNum), for the same reason the
// bidirectional check is: a caller may name the spec bare, slug-suffixed or
// zero-padded, and all three name one record.
func otherOpenSpecs(claimers []spec.Spec, closingID string) []spec.Spec {
	var out []spec.Spec
	for _, c := range claimers {
		if c.Status != spec.StatusOpen || spec.SameNum(c.ID, closingID) {
			continue
		}
		out = append(out, c)
	}
	return out
}

// openRemainderWithSlug finds an OPEN spec already realising this intent under
// the requested slug, excluding the spec being closed. It is what makes the
// remainder mint idempotent: a retry after a failure downstream of the mint
// recognises the spec the previous attempt left behind instead of minting a
// second one for the same remaining work.
//
// The slug is the whole identity test on purpose. The operator names the
// remainder by slug and by nothing else, so two closes asking for `the-rest` on
// one intent are one request repeated; a genuinely different second remainder is
// asked for under a different slug and mints normally.
func openRemainderWithSlug(claimers []spec.Spec, slug, closingID string) (spec.Spec, bool) {
	for _, c := range claimers {
		if c.Status != spec.StatusOpen || spec.SameNum(c.ID, closingID) {
			continue
		}
		if c.Slug == slug {
			return c, true
		}
	}
	return spec.Spec{}, false
}

// specIDs renders a spec set as its ids, for a refusal or a result.
func specIDs(specs []spec.Spec) []string {
	if len(specs) == 0 {
		return nil
	}
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.ID
	}
	return out
}

// shipImpactValues is the vocabulary a shipping intent may declare, spelled for
// a human reading a refusal. `internal` is legal on an issue and a category
// error on an intent — an intent is press-release-first, so "invisible to
// users" is not a judgement it can hold — which is exactly the rule
// intent_impact_valid applies at shipped/ and CreateFromText applies at the
// seed.
const shipImpactValues = "additive|breaking|fix"

// resolveShipImpact decides the impact a planned intent will carry into
// shipped/, and returns the value to STAMP — empty when the record already
// carries its own judgement and nothing needs writing.
//
// The judgement itself is a human's, never the tool's: there is no default,
// because the impact decides the derived version of the release this intent
// lands in. So the four cases are settled without ever guessing one:
//
//   - neither the record nor the caller has one → refuse, naming the flag. This
//     is iss-126: without the refusal a seed that never got a judgement travels
//     drafts → planned → shipped through abcd's own verbs and lands in the one
//     bucket abcd's own record-lint refuses it in.
//   - only the caller has one → stamp it, after the same validation the seed
//     path applies, so the tool cannot write a value the gate would reject.
//   - only the record has one → validate it and write nothing. A record
//     carrying a misspelling or `internal` is refused here rather than moved
//     into the bucket where the blocker would catch it as archaeology.
//   - both → they must agree. A close is not the place to revise a recorded
//     judgement: silently overwriting it would let `--impact` rewrite history
//     as a side effect of shipping, so a disagreement is refused and the human
//     edits the record they meant to change.
func resolveShipImpact(repoRoot string, it Intent, supplied string) (string, error) {
	abs := filepath.Join(repoRoot, it.Path)
	data, err := readRepoFile(abs, it.Path)
	if err != nil {
		return "", err
	}
	recorded := recordedImpact(string(data))
	supplied = strings.TrimSpace(supplied)

	switch {
	case recorded == "" && supplied == "":
		return "", fmt.Errorf("intent: %s has no impact and none was supplied; shipped/ requires one of %s (it decides the derived version, and there is no default) — re-run with --impact, or record the judgement in %s first", it.ID, shipImpactValues, it.Path)
	case supplied == "":
		if err := validShipImpact(recorded); err != nil {
			return "", fmt.Errorf("intent: %s records %w; refusing to ship a record its own record-lint would refuse", it.ID, err)
		}
		return "", nil
	case recorded == "":
		if err := validShipImpact(supplied); err != nil {
			return "", fmt.Errorf("intent: --impact %w", err)
		}
		return supplied, nil
	case recorded != supplied:
		return "", impactDisagreement(it, recorded, supplied, "close")
	default:
		if err := validShipImpact(recorded); err != nil {
			return "", fmt.Errorf("intent: %s records %w; refusing to ship a record its own record-lint would refuse", it.ID, err)
		}
		return "", nil
	}
}

// resolvePlanImpact decides the impact `intent plan --impact` writes onto the
// record — the value to STAMP, or empty when nothing needs writing. Plan is the
// verb that runs when the judgement is made (the planning interview settles the
// impact class), so it is where a draft filed without one gets it, and the
// identity-step re-run over a planned record takes it for the same reason: a
// planned record without a judgement is the record the close refuses
// (iss-2609170726457256). The rules are the create path's and the close's, not
// a third set:
//
//   - nothing supplied → nothing written, whatever the record carries. The bare
//     verb behaves as it always has, and the judgement stays owed to the close.
//   - supplied, record has none → validated at the one bar CreateFromText and
//     resolveShipImpact apply (validShipImpact: a legal member of the vocabulary,
//     never `internal`), then stamped. The value is taken as typed, as the create
//     path takes it: `Fix` and ` fix` are refused, not repaired.
//   - supplied, record disagrees → refused before anything is written, in the
//     shape the close uses: a plan does not revise a recorded judgement either.
//   - supplied, record agrees → a no-op, validated so a hand-typed value the gate
//     would refuse is named here rather than carried on.
func resolvePlanImpact(it Intent, content, supplied string) (string, error) {
	if supplied == "" {
		return "", nil
	}
	recorded := recordedImpact(content)
	switch {
	case recorded == "":
		if err := validShipImpact(supplied); err != nil {
			return "", fmt.Errorf("intent: --impact %w", err)
		}
		return supplied, nil
	case recorded != supplied:
		return "", impactDisagreement(it, recorded, supplied, "plan")
	default:
		if err := validShipImpact(supplied); err != nil {
			return "", fmt.Errorf("intent: --impact %w", err)
		}
		return "", nil
	}
}

// impactDisagreement is the one refusal every verb that takes --impact over an
// existing record shares: the record already holds a judgement and the flag
// says otherwise. Neither verb revises it as a side effect — the human edits the
// record they meant to change — and the two refusals read the same so a reader
// of one recognises the other.
func impactDisagreement(it Intent, recorded, supplied, verb string) error {
	return fmt.Errorf("intent: %s already records impact %q but --impact says %q; a %s does not revise a recorded judgement — edit %s if the judgement changed", it.ID, recorded, supplied, verb, it.Path)
}

// recordedImpact reads the impact a record's frontmatter carries, with a null
// or absent field read as "none".
func recordedImpact(content string) string {
	recorded := frontmatter.Fields(strings.Split(content, "\n"))["impact"].Value
	if frontmatter.IsNull(recorded) {
		return ""
	}
	return recorded
}

// validShipImpact applies the shipped/ bar to one impact value: a legal member
// of the changelog vocabulary, and not `internal`. It is the same pair of checks
// CreateFromText makes at the seed, so a value either boundary accepts survives
// unchanged into shipped/ and passes intent_impact_valid there.
func validShipImpact(value string) error {
	imp, err := changelog.ParseImpact(value)
	if err != nil {
		return fmt.Errorf("impact %q, which is not one of %s (lower-case, no surrounding whitespace)", value, shipImpactValues)
	}
	if imp == changelog.ImpactInternal {
		return fmt.Errorf("impact internal, which an intent may not hold — a press-release-first intent is user-facing by definition; declare one of %s, or record the work as an issue instead", shipImpactValues)
	}
	return nil
}

// writeIntentFile is the one way this package writes an intent record — the
// seed, the lifecycle rewrites and the audit-block upserts all go through it,
// so the claim is a fact rather than a description of most of them. It caps
// the FINAL bytes before writing them, because the cap belongs at the write and
// not at any one producer: readRepoFile refuses a record over the cap, and every
// verb here loads the WHOLE corpus first, so one oversized record makes every
// intent command refuse every record until a human trims the file by hand. Each
// write is preceded by a growth step — an identity stamp, a `kind` rewrite, a
// `spec_id` rewrite — and guarding only the first of them left the others free to
// carry a record over the line (iss-2608300318192814).
//
// rel is repo-relative: an intent error never carries an absolute local path.
func writeIntentFile(abs, rel, content string) error {
	if len(content) > maxIntentFileBytes {
		return fmt.Errorf("intent: writing %s would produce %d bytes, past the %d-byte cap its own reader enforces; refusing", rel, len(content), maxIntentFileBytes)
	}
	if err := fsutil.WriteFileAtomic(abs, []byte(content), 0o644); err != nil {
		return fmt.Errorf("intent: writing %s: %w", rel, err)
	}
	return nil
}

// moveIntentToBucket moves the intent file at srcRel into dstBucket via os.Rename
// (atomic on one filesystem), behind the store's trust guards: it refuses to
// follow a symlinked destination directory and refuses to clobber an existing
// destination file. It returns the new repo-relative path. This is the single
// canonical intent move, shared by Plan (drafts → planned) and Reconcile
// (planned → shipped).
func moveIntentToBucket(repoRoot, srcRel, dstBucket string) (string, error) {
	name := filepath.Base(srcRel)
	dstRelDir := filepath.Join(IntentsRelDir, dstBucket)
	dstRel := filepath.Join(dstRelDir, name)
	dstAbs := filepath.Join(repoRoot, dstRel)
	if err := ensureRecordDir(repoRoot, dstRelDir); err != nil {
		return "", err
	}
	if _, err := os.Lstat(dstAbs); err == nil {
		return "", fmt.Errorf("intent: refusing to overwrite existing %s", dstRel)
	}
	srcBucket := filepath.Base(filepath.Dir(srcRel))
	if err := os.Rename(filepath.Join(repoRoot, srcRel), dstAbs); err != nil {
		return "", fmt.Errorf("intent: moving %s %s->%s: %w", name, srcBucket, dstBucket, err)
	}
	return dstRel, nil
}

// Status builds the read-only lifecycle summary: intent counts by bucket, spec
// counts by status, and the intent↔spec links (every intent whose spec_id is
// non-null). Linked pairs are ordered by the corpus load order (bucket, then
// directory), which is deterministic.
func Status(repoRoot string) (StatusView, error) {
	corpus, err := Load(repoRoot)
	if err != nil {
		return StatusView{}, err
	}
	store, err := spec.Load(repoRoot)
	if err != nil {
		return StatusView{}, err
	}

	v := StatusView{Buckets: map[string]int{}, Linked: []LinkedPair{}, Intents: []IntentListing{}}
	for _, b := range Buckets {
		v.Buckets[b] = 0
	}
	for _, it := range corpus.Intents {
		v.Buckets[it.Bucket]++
		if !frontmatter.IsNull(it.SpecID) {
			v.Linked = append(v.Linked, LinkedPair{Intent: it.ID, Spec: it.SpecID})
		}
		l, err := listIntent(repoRoot, it)
		if err != nil {
			return StatusView{}, err
		}
		v.Intents = append(v.Intents, l)
	}
	bucketRank := map[string]int{}
	for i, b := range Buckets {
		bucketRank[b] = i
	}
	sort.SliceStable(v.Intents, func(i, j int) bool {
		a, b := v.Intents[i], v.Intents[j]
		if a.Bucket != b.Bucket {
			return bucketRank[a.Bucket] < bucketRank[b.Bucket]
		}
		return a.ID < b.ID
	})
	for _, sp := range store.Specs {
		if sp.Status == spec.StatusClosed {
			v.SpecsClosed++
		} else {
			v.SpecsOpen++
		}
	}
	return v, nil
}

// listIntent reads one intent for the status listing (iss-242): its H1 title,
// masked for the terminal a JSON consumer may print it to; whether its
// Acceptance Criteria clear the bar plan applies; and the date its id encodes.
func listIntent(repoRoot string, it Intent) (IntentListing, error) {
	data, err := readRepoFile(filepath.Join(repoRoot, it.Path), it.Path)
	if err != nil {
		return IntentListing{}, err
	}
	content := string(data)
	l := IntentListing{ID: it.ID, Bucket: it.Bucket, ACState: ACStateSeeded, Filed: filedFromID(it.ID)}
	if hasAcceptanceCriteria(content) {
		l.ACState = ACStateReal
	}
	for _, ln := range strings.Split(content, "\n") {
		if t, ok := strings.CutPrefix(strings.TrimRight(ln, "\r"), "# "); ok {
			l.Title = termsafe.Sanitize(strings.TrimSpace(t))
			break
		}
	}
	return l, nil
}

// filedFromID is the YYYY-MM-DD a timestamp intent id encodes in its
// yymmdd head (adr-45: itd-<yymmddHHMMSS><rrrr>), or nil for an ordinal id.
func filedFromID(id string) *string {
	num := strings.TrimPrefix(id, "itd-")
	if len(num) != 16 {
		return nil
	}
	d := "20" + num[0:2] + "-" + num[2:4] + "-" + num[4:6]
	return &d
}

// readRepoFile reads a repo file behind the trust-boundary guards: refuse a
// symlinked leaf, require a regular file, and cap the size. (Mirrors the spec
// store's private guard; a shared read-guard is a flagged consolidation target
// alongside ensureRecordDir.)
func readRepoFile(abs, rel string) ([]byte, error) {
	fi, err := os.Lstat(abs)
	if err != nil {
		return nil, fmt.Errorf("intent: stat %s: %w", rel, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("intent: %s is a symlink (refusing to follow)", rel)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("intent: %s is not a regular file", rel)
	}
	if fi.Size() > maxIntentFileBytes {
		return nil, fmt.Errorf("intent: %s exceeds the %d-byte cap", rel, maxIntentFileBytes)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("intent: reading %s: %w", rel, err)
	}
	return data, nil
}

// recordDirPerm is the mode a record directory is created with. 0o755, not the
// 0o700 the home-scoped stores use: these directories live in a shared worktree
// and are committed, so they carry the mode a checkout of them would have.
const recordDirPerm = 0o755

// ensureRecordDir creates repoRoot/rel, proving every level it creates is a real
// directory. It is a two-line adapter over fsutil.EnsureRealDirAll — the record
// store's error wording, and nothing else. The create-and-prove sequence itself
// used to live here, as one of three copies in the tree, and this one was the
// weak copy: it lstat'd the leaf and then called os.MkdirAll, which follows a
// symlinked ANCESTOR and creates the rest of the chain under its target. The
// doc comment recorded that hole rather than closing it. Routing through the
// canonical primitive closes it, because the walk proves each level as it goes
// (iss-2609091128479544).
func ensureRecordDir(repoRoot, rel string) error {
	if err := fsutil.EnsureRealDirAll(repoRoot, filepath.ToSlash(rel), recordDirPerm); err != nil {
		return fmt.Errorf("intent: creating %s: %w", rel, err)
	}
	return nil
}
