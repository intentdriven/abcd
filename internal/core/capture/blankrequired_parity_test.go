package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/lint"
)

// blankSpellings are the ways a record writes a required property that carries
// nothing on the key's own line while still being PRESENT. They are held apart
// rather than folded into one case because the readers part them: `""` decodes to
// the empty string, a bare single-quote pair survives decoding intact (this
// reader unquotes double quotes only), `" "` decodes to padding, an indented
// block sequence puts no scalar on the key's line at all, and the empty flow
// mapping and the explicit null tag are both left as the strings they are spelled
// with — different values, and the reader reaches different verdicts on some of
// the fields.
//
// The last two are here because the gate read one empty flow collection as
// nothing and the other as a value, so `grounds: {}` passed a rule armed for
// exactly that blank (iss-2608301649337965). A set that enumerates the spellings
// is only as good as the enumeration.
//
// A spelling that begins with a newline is a CONTINUATION: the key is written
// bare and the lines below it carry the shape.
var blankSpellings = []struct {
	name string
	yaml string
}{
	{"double-quoted", `""`},
	{"single-quoted", `''`},
	{"quoted padding", `" "`},
	{"block sequence", "\n  - a"},
	{"empty flow mapping", `{}`},
	{"explicit null tag", `!!null`},
}

// blankRequiredRecord renders a well-formed open-issue record with exactly one
// required property blanked, so a single verdict is the whole delta.
func blankRequiredRecord(field, spelling string) []string {
	values := map[string]string{
		"schema_version": "1",
		"id":             "iss-1",
		"slug":           "a-finding",
		"severity":       "minor",
		"category":       "bug",
		"source":         "user-observation",
		"found_during":   "a session",
	}
	values[field] = spelling
	lines := make([]string, 0, len(issueschema.Required))
	for _, k := range issueschema.Required {
		if strings.HasPrefix(values[k], "\n") {
			lines = append(lines, k+":")
			lines = append(lines, strings.Split(values[k], "\n")[1:]...)
			continue
		}
		lines = append(lines, k+": "+values[k])
	}
	return lines
}

// claim is one statement a finding message can make about what the LEDGER READER
// does with the record it reports. Each is a claim the operator acts on: a
// refusal sends them to look for a record that is not being read, an acceptance
// tells them the record is read and the blank is rendering as an answer. A
// message that makes the claim the reader disproves sends them to the wrong
// remedy, which is the whole reason this parity is pinned rather than reviewed.
type claim struct {
	phrase string
	// refusal is what the phrase asserts: true for "the reader refuses and skips
	// this record", false for "the reader reads it".
	refusal bool
}

var readerClaims = []claim{
	{"skips the record", true},
	{"is skipped", true},
	{"capture refuses", true},
	{"this record is read", false},
	{"renders the property as answered", false},
}

// blankJudgedByAnotherLeg names the required properties whose present-but-blank
// SCALAR another leg of the record_schema rule reports on its own terms, so the
// generic required-fields leg must stay silent about them rather than adding a
// second and weaker finding on the same line.
//
// The value says whether that leg can also state the READER's verdict. The enum
// legs and the slug grammar are capture's own, read from the one shared schema,
// so they can and must: for those it is not enough to report that the property is
// blank — a gate that can establish the consequence and does not sends the author
// to find it themselves. The id leg reports a disagreement with the FILENAME,
// which is its own question and no claim about the reader, so it states none.
//
// `schema_version` and `found_during` are absent entirely: no leg judges them, so
// the generic finding is the only one, and it can say nothing about the reader
// without inventing it.
var blankJudgedByAnotherLeg = map[string]bool{
	"id":       false,
	"slug":     true,
	"severity": true,
	"category": true,
	"source":   true,
}

// genericBlankFinding is the required-fields leg's own account of a blank. It is
// quoted here to pin the SUPPRESSION: where another leg has spoken, this must not
// also appear.
const genericBlankFinding = "carries no value once its YAML scalar is read"

