package capture

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/provenance"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// PromoteRequest is the input to Promote: graduate an issue into an intent.
// LinkIntent, when non-empty, selects the stamp-only mode that links an
// EXISTING draft (itd-N) instead of minting one — the repair path after a
// stamp failure, and the "I already filed the intent by hand" path.
type PromoteRequest struct {
	RepoRoot   string
	IssuesRoot string
	// ID is the record to graduate: an iss-N, or the rdi-N of a reading item
	// that has been dispositioned.
	ID         string
	LinkIntent string // itd-N; "" mints
	// Grounds is the REQUIRED conjecture behind the promotion, in the shared
	// `<token>: <text>` grammar (core/grounds). A capture routed to an intent
	// draft is a conjecture being pursued, and there is nothing to stage here:
	// promote mints the value in the same call, so it has no corpus to fix.
	Grounds string
	// ProductionMode is how the MINTED DRAFT's seed text was produced (itd-178),
	// or empty for the vocabulary's default. It has no effect in stamp-only mode,
	// where nothing is minted. The draft's `origin` is not a member here: promote
	// is the one shipped path that derives a record from another record, so it
	// derives extracted-from-record from what it did rather than from what it was
	// told.
	ProductionMode string
}

// PromoteResult is the outcome of a successful Promote. Paths are
// repo-relative. Linked reports stamp-only mode (no draft minted this call).
type PromoteResult struct {
	IssueID string `json:"issue_id"`
	// IssueStatus is the source record's status. For an issue that is its status
	// directory; for a reading item it is the STANDING DISPOSITION's state,
	// because that family's status signal is the presence of the keyed
	// disposition and never folder membership.
	IssueStatus State  `json:"issue_status"`
	IssuePath   string `json:"issue_path"`
	IntentID    string `json:"intent_id"`
	IntentPath  string `json:"intent_path"`
	Linked      bool   `json:"linked"`
	// BackEdgeKept names the record a linked draft's `related_issues` ALREADY
	// carried first, when this call appended beside it. An intent occasioned by
	// several records is promoted from one and joined to the others, so an
	// existing back-edge is kept first and reported rather than refused or
	// overwritten (itd-2609020625400169, first scope condition). Empty on every
	// other outcome, the mint included.
	BackEdgeKept string `json:"back_edge_kept,omitempty"`
	// Redacted / Degraded mirror TransitionResult: the grounds text is free prose
	// written to the same committed ledger, so it goes through the same redactor
	// and reports the same way. Rewriting somebody's reasoning in silence is worse
	// than not recording it.
	Redacted int    `json:"redacted,omitempty"`
	Degraded string `json:"redaction_degraded,omitempty"`
}

// stampWriteHook, when non-nil, replaces the atomic in-place write inside
// Promote's stamp step. It is a test-only seam (nil in production, zero
// overhead) used to force a deterministic post-mint stamp failure without
// relying on platform- or uid-dependent filesystem tricks (a chmod'd status
// dir is a no-op for root), mirroring removeSourceHook in commitTransition.
var stampWriteHook func(path string, data []byte) error

// beforeStampHook, when non-nil, fires between the pre-flight and the moment the
// stamp closure takes the ledger lock. It is a test-only seam (nil in production,
// zero overhead) that forces exactly the window a concurrent write would land in,
// so the under-lock re-checks are exercised rather than asserted. It fires
// OUTSIDE the lock on purpose: a hook that ran inside it could not write to the
// ledger it is meant to change.
var beforeStampHook func()

