package oracle

// connect_guide.go is the guided path of `abcd ahoy connect`
// (spc-2610031241482088, "The guided path in the session";
// itd-2610030821294016): inside a session, abcd works the setup's values out
// with the person one question at a time and ends by printing the one command
// that sets the connection up, with every path that command writes, for the
// person to paste into a terminal. Nothing is set up from inside the session
// (decision 1): the guide never calls Connect, never writes a file, never
// reads a key home and never asks for the key.
//
// Each call is one turn. The guide is a function of what it was given (the
// provider's name and the address, when given), the answers so far, and the
// models a look-up listed: every turn replays the answers from the first
// question, re-deriving each question and checking each answer against it, so
// an edited resume object cannot skip a question. The listed models are the
// one thing carried rather than re-derived (open question 2, decided (a)):
// the look-up runs once, keyless, at most MaxCarriedBytes of its ids are
// carried, and each carried id is checked again on the way back in.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// GuideSchemaVersion is the resume object's shape.
const GuideSchemaVersion = 1

// The guide's bounds on what a resume object may carry back.
const (
	// maxGuideAnswers bounds the answers one guided setup replays: six
	// questions and the narrowing rounds of the model question.
	maxGuideAnswers = 64
	// maxGuideAnswerBytes bounds one answer as the person typed it.
	maxGuideAnswerBytes = 2048
	// maxSuggestions is how many models the person already uses the model
	// question offers (G2: at most three).
	maxSuggestions = 3
	// maxNarrowShown is how many matches a narrowing question offers.
	maxNarrowShown = 3
	// maxEnvNames is how many environment variables' names the external
	// home's question offers (open question 4, decided (a)).
	maxEnvNames = 3
	// guideEcho bounds a typed answer quoted back in a question.
	guideEcho = 64
)

// MaxCarriedBytes bounds the listed ids a resume object carries, as JSON:
// the first ids in the service's order that fit, and the count of the rest.
// A look-up keeps up to 5,000 ids, about 270 KB of JSON a turn: more than one
// argument may hold on linux (MAX_ARG_STRLEN, 128 KiB) when the host runs the
// page's command as sh -c, and more than a session should re-emit every turn.
// Real services list tens to a few hundred ids, which fit whole.
const MaxCarriedBytes = 32 << 10

// The ids of the guide's questions, as the resume object's answers name them.
const (
	GuideQAddress = "address"
	GuideQLookup  = "lookup"
	GuideQModel   = "model"
	GuideQNarrow  = "model-narrow"
	GuideQTyped   = "model-name"
	GuideQTakes   = "takes-key"
	GuideQHome    = "home"
	GuideQEnv     = "env"
)

// guideLater is every guided question's decide-later value: no model id,
// address or variable name can take it, since none admits a '#'.
const guideLater = "#later"

// The look-up's outcomes, as the resume object carries them.
const (
	ListedListed   = "listed"    // the service listed its models, keyless
	ListedNeedsKey = "needs_key" // it lists them only for a key
	ListedNone     = "none"      // it gave no usable list; Reason says why
)

