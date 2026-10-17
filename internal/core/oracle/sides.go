package oracle

// sides.go is the vocabulary every model call's record shares (itd-166): the
// side abcd asked for and the side that answered, on whichever route the call
// took. The provider adapter's record (CallRecord), the harness hand-off's
// receipt (ReceiptRoute) and the command-line runners' route record
// (runner.RouteRecord) each carry the same token counts, the same cost and the
// same list of the fields they could not pin, so a reader compares two calls
// field by field whatever routes they took. The runner package imports this
// one; nothing here imports the runner.

import "github.com/intentdriven/abcd/internal/adapter/openaiapi"

// Field is one field of the fixed lists a call's record is held to (itd-166
// decision 10). Every route records each field or lists it in its Unpinned
// with the reason, so a field is never silently absent.
type Field string

// The asked side: what the dispatching layer requested.
const (
	AskedModel          Field = "asked.model"
	AskedProviderPin    Field = "asked.provider_pin"
	AskedTemperature    Field = "asked.temperature"
	AskedTopP           Field = "asked.top_p"
	AskedSeed           Field = "asked.seed"
	AskedEffort         Field = "asked.effort"
	AskedContext        Field = "asked.context"
	AskedPromptSHA256   Field = "asked.prompt_sha256"
	AskedHarnessVersion Field = "asked.harness_version"
	// The local fields are a local model's (decision 7): the weight file's
	// fingerprint, its quantisation and the runtime's version, which a local
	// route lists as not configured until a local route can be pinned
	// (iss-2610161734428291).
	AskedWeightsSHA256  Field = "asked.weights_sha256"
	AskedQuantisation   Field = "asked.quantisation"
	AskedRuntimeVersion Field = "asked.runtime_version"
)

// The served side: what actually answered.
const (
	ServedModel            Field = "served.model"
	ServedProvider         Field = "served.provider"
	ServedGenerationID     Field = "served.generation_id"
	ServedFinishReason     Field = "served.finish_reason"
	ServedPromptTokens     Field = "served.prompt_tokens"
	ServedCompletionTokens Field = "served.completion_tokens"
	ServedReasoningTokens  Field = "served.reasoning_tokens"
	ServedCacheReadTokens  Field = "served.cache_read_tokens"
	ServedCacheWriteTokens Field = "served.cache_write_tokens"
	ServedCost             Field = "served.cost"
)

// AskedFields and ServedFields are the fixed lists, in the order a record
// lists them. LocalFields extends the asked list on a local route.
var (
	AskedFields = []Field{AskedModel, AskedProviderPin, AskedTemperature, AskedTopP, AskedSeed,
		AskedEffort, AskedContext, AskedPromptSHA256, AskedHarnessVersion}
	LocalFields  = []Field{AskedWeightsSHA256, AskedQuantisation, AskedRuntimeVersion}
	ServedFields = []Field{ServedModel, ServedProvider, ServedGenerationID, ServedFinishReason,
		ServedPromptTokens, ServedCompletionTokens, ServedReasoningTokens,
		ServedCacheReadTokens, ServedCacheWriteTokens, ServedCost}
)

// UnpinnedReason is why a field is not pinned (decision 8).
type UnpinnedReason string

const (
	// NotReported: what the route returned does not carry the field.
	NotReported UnpinnedReason = "not_reported"
	// SelfReported: the model, or the payload it wrote, declared the value,
	// which nothing outside it confirms.
	SelfReported UnpinnedReason = "self_reported"
	// HarnessComputed: the harness computed the value (a cost from its own
	// price table), not the provider that served the call.
	HarnessComputed UnpinnedReason = "harness_computed"
	// NotConfigured: no setting or pin names the value, so the server's
	// default applied, whatever it was.
	NotConfigured UnpinnedReason = "not_configured"
	// NotVerified: abcd configured the value and the record carries it, but
	// nothing on the route shows it reached the provider: a harness was
	// handed it and its answer does not say what the provider was sent.
	NotVerified UnpinnedReason = "not_verified"
)

// Unpinned is one field a record could not pin, and why.
type Unpinned struct {
	Field  Field          `json:"field"`
	Reason UnpinnedReason `json:"reason"`
}

// Usage is the token counts a call's record carries. A count the route did
// not report is zero here and listed in the record's Unpinned.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	ReasoningTokens  int `json:"reasoning_tokens"`
	CacheReadTokens  int `json:"cache_read_tokens"`
	CacheWriteTokens int `json:"cache_write_tokens"`
}

// Cost is what a call cost, in the unit its route reports: CostUSD for the
// runners' harnesses, CostCredits for a provider's usage member. A record
// carries a *Cost, so a cost of zero is never confused with one not reported.
type Cost struct {
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

// ReadableCost reports whether a cost a route reported, or a sum of them, is
// one a record keeps: zero or more, and finite. It is the adapter's policy
// (openaiapi.ReadableCost), the one every route's cost is held to: a cost
// that fails it is not reported.
func ReadableCost(c float64) bool { return openaiapi.ReadableCost(c) }

// The units a Cost is reported in.
const (
	CostUSD     = "usd"
	CostCredits = "credits"
)

// Unpin is a record's list of the fields it could not pin, built in the
// fixed lists' order.
type Unpin []Unpinned

// Add lists f with reason r.
func (u *Unpin) Add(f Field, r UnpinnedReason) { *u = append(*u, Unpinned{Field: f, Reason: r}) }

// Unless lists f with reason r when have is false.
func (u *Unpin) Unless(have bool, f Field, r UnpinnedReason) {
	if !have {
		u.Add(f, r)
	}
}

// List is the list as a record stores it: never nil, so a record that pins
// every field says so with an empty list.
func (u Unpin) List() []Unpinned {
	if u == nil {
		return []Unpinned{}
	}
	return []Unpinned(u)
}
