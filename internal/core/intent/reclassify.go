package intent

// reclassify.go is `abcd intent reclassify` (itd-34, spc-2609211859391533
// scopes 3 and 4): the one verb that changes an intent's binding kind after it
// was set, so a late change is one command whose refusals name what is missing
// rather than a hand edit that leaves a one-way supersession link or a kind that
// disagrees with its shelf.
//
// Two shapes of change, and what each writes:
//
//   - a KIND change, standalone ↔ bundle-member, on a draft or a planned
//     record. The shelf does not move; the kind (and the bundle, set or
//     cleared) is rewritten and the change appended to
//     `reclassification_history`.
//   - a SUPERSESSION, `--kind superseded --by <itd-M|adr-N> --reason`, on any
//     live record. The record moves to superseded/ carrying `superseded_by`,
//     `kind_at_supersession`, the history entry and the supersession note under
//     its title; the successor's `supersedes` gains the record in the SAME
//     write, so the link is never one-way; and when the record was one of a
//     bundle of two, the member left behind stays a bundle-member and its
//     history says the bundle now has one member (decision 3).
//
// Two rulings bound it. A shipped intent never changes kind (decision 4): a
// rule discovered after the fact is filed as a discipline that supersedes it,
// so `--kind discipline` on a shipped record is refused with that remedy — and
// reclassify writes no discipline at all, because a discipline is a `## Rule`
// record, not a relabelled press release. Dissolving a bundle is out of scope
// (decision 3), so a planned member whose spec is its bundle's shared spec does
// not leave it by a kind change.
//
// One acquisition of the intent store's mint lock holds the whole verb: the
// corpus every judgement reads is loaded under it and every write is made from
// bytes read under it, every refusal fires before the first write, and a
// failure part way through puts back every file already written.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/decide"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/relink"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// KindSuperseded is the reclassify target that retires a record to
// superseded/. It is a target, never a persisted kind: the retired record
// keeps the kind it had, and records it again as kind_at_supersession.
const KindSuperseded = "superseded"

// ReclassificationHistoryKey is the append-only kind-change log.
const ReclassificationHistoryKey = "reclassification_history"

// BundleOfOnePhrase is how a survivor's history line states that its bundle
// now has one member (decision 3), with the bundle's name in place of %s.
// record_schema reads the same words as the bundle-of-one declaration.
const BundleOfOnePhrase = "bundle %s now has one member"

// ReclassifyRequest parameterises Reclassify.
type ReclassifyRequest struct {
	// Kind is the target: standalone, bundle-member, superseded (or discipline,
	// which is refused with its remedy).
	Kind string
	// Bundle names the bundle a bundle-member target joins; required there and
	// refused anywhere else.
	Bundle string
	// By is the successor of a superseded target, an intent (itd-M) or an ADR
	// (adr-N); required there and refused anywhere else.
	By string
	// Reason is why, one line, redacted before it is written; required for a
	// supersession and optional for a kind change.
	Reason string
	// Date is the history entry's date, YYYY-MM-DD; empty means today (UTC).
	Date string
}

// PathMove is one record a reclassify moved.
type PathMove struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ReclassifyResult reports a completed Reclassify.
type ReclassifyResult struct {
	IntentID string `json:"intent_id"`
	FromKind string `json:"from_kind"`
	ToKind   string `json:"to_kind"`
	Bucket   string `json:"bucket"`
	Path     string `json:"path"`
	// Moved names every record this call moved, and Written every record it
	// wrote (the moved record at its new path, the successor, the survivor).
	Moved   []PathMove `json:"moved"`
	Written []string   `json:"written"`
	// Successor is the record a supersession named, and Survivor the member it
	// left alone in its bundle (empty when it left none, or several).
	Successor string `json:"successor,omitempty"`
	Survivor  string `json:"survivor,omitempty"`
	// OpenSpecs names the open specs that still name a record this call
	// superseded: a fact to act on, not a refusal.
	OpenSpecs   []string         `json:"open_specs,omitempty"`
	Reason      string           `json:"reason,omitempty"`
	Redacted    int              `json:"redacted,omitempty"`
	Relinked    []relink.Rewrite `json:"relinked,omitempty"`
	RelinkError string           `json:"relink_error,omitempty"`
}

