package cli

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/intentdriven/abcd/internal/core/docfidelity"
	"github.com/intentdriven/abcd/internal/core/ideate"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/lifeboat"
	"github.com/intentdriven/abcd/internal/core/memory"
	"github.com/intentdriven/abcd/internal/core/reading"
	"github.com/intentdriven/abcd/internal/core/release"
	"github.com/intentdriven/abcd/internal/core/scribe"
	"github.com/intentdriven/abcd/internal/core/surface"
)

// inputs.go — the input registry (spc-2610301431075964 step 1, approach 2):
// one entry for every filled-in input a command reads, a flag ending `-json`.
//
// The shape of such an input had no home in code. Its Go type and its version
// constant lived in the owning core package, and its example was written only
// in the command page's prose, which nothing checked: on 2026-10-30 an agent
// composed an ideate verdict from a page at schema version 1 and the binary
// refused it. Each entry here names, for one command path and flag:
//
//   - the owning package's Go type and schema-version constant;
//   - where the document carries that version (VersionIn);
//   - an example, held as a constant beside the type in the owning package and
//     never here, so a change to the shape breaks the example where the shape
//     is;
//   - the owning package's parse-and-validate function, the one its writing
//     command calls, run without writing.
//
// The registry is held whole by a test (inputs_test.go): every visible flag
// ending `-json` that a command reads is in it, every entry names a real
// command and flag, every example passes its check, and an output-only flag
// is declared as such, once, in outputOnlyJSONFlags. So a new input cannot
// ship without its shape.

// inputEntry is one registered input.
type inputEntry struct {
	path, flag, name string
	// shape is the Go type the document decodes into; root says whether the
	// document is that object or an array of it.
	shape reflect.Type
	root  string
	// version is the owning package's schema-version constant, and versionIn
	// where the document carries it.
	version   int
	versionIn string
	example   string
	// check is the owning package's parse-and-validate function. repoRoot is
	// the checkout a refusal is redacted against and a citation resolved in.
	check func(repoRoot string, raw []byte) error
}

// ignoreRoot adapts a check that needs no checkout.
func ignoreRoot(check func([]byte) error) func(string, []byte) error {
	return func(_ string, raw []byte) error { return check(raw) }
}