// GuideAnswer is one answer the person gave: the question's id and the value
// recorded for it.
type GuideAnswer struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// GuideListed is what the one look-up found.
type GuideListed struct {
	Status string   `json:"status"`
	Host   string   `json:"host"`
	Models []string `json:"models,omitempty"`
	// More is how many of the ids listed are not carried (MaxCarriedBytes).
	More   int    `json:"more,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// GuideState is the resume object: everything the next turn needs, written
// out on stdout and passed back on stdin. It never carries a key.
type GuideState struct {
	SchemaVersion int           `json:"schema_version"`
	Provider      string        `json:"provider,omitempty"`
	BaseURL       string        `json:"base_url,omitempty"`
	Answers       []GuideAnswer `json:"answers"`
	Listed        *GuideListed  `json:"listed,omitempty"`
	// Open is the question the next answer is for.
	Open string `json:"open,omitempty"`
}

// GuideDone is the guide's end: the one command and every path it writes,
// in the tilde form and in the order a run reports them (G4).
type GuideDone struct {
	Command  string   `json:"command"`
	Writes   []string `json:"writes"`
	Provider string   `json:"provider"`
	Home     string   `json:"home"`
	// PicksInTerminal is a command with no --model: the service lists its
	// models only for a key, so the command lists them in the terminal once it
	// holds the key, and the person picks there (decision 2).
	PicksInTerminal bool `json:"picks_in_terminal"`
}

// GuideTurn is one turn: a question to ask, the end, or a stop (decide
// later), and the resume object for the next turn.
type GuideTurn struct {
	Ask     *question.Ask `json:"ask,omitempty"`
	Done    *GuideDone    `json:"done,omitempty"`
	Stopped string        `json:"stopped,omitempty"`
	Resume  GuideState    `json:"resume"`
}

// GuideStopped is what a decide-later answer ends the guide with.
const GuideStopped = "Nothing was set up; run the guide again to pick up where this left off."

// GuideRequest is one turn's input.
type GuideRequest struct {
	// Roots are where the configuration in force is read: the provider names
	// already taken, the denylist, and the models the person already uses.
	Roots layered.Roots
	// Provider and BaseURL are what the first turn was given; a resumed turn
	// takes them from Resume.
	Provider string
	BaseURL  string
	// Resume is the previous turn's resume object; nil starts the guide.
	Resume *GuideState
	// Answer is the answer to Resume.Open; nil asks the open question again.
	Answer *string
	// EnvNames are the names of the variables in the guide's environment,
	// never their values: the external home's question offers the ones that
	// end in _API_KEY.
	EnvNames []string
}

// GuideRefusal is a turn the guide refuses: a resume object or an answer it
// cannot admit. Nothing was asked of the service and nothing written.
type GuideRefusal struct{ msg string }

func (e *GuideRefusal) Error() string { return e.msg }

func refuse(format string, args ...any) error {
	return &GuideRefusal{msg: "oracle adapter: guided connect: " + fmt.Sprintf(format, args...)}
}

// Guide returns the next turn of the guided setup. It writes nothing and
// reads no key home; its one request is the keyless look-up of the service's
// models, made only on the person's yes and only once.
func Guide(ctx context.Context, req GuideRequest) (GuideTurn, error) {
	st := GuideState{SchemaVersion: GuideSchemaVersion, Provider: req.Provider, BaseURL: req.BaseURL}
	if req.Resume != nil {
		if req.Provider != "" || req.BaseURL != "" {
			return GuideTurn{}, refuse("a resumed guide takes its provider and address from the resume object, not from the command line")
		}
		st = *req.Resume
		if st.SchemaVersion != GuideSchemaVersion {
			return GuideTurn{}, refuse("the resume object's schema_version is %d; this abcd reads %d", st.SchemaVersion, GuideSchemaVersion)
		}
		if len(st.Answers) > maxGuideAnswers {
			return GuideTurn{}, refuse("the resume object carries %d answers; a guided setup takes at most %d", len(st.Answers), maxGuideAnswers)
		}
	} else if req.Answer != nil {
		return GuideTurn{}, refuse("an answer needs the resume object of the turn it answers")
	}
	if req.Answer != nil && st.Open == "" {
		return GuideTurn{}, refuse("the resume object names no open question for the answer")
	}
	cfg, err := LoadAPI(req.Roots)
	if err != nil {
		return GuideTurn{}, fmt.Errorf("%w; fix the configuration before adding a provider to it", err)
	}
	if st.Provider != "" {
		probe := ConnectRequest{Provider: st.Provider, BaseURL: "https://example.com", Home: KeyHomeNone, Models: []string{"m"}}
		if err := checkConnect(&probe, false); err != nil {
			return GuideTurn{}, err
		}
		if _, exists := cfg.providers[st.Provider]; exists {
			return GuideTurn{}, fmt.Errorf("oracle adapter: provider %s is already configured in %s; "+
				"abcd never replaces a block unasked, so edit or remove it there to change it", st.Provider, layered.Config.MachineOrigin())
		}
	}
	if st.BaseURL != "" {
		if err := openaiapi.ValidateBaseURL(st.BaseURL); err != nil {
			return GuideTurn{}, fmt.Errorf("oracle adapter: %w", err)
		}
	}
	g := &guide{ctx: ctx, cfg: cfg, st: st, hist: st.Answers, fresh: req.Answer, env: req.EnvNames}
	g.st.Answers = []GuideAnswer{}
	return g.run()
}

// guide is one replay.
type guide struct {
	ctx   context.Context
	cfg   *APIConfig
	st    GuideState    // the resume object being rebuilt
	hist  []GuideAnswer // the answers given before this turn
	pos   int           // the next of hist to replay
	fresh *string       // this turn's answer, for the question hist ends at
	env   []string
	n     int // questions put so far, for the chip
}

// admitFunc judges an answer to one question: the value recorded, or a
// reason to ask the question again (a typed answer the question cannot take),
// or an error refusing it (an answer the question does not admit at all).
type admitFunc func(string) (value, retry string, err error)

// stop is a turn ending the replay: a question to ask, a stop, or a refusal.
type stop struct {
	turn GuideTurn
	err  error
}

// put replays or takes the answer to q. It returns the value recorded, or a
// stop: the question asked (no answer yet, or a typed answer to ask again),
// the guide stopped on decide later, or a refusal.
func (g *guide) put(q question.Question, admit admitFunc) (string, *stop) {
	g.n++
	q.Chip = fmt.Sprintf("Setup Q%d", g.n)
	q.Later = question.Option{Value: guideLater, Label: "Decide later",
		Meaning: "Nothing is set up or written. Run the guide again to pick up here."}
	q.Now = "nothing is set up for this service"
	q.ChangeLater = "run the guide again"
	isLater := func(v string) bool {
		v = strings.TrimSpace(v)
		return v == guideLater || strings.EqualFold(v, q.Later.Label)
	}
	if g.pos < len(g.hist) {
		a := g.hist[g.pos]
		g.pos++
		if a.ID != q.ID {
			return "", &stop{err: refuse("the resume object answers %q where the guide asks %q", layered.BoundKey(a.ID), q.ID)}
		}
		if isLater(a.Value) {
			return "", &stop{err: refuse("the resume object records decide later for %q; run the guide again from that turn", q.ID)}
		}
		if keyShaped(a.Value) {
			return "", &stop{err: refuse("the answer recorded for %q: %s", q.ID, lookedLikeKey)}
		}
		v, retry, err := admit(a.Value)
		if err == nil && retry != "" {
			err = refuse("the answer recorded for %q is one it does not take: %s", q.ID, retry)
		}
		if err != nil {
			return "", &stop{err: err}
		}
		g.st.Answers = append(g.st.Answers, GuideAnswer{ID: q.ID, Value: v})
		return v, nil
	}
	if g.fresh != nil {
		ans := *g.fresh
		g.fresh = nil
		if g.st.Open != "" && g.st.Open != q.ID {
			return "", &stop{err: refuse("the answer is for %q, and the guide asks %q", layered.BoundKey(g.st.Open), q.ID)}
		}
		if isLater(ans) {
			g.st.Open = q.ID
			return "", &stop{turn: GuideTurn{Stopped: GuideStopped, Resume: g.st}}
		}
		var v, retry string
		var err error
		switch {
		case len(ans) > maxGuideAnswerBytes:
			retry = fmt.Sprintf("The answer is %d bytes; an answer here is at most %d.", len(ans), maxGuideAnswerBytes)
		case keyShaped(ans):
			retry = "That " + lookedLikeKey + "."
		case strings.EqualFold(strings.TrimSpace(ans), question.TypedRowLabel) && q.Typed != "":
			retry = "That choice points at the row for typing: type the answer itself there."
		default:
			v, retry, err = admit(ans)
		}
		if err != nil {
			return "", &stop{err: err}
		}
		if retry != "" {
			q.Material = append([]question.Block{{Kind: question.KindParagraph, Text: retry}}, q.Material...)
			return "", g.ask(q)
		}
		g.st.Answers = append(g.st.Answers, GuideAnswer{ID: q.ID, Value: v})
		return v, nil
	}
	return "", g.ask(q)
}

// ask is the stop that asks q.
func (g *guide) ask(q question.Question) *stop {
	g.st.Open = q.ID
	return &stop{turn: GuideTurn{Ask: &question.Ask{Questions: []question.Question{q}}, Resume: g.st}}
}

// options admits an answer that is one of q's options, by its value or,
// as the host's question tool returns it, its label.
func options(q question.Question, typed func(string) (string, string, error)) admitFunc {
	return func(ans string) (string, string, error) {
		t := strings.TrimSpace(ans)
		for _, o := range q.Options {
			if t == o.Value || strings.EqualFold(t, o.Label) {
				return o.Value, "", nil
			}
		}
		if typed != nil && q.Typed != "" {
			return typed(t)
		}
		return "", "", refuse("%q is not an answer the question %q takes; its answers are %s", layered.BoundKey(ans), q.ID, optionList(q))
	}
}

func optionList(q question.Question) string {
	vals := make([]string, 0, len(q.Options)+1)
	for _, o := range q.Options {
		vals = append(vals, fmt.Sprintf("%q", o.Value))
	}
	return strings.Join(append(vals, fmt.Sprintf("%q", guideLater)), ", ")
}

func para(s string) question.Block { return question.Block{Kind: question.KindParagraph, Text: s} }

// echoed is a typed answer as a question quotes it back: one sanitised line,
// bounded. Only the narrowing question quotes one, the part of a name it
// narrows by; a retry never does, so a value pasted in the wrong place is not
// carried into the next turn.
func echoed(s string) string { return termsafe.CleanProseLine(s, guideEcho) }

// lookedLikeKey is why the guide refuses a typed value the secret scanner
// knows as a credential.
const lookedLikeKey = "looks like a key; the guide takes a name, never a key"

// secretPatterns is the secret scanner's pattern set, built once.
var secretPatterns = sync.OnceValue(scanner.DefaultPatterns)

// keyLines are the indexes of the lines among lines that hold a value the
// secret scanner knows as a credential (a token: kind; its network and
// identity kinds are no key), judged in one scan. The guide asks for names and
// never for a key, so such a value is the key pasted in the wrong place, or an
// id that carries one.
func keyLines(lines []string) map[int]bool {
	out := map[int]bool{}
	if len(lines) == 0 {
		return out
	}
	for _, f := range scanner.ScanText(strings.Join(lines, "\n"), scanner.Identity{}, secretPatterns(), nil, "") {
		if scanner.IsTokenKind(f.Kind) {
			out[f.Line-1] = true
		}
	}
	return out
}

// keyShaped reports whether a typed value holds a key (keyLines).
func keyShaped(s string) bool { return len(keyLines([]string{s})) != 0 }

// dropKeyShaped is ids without those that carry a key (keyLines). Each id is
// one line: none holds a line break.
func dropKeyShaped(ids []string) []string {
	bad := keyLines(ids)
	if len(bad) == 0 {
		return ids
	}
	out := make([]string, 0, len(ids)-len(bad))
	for i, id := range ids {
		if !bad[i] {
			out = append(out, id)
		}
	}
	return out
}

// run replays the guide from its first question.
func (g *guide) run() (GuideTurn, error) {
	turn, s := g.walk()
	if s != nil {
		return s.turn, s.err
	}
	if g.pos < len(g.hist) {
		return GuideTurn{}, refuse("the resume object carries answers past the guide's end")
	}
	if g.fresh != nil {
		return GuideTurn{}, refuse("the guide has ended, so there is no question for the answer")
	}
	return turn, nil
}

// walk is the questions in order; it returns the done turn, or the stop that
// ended the replay first.
func (g *guide) walk() (GuideTurn, *stop) {
	// 1. The address, when none was given.
	base := g.st.BaseURL
	if base == "" {
		q := question.Question{
			ID:       GuideQAddress,
			Material: []question.Block{para("abcd reaches a model service at the OpenAI-compatible address it publishes.")},
			Ask:      "What is the service's address?",
			Typed:    "the address, https://, or http:// on this machine",
		}
		v, s := g.put(q, func(ans string) (string, string, error) {
			t := strings.TrimSpace(ans)
			if err := openaiapi.ValidateBaseURL(t); err != nil {
				return "", fmt.Sprintf("That address is refused: %s.", err), nil
			}
			return t, "", nil
		})
		if s != nil {
			return GuideTurn{}, s
		}
		base = v
	}
	u, _ := url.Parse(base)
	host := u.Host
	provider := g.st.Provider
	if provider == "" {
		provider = g.deriveProvider(u.Hostname())
	}

	// 2. The look-up (G1): the scheme and host first, a request only on yes.
	lookup := question.Question{
		ID: GuideQLookup,
		Material: []question.Block{para(fmt.Sprintf("abcd can ask %s://%s for the list of models it offers. "+
			"The request carries no key, follows no redirect, and gives up after %d seconds.",
			u.Scheme, host, int(openaiapi.ListTimeout.Seconds())))},
		Ask: "Should abcd look up the service's models?",
		Options: []question.Option{
			{Value: "lookup", Label: "Look up the list", Meaning: "One request to the service, with no key. You then pick from its models."},
			{Value: "type", Label: "Type the name myself", Meaning: "No request is sent. You type the model's exact name."},
		},
	}
	how, s := g.put(lookup, options(lookup, nil))
	if s != nil {
		return GuideTurn{}, s
	}
	var listed *GuideListed
	if how == "lookup" {
		var err error
		if listed, err = g.listed(base, host); err != nil {
			return GuideTurn{}, &stop{err: err}
		}
		g.st.Listed = listed
	} else {
		g.st.Listed = nil
	}

	// 3. The model.
	model, picks := "", false
	switch {
	case listed != nil && listed.Status == ListedNeedsKey:
		picks = true
	case listed != nil && listed.Status == ListedListed:
		if model, s = g.pickListed(host, listed.Models, listed.More); s != nil {
			return GuideTurn{}, s
		}
	default:
		why := "You chose to type the name rather than look it up."
		if listed != nil {
			why = fmt.Sprintf("%s publishes no model list: it %s.", host, listed.Reason)
		}
		q := question.Question{
			ID:       GuideQTyped,
			Material: []question.Block{para(why)},
			Ask:      "Which model should abcd set up?",
			Typed:    "the model's name, exactly as the service spells it",
		}
		if model, s = g.put(q, g.typedModel); s != nil {
			return GuideTurn{}, s
		}
	}

	// 4. Whether the service takes a key, unless the look-up said it does.
	home := ""
	needsKeyNote := ""
	if picks {
		needsKeyNote = fmt.Sprintf("%s lists its models only for a key, so you pick one in your terminal "+
			"once the command has it.", host)
	} else {
		q := question.Question{
			ID:       GuideQTakes,
			Material: []question.Block{para("A hosted service takes a key with every request. A server on this machine often takes none.")},
			Ask:      "Does the service take a key?",
			Options: []question.Option{
				{Value: "key", Label: "It needs a key", Meaning: "The next question asks where the key lives. abcd never asks for the key itself here."},
				{Value: KeyHomeNone, Label: "No key needed", Meaning: "A server that takes no key, such as one on this machine. Nothing is stored for it."},
			},
		}
		v, s := g.put(q, options(q, nil))
		if s != nil {
			return GuideTurn{}, s
		}
		if v == KeyHomeNone {
			home = KeyHomeNone
		}
	}

	// 5. Where the key lives (G8): the store's three homes, never an item.
	env := ""
	if home == "" {
		// The store's prose (KeyHomesProse) recommends the keychain in prose,
		// never as a marked option; it is too long for the 24 rows a question
		// is held to, so its gist is said here and each home's meaning is on
		// its option.
		text := "Where the key lives is your choice; abcd recommends the keychain, the system's own store."
		if needsKeyNote != "" {
			text = needsKeyNote + " " + text
		}
		material := []question.Block{para(text)}
		q := question.Question{
			ID:       GuideQHome,
			Material: material,
			Ask:      "Where should the key live?",
			Options: []question.Option{
				{Value: KeyHomeExternal, Label: "An environment variable", Meaning: "A variable you set holds the key; abcd keeps only its name."},
				{Value: KeyHomeABCD, Label: "abcd's own file", Meaning: "You paste it hidden in your terminal; a file only you can read keeps it."},
				{Value: KeyHomeKeychain, Label: "The system keychain", Meaning: "You paste it hidden in your terminal; the system keychain keeps it."},
			},
		}
		v, s := g.put(q, options(q, nil))
		if s != nil {
			return GuideTurn{}, s
		}
		home = v
	}

	// 6. The variable, for the external home.
	if home == KeyHomeExternal {
		q := question.Question{
			ID: GuideQEnv,
			Material: []question.Block{para("The variable must be set in the terminal you paste the command into. " +
				"abcd keeps only its name, never its value.")},
			Ask:   "Which environment variable holds the key?",
			Typed: "the variable's name",
		}
		for _, n := range g.envNames(provider) {
			q.Options = append(q.Options, question.Option{Value: n, Label: n, Meaning: "A variable set where this guide runs; its value is never read."})
		}
		v, s := g.put(q, options(q, func(t string) (string, string, error) {
			if !credential.ValidEnvName(t) {
				return "", "That is not a variable's name: letters, digits and _, not starting with a digit.", nil
			}
			return t, "", nil
		}))
		if s != nil {
			return GuideTurn{}, s
		}
		env = v
	}

	done, err := guideCommand(provider, base, model, home, env, picks)
	if err != nil {
		return GuideTurn{}, &stop{err: err}
	}
	g.st.Open = ""
	return GuideTurn{Done: done, Resume: g.st}, nil
}

// listed is the look-up's outcome: the one the resume object carries, its
// ids checked again, or, when it carries none, the one keyless request.
func (g *guide) listed(base, host string) (*GuideListed, error) {
	if l := g.st.Listed; l != nil {
		if l.Host != host {
			return nil, refuse("the resume object's model list is for %q, not %q", layered.BoundKey(l.Host), host)
		}
		if l.More < 0 || (l.More > 0 && l.Status != ListedListed) {
			return nil, refuse("the resume object's model list counts %d more names than it carries", l.More)
		}
		switch l.Status {
		case ListedListed:
			if n := len(l.Models) + l.More; n > openaiapi.MaxListedModels {
				return nil, refuse("the resume object's model list holds %d names; a look-up keeps at most %d", n, openaiapi.MaxListedModels)
			}
			kept := make([]string, 0, len(l.Models))
			seen := map[string]bool{}
			for _, id := range l.Models {
				if g.modelRefusal(id) == "" && !seen[id] {
					seen[id] = true
					kept = append(kept, id)
				}
			}
			if kept = dropKeyShaped(kept); len(kept) == 0 {
				return &GuideListed{Status: ListedNone, Host: host, Reason: "listed no usable models"}, nil
			}
			kept, more := carry(kept)
			return &GuideListed{Status: ListedListed, Host: host, Models: kept, More: l.More + more}, nil
		case ListedNeedsKey, ListedNone:
			return &GuideListed{Status: l.Status, Host: host, Reason: termsafe.CleanProseLine(l.Reason, 256)}, nil
		}
		return nil, refuse("the resume object's model list has the status %q", layered.BoundKey(l.Status))
	}
	client, err := openaiapi.New(base, "")
	if err != nil {
		return nil, fmt.Errorf("oracle adapter: %w", err)
	}
	listing, err := client.Models(g.ctx, func(id string) bool { return g.modelRefusal(id) == "" })
	var le *openaiapi.ListError
	switch {
	case errors.As(err, &le) && le.NeedsKey:
		return &GuideListed{Status: ListedNeedsKey, Host: host, Reason: le.Reason}, nil
	case errors.As(err, &le):
		return &GuideListed{Status: ListedNone, Host: host, Reason: le.Reason}, nil
	case err != nil:
		return &GuideListed{Status: ListedNone, Host: host, Reason: "could not be read"}, nil
	}
	seen := map[string]bool{}
	ids := make([]string, 0, len(listing.IDs))
	for _, id := range listing.IDs {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if ids = dropKeyShaped(ids); len(ids) == 0 {
		return &GuideListed{Status: ListedNone, Host: host, Reason: "listed no usable models"}, nil
	}
	ids, more := carry(ids)
	return &GuideListed{Status: ListedListed, Host: host, Models: ids, More: more}, nil
}

// carry is the ids a resume object carries: the first, in the order given,
// whose JSON array fits MaxCarriedBytes, and how many of the rest it does not
// carry.
func carry(ids []string) (kept []string, more int) {
	size := len("[]")
	for i, id := range ids {
		b, err := json.Marshal(id)
		if err != nil {
			return ids[:i], len(ids) - i
		}
		if i > 0 {
			size++ // the comma before it
		}
		if size += len(b); size > MaxCarriedBytes {
			return ids[:i], len(ids) - i
		}
	}
	return ids, 0
}

// typedModel admits a model's name as typed: one validModel and the
// denylist take (modelRefusal), else the question is asked again saying why.
func (g *guide) typedModel(ans string) (string, string, error) {
	t := strings.TrimSpace(ans)
	if retry := g.modelRefusal(t); retry != "" {
		return "", retry, nil
	}
	return t, "", nil
}

// modelRefusal is why a model name cannot be offered or printed: one
// validModel refuses (the adapter keeps characters a shell acts on), or one
// the denylist refuses; "" admits it. It never quotes the name: a typed name
// is asked for again without it. An id carrying a key is dropped apart, in
// one scan of the whole list (dropKeyShaped).
func (g *guide) modelRefusal(m string) string {
	if !validModel(m) {
		return "That is not a model name abcd accepts: letters, digits and . _ : @ + ~ -, in parts separated by /."
	}
	if e, denied := Denied(g.cfg.denylist, m); denied {
		return fmt.Sprintf("That name is refused by %s (%s, from %s), even when a service lists it.", denylistKey, e.Pattern, e.Origin)
	}
	return ""
}

// pickListed is the model question over a listed service (G2): the models
// the person already uses that the service lists, then typing part of a name
// to narrow the carried list, with no second request. more is how many ids
// the service listed that are not carried: a part of a name matching none
// carried then asks for the model's full name, taken as any typed name is.
func (g *guide) pickListed(host string, ids []string, more int) (string, *stop) {
	in := make(map[string]bool, len(ids))
	for _, id := range ids {
		in[id] = true
	}
	exact := func(t string) (string, bool) {
		if in[t] {
			return t, true
		}
		return "", false
	}
	q := question.Question{
		ID:       GuideQModel,
		Material: []question.Block{para(fmt.Sprintf("%s lists %d models.", host, len(ids)))},
		Ask:      "Which model should abcd set up?",
		Typed:    "part of a model's name",
	}
	if more > 0 {
		q.Material = []question.Block{para(fmt.Sprintf("%s lists %d models; the guide carries the first %d.", host, len(ids)+more, len(ids)))}
	}
	for _, m := range g.suggestions(in) {
		q.Options = append(q.Options, question.Option{Value: m, Label: m, Meaning: "A model one of your connections already uses."})
	}
	// A fragment that is exactly one listed id chooses it; any other text
	// narrows. The recorded value is the text, so the replay re-derives which.
	text := func(t string) (string, string, error) {
		if t == "" {
			return "", "Type part of a model's name, or choose one.", nil
		}
		return t, "", nil
	}
	v, s := g.put(q, options(q, text))
	if s != nil {
		return "", s
	}
	for {
		if m, ok := exact(v); ok {
			return m, nil
		}
		var matches []string
		for _, id := range ids {
			if question.Matches(v, question.Option{Value: id, Label: id}) {
				matches = append(matches, id)
			}
		}
		nq := question.Question{
			ID:    GuideQNarrow,
			Ask:   "Which model should abcd set up?",
			Typed: "more of the name, or another part",
		}
		switch n := len(matches); {
		case n == 0 && more > 0:
			tq := question.Question{
				ID: GuideQTyped,
				Material: []question.Block{
					para(fmt.Sprintf("None of the %d models the guide carries matches %q.", len(ids), echoed(v))),
					para(fmt.Sprintf("%s listed %d more models than the guide carries; type the model's full name.", host, more)),
				},
				Ask:   "Which model should abcd set up?",
				Typed: "the model's full name, exactly as the service spells it",
			}
			return g.put(tq, g.typedModel)
		case n == 0:
			nq.Material = []question.Block{para(fmt.Sprintf("None of the %d models %s lists matches %q.", len(ids), host, echoed(v)))}
		case n > maxNarrowShown:
			nq.Material = []question.Block{para(fmt.Sprintf("%d of %d match %q; the first %d are shown.", n, len(ids), echoed(v), maxNarrowShown))}
		default:
			nq.Material = []question.Block{para(fmt.Sprintf("%d of %d match %q.", n, len(ids), echoed(v)))}
		}
		for _, m := range matches[:min(len(matches), maxNarrowShown)] {
			nq.Options = append(nq.Options, question.Option{Value: m, Label: m, Meaning: "A model " + host + " lists."})
		}
		if v, s = g.put(nq, options(nq, text)); s != nil {
			return "", s
		}
	}
}

// suggestions are the models the person already uses (decision 6): each
// route's model in route order, then each configured provider's allowlist in
// name order, kept only where the service lists the id exactly (open question
// 3, decided (a)), at most three.
func (g *guide) suggestions(listed map[string]bool) []string {
	var out []string
	seen := map[string]bool{}
	add := func(m string) {
		if len(out) < maxSuggestions && listed[m] && !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	for _, r := range g.cfg.Routes() {
		add(r.Target.Model)
	}
	for _, p := range g.cfg.Providers() {
		for _, m := range p.Models {
			add(m)
		}
	}
	return out
}

// envNames are at most three names from the guide's environment that end in
// _API_KEY (open question 4, decided (a)), those naming the provider first,
// each a name a pointer takes. Values are never read.
func (g *guide) envNames(provider string) []string {
	var named, other []string
	seen := map[string]bool{}
	want := strings.ToUpper(strings.NewReplacer("-", "_").Replace(provider))
	for _, n := range g.env {
		if !strings.HasSuffix(n, "_API_KEY") || !credential.ValidEnvName(n) || seen[n] {
			continue
		}
		seen[n] = true
		if strings.Contains(strings.ToUpper(n), want) {
			named = append(named, n)
		} else {
			other = append(other, n)
		}
	}
	sort.Strings(named)
	sort.Strings(other)
	all := dropKeyShaped(append(named, other...))
	return all[:min(len(all), maxEnvNames)]
}

// guideSkippedLabels are the host labels a derived provider name passes over.
var guideSkippedLabels = map[string]bool{"api": true, "www": true}

// nonName is a run of characters a provider name cannot hold.
var nonName = regexp.MustCompile(`[^a-z0-9_-]+`)

// deriveProvider names the provider after the address when none was given:
// local for this machine; otherwise the host's label before its last, api and
// www passed over; "-2" and on appended while the name is taken.
func (g *guide) deriveProvider(hostname string) string {
	h := strings.ToLower(strings.TrimSuffix(hostname, "."))
	name := ""
	switch {
	case h == "localhost" || isLoopbackIP(h):
		name = "local"
	case net.ParseIP(h) == nil:
		labels := strings.Split(h, ".")
		for i := len(labels) - 2; i >= 0; i-- {
			if !guideSkippedLabels[labels[i]] && labels[i] != "" {
				name = labels[i]
				break
			}
		}
	}
	name = strings.Trim(nonName.ReplaceAllString(name, "-"), "-_")
	if len(name) > 24 {
		name = name[:24]
	}
	if !providerNameRe.MatchString(name) {
		name = "provider"
	}
	taken := func(n string) bool { _, ok := g.cfg.providers[n]; return ok || n == Harness }
	cand := name
	for i := 2; taken(cand); i++ {
		cand = fmt.Sprintf("%s-%d", name, i)
	}
	return cand
}

func isLoopbackIP(h string) bool {
	ip := net.ParseIP(strings.Trim(h, "[]"))
	return ip != nil && ip.IsLoopback()
}

// guideCommand is the done turn: the command, every value shell-quoted, and
// the paths it writes in the order a run reports them, the key's home's
// (credential.WritesFor) and then the settings file.
func guideCommand(provider, base, model, home, env string, picks bool) (*GuideDone, error) {
	if model != "" && !validModel(model) {
		return nil, refuse("the model %q is not a model identifier", layered.BoundKey(model))
	}
	req := ConnectRequest{Provider: provider, BaseURL: base, Home: home, Pointer: credential.Pointer{Env: env}}
	if model != "" {
		req.Models = []string{model}
	}
	if err := checkConnect(&req, picks); err != nil {
		return nil, err
	}
	parts := []string{"abcd", "ahoy", "connect", shellQuote(provider), "--base-url", shellQuote(base)}
	if model != "" {
		parts = append(parts, "--model", shellQuote(model))
	}
	parts = append(parts, "--home", shellQuote(home))
	if env != "" {
		parts = append(parts, "--env", shellQuote(env))
	}
	writes := append([]string{}, credential.WritesFor(home, req.KeyName)...)
	writes = append(writes, layered.Config.MachineOrigin())
	return &GuideDone{Command: strings.Join(parts, " "), Writes: writes, Provider: provider, Home: home, PicksInTerminal: picks}, nil
}

// shellSafe is a word a POSIX shell takes as it is.
var shellSafe = regexp.MustCompile(`^[A-Za-z0-9_./:@+~=,%-]+$`)

// shellQuote is s as one word of a POSIX shell command: as it is when every
// character is plain, else in single quotes, a quote inside closed, escaped
// and reopened.
func shellQuote(s string) string {
	// A word opening with ~ is a home directory to the shell, so it is quoted.
	if shellSafe.MatchString(s) && !strings.HasPrefix(s, "~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