// Reclassify changes one intent's kind, or retires it as superseded, in one
// write (criteria 3 and 4). See the file comment for what each shape writes
// and what it refuses.
func Reclassify(repoRoot, intentID string, req ReclassifyRequest) (ReclassifyResult, error) {
	if !recordid.ValidIntentID(intentID) {
		return ReclassifyResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
	}
	switch req.Kind {
	case KindStandalone, KindBundleMember, KindSuperseded, KindDiscipline:
	default:
		return ReclassifyResult{}, fmt.Errorf("intent: --kind %q is not one of standalone, bundle-member, superseded (nothing written)", req.Kind)
	}
	if req.By != "" && req.Kind != KindSuperseded {
		return ReclassifyResult{}, fmt.Errorf("intent: --by names a successor, which only --kind superseded takes (nothing written)")
	}
	if req.Bundle != "" && req.Kind != KindBundleMember {
		return ReclassifyResult{}, fmt.Errorf("intent: --bundle names the bundle a bundle-member joins, which only --kind bundle-member takes (nothing written)")
	}
	date := req.Date
	if date == "" {
		date = time.Now().UTC().Format(time.DateOnly)
	}
	if _, err := time.Parse(time.DateOnly, date); err != nil {
		return ReclassifyResult{}, fmt.Errorf("intent: date %q must be YYYY-MM-DD (nothing written)", date)
	}
	// The corpus is loaded, and every judgement made on it, under the store
	// lock the write takes: the bundle's other namers a join depends on, the
	// survivor a supersession writes to, the successor and the record's own
	// kind. Judged on a corpus loaded before the lock, a reclassify landing in
	// the window went unseen — a join into a bundle its last other namer had
	// just left, or a supersession leaving a member alone with no line saying
	// so (iss-2609261215159796).
	var res ReclassifyResult
	if err := withIntentMintLock(repoRoot, func() error {
		corpus, err := Load(repoRoot)
		if err != nil {
			return err
		}
		it, ok := corpus.Lookup(intentID)
		if !ok {
			return fmt.Errorf("intent: %s not found in any bucket", intentID)
		}
		if req.Kind == KindDiscipline {
			if it.Bucket == BucketShipped {
				return fmt.Errorf("intent: %s is shipped, and a shipped intent never changes kind; a rule found after the fact is filed as a new discipline — file a discipline that supersedes it, then `abcd intent reclassify %s --kind superseded --by <that discipline> --reason \"…\"` (nothing written)", it.ID, it.ID)
			}
			return fmt.Errorf("intent: reclassify writes no discipline — a discipline is a `## Rule` record on disciplines/, not a relabelled %s record; file a discipline that supersedes it, then `abcd intent reclassify %s --kind superseded --by <that discipline> --reason \"…\"` (nothing written)", it.Bucket, it.ID)
		}
		if it.Bucket == BucketSuperseded {
			return fmt.Errorf("intent: %s is already superseded (nothing written)", it.ID)
		}
		if err := refuseIfHeld(it, "reclassify"); err != nil {
			return err
		}
		reason, redacted, err := reclassifyReason(repoRoot, req)
		if err != nil {
			return err
		}
		if req.Kind == KindSuperseded {
			res, err = supersede(repoRoot, corpus, it, req.By, reason, redacted, date)
		} else {
			res, err = changeKind(repoRoot, corpus, it, req, reason, redacted, date)
		}
		return err
	}); err != nil {
		return ReclassifyResult{}, err
	}
	if res.ToKind != KindSuperseded {
		return res, nil
	}
	// After the lock, as every close's repoint is: the record has moved and the
	// supersession stands, so what follows is reported, never raised.
	if store, err := spec.Load(repoRoot); err == nil {
		res.OpenSpecs = specIDs(store.OpenSpecsForIntent(res.IntentID))
	}
	relinked, err := relink.Repoint(repoRoot, []relink.Move{{From: res.Moved[0].From, To: res.Moved[0].To, MovedNow: true}})
	res.Relinked = relinked
	if err != nil {
		res.RelinkError = err.Error()
	}
	return res, nil
}

