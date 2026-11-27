package refusal

// codes_capture.go — capture's codes beyond the table's: the refusals every
// capture verb shares with the other record verbs (spc-2610301431108175
// step 3).

const (
	CaptureOutsideCheckout Code = "capture/outside-checkout"
	CaptureHeldByPeer      Code = "capture/held-by-peer"
)

func init() {
	register(
		Entry{
			Code:        CaptureOutsideCheckout,
			Summary:     "the issue ledger is per-repository, and this call was made outside a checkout",
			Explanation: outsideCheckout,
			Next:        NoNext,
			RaisedBy: []string{"abcd capture", "abcd capture admit", "abcd capture defer", "abcd capture disposition",
				"abcd capture link", "abcd capture list", "abcd capture mentions", "abcd capture migrate", "abcd capture owed",
				"abcd capture promote", "abcd capture reframe", "abcd capture remedy", "abcd capture resolve",
				"abcd capture surprise", "abcd capture wontfix"},
		},
		Entry{
			Code:    CaptureHeldByPeer,
			Summary: "the issue is not in this checkout, and a peer's checkout holds it",
			Explanation: "The record is on another branch or worktree, not here. The refusal names each\n" +
				"holder's branch, path and folder. Work on it where it is, or bring that branch's\n" +
				"record here; nothing was written.",
			Next:     "abcd peers",
			RaisedBy: []string{"abcd capture resolve"},
		},
	)
}
