package ideate

// Example is one verdict payload at SchemaVersion that passes Check: the
// worked example the input registry shows an agent (spc-2610301431075964
// step 1). It lives beside the type it is an example of, so a change to the
// shape that the example no longer meets fails that package's own test rather
// than a page nobody re-reads. It cites no record and no corpus key, so it
// holds in any checkout.
const Example = `{
  "schema_version": 2,
  "prompt_version": "1.0.0",
  "idea": "abcd should gate a new idea before it becomes a record entry",
  "origin_sources": ["https://example.com/origin-post"],
  "legs": [
    {"kind": "research", "claims": [
      {"claim": "fresh-context evaluation is the only measured debiasing effect", "primary_source": "https://example.com/paper.pdf", "status": "verified"}
    ]},
    {"kind": "record-grill",
     "hits": [],
     "searches": [
       {"terms": ["admission gate", "ideate"], "places": [".abcd/development/intents/"], "count": 0}
     ]},
    {"kind": "adversarial-review",
     "reviewers": [
       {"label": "reviewer-a", "verdict": "survives",
        "kill_attempts": [{"attempt": "the gate adds friction to capture", "outcome": "survived"}],
        "sources": ["https://example.com/paper.pdf"]},
       {"label": "reviewer-b", "verdict": "reframed",
        "kill_attempts": [{"attempt": "no evidence the legs improve outcomes", "outcome": "partial"}],
        "sources": []}
     ],
     "ruling": {"by": "the product thinker", "verdict": "survives", "reason": "the friction is one line, and the evidence gap is the research leg's to close"}}
  ],
  "verdict": "survives",
  "rejected_alternatives": [
    {"alternative": "make ideate a blocking pre-capture gate", "why_rejected": "capture friction stays at one line"}
  ]
}`

// Check runs the decoder and the validator Record runs on a verdict payload,
// and writes nothing: the schema gate, the strict decode, every leg's checks,
// the redaction scanner the record would be written through, and the
// resolution of every record a grill hit cites in repoRoot. Source keys
// ("source:<key>") are not resolved, because the sources corpus is the
// person's and not the checkout's; Record resolves them against it.
func Check(repoRoot string, raw []byte) error {
	p, err := decodePayload(repoRoot, raw)
	if err != nil {
		return err
	}
	red, err := newRecordRedactor(repoRoot)
	if err != nil {
		return err
	}
	_, err = validate(repoRoot, "", red, p)
	return err
}
