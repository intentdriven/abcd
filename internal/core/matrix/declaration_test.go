package matrix

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
)

// goodDeclaration is a matrix of all three task kinds, two builders and one
// reviewer, with a metered cell and so a cap.
const goodDeclaration = `{
  "schema_version": 1,
  "name": "routing check",
  "tasks": [
    {"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518"},
    {"id": "doc-page", "kind": "writing", "source": ["docs/guide.md"], "criteria": "accurate against the source first"},
    {"id": "fidelity", "kind": "check", "material": ["docs/reference"]}
  ],
  "builders": [
    {"id": "host-big", "route": "host"},
    {"id": "or-cheap", "route": "builder=runner:opencode@openrouter/vendor/model", "settings": {"temperature": 0.6}}
  ],
  "reviewers": [
    {"id": "local-rev", "route": "reviewer=runner:opencode@local/qwen"}
  ],
  "repeats": {"pilot": 3, "max": 20},
  "spend_cap": {"amount": 25, "unit": "usd"},
  "load_cap": 4,
  "concurrency": 2,
  "judging": {"guests_may_judge": false}
}`

func TestADeclarationParsesIntoTasksCellsAndRepeats(t *testing.T) {
	d, err := parseDeclaration([]byte(goodDeclaration))
	if err != nil {
		t.Fatalf("parseDeclaration: %v", err)
	}
	if d.SchemaVersion != DeclarationSchema || d.Name != "routing check" {
		t.Fatalf("header = %d %q", d.SchemaVersion, d.Name)
	}
	if len(d.Tasks) != 3 || d.Tasks[0].Kind != KindCode || d.Tasks[0].Intent != "itd-2610161733316518" ||
		d.Tasks[1].Kind != KindWriting || d.Tasks[1].Criteria == "" || d.Tasks[2].Kind != KindCheck || d.Tasks[2].Material[0] != "docs/reference" {
		t.Fatalf("tasks = %+v", d.Tasks)
	}
	if len(d.Builders) != 2 || d.Builders[1].Route != "builder=runner:opencode@openrouter/vendor/model" ||
		string(d.Builders[1].Settings["temperature"]) != "0.6" || len(d.Reviewers) != 1 {
		t.Fatalf("cells = %+v %+v", d.Builders, d.Reviewers)
	}
	if d.Repeats != (Repeats{Pilot: 3, Max: 20}) || d.SpendCap == nil || d.SpendCap.Amount != 25 || d.SpendCap.Unit != "usd" ||
		d.LoadCap == nil || *d.LoadCap != 4 || d.Concurrency != 2 || d.Judging.GuestsMayJudge {
		t.Fatalf("run settings = %+v %+v %v %d %+v", d.Repeats, d.SpendCap, d.LoadCap, d.Concurrency, d.Judging)
	}
	if got := d.cells(); len(got) != 3 || got[2].role != RoleReviewer || got[0].role != RoleBuilder {
		t.Fatalf("cells() = %+v", got)
	}
}

func TestADeclarationWithAnUnknownKeyIsRefused(t *testing.T) {
	for name, doc := range map[string]string{
		"top level":  strings.Replace(goodDeclaration, `"load_cap": 4,`, `"load_cap": 4, "loadcap_typo": 1,`, 1),
		"in a cell":  strings.Replace(goodDeclaration, `{"id": "host-big", "route": "host"}`, `{"id": "host-big", "route": "host", "fallback": "host"}`, 1),
		"in a task":  strings.Replace(goodDeclaration, `"kind": "check",`, `"kind": "check", "seeded": true,`, 1),
		"in repeats": strings.Replace(goodDeclaration, `"max": 20`, `"max": 20, "min": 1`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseDeclaration([]byte(doc)); err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("parseDeclaration = %v; want an unknown-field refusal", err)
			}
		})
	}
}

func TestADeclarationThatRepeatsAKeyIsRefused(t *testing.T) {
	doc := strings.Replace(goodDeclaration, `"load_cap": 4,`, `"load_cap": 4, "Load_Cap": 64,`, 1)
	_, err := parseDeclaration([]byte(doc))
	var dk *jsonstrict.DuplicateKeyError
	if !errors.As(err, &dk) {
		t.Fatalf("parseDeclaration = %v; want a duplicate-key refusal", err)
	}
	if _, err := parseDeclaration([]byte(goodDeclaration + "{}")); !errors.Is(err, jsonstrict.ErrTrailing) {
		t.Fatalf("a trailing document = %v; want ErrTrailing", err)
	}
}