// Promote graduates an issue into an intent without retyping (spc-24, step 2
// of the record walk). Default mode mints an intent draft — slug reused from
// the issue, body carrying a by-id pointer to the issue rather than a copy
// (SSOT), the issue named in the draft's `related_issues` — then appends the
// minted itd-N to the issue's `related_intents`. The two halves are itd-4 AC3's
// bidirectional join, and the pair — an intent the record names that names the
// record back — is what "promoted" means (see promotedInto). Promotion is
// orthogonal to fix-status: the issue may sit in any status directory and never
// moves.
//
// Ordering + residue contract: mint first, stamp second. No cross-store lock
// is attempted — the ledger lock alone guards the stamp, exactly as transition
// does — so a failure after the mint leaves an orphan draft; the returned
// error names the draft and the stamp-only remedy
// (`capture promote <iss-N> --intent <itd-N> --grounds '...'`), carrying the
// promotion's own grounds, single-quoted, so the remedy runs as printed in any
// shell — an interactive one included.
//
// Every refusal that can be established from the bytes in hand is therefore
// raised BEFORE the mint: the grounds text, whether the record is one the
// validators would let the stamp write at all, and whether it can take the
// append. What is left to the stamp is what only a write under the lock can
// discover, which is the residue the remedy above is for — never a
// deterministic refusal that would leak one draft per attempt.
func Promote(req PromoteRequest) (PromoteResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return PromoteResult{}, err
	}
	if err := mutationPreamble(repoRoot, issuesRoot); err != nil {
		return PromoteResult{}, err
	}
	// The reading item is a DIFFERENT route with its own recorded reasoning: its
	// disposition record, which promoteReadingItem refuses to act without. It
	// takes no grounds argument, so the gate below would refuse it for a value
	// that route never writes — hence the dispatch runs first.
	if recordid.ValidReadingItemID(req.ID) {
		return promoteReadingItem(repoRoot, issuesRoot, req)
	}
	// BEFORE anything is minted or stamped. Promote's residue contract is
	// mint-first-stamp-second, so a refusal raised any later than here would leave
	// an orphan draft behind for a missing argument — the exact residue the rest
	// of this path works to avoid.
	g, gRedacted, gDegraded, err := optionalGrounds(repoRoot, "promote", req.Grounds)
	if err != nil {
		return PromoteResult{}, err
	}

	// Pre-flight outside the lock: locate and read the issue, refuse a
	// double-promote early (re-checked under the lock at stamp time).
	src, status, err := findIssue(issuesRoot, req.ID)
	if err != nil {
		return PromoteResult{}, err
	}
	content, _, err := readWithChecksum(src)
	if err != nil {
		return PromoteResult{}, err
	}
	fm, body, err := parseFrontmatterAndBody(content)
	if err != nil {
		return PromoteResult{}, err
	}
	// The validators run on the pre-flight bytes, not only under the lock after
	// the mint. A record the stamp could never validate — a frontmatter slug that
	// disagrees with its filename, a value outside an enum — used to reach the
	// mint anyway, fail the stamp on the invariant, and leave one orphan draft
	// per attempt with a repair verb that refused on the same invariant
	// (GHSA-cxmf-gw6r-2pf5). The stamp-step checks stay: they judge the
	// post-append bytes under the lock, which these cannot.
	if err := validateStrict(fm); err != nil {
		return PromoteResult{}, err
	}
	if err := validateInvariants(fm, status, src); err != nil {
		return PromoteResult{}, err
	}
	if into, err := promotedInto(repoRoot, req.ID, asStrList(fm["related_intents"]), ""); err != nil {
		return PromoteResult{}, err
	} else if into != "" {
		return PromoteResult{}, fmt.Errorf("%s is already promoted to %s; refusing to promote twice", req.ID, into)
	}
	// Establish that the RECORD can accept the append, before anything is minted.
	// requireGrounds above already gated the grounds TEXT; what it cannot answer
	// is whether the bytes it will be appended to can hold it. A record whose body
	// leaves a fence or a comment open masks everything appended below it, so the
	// stamp's read-back refuses — permanently, and identically on every retry,
	// including the retry through the repair verb the failure message names. With
	// the mint first, that left one orphan draft per attempt and a draft counter
	// climbing behind an operator who had no way to succeed (iss-2608301803423101).
	//
	// The dry run is the real append against the pre-flight bytes, discarded. It
	// is not a substitute for the guard under the lock — the file may change
	// between the two, and the write is judged again there — but the failure it
	// removes is the deterministic one, where the record could never have taken
	// the entry in the first place.
	if g != nil {
		if _, err := appendGrounds("promote", content, *g); err != nil {
			return PromoteResult{}, err
		}
	}

	var itdID, intentPath, backEdgeKept string
	linked := req.LinkIntent != ""
	if linked {
		// Stamp-only mode: the target intent must exist in the store (any bucket)
		// BEFORE anything is written — an unknown itd-N is a structural fault.
		// Existence is probed by filename through the shared record-id probe
		// (recordref.go), the same probe resolve's provenance flags use.
		if !reItdID.MatchString(req.LinkIntent) {
			return PromoteResult{}, fmt.Errorf("invalid itd-N identifier: %q", req.LinkIntent)
		}
		rel, ok := findRecordFile(repoRoot, intentStoreRelDirs(), req.LinkIntent)
		if !ok {
			return PromoteResult{}, fmt.Errorf("%s not found in the intent store; nothing stamped", req.LinkIntent)
		}
		itdID, intentPath = req.LinkIntent, rel
		// The intent half of the join, written before the ledger-locked stamp
		// exactly as the reading route writes it: idempotent, so a failure between
		// the two is completed by re-running the same command. A draft filed by
		// hand used to be linked from the issue side alone, which left a join that
		// read from one end only (itd-4 AC3).
		it, err := intent.AddRelatedIssue(repoRoot, req.LinkIntent, req.ID)
		if err != nil {
			return PromoteResult{}, err
		}
		if len(it.RelatedIssues) > 0 && it.RelatedIssues[0] != req.ID {
			backEdgeKept = it.RelatedIssues[0]
		}
	} else {
		// Mint mode: reuse the issue's slug and seed a draft that POINTS at the
		// issue by id — never a copy of its body (the issue record stays the
		// single source of the observation).
		slug := asString(fm["slug"])
		title := issueTitleLine(body, slug)
		seed := "Graduated from `" + req.ID + "`: " + title +
			". Read that issue record for the source observation."
		it, err := intent.CreateDraft(repoRoot, intent.DraftOptions{
			Slug:         slug,
			Title:        title,
			SeedBody:     seed,
			RelatedIssue: req.ID,
			// The one arrival path a command derives from what it did (itd-178).
			// An issue is something a PERSON noticed, so promoting one keeps
			// saying extracted-from-record; the reading route below is the one
			// that names a run and an item.
			Origin:         provenance.Origin{Kind: provenance.KindExtractedFromRecord},
			ProductionMode: req.ProductionMode,
		})
		if err != nil {
			return PromoteResult{}, err
		}
		itdID, intentPath = it.ID, it.Path
	}

	// Stamp second, under the ledger lock (the same flock every ledger mutation
	// takes). Re-find and checksum-re-read: the file may have transitioned
	// between the pre-flight and the lock.
	var stamped struct {
		path   string
		status State
	}
	stampErr := withLedgerLock(repoRoot, issuesRoot, func() error {
		src, status, err := findIssue(issuesRoot, req.ID)
		if err != nil {
			return err
		}
		content, _, err := readWithChecksum(src)
		if err != nil {
			return err
		}
		fm, _, err := parseFrontmatterAndBody(content)
		if err != nil {
			return err
		}
		// The intent this call is joining names the record back by now — the mint
		// and the link both wrote that half first — so it is left out of the
		// question: what is asked here is whether ANOTHER promotion landed between
		// the pre-flight and the lock.
		related := asStrList(fm["related_intents"])
		if into, err := promotedInto(repoRoot, req.ID, related, itdID); err != nil {
			return err
		} else if into != "" {
			return fmt.Errorf("%s is already promoted to %s; refusing to promote twice", req.ID, into)
		}
		newContent, err := setListField(content, "related_intents", appendUnique(related, itdID))
		if err != nil {
			return err
		}
		if g != nil {
			newContent, err = appendGrounds("promote", newContent, *g)
			if err != nil {
				return err
			}
		}
		newFM, _, err := parseFrontmatterAndBody(newContent)
		if err != nil {
			return err
		}
		if err := validateStrict(newFM); err != nil {
			return err
		}
		if err := validateInvariants(newFM, status, src); err != nil {
			return err
		}
		// In place, atomic — the file keeps its status directory (promotion is
		// not resolution). The write happens under the same lock as the re-read,
		// so no checksum window exists between them.
		write := fsutil.WriteFileAtomicPreserveMode
		if stampWriteHook != nil {
			write = stampWriteHook
		}
		if err := write(src, []byte(newContent)); err != nil {
			return err
		}
		stamped.path, stamped.status = src, status
		return nil
	})
	if stampErr != nil {
		if !linked {
			// The mint already happened; report the orphan and the repair verb. The
			// remedy carries the grounds this call was given: the issue route refuses
			// without them, so a remedy that named only --intent refused on its own
			// text for every orphan (iss-2609012037130181), and the repair stamps
			// the same conjecture the failed promotion was pursuing.
			return PromoteResult{}, fmt.Errorf(
				"%w — the minted draft %s (%s) is orphaned; complete the link with `abcd capture promote %s --intent %s --grounds %s`",
				stampErr, itdID, intentPath, req.ID, itdID, shellQuoted(g.String()))
		}
		return PromoteResult{}, stampErr
	}

	return PromoteResult{
		IssueID:      req.ID,
		IssueStatus:  stamped.status,
		IssuePath:    fsutil.RepoRel(repoRoot, stamped.path),
		IntentID:     itdID,
		IntentPath:   intentPath,
		Linked:       linked,
		BackEdgeKept: backEdgeKept,
		Redacted:     gRedacted,
		Degraded:     gDegraded,
	}, nil
}

