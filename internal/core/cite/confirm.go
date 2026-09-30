package cite

// The manual queue's other end: a human clears a link, and the confirm verb
// records it.
//
// One receipt schema serves both of spc-17's rungs. Today the maintainer clears
// the printed checklist and names the URLs on the command line; later, a
// generated checklist page hands back a receipt FILE. Those are two producers of
// the same input, not two pathways — which is why the schema is declared here,
// once, and the CLI's positional arguments are simply assembled into it before
// they reach this function. When the page arrives it changes nothing on this
// side.
//
// The schema records THAT a human confirmed a citation and WHEN. It is
// structurally incapable of recording HOW, enforced twice like the baseline's
// own refusal: no such field is declared, and decoding runs with
// DisallowUnknownFields so a hand-added one is a refusal rather than a silently
// dropped key. A maintainer's browser, credential, or library proxy is their
// business and never enters a committed record.

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// ReceiptSchemaVersion is the only receipt schema this build understands.
const ReceiptSchemaVersion = 1

// receiptSizeLimit caps a receipt read. A receipt is a short list of addresses;
// the guarded read is what keeps a hostile file from being slurped whole.
const receiptSizeLimit = 1 << 20 // 1 MiB

// ConfirmedCitation is one cleared queue line.
type ConfirmedCitation struct {
	// URL is the cited address, exactly as the docs write it.
	URL string `json:"url"`
	// FinalURL is where the human found the document, when it differs from the
	// cited address. Empty means it did not move.
	FinalURL string `json:"final_url,omitempty"`
	// VerifiedOn is the date the human checked, YYYY-MM-DD. Empty means today.
	// It is a declared date rather than an implicit "now" because a receipt for
	// last week's work must age from last week — the clock AC 3 puts human and
	// machine verifications on is the same one.
	VerifiedOn string `json:"verified_on,omitempty"`
}

// Receipt is what a human — or the generated checklist page — hands back.
type Receipt struct {
	SchemaVersion int                 `json:"schema_version"`
	Confirmed     []ConfirmedCitation `json:"confirmed"`
}

// ConfirmRequest is the input to one confirm call.
type ConfirmRequest struct {
	RepoRoot string
	Config   lint.Config
	Receipt  Receipt
	Now      time.Time
}

// ConfirmResult reports what was recorded.
type ConfirmResult struct {
	BaselinePath string   `json:"baseline_path"`
	Recorded     []string `json:"recorded"`
}

// ConfirmError is a refused confirmation — a distinct type so a caller can tell
// a bad receipt from an I/O fault without string matching.
type ConfirmError struct{ msg string }

func (e *ConfirmError) Error() string { return e.msg }

// parseReceipt decodes a receipt file, refusing anything it does not fully
// understand.
func parseReceipt(data []byte) (Receipt, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var r Receipt
	if err := dec.Decode(&r); err != nil {
		return Receipt{}, &ConfirmError{"citation receipt is not valid: " + err.Error()}
	}
	if r.SchemaVersion != ReceiptSchemaVersion {
		return Receipt{}, &ConfirmError{
			"citation receipt schema_version is " + strconv.Itoa(r.SchemaVersion) +
				"; this build understands only " + strconv.Itoa(ReceiptSchemaVersion)}
	}
	return r, nil
}

// LoadReceipt reads and parses a receipt file.
func LoadReceipt(path string) (Receipt, error) {
	data, err := fsutil.ReadGuarded(path, receiptSizeLimit)
	if err != nil {
		return Receipt{}, err
	}
	return parseReceipt(data)
}

// Confirm records a human's confirmations in the baseline.
//
// Every line is validated against the docs' cited set BEFORE anything is
// written, and one bad line refuses the whole receipt. Confirming is the single
// place a receipt enters the record without a fetch, so it must not become a
// door through which arbitrary entries arrive — and a batch that half-applied
// would leave a maintainer unable to tell which half.
func Confirm(req ConfirmRequest) (ConfirmResult, error) {
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	today := now.Format(dateLayout)

	if len(req.Receipt.Confirmed) == 0 {
		return ConfirmResult{}, &ConfirmError{"citation receipt confirms nothing"}
	}

	policy, err := lint.CitationPolicy(req.Config, req.RepoRoot)
	if err != nil {
		return ConfirmResult{}, err
	}
	relPath, absPath := policy.BaselineRel, policy.BaselinePath

	cited, err := lint.CollectCitedURLs(req.Config, req.RepoRoot)
	if err != nil {
		return ConfirmResult{}, err
	}
	citedSet := make(map[string]bool, len(cited))
	for _, c := range cited {
		citedSet[c.URL] = true
	}

	// Validate the whole receipt first: nothing is written until every line is
	// good.
	entries := make(map[string]lint.BaselineEntry, len(req.Receipt.Confirmed))
	recorded := make([]string, 0, len(req.Receipt.Confirmed))
	for _, c := range req.Receipt.Confirmed {
		if !citedSet[c.URL] {
			return ConfirmResult{}, &ConfirmError{
				"cannot confirm " + c.URL + ": it is not cited anywhere under the configured roots. " +
					"A receipt records a citation this repo publishes, never an arbitrary address"}
		}
		on := c.VerifiedOn
		if on == "" {
			on = today
		}
		when, perr := time.Parse(dateLayout, on)
		if perr != nil {
			return ConfirmResult{}, &ConfirmError{
				"cannot confirm " + c.URL + ": verified_on " + strconv.Quote(on) + " is not a " + dateLayout + " date"}
		}
		if lint.DaysBetween(when, now) < 0 {
			return ConfirmResult{}, &ConfirmError{
				"cannot confirm " + c.URL + ": verified_on " + on + " is in the future"}
		}
		final := c.FinalURL
		if final == "" {
			final = c.URL
		}
		entries[c.URL] = lint.BaselineEntry{
			URL: c.URL, FinalURL: final, LastChecked: on,
			Outcome: lint.OutcomeAlive, Verification: lint.VerificationManual, VerifiedOn: on,
		}
		recorded = append(recorded, c.URL)
	}

	// A missing baseline is fine — a repo may clear its queue before its first
	// refresh. A malformed one is a refusal, for the same reason a refresh
	// refuses it.
	merged := map[string]lint.BaselineEntry{}
	old, err := lint.LoadBaseline(absPath)
	switch {
	case err == nil:
		for k, v := range old.Entries {
			merged[k] = v
		}
	case errors.Is(err, os.ErrNotExist):
	default:
		return ConfirmResult{}, err
	}
	for k, v := range entries {
		merged[k] = v
	}

	if err := lint.SaveBaseline(absPath, lint.Baseline{SchemaVersion: lint.BaselineSchemaVersion, Entries: merged}); err != nil {
		return ConfirmResult{}, err
	}
	sort.Strings(recorded)
	return ConfirmResult{BaselinePath: relPath, Recorded: recorded}, nil
}