func TestADeclarationPastItsSizeBoundIsRefusedUnread(t *testing.T) {
	// Not JSON at all past the bound: the bound refuses before any parse.
	big := make([]byte, MaxDeclarationBytes+1)
	for i := range big {
		big[i] = '{'
	}
	_, err := parseDeclaration(big)
	if err == nil || !strings.Contains(err.Error(), "bytes") {
		t.Fatalf("parseDeclaration of %d bytes = %v; want the size refusal", len(big), err)
	}
}

func TestADeclarationWithAPilotOutsideThreeToFiveIsRefused(t *testing.T) {
	for _, pilot := range []string{`"pilot": 2`, `"pilot": 6`, `"pilot": 0`} {
		doc := strings.Replace(goodDeclaration, `"pilot": 3`, pilot, 1)
		if _, err := parseDeclaration([]byte(doc)); err == nil || !strings.Contains(err.Error(), "pilot") {
			t.Errorf("%s: parseDeclaration = %v; want a pilot refusal", pilot, err)
		}
	}
	doc := strings.Replace(goodDeclaration, `"max": 20`, `"max": 4`, 1)
	doc = strings.Replace(doc, `"pilot": 3`, `"pilot": 5`, 1)
	if _, err := parseDeclaration([]byte(doc)); err == nil || !strings.Contains(err.Error(), "max") {
		t.Errorf("max below the pilot: %v; want a refusal", err)
	}
}

// Each refusal names what is wrong; none admits a declaration the later steps
// would have to second-guess.
func TestADeclarationIsHeldToItsShape(t *testing.T) {
	cases := map[string]struct{ from, to, want string }{
		"schema":              {`"schema_version": 1`, `"schema_version": 2`, "schema_version"},
		"no name":             {`"name": "routing check"`, `"name": ""`, "name"},
		"unknown kind":        {`"kind": "check"`, `"kind": "essay"`, "kind"},
		"code without intent": {`, "intent": "itd-2610161733316518"`, ``, "intent"},
		"bad intent":          {`"intent": "itd-2610161733316518"`, `"intent": "iss-1"`, "intent"},
		"writing no criteria": {`, "criteria": "accurate against the source first"`, ``, "criteria"},
		"writing no source":   {`"source": ["docs/guide.md"], `, ``, "source"},
		"check no material":   {`, "material": ["docs/reference"]`, ``, "material"},
		"field of other kind": {`"kind": "check", "material"`, `"kind": "check", "criteria": "x", "material"`, "criteria"},
		"path escapes":        {`["docs/guide.md"]`, `["../secrets"]`, "path"},
		"absolute path":       {`["docs/guide.md"]`, `["/etc/passwd"]`, "path"},
		"duplicate task":      {`"id": "doc-page"`, `"id": "feature-a"`, "twice"},
		"bad task id":         {`"id": "doc-page"`, `"id": "Doc Page"`, "id"},
		"duplicate cell":      {`"id": "local-rev"`, `"id": "host-big"`, "twice"},
		"no builders": {`{"id": "host-big", "route": "host"},
    {"id": "or-cheap", "route": "builder=runner:opencode@openrouter/vendor/model", "settings": {"temperature": 0.6}}`, ``, "builder"},
		"empty route":            {`"route": "host"`, `"route": ""`, "route"},
		"control in route":       {`"route": "host"`, `"route": "ho\u001bst"`, "route"},
		"bidi in route":          {`"route": "host"`, `"route": "ho‮st"`, "route"},
		"bidi in name":           {`"name": "routing check"`, `"name": "routing ‮check"`, "name"},
		"bidi in criteria":       {`"criteria": "accurate against the source first"`, `"criteria": "accurate​ against the source first"`, "criteria"},
		"zero load cap":          {`"load_cap": 4`, `"load_cap": 0`, "load_cap"},
		"zero concurrency":       {`"concurrency": 2`, `"concurrency": 0`, "concurrency"},
		"concurrency past bound": {`"concurrency": 2`, `"concurrency": 100000`, "concurrency"},
		"negative load cap":      {`"load_cap": 4`, `"load_cap": -1`, "load_cap"},
		"cap unit":               {`"unit": "usd"`, `"unit": "eur"`, "unit"},
		"cap amount":             {`"amount": 25`, `"amount": 0`, "amount"},
		"too many repeats":       {`"max": 20`, `"max": 100000`, "max"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(goodDeclaration, c.from) {
				t.Fatalf("fixture lacks %q", c.from)
			}
			doc := strings.Replace(goodDeclaration, c.from, c.to, 1)
			_, err := parseDeclaration([]byte(doc))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("parseDeclaration = %v; want a refusal naming %q", err, c.want)
			}
		})
	}
}

