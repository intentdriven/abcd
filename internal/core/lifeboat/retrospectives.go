package lifeboat

// retrospectives.go is embark's reading of the release retrospectives a lifeboat
// carries (itd-24 criterion 6, decision 2): the predecessor's lessons, ranked
// against the new voyage's brief, the few most like it first and the rest as a
// list. The write-back itself is the retrospectives family of embarkFamilies;
// this file only reads.
//
// The lifeboat is untrusted input. It is gated and its manifest verified before
// any lesson is read, the retrospectives are read through the same guarded
// reader the embarker uses, the release a lesson is attributed to is the
// directory's validated tag (never the file's own frontmatter), and every lesson
// is cleaned to one inert, capped line before it reaches a terminal or a host.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corereflect "github.com/intentdriven/abcd/internal/core/reflect"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Lesson ceilings: a lesson is a line or two of prose, and a voyage's
// retrospectives hold tens of them, so anything past these is a hostile or
// broken lifeboat, capped rather than carried whole into a host's context.
const (
	maxPredecessorLessonBytes = 600
	maxPredecessorLessons     = 500
)

// LessonsView is the embark view of the predecessor's lessons: the ranking, how
// many retrospectives the lifeboat carried, and whether the ranking had a brief
// to rank against.
type LessonsView struct {
	corereflect.Ranking
	Retrospectives int `json:"retrospectives"`
	// Unranked is true when there was no brief text to rank against: the lessons
	// are listed in the order the lifeboat carries them, the top few first.
	Unranked bool `json:"unranked"`
	// Truncated is true when the lifeboat carried more than maxPredecessorLessons lessons
	// and only the first were read.
	Truncated bool `json:"truncated,omitempty"`
}

// PredecessorLessons reads every retrospective the lifeboat at lifeboatDir
// carries and ranks their lessons against framing, the new voyage's brief
// framing text (corereflect.RankLessons). It writes nothing. A lifeboat that is
// not one, is newer than this abcd, or fails its manifest is refused before any
// lesson is read.
func PredecessorLessons(lifeboatDir, framing string) (LessonsView, error) {
	abs, err := filepath.Abs(lifeboatDir)
	if err != nil {
		return LessonsView{}, err
	}
	if err := proveOperand("lifeboat", abs); err != nil {
		return LessonsView{}, err
	}
	if !fsutil.IsRealDir(abs) {
		return LessonsView{}, fmt.Errorf("lifeboat %s is not a directory", filepath.Base(abs))
	}
	if !isAbcdLifeboat(abs) {
		return LessonsView{}, fmt.Errorf("%s is not an abcd lifeboat (no parseable %s)", filepath.Base(abs), ProvenanceName)
	}
	prov, err := readProvenance(abs)
	if err != nil {
		return LessonsView{}, err
	}
	if prov.SchemaVersion > SchemaVersion {
		return LessonsView{}, update.TooNew("lifeboat", prov.SchemaVersion, SchemaVersion)
	}
	if err := VerifyManifest(abs); err != nil {
		return LessonsView{}, err
	}

	root, err := os.OpenRoot(abs)
	if err != nil {
		return LessonsView{}, err
	}
	defer root.Close()
	rels, err := walkLifeboatFiles(root)
	if err != nil {
		return LessonsView{}, err
	}

	var (
		lessons   []corereflect.Lesson
		retros    int
		truncated bool
	)
	for _, rel := range rels {
		family, _, disp, _ := resolveTarget(rel)
		if family != "retrospectives" || disp != dispPlanned {
			continue
		}
		data, err := readLifeboatFile(root, rel)
		if err != nil {
			return LessonsView{}, err
		}
		retros++
		// The release is the directory's tag, which resolveTarget has held to the
		// release-tag shape; the file's own frontmatter is the lifeboat's claim.
		tag := strings.Split(strings.TrimPrefix(rel, retrospectivesLifeboatDir+"/"), "/")[0]
		for _, l := range corereflect.ReadLessons(data) {
			if len(lessons) == maxPredecessorLessons {
				truncated = true
				break
			}
			text := termsafe.CleanProseLine(l.Text, maxPredecessorLessonBytes)
			if text == "" {
				continue
			}
			lessons = append(lessons, corereflect.Lesson{Release: tag, Text: text})
		}
	}
	view := LessonsView{
		Ranking:        corereflect.RankLessons(framing, lessons),
		Retrospectives: retros,
		Unranked:       strings.TrimSpace(framing) == "",
		Truncated:      truncated,
	}
	if view.Top == nil {
		view.Top = []corereflect.RankedLesson{}
	}
	return view, nil
}

// Render is the person's view of the lessons: the few first, each with the
// release it came from, then the rest as a list. Every lesson is already one
// cleaned line; the release is a validated tag.
func (v LessonsView) Render() string {
	var b strings.Builder
	if v.Retrospectives == 0 {
		b.WriteString("the lifeboat carries no retrospectives, so there are no predecessor lessons to show\n")
		return b.String()
	}
	n := len(v.Top) + len(v.Rest)
	fmt.Fprintf(&b, "predecessor lessons: %d from %d retrospective(s)", n, v.Retrospectives)
	if v.Unranked {
		b.WriteString(", unranked (no brief text to rank against)")
	} else {
		fmt.Fprintf(&b, ", ranked against the brief (%s)", sanitize(v.Heuristic))
	}
	b.WriteString("\n")
	if v.Truncated {
		fmt.Fprintf(&b, "  (only the first %d lessons were read)\n", maxPredecessorLessons)
	}
	b.WriteString("\nmost like this voyage — which of these apply?\n")
	for i, l := range v.Top {
		fmt.Fprintf(&b, "  %d. [%s] %s\n", i+1, sanitize(l.Release), sanitize(l.Text))
	}
	if len(v.Rest) > 0 {
		fmt.Fprintf(&b, "\nthe rest (%d):\n", len(v.Rest))
		for _, l := range v.Rest {
			fmt.Fprintf(&b, "  - [%s] %s\n", sanitize(l.Release), sanitize(l.Text))
		}
	}
	return b.String()
}