// reclassifyReason redacts and validates the reason: required for a
// supersession, optional for a kind change, one line either way.
func reclassifyReason(repoRoot string, req ReclassifyRequest) (string, int, error) {
	trimmed := strings.TrimSpace(req.Reason)
	if trimmed == "" {
		if req.Kind == KindSuperseded {
			return "", 0, fmt.Errorf("intent: --reason is required with --kind superseded; the supersession note and the history entry both carry it (nothing written)")
		}
		return "", 0, nil
	}
	out, n, err := redactIntentText(repoRoot, trimmed)
	if err != nil {
		return "", 0, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return "", 0, fmt.Errorf("intent: the reason is empty after redaction and trimming (nothing written)")
	}
	for _, r := range out {
		if unicode.IsControl(r) {
			return "", 0, fmt.Errorf("intent: the reason must be a single line with no control characters (found U+%04X) (nothing written)", r)
		}
	}
	return out, n, nil
}

// changeKind is the standalone ↔ bundle-member change on a draft or planned
// record: the shelf stays, the kind and bundle are rewritten, and the change is
// appended to the history. Reclassify calls it under the store lock, with the
// corpus loaded there.
func changeKind(repoRoot string, corpus Corpus, it Intent, req ReclassifyRequest, reason string, redacted int, date string) (ReclassifyResult, error) {
	switch it.Bucket {
	case BucketDrafts, BucketPlanned:
	case BucketShipped:
		return ReclassifyResult{}, fmt.Errorf("intent: %s is shipped, and a shipped intent never changes kind; supersede it with `--kind superseded --by <itd-M|adr-N>` if a later record replaced it (nothing written)", it.ID)
	default:
		return ReclassifyResult{}, fmt.Errorf("intent: %s is in %s, whose shelf is its kind; supersede it with `--kind superseded --by <itd-M|adr-N>` instead (nothing written)", it.ID, it.Bucket)
	}
	from := it.Kind
	if frontmatter.IsNull(from) {
		from = "null"
	}
	fields := map[string]string{"kind": req.Kind}
	switch req.Kind {
	case KindBundleMember:
		if req.Bundle == "" {
			return ReclassifyResult{}, fmt.Errorf("intent: --kind bundle-member needs --bundle <name>, the bundle %s joins (nothing written)", it.ID)
		}
		if !slugRe.MatchString(req.Bundle) {
			return ReclassifyResult{}, fmt.Errorf("intent: bundle name %q must be kebab-case (nothing written)", req.Bundle)
		}
		if from == KindBundleMember && it.Bundle == req.Bundle {
			return ReclassifyResult{}, fmt.Errorf("intent: %s is already a bundle-member of %s; nothing to change (nothing written)", it.ID, req.Bundle)
		}
		named := false
		for _, other := range corpus.Intents {
			named = named || (other.ID != it.ID && other.Bundle == req.Bundle)
		}
		if !named {
			return ReclassifyResult{}, fmt.Errorf("intent: no other record names bundle %q, so %s would be a bundle of one nobody planned; plan a new bundle with `abcd intent plan <itd-A> <itd-B> --bundle %s` (nothing written)", req.Bundle, it.ID, req.Bundle)
		}
		fields[BundleKey] = req.Bundle
	case KindStandalone:
		if from == KindStandalone {
			return ReclassifyResult{}, fmt.Errorf("intent: %s is already standalone; nothing to change (nothing written)", it.ID)
		}
		if it.Bundle != "" {
			fields[BundleKey] = "null"
		}
	}
	// A planned member whose spec is its bundle's shared spec cannot leave the
	// bundle, or move to another, by a kind change: that is dissolving the
	// bundle, which decision 3 leaves out of scope.
	if it.Bucket == BucketPlanned && it.Bundle != "" {
		store, err := spec.Load(repoRoot)
		if err != nil {
			return ReclassifyResult{}, err
		}
		if sp, ok := store.Lookup(it.SpecID); ok && sp.Bundle != "" {
			return ReclassifyResult{}, fmt.Errorf("intent: %s is planned on %s, bundle %s's shared spec, and leaving it would dissolve the bundle, which reclassify does not do; supersede the member instead (nothing written)", it.ID, sp.ID, sp.Bundle)
		}
	}

	abs := filepath.Join(repoRoot, it.Path)
	if err := func() error {
		content, err := readIntentRefusingHold(abs, it.Path, it.ID, "reclassify")
		if err != nil {
			return err
		}
		updated, err := setFrontmatterFields(content, fields)
		if err != nil {
			return err
		}
		if updated, err = appendFrontmatterBlockItem(updated, ReclassificationHistoryKey, historyEntry(date, from, req.Kind, reason)); err != nil {
			return err
		}
		return writeIntentFile(abs, it.Path, updated)
	}(); err != nil {
		return ReclassifyResult{}, err
	}
	return ReclassifyResult{
		IntentID: it.ID, FromKind: from, ToKind: req.Kind, Bucket: it.Bucket, Path: it.Path,
		Moved: []PathMove{}, Written: []string{it.Path}, Reason: reason, Redacted: redacted,
	}, nil
}