// TestBlankRequiredPropertyFindingsMatchTheReadersVerdict pins the record_schema
// gate's account of a PRESENT-BUT-BLANK required property against what the ledger
// reader actually does with it, for every required property × every blank
// spelling — forty-two combinations, of which the reader refuses thirty-nine and
// accepts three, every one of them a found_during: the bare single-quote pair, the
// empty flow mapping and the explicit null tag, each of which this reader's
// decoder leaves as the string it is spelled with and therefore reads as
// non-empty. `found_during` is the field no shape check judges, which is why it is
// the field where the two readings can differ at all.
//
// What a blank does to a record is a property of the FIELD, not of the store: the
// reader's required-property loop type-checks without judging, but the shape
// checks below it judge severity, category, source, slug and id, and the version
// check above it judges schema_version. A single store-wide sentence about "what
// the reader does with a blank" is therefore false in every combination it does
// not happen to match — and a gate that misdescribes why it refused sends the
// operator to look for a consequence that is not there (iss-2608301308369559).
//
// The pin is deliberately not "this exact message". It is: the gate must SAY
// something (never go silent on a blank required property), and nothing it says
// may make the reader-claim the reader disproves.
func TestBlankRequiredPropertyFindingsMatchTheReadersVerdict(t *testing.T) {
	for _, field := range issueschema.Required {
		for _, spelling := range blankSpellings {
			t.Run(field+"/"+spelling.name, func(t *testing.T) {
				lines := blankRequiredRecord(field, spelling.yaml)

				// The reader's verdict, from the same parse and the same validator the
				// ledger read path runs. A record the PARSER refuses is a record the
				// ledger never reads either, so a parse error is a refusal like any
				// other — the block-sequence spelling is refused there rather than at
				// the validator.
				fm, readerErr := parseFrontmatterBlock(lines)
				if readerErr == nil {
					readerErr = validateStrict(fm)
				}
				accepted := readerErr == nil

				// The gate's verdict on the same bytes.
				root := t.TempDir()
				writeRecord(t, root, "rec/README.md", "# rec\n")
				writeRecord(t, root, "work/issues/open/iss-1-a-finding.md",
					"---\n"+strings.Join(lines, "\n")+"\n---\n\nan issue\n")
				fs, err := lint.Lint(lint.Config{
					Roots: []string{"rec"},
					Rules: map[string]lint.RuleConfig{
						"record_schema": {Enabled: true, Severity: "blocker", RecordStores: map[string]string{
							"iss": "work/issues",
						}},
					},
				}, root)
				if err != nil {
					t.Fatal(err)
				}
				var said []string
				for _, f := range fs {
					if f.RuleID == "record_schema" {
						said = append(said, f.Message)
					}
				}
				t.Logf("reader=%s | gate=%d finding(s): %s",
					map[bool]string{true: "ACCEPTS", false: "REFUSES (" + errText(readerErr) + ")"}[accepted],
					len(said), strings.Join(said, " || "))

				if len(said) == 0 {
					t.Fatalf("a blank required property must be a finding; the gate said nothing")
				}
				// A blank SCALAR on the key's own line is a value the legs that judge
				// content can read. (A block sequence is not: the shared frontmatter
				// scanner reads same-line values, so there is no scalar there for them to
				// judge, and the generic leg is rightly the only voice.)
				statesConsequence, judgedElsewhere := blankJudgedByAnotherLeg[field]
				if judgedElsewhere && !strings.HasPrefix(spelling.yaml, "\n") {
					for _, msg := range said {
						if strings.Contains(msg, genericBlankFinding) {
							t.Errorf("%s is already reported by the leg that judges its value, so the generic "+
								"blank finding must not appear beside it: %s", field, strings.Join(said, " || "))
						}
					}
					if statesConsequence && !accepted {
						var stated bool
						for _, msg := range said {
							for _, c := range readerClaims {
								if c.refusal && strings.Contains(msg, c.phrase) {
									stated = true
								}
							}
						}
						if !stated {
							t.Errorf("the reader refuses this record (%v) and the gate mirrors its %s check, "+
								"so the finding must say what the reader does with the blank, not only that it is blank: %s",
								readerErr, field, strings.Join(said, " || "))
						}
					}
				}
				for _, msg := range said {
					for _, c := range readerClaims {
						if !strings.Contains(msg, c.phrase) {
							continue
						}
						if c.refusal != accepted {
							continue
						}
						if accepted {
							t.Errorf("the message claims the reader refuses this record (%q) but validateStrict ACCEPTS it: %s", c.phrase, msg)
						} else {
							t.Errorf("the message claims the reader reads this record (%q) but validateStrict refuses it (%v): %s", c.phrase, readerErr, msg)
						}
					}
				}
			})
		}
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// writeRecord writes one fixture file, creating its parents.
func writeRecord(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
