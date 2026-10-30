package cli

// embark_lessons.go is `abcd embark lessons`: the predecessor's lessons the
// press-release interview shows (itd-24 criterion 6), and embark's lesson
// ranking as a host judgement routed through the decision adapter
// (dcn-2610300523005947, spc-2609221011151661 step 2). With a brief, each
// lesson is one score question (embark-lesson-ranking, 0 to 4) the result
// lists; the agent running the embark page scores them and hands the scores
// back with --scores, and the lessons are ranked by those scores, term overlap
// breaking ties. The route of embark-lesson-ranking decides who is asked
// (core/lifeboat.RankPredecessorLessons).

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/decide/judge"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/lifeboat"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// embarkFramingRel is the new voyage's brief framing chapter, the text embark
// ranks predecessor lessons against (spc-2609211751376504 scope 7).
const embarkFramingRel = ".abcd/development/brief/01-product/06-framing.md"

// maxEmbarkBriefBytes caps the brief text a ranking reads.
const maxEmbarkBriefBytes = 256 * 1024

// loadLessonsRouter reads the decision adapter's configuration for the target
// at targetAbs: the repository it lies in, or the target itself, is where a
// shadow pair's lab is looked for. What resolving the layers refused (a
// foreign-owned root, a HOME inside a working tree) goes to stderr, one line
// each, as every front door that reads the layers says it. Tests replace it.
var loadLessonsRouter = func(stderr io.Writer, targetAbs string) (*judge.Router, error) {
	roots, notes := layered.RootsFor(targetAbs)
	for _, n := range notes {
		fmt.Fprintln(stderr, termsafe.Sanitize(fsutil.RedactHome(n)))
	}
	repo := roots.Repo
	if repo == "" {
		repo = targetAbs
	}
	return judge.Load(roots, repo)
}

// embarkLessonsView is the lessons view plus where the brief text came from
// (the --brief file, named "--brief", the target's framing chapter, or
// nothing), and the decision adapter's configuration's non-fatal reports.
type embarkLessonsView struct {
	lifeboat.LessonsView
	FramingSource string   `json:"framing_source"`
	Diagnostics   []string `json:"diagnostics,omitempty"`
}

// newEmbarkLessonsCommand is `embark lessons <lifeboat-dir> [target-dir]`.
func newEmbarkLessonsCommand(asJSON *bool, resolveDirs func([]string) (string, string, error)) *cobra.Command {
	var brief, scores string
	cmd := &cobra.Command{
		Use:  "lessons <lifeboat-dir> [target-dir]",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			lbAbs, tgtAbs, err := resolveDirs(args)
			if err != nil {
				return err
			}
			fail := func(err error) error {
				return &exitError{Code: 2, Msg: "embark lessons: " + scrubPaths(err) + " (nothing ranked)"}
			}
			framing, source, err := embarkLessonsFraming(tgtAbs, brief)
			if err != nil {
				return fail(err)
			}
			var raw []byte
			if cmd.Flags().Changed("scores") {
				if raw, err = readLessonScores(cmd, scores); err != nil {
					return fail(err)
				}
			}
			view, err := lifeboat.PredecessorLessons(lbAbs, framing)
			if err != nil {
				return fail(err)
			}
			out := embarkLessonsView{FramingSource: source}
			// The router is read only when a question is asked: with no
			// brief, or no lessons, nothing is judged.
			var router *judge.Router
			if len(view.Questions) > 0 {
				if router, err = loadLessonsRouter(cmd.ErrOrStderr(), tgtAbs); err != nil {
					return fail(fmt.Errorf("the decision adapter's configuration: %w", err))
				}
				out.Diagnostics = termsafe.SanitizeAll(router.Diagnostics)
				for _, d := range out.Diagnostics {
					fmt.Fprintf(cmd.ErrOrStderr(), "abcd: %s\n", d)
				}
			}
			if out.LessonsView, err = lifeboat.RankPredecessorLessons(cmd.Context(), view, router, raw); err != nil {
				return fail(err)
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				if source != "" {
					fmt.Fprintf(w, "brief: %s\n", termsafe.Sanitize(source))
				}
				fmt.Fprint(w, out.Render())
				if len(out.Questions) > 0 {
					fmt.Fprintln(w, "  hand the scores back with --scores <file>; each lesson's id and state_sha256 are in the --json result")
				}
			})
		},
	}
	cmd.Flags().StringVar(&brief, "brief", "",
		"rank against this file's text (the press release the interview is writing) instead of the target's framing chapter")
	cmd.Flags().StringVar(&scores, "scores", "",
		"rank by the host's scores in this file (a scores document answering every lesson's question; - reads stdin)")
	return cmd
}

// readLessonScores reads the host's scores file behind the trust guards the
// other payload readers use: a regular file that is no link, within
// lifeboat.MaxLessonScoresBytes, or stdin bounded to the same cap for "-".
func readLessonScores(cmd *cobra.Command, spec string) ([]byte, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, errors.New("--scores names no file; give the scores file's path, or - for stdin")
	}
	var (
		raw []byte
		err error
	)
	if spec == "-" {
		raw, err = readCappedStdin(cmd, lifeboat.MaxLessonScoresBytes)
	} else {
		raw, err = readGuardedOperand(spec, lifeboat.MaxLessonScoresBytes)
	}
	if err != nil {
		return nil, fmt.Errorf("reading --scores: %w", err)
	}
	return raw, nil
}

// embarkLessonsFraming reads the text the lessons are ranked against: the
// --brief file when one is named, else the target's framing chapter, else none
// (the lessons are then listed unranked). The reads are guarded: no symlink, a
// regular file, capped.
func embarkLessonsFraming(targetAbs, brief string) (text, source string, err error) {
	if brief != "" {
		data, err := fsutil.ReadGuarded(brief, maxEmbarkBriefBytes)
		if err != nil {
			return "", "", fmt.Errorf("reading --brief: %w", err)
		}
		return string(data), "--brief", nil
	}
	// Read inside the target: with a route set, the chapter is sent to the
	// decision adapter, so a directory on its path that leads out of the
	// target is refused, as a link at the leaf always was.
	root, err := os.OpenRoot(targetAbs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("opening the target: %w", err)
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, filepath.FromSlash(embarkFramingRel), maxEmbarkBriefBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("reading the target's framing chapter: %w", err)
	}
	return string(data), embarkFramingRel, nil
}