// pendingWrite is one file a supersession rewrites: its bytes as read and as
// they will be written, so a failure can put it back.
type pendingWrite struct {
	abs, rel      string
	orig, updated string
	written       bool
}

// supersede retires it to superseded/ and writes both directions of the link
// and the survivor line in one all-or-nothing write. Reclassify calls it under
// the store lock, with the corpus loaded there, so the survivor set is the one
// the write finds.
func supersede(repoRoot string, corpus Corpus, it Intent, by, reason string, redacted int, date string) (ReclassifyResult, error) {
	if by == "" {
		return ReclassifyResult{}, fmt.Errorf("intent: --kind superseded needs --by <itd-M|adr-N>, the record that supersedes %s (nothing written)", it.ID)
	}
	succRel, succID, err := resolveSuccessor(repoRoot, corpus, it, by)
	if err != nil {
		return ReclassifyResult{}, err
	}
	kindAt := it.Kind
	if frontmatter.IsNull(kindAt) {
		kindAt = KindStandalone
	}
	dstRel := filepath.Join(IntentsRelDir, BucketSuperseded, filepath.Base(it.Path))
	if _, err := os.Lstat(filepath.Join(repoRoot, dstRel)); err == nil {
		return ReclassifyResult{}, fmt.Errorf("intent: refusing to overwrite existing %s (nothing written)", dstRel)
	}
	// The member a supersession leaves alone in its bundle — the one live record
	// still naming the bundle — says so in its own history (decision 3).
	var survivor Intent
	if kindAt == KindBundleMember && it.Bundle != "" {
		var others []Intent
		for _, o := range corpus.Intents {
			if o.ID != it.ID && o.Bundle == it.Bundle && o.Bucket != BucketSuperseded {
				others = append(others, o)
			}
		}
		if len(others) == 1 {
			survivor = others[0]
		}
	}

	recFields := map[string]string{"superseded_by": succID, "kind_at_supersession": kindAt, "kind": kindAt}
	if kindAt == KindBundleMember && it.Bundle != "" {
		recFields[BundleKey] = "null"
		recFields["bundle_at_supersession"] = it.Bundle
	}
	recAbs := filepath.Join(repoRoot, it.Path)
	var writes []*pendingWrite
	moved := false
	if err := func() error {
		recContent, err := readIntentRefusingHold(recAbs, it.Path, it.ID, "reclassify")
		if err != nil {
			return err
		}
		rec, err := setFrontmatterFields(recContent, recFields)
		if err != nil {
			return err
		}
		if rec, err = appendFrontmatterBlockItem(rec, ReclassificationHistoryKey, historyEntry(date, kindAt, KindSuperseded, reason)); err != nil {
			return err
		}
		rec = insertSupersessionNote(rec, "> **Superseded by "+succID+"** on "+date+": "+reason)
		recWrite := &pendingWrite{abs: recAbs, rel: it.Path, orig: recContent, updated: rec}

		succAbs := filepath.Join(repoRoot, succRel)
		succData, err := readRepoFile(succAbs, succRel)
		if err != nil {
			return err
		}
		succ, err := appendFrontmatterListItem(string(succData), "supersedes", it.ID)
		if err != nil {
			return fmt.Errorf("intent: %s: %w", succRel, err)
		}
		writes = append(writes, &pendingWrite{abs: succAbs, rel: succRel, orig: string(succData), updated: succ})

		if survivor.ID != "" {
			survAbs := filepath.Join(repoRoot, survivor.Path)
			survData, err := readRepoFile(survAbs, survivor.Path)
			if err != nil {
				return err
			}
			line := fmt.Sprintf(BundleOfOnePhrase, it.Bundle) + ": " + it.ID + " was superseded by " + succID
			surv, err := appendFrontmatterBlockItem(string(survData), ReclassificationHistoryKey, historyEntry(date, KindBundleMember, KindBundleMember, line))
			if err != nil {
				return fmt.Errorf("intent: %s: %w", survivor.Path, err)
			}
			writes = append(writes, &pendingWrite{abs: survAbs, rel: survivor.Path, orig: string(survData), updated: surv})
		}
		writes = append(writes, recWrite)
		for _, w := range writes {
			if len(w.updated) > maxIntentFileBytes {
				return fmt.Errorf("intent: writing %s would produce %d bytes, past the %d-byte cap its own reader enforces; refusing before any write", w.rel, len(w.updated), maxIntentFileBytes)
			}
		}

		// The successor and the survivor first, the record last, so the move —
		// the step a reader sees — is the one that completes the write.
		for _, w := range writes {
			w.written = true
			if err := writeIntentFile(w.abs, w.rel, w.updated); err != nil {
				return err
			}
		}
		if _, err := moveIntentToBucket(repoRoot, it.Path, BucketSuperseded); err != nil {
			return err
		}
		moved = true
		return nil
	}(); err != nil {
		if moved {
			_ = os.Rename(filepath.Join(repoRoot, dstRel), recAbs)
		}
		for i := len(writes) - 1; i >= 0; i-- {
			if writes[i].written {
				_ = writeIntentFile(writes[i].abs, writes[i].rel, writes[i].orig)
			}
		}
		return ReclassifyResult{}, err
	}

	res := ReclassifyResult{
		IntentID: it.ID, FromKind: kindAt, ToKind: KindSuperseded, Bucket: BucketSuperseded, Path: dstRel,
		Moved:     []PathMove{{From: it.Path, To: dstRel}},
		Written:   []string{dstRel, succRel},
		Successor: succID, Survivor: survivor.ID, Reason: reason, Redacted: redacted,
	}
	if survivor.ID != "" {
		res.Written = append(res.Written, survivor.Path)
	}
	return res, nil
}