// Decision 12 ties the load cap to local cells: a declaration may leave it
// out, and the run record asks for it only when a cell runs locally.
func TestADeclarationMayLeaveTheLoadCapOut(t *testing.T) {
	doc := strings.Replace(goodDeclaration, `"load_cap": 4,
`, "", 1)
	d, err := parseDeclaration([]byte(doc))
	if err != nil {
		t.Fatalf("parseDeclaration without a load cap: %v", err)
	}
	if d.LoadCap != nil {
		t.Fatalf("load cap = %v; want none named", *d.LoadCap)
	}
}

// The declaration bounds how many tasks and cells it names, so the run record
// built from it always fits the bound its lab reads a record back within.
func TestADeclarationBoundsItsTasksAndCells(t *testing.T) {
	var cells, tasks []string
	for i := range maxCells + 1 {
		cells = append(cells, fmt.Sprintf(`{"id": "c-%d", "route": "host"}`, i))
	}
	for i := range maxTasks + 1 {
		tasks = append(tasks, fmt.Sprintf(`{"id": "t-%d", "kind": "code", "intent": "itd-1"}`, i))
	}
	for name, doc := range map[string]string{
		"cells": strings.Replace(goodDeclaration, `{"id": "local-rev", "route": "reviewer=runner:opencode@local/qwen"}`, strings.Join(cells, ","), 1),
		"tasks": strings.Replace(goodDeclaration, `{"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518"},`, strings.Join(tasks, ",")+",", 1),
	} {
		if _, err := parseDeclaration([]byte(doc)); err == nil || !strings.Contains(err.Error(), "at most") {
			t.Errorf("%s past the bound: %v; want a refusal naming the bound", name, err)
		}
	}
}

// A cell's settings are held to the oracle's own settings check, the one a
// routing row and a --route pass: a parameter-shaped name and a scalar value.
func TestACellSettingTheOracleRefusesIsRefused(t *testing.T) {
	for name, settings := range map[string]string{
		"object value":     `{"temperature": {"x": 1}}`,
		"array value":      `{"temperature": [0.6]}`,
		"null value":       `{"temperature": null}`,
		"not a param name": `{"Top-P": 0.9}`,
		"escape in value":  `{"stop": "end\u001b[2J"}`,
	} {
		doc := strings.Replace(goodDeclaration, `"settings": {"temperature": 0.6}`, `"settings": `+settings, 1)
		if _, err := parseDeclaration([]byte(doc)); err == nil {
			t.Errorf("%s: the declaration was admitted", name)
		}
	}
}

// A declaration may name the smallest difference in pass rate that would change
// a choice, which sizes the repeats by a power estimate; it is a difference in
// a rate, so above nought and at most one.
func TestADeclarationMayNameTheSmallestDifferenceThatMatters(t *testing.T) {
	doc := strings.Replace(goodDeclaration, `"max": 20}`, `"max": 20, "min_difference": 0.2}`, 1)
	d, err := parseDeclaration([]byte(doc))
	if err != nil || d.Repeats.MinDifference == nil || *d.Repeats.MinDifference != 0.2 {
		t.Fatalf("parseDeclaration = %+v, %v; want a smallest difference of 0.2", d.Repeats, err)
	}
	for _, bad := range []string{"0", "-0.1", "1.5"} {
		doc := strings.Replace(goodDeclaration, `"max": 20}`, `"max": 20, "min_difference": `+bad+`}`, 1)
		if _, err := parseDeclaration([]byte(doc)); err == nil || !strings.Contains(err.Error(), "min_difference") {
			t.Errorf("min_difference %s: %v; want a refusal naming it", bad, err)
		}
	}
}

// withJudges is goodDeclaration with judge, a JSON fragment of top-level
// fields naming a matcher or a proposer, added after its judging settings.
func withJudges(judge string) string {
	return strings.Replace(goodDeclaration, `"judging": {"guests_may_judge": false}`, `"judging": {"guests_may_judge": false}, `+judge, 1)
}