// promotedInto names the intent the ledger record id was promoted into, or ""
// when it was promoted into none. "Promoted" is the PAIR, not either half: an
// intent the record's related_intents names that names the record back in its
// own related_issues (itd-4 AC3). Neither half alone says it — an issue may be
// captured already related to an intent it was never promoted into, and an
// intent names the record the moment it is minted, before the stamp lands.
//
// except is left out of the question: the intent the calling promote is
// joining, whose half is already written by the time the lock is taken.
//
// It reads the intent store only when the record names an intent at all, so a
// promote of a record naming none costs what it did before the join was two-sided.
func promotedInto(repoRoot, id string, related []string, except string) (string, error) {
	if len(related) == 0 {
		return "", nil
	}
	corpus, err := intent.Load(repoRoot)
	if err != nil {
		return "", err
	}
	for _, itd := range related {
		if itd == except {
			continue
		}
		it, ok := corpus.Lookup(itd)
		if !ok {
			continue
		}
		for _, back := range it.RelatedIssues {
			if back == id {
				return it.ID, nil
			}
		}
	}
	return "", nil
}

// PromotedInto names the intent iss was promoted into, or "" — the read-side
// form of the question promote asks before it writes, for a surface that
// renders a record's next move.
func PromotedInto(repoRoot string, iss Issue) (string, error) {
	return promotedInto(repoRoot, iss.ID, iss.RelatedIntents, "")
}