// inputRegistry is every input, in the order a reader of the table finds them
// by command.
var inputRegistry = []inputEntry{
	{
		path: "abcd disembark graveyard", flag: "lessons-json", name: "graveyard lessons",
		shape: lifeboat.LessonsFileType, root: surface.RootObject,
		version: lifeboat.LessonsSchemaVersion, versionIn: surface.VersionField,
		example: lifeboat.LessonsExample, check: lifeboat.CheckLessons,
	},
	{
		path: "abcd disembark press-release", flag: "press-release-json", name: "lifeboat press release",
		shape: lifeboat.PressReleaseFileType, root: surface.RootObject,
		version: lifeboat.PressReleaseSchemaVersion, versionIn: surface.VersionField,
		example: lifeboat.PressReleaseExample, check: lifeboat.CheckPressRelease,
	},
	{
		path: "abcd disembark principles", flag: "principles-json", name: "lifeboat principles",
		shape: lifeboat.PrinciplesFileType, root: surface.RootObject,
		version: lifeboat.PrinciplesSchemaVersion, versionIn: surface.VersionField,
		example: lifeboat.PrinciplesExample, check: lifeboat.CheckPrinciples,
	},
	{
		path: "abcd disembark review", flag: "review-json", name: "lifeboat review verdict",
		shape: lifeboat.ReviewArtefactType, root: surface.RootObject,
		version: lifeboat.ReviewSchemaVersion, versionIn: surface.VersionField,
		example: lifeboat.ReviewExample, check: lifeboat.CheckReview,
	},
	{
		path: "abcd docs fidelity record", flag: "verdict-json", name: "docs review verdict",
		shape: docfidelity.VerdictPayloadType, root: surface.RootObject,
		version: docfidelity.VerdictSchemaVersion, versionIn: surface.VersionField,
		example: docfidelity.VerdictExample, check: ignoreRoot(docfidelity.Check),
	},
	{
		path: "abcd ideate record", flag: "verdict-json", name: "ideate verdict",
		shape: reflect.TypeOf(ideate.Payload{}), root: surface.RootObject,
		version: ideate.SchemaVersion, versionIn: surface.VersionField,
		example: ideate.Example, check: ideate.Check,
	},
	{
		path: "abcd intent audit ingest", flag: "verdict-json", name: "intent audit verdict",
		shape: intent.VerdictShape, root: surface.RootObject,
		version: intent.VerdictSchemaVersion, versionIn: surface.VersionField,
		example: intent.VerdictExample, check: ignoreRoot(intent.CheckVerdict),
	},
	{
		path: "abcd intent consistency ingest", flag: "findings-json", name: "consistency findings",
		shape: intent.ConsistencyFindingsShape, root: surface.RootObject,
		version: intent.ConsistencySchemaVersion, versionIn: surface.VersionField,
		example: intent.ConsistencyFindingsExample, check: intent.CheckConsistencyFindings,
	},
	{
		path: "abcd intent prepass", flag: "findings-json", name: "pre-pass findings",
		shape: intent.PrepassFindingsShape, root: surface.RootObject,
		version: intent.PrepassFindingsSchemaVersion, versionIn: surface.VersionField,
		example: intent.PrepassFindingsExample, check: ignoreRoot(intent.CheckPrepassFindings),
	},
	{
		path: "abcd launch ship", flag: "changelog-json", name: "composed changelog",
		shape: release.ChangelogPayloadType, root: surface.RootObject,
		version: release.ChangelogSchemaVersion, versionIn: surface.VersionField,
		example: release.ChangelogExample, check: release.CheckChangelog,
	},
	{
		path: "abcd library ask", flag: "page-json", name: "answer page",
		shape: memory.AnswerPageType, root: surface.RootObject,
		version: memory.AnswerPageSchemaVersion, versionIn: surface.VersionField,
		example: memory.AnswerPageExample, check: memory.CheckAnswerPage,
	},
	{
		path: "abcd library ingest", flag: "pages-json", name: "distilled pages",
		shape: memory.PagesType, root: surface.RootArray,
		version: memory.PagesSchemaVersion, versionIn: surface.VersionNone,
		example: memory.PagesExample, check: memory.CheckPages,
	},
	{
		path: "abcd reading ingest", flag: "reading-json", name: "reading output",
		shape: reading.OutputShape, root: surface.RootObject,
		version: reading.OutputSchemaVersion, versionIn: surface.VersionType,
		example: reading.OutputExample, check: reading.CheckOutput,
	},
	{
		path: "abcd scribe ingest", flag: "scribe-json", name: "scribe output",
		shape: scribe.OutputShape, root: surface.RootObject,
		version: scribe.OutputSchemaVersion, versionIn: surface.VersionType,
		example: scribe.OutputExample, check: scribe.CheckOutput,
	},
}

// outputOnlyJSONFlags are the flags ending `-json` that a command WRITES
// rather than reads, keyed by command path and flag. They need no shape here;
// declaring one is the whole of the exemption, so a reading flag cannot slip
// past the registry test by looking like an output. None does today.
var outputOnlyJSONFlags = map[[2]string]bool{}

var (
	inputShapesOnce sync.Once
	inputShapes     map[[2]string]surface.InputShape
	inputShapesErr  error
)

// loadInputShapes builds every registered input's shape once. A declaration
// that cannot be one (a root its type is not, a version carried in a member
// the type lacks) is an error the registry test reports.
func loadInputShapes() (map[[2]string]surface.InputShape, error) {
	inputShapesOnce.Do(func() {
		inputShapes = make(map[[2]string]surface.InputShape, len(inputRegistry))
		for _, e := range inputRegistry {
			key := [2]string{e.path, e.flag}
			if _, dup := inputShapes[key]; dup {
				inputShapesErr = fmt.Errorf("input %s --%s is registered twice", e.path, e.flag)
				return
			}
			s, err := surface.NewInputShape(e.path, e.flag, e.name, e.shape, e.root, e.version, e.versionIn, e.example)
			if err != nil {
				inputShapesErr = err
				return
			}
			inputShapes[key] = s
		}
	})
	return inputShapes, inputShapesErr
}

// inputFor is the registered shape of the input a command reads through flag.
// A registry that does not build answers nothing, and the registry test names
// why.
func inputFor(path, flag string) (surface.InputShape, bool) {
	shapes, err := loadInputShapes()
	if err != nil {
		return surface.InputShape{}, false
	}
	s, ok := shapes[[2]string{path, flag}]
	return s, ok
}

// InputShapes is every registered input's shape, sorted by command path and
// flag.
func InputShapes() ([]surface.InputShape, error) {
	shapes, err := loadInputShapes()
	if err != nil {
		return nil, err
	}
	out := make([]surface.InputShape, 0, len(shapes))
	for _, s := range shapes {
		out = append(out, s)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Path != out[b].Path {
			return out[a].Path < out[b].Path
		}
		return out[a].Flag < out[b].Flag
	})
	return out, nil
}
