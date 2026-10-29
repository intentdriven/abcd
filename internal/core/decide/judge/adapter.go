package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Caller is one call through the OpenAI-compatible API adapter;
// *oracle.APIConfig implements it. Its Call admits the target against the
// allowlist and oracle.denylist again and resolves the key by name, so a
// target built anywhere is held to adr-2609221009491186 before any call.
type Caller interface {
	Call(ctx context.Context, creds credential.Source, req oracle.CallRequest, opts ...openaiapi.Option) ([]byte, oracle.CallRecord, error)
}

// Adapter is the decision model through the one interface: the question is
// rendered as the host's is, sent to Target through Calls, and the answer is
// read as a value and a probability. The model cannot abstain, so an answer
// outside the question's choices, or a probability outside 0 to 1, is no
// answer at all.
type Adapter struct {
	Calls   Caller
	Creds   credential.Source
	Target  oracle.Target
	Options []openaiapi.Option
}

// adapterInstructions frame every typed question the adapter is sent.
const adapterInstructions = "You answer one typed judgement. Read the question and the state, then reply with " +
	"a single JSON object and nothing else: {\"value\": \"<one of the allowed answers, exactly as written>\", " +
	"\"probability\": <your calibrated probability that the value is right, from 0 to 1>}."

// maxAnswerBytes bounds the answer the adapter parses: a value and a number.
const maxAnswerBytes = 4 << 10

// typed is the answer's shape.
type typed struct {
	Value       *string  `json:"value"`
	Probability *float64 `json:"probability"`
}

// Judge sends the question through the API adapter and parses the answer.
func (a Adapter) Judge(ctx context.Context, s State, q Question) (Answer, error) {
	if err := q.Check(); err != nil {
		return Answer{}, err
	}
	if a.Calls == nil {
		return Answer{}, fmt.Errorf("judge: no adapter configured for %s", q.Type)
	}
	var got Answer
	contract := func(content []byte) error {
		v, err := parseTyped(q, content)
		got = v
		return err
	}
	brief := openaiapi.Brief{Instructions: adapterInstructions, Input: render(s, q)}
	_, rec, err := a.Calls.Call(ctx, a.Creds, oracle.CallRequest{Target: a.Target, Brief: brief, Contract: contract}, a.Options...)
	if err != nil {
		return Answer{}, err
	}
	got.Model = rec.ModelReported
	return got, nil
}

// parseTyped reads the adapter's answer: one JSON object, perhaps inside a
// code fence, whose value is one of the question's choices and whose
// probability is a number from 0 to 1.
func parseTyped(q Question, content []byte) (Answer, error) {
	if len(content) > maxAnswerBytes {
		return Answer{}, fmt.Errorf("judge: the adapter's answer is %d bytes, more than the %d a typed answer takes", len(content), maxAnswerBytes)
	}
	body := bytes.TrimSpace(content)
	if i, j := bytes.IndexByte(body, '{'), bytes.LastIndexByte(body, '}'); i >= 0 && j > i {
		body = body[i : j+1]
	}
	var t typed
	if err := json.Unmarshal(body, &t); err != nil {
		return Answer{}, fmt.Errorf("judge: the adapter's answer is not a typed answer: %s", termsafe.Sanitize(err.Error()))
	}
	if t.Value == nil || t.Probability == nil {
		return Answer{}, fmt.Errorf("judge: the adapter's answer lacks its value or its probability")
	}
	p := *t.Probability
	if math.IsNaN(p) || p < 0 || p > 1 {
		return Answer{}, fmt.Errorf("judge: the adapter's probability %v is not from 0 to 1", p)
	}
	v, ok := q.match(*t.Value)
	if !ok {
		return Answer{}, fmt.Errorf("judge: the adapter's answer %q is not one of %s", termsafe.Sanitize(clip(*t.Value)), strings.Join(q.Choices(), ", "))
	}
	return Answer{Value: v, Probability: p}, nil
}