// appendUnique appends id to list unless list already carries it, returning a
// fresh slice either way so the caller's copy is never aliased.
func appendUnique(list []string, id string) []string {
	out := append([]string{}, list...)
	for _, have := range out {
		if have == id {
			return out
		}
	}
	return append(out, id)
}

// shellQuoted wraps s in SINGLE quotes for the shell a remedy is pasted into,
// spelling an embedded quote the only way single quoting can ('\”: close,
// escaped quote, reopen). It exists so the orphan remedy runs as printed: a
// repair command a person has to re-quote by hand is a remedy that fails on its
// own text.
//
// Single, not double: inside double quotes a POSIX shell still interprets four
// characters, which can be escaped one by one — but an INTERACTIVE bash or zsh
// also expands `!word` history there, and that one cannot be escaped by the
// writer of the string. Inside single quotes a shell interprets nothing at all,
// so the grounds arrive as one literal argument whatever they carry.
func shellQuoted(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// issueTitleLine derives the minted draft's title — the issue's one-line
// summary — from the first non-blank body line, whitespace-collapsed. A
// degenerate empty body falls back to the slug so the draft still carries an
// honest heading.
func issueTitleLine(body, fallback string) string {
	for _, line := range strings.Split(body, "\n") {
		if fields := strings.Fields(line); len(fields) > 0 {
			return strings.Join(fields, " ")
		}
	}
	return fallback
}

// promoteReadingItem graduates a DISPOSITIONED reading item into an intent
// draft. It is the same act as promoting an issue and shares its ordering
// (mint first, stamp second, no cross-store lock, the orphan named on failure)
// — with one refusal of its own in front.
//
// Item-to-intent without a disposition is the collapse this whole record family
// exists to prevent: it makes the action the answer, and leaves nothing able to
// show that the finding was ever weighed. So the disposition directory is probed
// BEFORE anything is minted, and its absence refuses. Circumventing the verb —
// writing the draft and the stamp by hand — is a lapse-log entry, not something
// this gate can see.
func promoteReadingItem(repoRoot, issuesRoot string, req PromoteRequest) (PromoteResult, error) {
	// Grounds belong to the ISSUE route, which has nowhere else to say why. A
	// reading item records its conjecture in its DISPOSITION, which this route
	// already refuses to act without, and nothing here writes req.Grounds — so an
	// operand supplied to this route is refused rather than written and ignored,
	// on the same rule an exit condition outside `held` is (reading.go). Accepting
	// it would report success over a conjecture that reached no record, which is
	// the evaporation the grounds argument exists to close.
	if strings.TrimSpace(req.Grounds) != "" {
		return PromoteResult{}, fmt.Errorf(
			"promote: %w: %s records its conjecture in its disposition, which this route already refuses to act without, so there is nothing here for grounds to say; nothing written",
			ErrGroundsRefused, req.ID)
	}
	src, err := findReadingItem(issuesRoot, req.ID)
	if err != nil {
		return PromoteResult{}, err
	}
	content, err := readRecordGuarded(src)
	if err != nil {
		return PromoteResult{}, err
	}
	fm, _, err := parseFrontmatterAndBody(content)
	if err != nil {
		return PromoteResult{}, err
	}
	if err := validateReadingStrict(fm); err != nil {
		return PromoteResult{}, err
	}
	// A reading item carries no loose relation — only promote writes its
	// related_intents — so any entry at all is the forward half of a promotion.
	if existing := asStrList(fm["related_intents"]); len(existing) > 0 {
		return PromoteResult{}, fmt.Errorf("%s is already promoted to %s; refusing to promote twice", req.ID, existing[0])
	}
	// The run half of the origin pair, taken from WHERE THE ITEM WAS FOUND: the
	// item's bucket IS its run directory, which is the same join the provenance
	// lint performs, so the value this path stamps resolves by construction. Belt
	// and braces, the record's own mandatory `run` must agree with the directory
	// it sits in; a disagreement is a ledger fault rather than a choice between
	// two answers, and it refuses before anything is minted.
	run := filepath.Base(filepath.Dir(src))
	if declared := asString(fm["run"]); declared != run {
		return PromoteResult{}, fmt.Errorf(
			"%w: %s sits in run %s but its record names %s; nothing minted",
			ErrInvariantViolation, req.ID, run, declared)
	}

	standing, err := standingDispositions(filepath.Join(issuesRoot, issueschema.DispositionsDir, req.ID))
	if err != nil {
		return PromoteResult{}, err
	}
	if err := refuseUnlessAcceptedGiven(issuesRoot, req.ID, standing); err != nil {
		return PromoteResult{}, err
	}
	state := issueschema.DispositionAccepted

	var itdID, intentPath, backEdgeKept string
	linked := req.LinkIntent != ""
	if linked {
		if !reItdID.MatchString(req.LinkIntent) {
			return PromoteResult{}, fmt.Errorf("invalid itd-N identifier: %q", req.LinkIntent)
		}
		rel, ok := findRecordFile(repoRoot, intentStoreRelDirs(), req.LinkIntent)
		if !ok {
			return PromoteResult{}, fmt.Errorf("%s not found in the intent store; nothing stamped", req.LinkIntent)
		}
		itdID, intentPath = req.LinkIntent, rel
		// The draft half of the join. It runs after the pre-flight and BEFORE the
		// ledger-locked stamp, so a failure between the two leaves a draft naming
		// the item and an item not yet naming the draft; re-running the same
		// command completes it, because this write is idempotent and the stamp is
		// the step that was missing.
		//
		// A back-edge already naming another record is not a refusal here: an
		// intent occasioned by several items is promoted from one, and the others
		// join it in the list. The existing edge stays first, the forward stamp is
		// still written, and the kept record is reported.
		it, err := intent.AddRelatedIssue(repoRoot, req.LinkIntent, req.ID)
		if err != nil {
			return PromoteResult{}, err
		}
		if len(it.RelatedIssues) > 0 && it.RelatedIssues[0] != req.ID {
			backEdgeKept = it.RelatedIssues[0]
		}
	} else {
		// The pattern named is the item's one durable one-liner and the only body
		// field every position carries, so it is what the draft is titled and
		// slugged from. The seed POINTS at the item rather than copying it: the
		// reading record stays the single source of what the instrument returned.
		title := asString(fm["pattern"])
		slug, err := normaliseSlug(deriveSlug(title))
		if err != nil {
			return PromoteResult{}, err
		}
		seed := "Graduated from `" + req.ID + "` (" + state + "): " + title +
			". Read that reading record for the instrument's own text."
		it, err := intent.CreateDraft(repoRoot, intent.DraftOptions{
			Slug:         slug,
			Title:        title,
			SeedBody:     seed,
			RelatedIssue: req.ID,
			// The third arrival path, and the one this command is the sole minter
			// of (itd-178's `contributed-by-reading`): the run and the item are
			// the pair read off `readings/<run>/<item>.md` above, so the value
			// resolves back to the record that occasioned the draft.
			Origin: provenance.Origin{
				Kind: provenance.KindContributedByReading, Run: run, Item: req.ID,
			},
			ProductionMode: req.ProductionMode,
		})
		if err != nil {
			return PromoteResult{}, err
		}
		itdID, intentPath = it.ID, it.Path
	}

	if beforeStampHook != nil {
		beforeStampHook()
	}
	stampErr := withLedgerLock(repoRoot, issuesRoot, func() error {
		src, err := findReadingItem(issuesRoot, req.ID)
		if err != nil {
			return err
		}
		content, err := readRecordGuarded(src)
		if err != nil {
			return err
		}
		fm, _, err := parseFrontmatterAndBody(content)
		if err != nil {
			return err
		}
		if existing := asStrList(fm["related_intents"]); len(existing) > 0 {
			return fmt.Errorf("%s is already promoted to %s; refusing to promote twice", req.ID, existing[0])
		}
		// Re-read the standing answer HERE, not only in the pre-flight. A
		// disposition landing between the two — an acceptance superseded by a
		// rejection while the mint runs — would otherwise leave a standing
		// `rejected` beside a forward stamp, a ledger holding both a refusal and
		// the admission it refused. Nothing can land after this check, because the
		// lock is held from here to the write.
		if err := refuseUnlessAccepted(issuesRoot, req.ID); err != nil {
			return err
		}
		newContent, err := setListField(content, "related_intents", []string{itdID})
		if err != nil {
			return err
		}
		newFM, _, err := parseFrontmatterAndBody(newContent)
		if err != nil {
			return err
		}
		if err := validateReadingStrict(newFM); err != nil {
			return err
		}
		write := fsutil.WriteFileAtomicPreserveMode
		if stampWriteHook != nil {
			write = stampWriteHook
		}
		return write(src, []byte(newContent))
	})
	if stampErr != nil {
		if !linked {
			return PromoteResult{}, fmt.Errorf(
				"%w — the minted draft %s (%s) is orphaned; complete the link with `abcd capture promote %s --intent %s`",
				stampErr, itdID, intentPath, req.ID, itdID)
		}
		return PromoteResult{}, stampErr
	}

	return PromoteResult{
		IssueID:      req.ID,
		IssueStatus:  State(state),
		IssuePath:    fsutil.RepoRel(repoRoot, src),
		IntentID:     itdID,
		IntentPath:   intentPath,
		Linked:       linked,
		BackEdgeKept: backEdgeKept,
	}, nil
}

// standingDispositionState reads the state of the item's standing disposition.
// More than one standing answer is a ledger fault the write path refuses, so it
// is reported here rather than silently resolved by picking one.
func standingDispositionState(issuesRoot, item string, standing []string) (string, error) {
	if len(standing) == 0 {
		return "", fmt.Errorf("%w: %s carries no standing disposition", ErrInvariantViolation, item)
	}
	if len(standing) > 1 {
		return "", fmt.Errorf("%w: %s carries %d standing dispositions (%s); exactly one answer is in force at a time",
			ErrInvariantViolation, item, len(standing), renderList(standing))
	}
	path := filepath.Join(issuesRoot, issueschema.DispositionsDir, item, standing[0]+".md")
	// The third reader of a disposition file, and it needs the same guard as the
	// other two: the state read here is what licenses the stamp, so a symlinked
	// record would license it from outside the ledger. One primitive, opened once
	// with O_NOFOLLOW and validated on the same descriptor — no stat-then-read
	// window for a racing writer to swap a link into.
	content, err := readRecordGuarded(path)
	if err != nil {
		return "", err
	}
	fm, _, err := parseFrontmatterAndBody(content)
	if err != nil {
		return "", err
	}
	return asString(fm["state"]), nil
}

// refuseUnlessAccepted re-reads the item's standing answer and refuses anything
// but an acceptance. It is the under-lock form of the pre-flight check, sharing
// its wording so the two can never describe the rule differently.
func refuseUnlessAccepted(issuesRoot, item string) error {
	standing, err := standingDispositions(filepath.Join(issuesRoot, issueschema.DispositionsDir, item))
	if err != nil {
		return err
	}
	return refuseUnlessAcceptedGiven(issuesRoot, item, standing)
}

// refuseUnlessAcceptedGiven is the rule itself, over a standing set the caller
// has already read.
//
// `accepted` is the one standing state a promotion follows from: acceptance is
// the record, and the action it licenses is a separate admission. An
// undispositioned item collapses the two acts into one, so nothing could show
// the finding was weighed before it was acted on. A `rejected` or `declined`
// one would let the action contradict the record it is supposed to follow from.
// A `held` one would settle by action exactly what the hold left open.
func refuseUnlessAcceptedGiven(issuesRoot, item string, standing []string) error {
	if len(standing) == 0 {
		return fmt.Errorf(
			"%s carries no disposition; an item is answered before it is acted on, and promoting an undispositioned item collapses the two acts into one — record the disposition first (abcd capture disposition %s --state <state> ...)",
			item, item)
	}
	state, err := standingDispositionState(issuesRoot, item, standing)
	if err != nil {
		return err
	}
	if state != issueschema.DispositionAccepted {
		return fmt.Errorf(
			"%s carries a standing disposition of %q, and only %q licenses an action; supersede it with a new disposition first (abcd capture disposition %s --state accepted --grounds \"...\" --supersedes %s)",
			item, state, issueschema.DispositionAccepted, item, standing[0])
	}
	return nil
}
