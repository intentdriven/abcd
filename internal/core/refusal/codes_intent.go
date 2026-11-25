package refusal

// codes_intent.go — the codes of the intent store's verbs, intent and spec
// (spc-2610301431108175 step 3). The intent core names no sentinel for most of
// its refusals, so the front door reads the few it can tell apart (an id no
// bucket holds, an impact a close needs) and gives every other refusal of the
// store its verb's invalid-request code; the message names the reason.

const (
	// intent's own refusals.
	IntentOutsideCheckout     Code = "intent/outside-checkout"
	IntentUnknownIntent       Code = "intent/unknown-intent"
	IntentHeldByPeer          Code = "intent/held-by-peer"
	IntentInsufficientGrounds Code = "intent/insufficient-grounds"
	IntentInvalidVerdict      Code = "intent/invalid-verdict"
	IntentInvalidFindings     Code = "intent/invalid-findings"
	IntentInvalidRequest      Code = "intent/invalid-request"

	// spec's own refusals.
	SpecOutsideCheckout Code = "spec/outside-checkout"
	SpecUnknownSpec     Code = "spec/unknown-spec"
	SpecMissingImpact   Code = "spec/missing-impact"
	SpecInvalidRequest  Code = "spec/invalid-request"
)

func init() {
	register(
		Entry{
			Code:        IntentOutsideCheckout,
			Summary:     "the intent store is per-repository, and this call was made outside a checkout",
			Explanation: outsideCheckout,
			Next:        NoNext,
			RaisedBy:    []string{"abcd intent", "abcd intent plan"},
		},
		Entry{
			Code:    IntentUnknownIntent,
			Summary: "no intent in this checkout's store has that id",
			Explanation: "The id (itd-N) is in none of the intent store's folders here, or is not an\n" +
				"intent id at all. Check the id against the store's listing; when the record is\n" +
				"on another branch, `abcd peers` shows where. Nothing was written.",
			Next: "abcd intent",
			RaisedBy: []string{"abcd intent condition", "abcd intent edge", "abcd intent hold", "abcd intent interview",
				"abcd intent plan", "abcd intent prepass", "abcd intent ready", "abcd intent target", "abcd intent unhold"},
		},
		Entry{
			Code:    IntentHeldByPeer,
			Summary: "the intent is not in this checkout, and a peer's checkout holds it",
			Explanation: "The record is on another branch or worktree, not here. The refusal names each\n" +
				"holder's branch, path and folder. Work on it where it is, or bring that branch's\n" +
				"record here; nothing was written.",
			Next:     "abcd peers",
			RaisedBy: []string{"abcd intent audit"},
		},
		Entry{
			Code:    IntentInsufficientGrounds,
			Summary: "the grounds given are outside their vocabulary or too thin",
			Explanation: "--grounds records the conjecture an act rests on, as \"<kind>: <what is\n" +
				"expected, and what would show it wrong>\", and the value given is not one of the\n" +
				"accepted kinds or is below the substance floor. Nothing was recorded.",
			Next:     "abcd explain intent ready",
			RaisedBy: []string{"abcd intent ready"},
		},
		Entry{
			Code:    IntentInvalidVerdict,
			Summary: "the audit verdict given cannot be read or recorded as it stands",
			Explanation: "The verdict file is unreadable, is not the verdict JSON the intent auditor\n" +
				"writes, or names an intent or criteria the store does not hold: the refusal\n" +
				"names which. Nothing was recorded; fix the verdict and ingest it again.",
			Next:     "abcd explain intent audit ingest",
			RaisedBy: []string{"abcd intent audit ingest"},
		},
		Entry{
			Code:    IntentInvalidFindings,
			Summary: "the findings given cannot be read or recorded as they stand",
			Explanation: "The findings file is unreadable or is not the findings JSON the command's\n" +
				"contract names: the refusal names the first part that is wrong. Nothing was\n" +
				"written; `abcd explain` on the command prints the contract.",
			Next:     explainCommand,
			RaisedBy: []string{"abcd intent consistency ingest", "abcd intent prepass"},
		},
		Entry{
			Code:    IntentInvalidRequest,
			Summary: "the intent store refused the act on its meaning; the refusal names why",
			Explanation: "The record is in a state the act does not apply to (a hold on a held intent, a\n" +
				"plan of one already planned), a value is outside its shape (a version, an\n" +
				"impact, a kind, a link), or the store could not be read: the refusal names which.\n" +
				"Nothing was written.",
			Next:     explainCommand,
			RaisedBy: []string{"abcd intent interview", "abcd intent link", "abcd intent target", "abcd intent unhold"},
		},
		Entry{
			Code:        SpecOutsideCheckout,
			Summary:     "the spec store is per-repository, and this call was made outside a checkout",
			Explanation: outsideCheckout,
			Next:        NoNext,
			RaisedBy:    []string{"abcd spec close"},
		},
		Entry{
			Code:    SpecUnknownSpec,
			Summary: "no spec in this checkout's store has that id",
			Explanation: "The spec id (spc-N) is in neither specs/open/ nor specs/closed/ here. Check the\n" +
				"id; when the spec is on another branch, `abcd peers` shows where. Nothing was\n" +
				"written.",
			Next:     "abcd spec",
			RaisedBy: []string{"abcd spec close"},
		},
		Entry{
			Code:    SpecMissingImpact,
			Summary: "the close ships an intent that records no impact, and none was given",
			Explanation: "A close that ships its intent moves it to shipped/, which requires an impact:\n" +
				"it decides the derived version, and there is no default. Re-run the close with\n" +
				"--impact additive, breaking or fix, or record the judgement in the intent first.\n" +
				"Nothing moved.",
			Next:     "abcd spec close <spc-n> --impact <impact>",
			RaisedBy: []string{"abcd spec close"},
		},
		Entry{
			Code:    SpecInvalidRequest,
			Summary: "the spec store refused the close on its meaning; the refusal names why",
			Explanation: "The spec is already closed, an --impact or a remainder does not fit what the\n" +
				"close does, or the intent it delivers is in a state the close does not apply\n" +
				"to: the refusal names which. Nothing moved.",
			Next:     "abcd explain spec close",
			RaisedBy: []string{"abcd spec close"},
		},
	)
}