// resolveSuccessor finds the record `--by` names: an intent anywhere but
// superseded/ and other than the record itself, or an ADR in the decision
// store. It returns the successor's path and its id as the tree spells it.
func resolveSuccessor(repoRoot string, corpus Corpus, it Intent, by string) (string, string, error) {
	switch {
	case recordid.ValidIntentID(by):
		succ, ok := corpus.Lookup(by)
		if !ok {
			return "", "", fmt.Errorf("intent: successor %s not found in any bucket; a successor must be present (nothing written)", by)
		}
		if succ.ID == it.ID {
			return "", "", fmt.Errorf("intent: %s cannot supersede itself (nothing written)", it.ID)
		}
		if succ.Bucket == BucketSuperseded {
			return "", "", fmt.Errorf("intent: successor %s is itself superseded; name the record in force (nothing written)", succ.ID)
		}
		return succ.Path, succ.ID, nil
	case recordid.CanonADRID(by) != "":
		canonical := recordid.CanonADRID(by)
		entries, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(decide.ADRsRelDir)))
		if err != nil && !os.IsNotExist(err) {
			return "", "", fmt.Errorf("intent: reading %s: %w", decide.ADRsRelDir, err)
		}
		for _, e := range entries {
			if e.Type().IsRegular() && recordid.ADRFileID(e.Name()) == canonical {
				return filepath.Join(filepath.FromSlash(decide.ADRsRelDir), e.Name()), canonical, nil
			}
		}
		return "", "", fmt.Errorf("intent: successor %s not found in %s; a successor must be present (nothing written)", canonical, decide.ADRsRelDir)
	}
	return "", "", fmt.Errorf("intent: --by %q names neither an intent (itd-N) nor an ADR (adr-N) (nothing written)", by)
}

// historyEntry renders one reclassification_history item in the flow-mapping
// shape the record's entries carry.
func historyEntry(date, from, to, reason string) string {
	entry := "{ date: " + date + ", from: " + from + ", to: " + to
	if reason != "" {
		entry += ", reason: " + frontmatter.QuoteScalar(reason)
	}
	return entry + " }"
}

