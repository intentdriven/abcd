package intent

// delivery_audit.go is the fidelity audit the implement loop runs before the
// close (spc-2609202134338445 piece 8, under ruling AI of 2026-09-29: "audit
// ONCE, on the lane that closes the spec, over the whole delivery"). The ship
// transition's own emit parks its request only once the intent has shipped, which
// is after the landing the audit has to gate; this composes that same request
// earlier, so the audit runs once, on the closing lane, and the close consumes
// its verdict rather than asking for a second one.
//
// The request is the emit's own, byte for byte, as the close will issue it: the
// receipt id is a function of the intent id, the spec id and the Acceptance
// Criteria alone (receiptFor), and the prompt is composed against the path the
// close moves the intent to and the specs that are closed once this one is. So
// the verdict the auditor returns here echoes the policy hashes the close's
// receipt issues, and `abcd intent audit ingest` accepts it against the OWED
// marker the close parks. What the loop adds — the delivered range — sits in a
// section after the Provenance block, outside the hashed prompt, as the routing
// section does.

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// DeliveryAudit is one fidelity request issued before the close.
type DeliveryAudit struct {
	// ReceiptID is the receipt the close parks for the same record.
	ReceiptID string
	IntentID  string
	// ShippedPath is where the close moves the intent, the path the prompt names.
	ShippedPath string
	// Specs are the specs the delivery realises once the closing spec is closed.
	Specs []string
	// Criteria is the number of Acceptance Criteria the verdict judges.
	Criteria int
	// RubricHash and PromptHash are the provenance the auditor echoes.
	RubricHash, PromptHash string

	prompt, content string
}

// DeliveryVerdict is what the loop reads from a verdict the audit accepted.
type DeliveryVerdict struct {
	// Rollup is the verdict's acceptance rollup.
	Rollup map[string]int
	// Worst is the worst acceptance verdict any criterion carries, in the order
	// NOT_MET, INCONCLUSIVE, MET_WITH_CONCERNS, MET.
	Worst string
	// NotMet and Inconclusive name the criteria carrying those verdicts.
	NotMet, Inconclusive []string
}

// ComposeDeliveryAudit composes the fidelity request for intentID before its
// close. plannedRel is the intent's repository-relative path in planned/ and
// content its bytes as the delivery leaves them; realised are the specs closed
// once the delivery's last open spec closes, in the order the spec store lists
// them. The intent's id and spec_id are read from content, as the close reads
// them: the receipt is keyed on those. An intent whose content names another
// id, a spec_id that is not one, an intent outside planned/, one with no
// criteria to judge, and an empty realised list are refused.
func ComposeDeliveryAudit(intentID, plannedRel, content string, realised []string) (DeliveryAudit, error) {
	if filepath.Base(filepath.Dir(filepath.FromSlash(plannedRel))) != BucketPlanned {
		return DeliveryAudit{}, fmt.Errorf("intent: %s is at %s, not in %s/; only a planned intent is audited before its close", intentID, plannedRel, BucketPlanned)
	}
	it, err := parseIntent(plannedRel, content, BucketPlanned)
	switch {
	case err != nil:
		return DeliveryAudit{}, err
	case !recordid.ValidIntentID(intentID) || !recordid.SameID(it.ID, intentID):
		return DeliveryAudit{}, fmt.Errorf("intent: %s names id %q, not %s", plannedRel, it.ID, intentID)
	case !spec.HasNum(it.SpecID):
		return DeliveryAudit{}, fmt.Errorf("intent: %s has no well-formed spec_id (%q); refusing to compose a review", it.ID, it.SpecID)
	case len(realised) == 0:
		return DeliveryAudit{}, fmt.Errorf("intent: %s's delivery names no spec", it.ID)
	}
	k := countAcceptanceCriteria(content)
	if k == 0 {
		return DeliveryAudit{}, errors.New("intent: " + it.ID + " has no Acceptance Criteria bullets to audit")
	}
	it.Bucket = BucketShipped
	it.Path = filepath.Join(IntentsRelDir, BucketShipped, filepath.Base(filepath.FromSlash(plannedRel)))
	rcp := receiptFor(it.ID, it.SpecID, content)
	policy := auditPolicyFor(it, rcp, content, realised)
	return DeliveryAudit{
		ReceiptID: rcp, IntentID: it.ID, ShippedPath: it.Path, Specs: append([]string(nil), realised...),
		Criteria: k, RubricHash: policy.RubricHash, PromptHash: policy.PromptHash,
		prompt: auditPromptBody(it, rcp, content, realised), content: content,
	}, nil
}

// Request renders the request the auditor is handed: the prompt the close's emit
// composes, its Provenance block, and the delivered range, which the prompt
// leaves to the host to supply.
func (a DeliveryAudit) Request(delivered string) string {
	var b strings.Builder
	b.WriteString(a.prompt)
	b.WriteString(auditProvenanceBlock(auditPolicy{RubricHash: a.RubricHash, PromptHash: a.PromptHash}))
	b.WriteString("\n## Delivered (the range the host supplies; outside the prompt hash)\n\n")
	fmt.Fprintf(&b, "The intent is audited before the landing that ships it: it is in %s/ until that\n", BucketPlanned)
	fmt.Fprintf(&b, "landing closes its last open spec and moves it to %s, the path the\n", a.ShippedPath)
	b.WriteString("prompt names. The delivery the criteria are judged against is:\n\n")
	b.WriteString(strings.TrimRight(delivered, "\n") + "\n")
	return b.String()
}

// Check validates a verdict the auditor returned against this request, as the
// ingest does: the strict schema, every criterion judged once with cited
// evidence, every scope condition disposed, and the two policy hashes the ones
// this request issued. It returns what the loop records.
func (a DeliveryAudit) Check(raw []byte) (DeliveryVerdict, error) {
	if len(raw) > maxVerdictBytes {
		return DeliveryVerdict{}, fmt.Errorf("intent: the verdict is %d bytes, over the %d-byte cap", len(raw), maxVerdictBytes)
	}
	v, err := validateVerdict(raw, a.ReceiptID, a.content)
	if err != nil {
		return DeliveryVerdict{}, err
	}
	if v.Policy.RubricHash != a.RubricHash || v.Policy.PromptHash != a.PromptHash {
		return DeliveryVerdict{}, issuedPolicyRefusal("verdict", a.ReceiptID, v.Policy,
			auditPolicy{RubricHash: a.RubricHash, PromptHash: a.PromptHash}, "abcd implement step")
	}
	out := DeliveryVerdict{Rollup: map[string]int{}, Worst: "MET"}
	for k, n := range v.AcceptanceRollup {
		out.Rollup[k] = n
	}
	rank := map[string]int{"MET": 0, "MET_WITH_CONCERNS": 1, "INCONCLUSIVE": 2, "NOT_MET": 3}
	for _, c := range v.Criteria {
		switch c.Verdict {
		case "NOT_MET":
			out.NotMet = append(out.NotMet, c.CriterionID)
		case "INCONCLUSIVE":
			out.Inconclusive = append(out.Inconclusive, c.CriterionID)
		}
		if rank[c.Verdict] > rank[out.Worst] {
			out.Worst = c.Verdict
		}
	}
	return out, nil
}
