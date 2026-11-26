package refusal

// codes_decide.go — the codes of the verbs that record a judgement: decide and
// ideate (spc-2610301431108175 step 3).

const (
	// decide's own refusals.
	DecideOutsideCheckout   Code = "decide/outside-checkout"
	DecideMissingSharedTier Code = "decide/missing-shared-tier"
	DecideInvalidText       Code = "decide/invalid-text"

	// ideate's own refusals.
	IdeateOutsideCheckout   Code = "ideate/outside-checkout"
	IdeateInvalidVerdict    Code = "ideate/invalid-verdict"
	IdeateShallowCheckout   Code = "ideate/shallow-checkout"
	IdeateMissingRootCommit Code = "ideate/missing-root-commit"
)

func init() {
	register(
		Entry{
			Code:        DecideOutsideCheckout,
			Summary:     "the decision record is per-repository, and this call was made outside a checkout",
			Explanation: outsideCheckout,
			Next:        NoNext,
			RaisedBy:    []string{"abcd decide"},
		},
		Entry{
			Code:    DecideMissingSharedTier,
			Summary: "the repository has no shared working tier to keep the decision in",
			Explanation: "A decision is kept under .abcd/work/decisions/, and this repository has no\n" +
				".abcd/work/. `abcd ahoy install` lays the tiers out; nothing was recorded.",
			Next:     "abcd ahoy install",
			RaisedBy: []string{"abcd decide"},
		},
		Entry{
			Code:    DecideInvalidText,
			Summary: "the text or title has nothing to name a record by",
			Explanation: "A decision's or an ADR's file is named from its text, and the text given is\n" +
				"empty or holds no letter or digit, or the record could not be minted from it:\n" +
				"the refusal says which. Nothing was recorded.",
			Next:     `abcd decide "<text>"`,
			RaisedBy: []string{"abcd decide"},
		},
		Entry{
			Code:        IdeateOutsideCheckout,
			Summary:     "a verdict is filed per repository, and this call was made outside a checkout",
			Explanation: outsideCheckout,
			Next:        NoNext,
			RaisedBy:    []string{"abcd ideate record"},
		},
		Entry{
			Code:    IdeateInvalidVerdict,
			Summary: "the verdict given cannot be read or recorded as it stands",
			Explanation: "The verdict file is unreadable, is not the verdict JSON the admission gauntlet\n" +
				"writes, or does not fit the idea named: the refusal names the first part that\n" +
				"is wrong. Nothing was written; --dry-run checks a verdict without writing.",
			Next:     "abcd explain ideate record",
			RaisedBy: []string{"abcd ideate record"},
		},
		Entry{
			Code:    IdeateShallowCheckout,
			Summary: "the checkout is shallow, so the root commit a verdict is filed under is missing",
			Explanation: "A verdict is filed under the repository's root commit, which a shallow clone\n" +
				"does not hold, or whether the clone is shallow could not be read. Fetch the full\n" +
				"history (git fetch --unshallow), then record. Nothing was written.",
			Next:     NoNext,
			RaisedBy: []string{"abcd ideate record"},
		},
		Entry{
			Code:    IdeateMissingRootCommit,
			Summary: "the repository has no root commit yet to file a verdict under",
			Explanation: "A verdict is filed under the repository's root commit, and the repository has\n" +
				"no commit yet, or its root commit could not be read. Commit first, then record.\n" +
				"Nothing was written.",
			Next:     NoNext,
			RaisedBy: []string{"abcd ideate record"},
		},
	)
}