// frontmatterClose returns the index of the frontmatter block's closing
// delimiter in lines, with setFrontmatterFields's delimiter tolerance.
func frontmatterClose(lines []string) (int, error) {
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t\r") != "---" {
		return 0, fmt.Errorf("intent: file has no leading frontmatter block")
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t\r") == "---" {
			return i, nil
		}
	}
	return 0, fmt.Errorf("intent: frontmatter block is not closed")
}

// frontmatterKeyLine returns the index of key's top-level line, or -1.
func frontmatterKeyLine(lines []string, closing int, key string) int {
	for i := 1; i < closing; i++ {
		if m := fmKeyRe.FindStringSubmatch(strings.TrimRight(lines[i], "\r")); m != nil && m[1] == key {
			return i
		}
	}
	return -1
}

// blockEnd returns the index just past the last item of the block sequence
// under the key at index k, and the indentation its items carry.
func blockEnd(lines []string, k, closing int) (int, string) {
	end, indent := k+1, "  "
	for i := k + 1; i < closing; i++ {
		raw := strings.TrimRight(lines[i], "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(trimmed, "- ") && !strings.HasPrefix(raw, " ") && !strings.HasPrefix(raw, "\t") {
			break
		}
		if strings.HasPrefix(trimmed, "- ") {
			indent = raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))]
		}
		end = i + 1
	}
	return end, indent
}

// appendFrontmatterBlockItem appends `- item` to the block sequence under key:
// an absent key, or one holding an empty list or null, becomes a block holding
// the one item; a block has the item appended after its last line, at its
// items' indentation. A non-empty inline list is refused rather than rewritten,
// since its items are not ours to re-spell.
func appendFrontmatterBlockItem(content, key, item string) (string, error) {
	lines := strings.Split(content, "\n")
	closing, err := frontmatterClose(lines)
	if err != nil {
		return "", err
	}
	k := frontmatterKeyLine(lines, closing, key)
	insert := func(at int, add ...string) string {
		out := make([]string, 0, len(lines)+len(add))
		out = append(out, lines[:at]...)
		out = append(out, add...)
		out = append(out, lines[at:]...)
		return strings.Join(out, "\n")
	}
	if k < 0 {
		return insert(closing, key+":", "  - "+item), nil
	}
	value := strings.TrimSpace(frontmatter.StripComment(strings.TrimRight(lines[k], "\r")[len(key)+1:]))
	switch {
	case value == "":
		end, indent := blockEnd(lines, k, closing)
		return insert(end, indent+"- "+item), nil
	case value == "[]" || frontmatter.IsNull(value):
		lines[k] = key + ":"
		return insert(k+1, "  - "+item), nil
	}
	return "", fmt.Errorf("`%s` is an inline list; rewrite it as a block sequence (one `- ` item per line) and re-run (nothing written)", key)
}

// appendFrontmatterListItem adds id to the list under key unless the list
// already names it (canonically): an absent, null or inline list is written
// inline, and a block sequence has the item appended.
func appendFrontmatterListItem(content, key, id string) (string, error) {
	lines := strings.Split(content, "\n")
	closing, err := frontmatterClose(lines)
	if err != nil {
		return "", err
	}
	for _, have := range frontmatterList(content, key) {
		if recordid.SameID(have, id) {
			return content, nil
		}
	}
	k := frontmatterKeyLine(lines, closing, key)
	if k >= 0 && strings.TrimSpace(frontmatter.StripComment(strings.TrimRight(lines[k], "\r")[len(key)+1:])) == "" {
		return appendFrontmatterBlockItem(content, key, id)
	}
	items := append(frontmatterList(content, key), id)
	return setFrontmatterFields(content, map[string]string{key: "[" + strings.Join(items, ", ") + "]"})
}

// insertSupersessionNote puts the note under the record's title, as the
// superseded records carry it; a record with no title has it at the top of
// its body.
func insertSupersessionNote(content, note string) string {
	lines := strings.Split(content, "\n")
	closing, err := frontmatterClose(lines)
	if err != nil {
		return content
	}
	at := closing + 1
	for i := closing + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "# ") {
			at = i + 1
			break
		}
	}
	add := []string{"", note}
	if at >= len(lines) || strings.TrimSpace(lines[at]) != "" {
		add = append(add, "")
	}
	out := make([]string, 0, len(lines)+len(add))
	out = append(out, lines[:at]...)
	out = append(out, add...)
	out = append(out, lines[at:]...)
	return strings.Join(out, "\n")
}