// A check's matcher and proposer are cells the person declares, each
// optional, each a judge outside the matrix (spec step 9): either named as a
// builder of the same run, on a builder's route or model or under a builder's
// id, is refused, and so is either where the run has no check to judge.
func TestADeclarationNamesAnOptionalMatcherAndProposerOutsideItsBuilders(t *testing.T) {
	// A declaration written before the two fields existed reads as it did.
	d, err := parseDeclaration([]byte(goodDeclaration))
	if err != nil || d.Matcher != nil || d.Proposer != nil || len(d.cells()) != 3 {
		t.Fatalf("parseDeclaration(goodDeclaration) = %+v, %v; want no matcher, no proposer and three cells", d, err)
	}
	doc := withJudges(`"matcher": {"id": "match-or", "route": "matcher=runner:opencode@openrouter/vendor/judge"},
  "proposer": {"id": "propose-host", "route": "proposer=runner:opencode@openrouter/vendor/other"}`)
	d, err = parseDeclaration([]byte(doc))
	if err != nil {
		t.Fatalf("parseDeclaration: %v", err)
	}
	if d.Matcher == nil || d.Matcher.ID != "match-or" || d.Proposer == nil || d.Proposer.ID != "propose-host" {
		t.Fatalf("judges = %+v %+v; want the matcher and the proposer", d.Matcher, d.Proposer)
	}
	roles := map[string]Role{}
	for _, c := range d.cells() {
		roles[c.ID] = c.role
	}
	if roles["match-or"] != RoleMatcher || roles["propose-host"] != RoleProposer || len(roles) != 5 {
		t.Fatalf("cells = %v; want the two judges beside the three cells", roles)
	}
	for name, tc := range map[string]struct{ judge, want string }{
		"a builder's route": {`"matcher": {"id": "match-host", "route": "host"}`, "named as the builder cell host-big"},
		"a builder's model": {`"proposer": {"id": "propose-or", "route": "proposer=runner:claude@openrouter/vendor/model"}`, "named as the builder cell or-cheap"},
		"a builder's id":    {`"matcher": {"id": "or-cheap", "route": "matcher=runner:opencode@openrouter/vendor/judge"}`, "named twice"},
		"an unknown key":    {`"matcher": {"id": "match-or", "route": "matcher=runner:opencode@openrouter/vendor/judge", "weight": 2}`, "weight"},
		"a route of none":   {`"proposer": {"id": "propose-or", "route": ""}`, "names no route"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseDeclaration([]byte(withJudges(tc.judge))); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("parseDeclaration = %v; want a refusal saying %q", err, tc.want)
			}
		})
	}
	noCheck := strings.Replace(withJudges(`"matcher": {"id": "match-or", "route": "matcher=runner:opencode@openrouter/vendor/judge"}`),
		`,
    {"id": "fidelity", "kind": "check", "material": ["docs/reference"]}`, "", 1)
	if _, err := parseDeclaration([]byte(noCheck)); err == nil || !strings.Contains(err.Error(), "no check task") {
		t.Fatalf("a matcher with no check to judge = %v; want it refused", err)
	}
}

// A code task names its hidden tests by a folder and the command that runs
// them, the two together (spec step 9 lands them as schema; step 10 reads
// them): an absolute, clean folder and a command of plain words.
func TestACodeTasksHiddenTestsAndTestCommandParse(t *testing.T) {
	code := `{"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518"}`
	with := func(task string) string { return strings.Replace(goodDeclaration, code, task, 1) }
	d, err := parseDeclaration([]byte(with(`{"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518",
      "hidden_tests": "/work/hidden/feature-a", "test_command": ["go", "test", "-run", "TestHidden", "./..."]}`)))
	if err != nil {
		t.Fatalf("parseDeclaration: %v", err)
	}
	if got := d.Tasks[0]; got.HiddenTests != "/work/hidden/feature-a" || len(got.TestCommand) != 5 || got.TestCommand[4] != "./..." {
		t.Fatalf("task = %+v; want its hidden tests and command", got)
	}
	for name, tc := range map[string]struct{ task, want string }{
		"tests with no command":   {`{"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518", "hidden_tests": "/work/hidden"}`, "come together"},
		"a command with no tests": {`{"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518", "test_command": ["go", "test"]}`, "come together"},
		"a relative folder":       {`{"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518", "hidden_tests": "hidden", "test_command": ["go"]}`, "absolute, clean path"},
		"an unclean folder":       {`{"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518", "hidden_tests": "/work/../hidden", "test_command": ["go"]}`, "absolute, clean path"},
		"an empty word":           {`{"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518", "hidden_tests": "/work/hidden", "test_command": ["go", ""]}`, "word 2"},
		"a control word":          {`{"id": "feature-a", "kind": "code", "intent": "itd-2610161733316518", "hidden_tests": "/work/hidden", "test_command": ["go\u001b[2J"]}`, "word 1"},
		"on a check task":         {`{"id": "feature-a", "kind": "check", "material": ["docs"], "hidden_tests": "/work/hidden", "test_command": ["go"]}`, "does not apply"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseDeclaration([]byte(with(tc.task))); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("parseDeclaration = %v; want a refusal saying %q", err, tc.want)
			}
		})
	}
}

