package runner

// sides.go is what a runner records of both sides of a run (itd-166): the
// model it passed the harness and the prompt it handed it, beside what the
// harness's events say answered, with the token counts and the cost they
// carry. The vocabulary (the fixed field lists, the four reasons, the usage
// and the cost) is internal/core/oracle's, shared with the provider adapter's
// record, so two runs compare field by field whatever routes they took.
//
// A harness computes the cost it reports from its own price table, so a
// runner's cost is listed as computed by the harness, never as the
// provider's. Nothing here asks the model about itself: every value is the
// runner's own or read from the harness's event stream.

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// maxHarnessText bounds one string a harness supplied to a record: a served
// model, provider, finish reason or version.
const maxHarnessText = 128

// maxServedModels bounds the served models one run records.
const maxServedModels = 16

// Sides is both sides of a run as a runner records them. Every string the
// harness supplied is cleaned and bounded (cleanHarness) before it is set.
type Sides struct {
	// ModelAsked is the model the runner passed the harness, "" when it
	// passed none and the harness chose.
	ModelAsked string `json:"model_asked,omitempty"`
	// ServedModels is every model the harness says served the run (claude's
	// per-model usage), sorted; empty when it names none.
	ServedModels []string `json:"served_models,omitempty"`
	// ServedProvider is the provider the harness names as serving the
	// canonical model, "" when it names none.
	ServedProvider string `json:"served_provider,omitempty"`
	// FinishReason is why the run's last turn stopped, as the harness says.
	FinishReason string `json:"finish_reason,omitempty"`
	// HarnessVersion is the harness's own version, as it reports it.
	HarnessVersion string `json:"harness_version,omitempty"`
	// PromptSHA256 is the SHA-256 of the prompt the runner handed the harness
	// (prompt), never its text; the brief it names is the run's own file.
	PromptSHA256 string `json:"prompt_sha256,omitempty"`
	// Usage and Cost are the run's token counts and cost, nil when the
	// harness reported none. Each count is the harness's own: claude's
	// input_tokens and opencode's input exclude the cached prompt tokens,
	// which are counted apart.
	Usage *oracle.Usage `json:"usage,omitempty"`
	Cost  *oracle.Cost  `json:"cost,omitempty"`
	// Unpinned lists every field of the fixed lists the run could not pin.
	Unpinned []oracle.Unpinned `json:"unpinned,omitempty"`
}

// RecordOf is the route record of a role ran ran, routed to asked, whose
// harness answered ans: the served model as the canonical one, and both sides
// beside it.
func RecordOf(asked, ran string, ans Answer) RouteRecord {
	return RouteRecord{Asked: asked, Ran: ran, Model: ans.Model, Sides: ans.Sides}
}

// promptSHA256 is the hash of the prompt a route hands req's role.
func promptSHA256(req Request) string {
	sum := sha256.Sum256([]byte(prompt(req)))
	return hex.EncodeToString(sum[:])
}

// cleanHarness bounds a string a harness supplied and percent-encodes any
// hidden or control rune, so it is recorded but cannot reorder or escape the
// record it lands in.
func cleanHarness(s string) string {
	s = termsafe.EncodeHiddenRunes(s)
	if len(s) > maxHarnessText {
		cut := maxHarnessText
		for cut > 0 && s[cut]&0xC0 == 0x80 {
			cut--
		}
		s = s[:cut] + "..."
	}
	return s
}

// askedUnpinned lists the asked side's fields neither command-line runner
// can pin: no provider pin, sampling setting, seed, effort or context variant
// is passed to a harness (each is the harness's or the owner's configuration),
// nor, when the runner passed none, a model.
func askedUnpinned(un *oracle.Unpin, s Sides) {
	un.Unless(s.ModelAsked != "", oracle.AskedModel, oracle.NotConfigured)
	for _, f := range []oracle.Field{oracle.AskedProviderPin, oracle.AskedTemperature, oracle.AskedTopP,
		oracle.AskedSeed, oracle.AskedEffort, oracle.AskedContext} {
		un.Add(f, oracle.NotConfigured)
	}
	un.Unless(s.PromptSHA256 != "", oracle.AskedPromptSHA256, oracle.NotReported)
	un.Unless(s.HarnessVersion != "", oracle.AskedHarnessVersion, oracle.NotReported)
}

// counted is which of a usage's counts the harness reported.
type counted struct {
	prompt, completion, reasoning, cacheRead, cacheWrite bool
}

// servedUnpinned lists the served side's fields the run could not pin, for a
// harness that reports the counts have names and computes the cost itself.
func servedUnpinned(un *oracle.Unpin, model string, s Sides, c counted) {
	un.Unless(model != "", oracle.ServedModel, oracle.NotReported)
	un.Unless(s.ServedProvider != "", oracle.ServedProvider, oracle.NotReported)
	// Neither harness names a generation id for the run: a run is many
	// generations, and their ids are not in the events read.
	un.Add(oracle.ServedGenerationID, oracle.NotReported)
	un.Unless(s.FinishReason != "", oracle.ServedFinishReason, oracle.NotReported)
	un.Unless(c.prompt, oracle.ServedPromptTokens, oracle.NotReported)
	un.Unless(c.completion, oracle.ServedCompletionTokens, oracle.NotReported)
	un.Unless(c.reasoning, oracle.ServedReasoningTokens, oracle.NotReported)
	un.Unless(c.cacheRead, oracle.ServedCacheReadTokens, oracle.NotReported)
	un.Unless(c.cacheWrite, oracle.ServedCacheWriteTokens, oracle.NotReported)
	if s.Cost != nil {
		un.Add(oracle.ServedCost, oracle.HarnessComputed)
	} else {
		un.Add(oracle.ServedCost, oracle.NotReported)
	}
}