// The builder separation compares the model a route names, not its text: the
// same model named through another provider, in another case, is still the
// builder's own (spec step 9).
func TestAJudgeNamingABuildersModelThroughAnotherProviderIsRefused(t *testing.T) {
	for name, judge := range map[string]string{
		"another provider": `"matcher": {"id": "match-x", "route": "matcher=runner:other@vendor/model"}`,
		"another case":     `"proposer": {"id": "propose-x", "route": "proposer=runner:opencode@openrouter/Vendor/MODEL"}`,
		"a variant tag":    `"matcher": {"id": "match-x", "route": "matcher=runner:opencode@openrouter/vendor/model:free"}`,
		"a dated snapshot": `"matcher": {"id": "match-x", "route": "matcher=runner:other@vendor/model-20260101"}`,
		"a dashed date":    `"proposer": {"id": "propose-x", "route": "proposer=runner:other@vendor/Model-2026-01-01"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseDeclaration([]byte(withJudges(judge))); err == nil || !strings.Contains(err.Error(), "named as the builder cell or-cheap") {
				t.Fatalf("parseDeclaration = %v; want the judge refused as the builder or-cheap's model", err)
			}
		})
	}
	if _, err := parseDeclaration([]byte(withJudges(`"matcher": {"id": "match-x", "route": "matcher=runner:other@vendor/judge"}`))); err != nil {
		t.Fatalf("a judge on another model = %v; want it taken", err)
	}
}

// One model under each spelling carriers give it is one key, and models that
// differ in their weights or their version number stay apart (spec step 9).
func TestModelKeyReadsOneModelUnderEachSpellingAndKeepsOtherModelsApart(t *testing.T) {
	same := [][2]string{
		{"vendor/model-4-5", "openrouter/vendor/model-4.5"},
		{"vendor/model-x", "openrouter/vendor/Model-X"},
		{" model-x ", "model-x"},
		{"model-x", "model-x@20261001"},
		{"model-x", "model-x-20261001"},
		{"model-x", "model-x-2026-10-01"},
		{"model-x", "model-x-2026-10"},
		{"model-x", "model-x-latest"},
		{"anthropic/model-x", "us.anthropic.model-x-20261001-v1:0"},
		{"meta/model-4-5", "eu.meta.model-4.5-v2:0"},
	}
	for _, v := range routingVariants {
		same = append(same, [2]string{"vendor/model-x", "openrouter/vendor/model-x:" + v})
	}
	for _, p := range same {
		if a, b := modelKey(p[0]), modelKey(p[1]); a != b || a == "" {
			t.Errorf("modelKey(%q) = %q and modelKey(%q) = %q; want one model", p[0], a, p[1], b)
		}
	}
	for _, p := range [][2]string{
		{"local/qwen2.5-coder:7b", "local/qwen2.5-coder:32b"},
		{"local/model:q4_K_M", "local/model:q8_0"},
		{"vendor/model-4", "vendor/model-4-5"},
		{"vendor/model-4.5", "vendor/model-4.6"},
		{"vendor/model-x", "vendor/model-y"},
	} {
		if a, b := modelKey(p[0]), modelKey(p[1]); a == b {
			t.Errorf("modelKey(%q) and modelKey(%q) are both %q; want two models", p[0], p[1], a)
		}
	}
	// The pair a size tag tells apart is two cells a declaration admits.
	local := strings.Replace(withJudges(`"matcher": {"id": "match-local", "route": "matcher=runner:opencode@local/qwen2.5-coder:32b"}`),
		`"route": "builder=runner:opencode@openrouter/vendor/model"`, `"route": "builder=runner:opencode@local/qwen2.5-coder:7b"`, 1)
	if _, err := parseDeclaration([]byte(local)); err != nil {
		t.Fatalf("a matcher on another size of the builder's local model = %v; want it taken", err)
	}
}

// A latest alias names the model it aliases, written after a dash or as a
// colon tag (`model:latest`, a local server's default tag), and so keys as
// that model.
func TestModelKeyReadsALatestAliasAsTheModel(t *testing.T) {
	for _, p := range [][2]string{
		{"foo", "foo-latest"},
		{"foo", "foo:latest"},
		{"local/qwen2.5-coder", "local/qwen2.5-coder:latest"},
		{"vendor/model-4-5", "openrouter/vendor/model-4.5-latest:free"},
	} {
		if a, b := modelKey(p[0]), modelKey(p[1]); a != b || a == "" {
			t.Errorf("modelKey(%q) = %q and modelKey(%q) = %q; want one model", p[0], a, p[1], b)
		}
	}
	if a, b := modelKey("local/qwen2.5-coder:latest"), modelKey("local/qwen2.5-coder:7b"); a == b {
		t.Errorf("modelKey of the latest tag and of a size tag are both %q; want two models", a)
	}
}

// A bracketed context suffix (`[1m]`) names the same model with a longer
// context window, so it keys as that model, as samemodel reads it.
func TestModelKeyReadsAContextSuffixAsTheModel(t *testing.T) {
	for _, p := range [][2]string{
		{"claude-opus-5-5[1m]", "anthropic/claude-opus-5-5"},
		{"model-x [1m]", "model-x"},
		{"model-x-20261001[1m]", "vendor/model-x"},
	} {
		if a, b := modelKey(p[0]), modelKey(p[1]); a != b || a == "" {
			t.Errorf("modelKey(%q) = %q and modelKey(%q) = %q; want one model", p[0], a, p[1], b)
		}
	}
	if k := modelKey("[1m]"); k == "" {
		t.Error("modelKey of a lone suffix is empty; want a key")
	}
}

// A carrier's revision tail is cut only after its dotted prefix was, with or
// without the v (openai.gpt-oss-120b-1:0), so a bare name that ends in digits
// and a colon keeps them.
func TestModelKeyCutsARevisionTailOnlyAfterACarrierPrefix(t *testing.T) {
	for _, p := range [][2]string{
		{"openai/gpt-oss-120b", "openai.gpt-oss-120b-1:0"},
		{"openai/gpt-oss-120b", "us.openai.gpt-oss-120b-1:0"},
		{"anthropic/model-x", "us.anthropic.model-x-v1:0"},
	} {
		if a, b := modelKey(p[0]), modelKey(p[1]); a != b || a == "" {
			t.Errorf("modelKey(%q) = %q and modelKey(%q) = %q; want one model", p[0], a, p[1], b)
		}
	}
	for _, p := range [][2]string{
		{"local/model-x-1:0", "local/model-x"},
		{"local/model-x-v1:0", "local/model-x"},
	} {
		if a, b := modelKey(p[0]), modelKey(p[1]); a == b {
			t.Errorf("modelKey(%q) and modelKey(%q) are both %q; want two models", p[0], p[1], a)
		}
	}
}

// A dotted prefix is cut only where a carrier writes it, its region and its
// vendor before the model (us.anthropic.), so a model's own dotted name keeps
// what tells it apart, and no name keys as nothing.
func TestModelKeyCutsOnlyACarriersDottedPrefix(t *testing.T) {
	for _, p := range [][2]string{
		{"anthropic/model-x", "us.anthropic.model-x-20261001-v1:0"},
		{"meta/model-x", "eu.meta.model-x-v1:0"},
		{"model-x", "anthropic.model-x-v2:0"},
		{"model-x", "apac.amazon.model-x"},
		{"model-x", "us-gov.anthropic.model-x"},
		{"model-x", "global.anthropic.model-x"},
		{"model-x", "US.Anthropic.Model-X"},
	} {
		if a, b := modelKey(p[0]), modelKey(p[1]); a != b || a == "" {
			t.Errorf("modelKey(%q) = %q and modelKey(%q) = %q; want one model", p[0], a, p[1], b)
		}
	}
	for _, p := range [][2]string{
		{"aya.expanse", "yi.expanse"},
		{"local/aya.expanse", "expanse"},
		{"us.anthropic.aya.expanse", "us.anthropic.yi.expanse"},
	} {
		if a, b := modelKey(p[0]), modelKey(p[1]); a == b {
			t.Errorf("modelKey(%q) and modelKey(%q) are both %q; want two models", p[0], p[1], a)
		}
	}
	for _, s := range []string{"us.anthropic.", "anthropic.", "us.", "a.b.", "-latest", ":free", ":latest", "@20261001", "-20261001", "-v1:0", "vendor/", "us.anthropic.-latest"} {
		if k := modelKey(s); k == "" {
			t.Errorf("modelKey(%q) is empty; want a key", s)
		}
	}
}
