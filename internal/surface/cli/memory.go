package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/memnote"
	"github.com/intentdriven/abcd/internal/core/recall"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// memory.go — the `memory` verb, the front door onto internal/core/memnote
// (itd-2610091918433290, spc-2610111627213138 step 1). An agent keeps short
// notes with it; the agents' page commands/memory.md carries the judgement
// half (when to write one, which recall words, never a secret). Every write
// goes through the core's gate and touches only the machine layer,
// ~/.abcd.noindex/memory.json; the project's .abcd/memory.json is read, never
// written. Until the dashboard's review ships (itd-2610111655362931), the
// review is `memory list` and `memory remove`.

// maxMemoryStdinBytes caps the body read from stdin; the note's own limit
// (memnote.MaxBodyBytes) is judged after the surrounding white space is
// trimmed, and names itself.
const maxMemoryStdinBytes = 64 << 10

// The reach a listed note has, as the list names it.
const (
	reachThisRepository    = "this repository"
	reachEverywhere        = "everywhere"
	reachAnotherRepository = "another repository"
	reachProject           = "the project's file"
)

// memoryBareOutput is bare `memory`'s --json envelope.
type memoryBareOutput struct {
	ThisRepository    int              `json:"this_repository"`
	Everywhere        int              `json:"everywhere"`
	OtherRepositories int              `json:"other_repositories"`
	Project           int              `json:"project"`
	Unreadable        []recall.Refusal `json:"unreadable"`
}

// memoryNoteView is one listed note.
type memoryNoteView struct {
	Layer     string   `json:"layer"`
	Reach     string   `json:"reach"`
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Recall    []string `json:"recall"`
	Origin    string   `json:"origin"`
	WrittenBy string   `json:"written_by"`
	WrittenAt string   `json:"written_at"`
	Session   *string  `json:"session"`
}

// memoryListOutput is `memory list`'s --json envelope.
type memoryListOutput struct {
	Notes      []memoryNoteView `json:"notes"`
	Unreadable []recall.Refusal `json:"unreadable"`
}

// memoryAddOutput is `memory add`'s --json envelope.
type memoryAddOutput struct {
	ID      string   `json:"id"`
	File    string   `json:"file"`
	Notices []string `json:"notices"`
}

// memoryRemoveOutput is `memory remove`'s --json envelope.
type memoryRemoveOutput struct {
	Removed string `json:"removed"`
	File    string `json:"file"`
}

// The notices add prints, each on the write it concerns.
const (
	memorySessionUnknownNotice = "no --session was passed, so the note records its session as unknown (session unknown)"
	memoryEverywhereNotice     = "this note is recalled in every repository, but only this checkout's private names were checked; a name another project bans is yours to keep out of it"
)

// readMemoryLayers reads both layers through the recall's one reader
// (recall.ReadLayers), for the repository the rules loader resolved, and
// that repository's root commit, the key a personal note is tied by. A layer
// that cannot be read is named, never fatal here.
func readMemoryLayers(home, repoRoot string) (l recall.Layers, root string, unreadable []recall.Refusal) {
	l, unreadable = recall.ReadLayers(home, repoRoot)
	if unreadable == nil {
		unreadable = []recall.Refusal{}
	}
	return l, memoryRootCommit(repoRoot), unreadable
}

// memoryRootCommit is the repository's root commit, the key a personal note
// is tied by; "" outside a repository with commits.
var memoryRootCommit = gitutil.RootCommit

// reachOf names a personal note's reach from the repository root key.
func reachOf(n memnote.Note, root string) string {
	switch {
	case n.Root == nil:
		return reachEverywhere
	case root != "" && *n.Root == root:
		return reachThisRepository
	}
	return reachAnotherRepository
}

// memoryWhere is the home and the repository root a memory verb works from.
// The root is resolved through rulesRoot, which prints the resolution's notes
// to w: a refused foreign-uid root is named there, before any refusal of the
// verb's own (rules.Resolution).
func memoryWhere(w io.Writer) (home, repoRoot string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	home, _ = os.UserHomeDir()
	return home, rulesRoot(cwd, w), nil
}

// memoryRefusal maps the core's refusals: a malformed field, the gate, no
// root commit, no home or an absent note is exit 2 with nothing written;
// anything else is exit 1, a machine file the read refuses among them, whether
// it fails a trust check or does not parse — as `abcd rules` exits 1 on a
// rules.json layer that fails to load.
func memoryRefusal(verb string, err error) error {
	var fe *memnote.FieldError
	var ge *memnote.GateError
	msg := "abcd memory " + verb + ": " + strings.TrimPrefix(recall.Reason(err), "memory: ")
	if errors.As(err, &fe) || errors.As(err, &ge) || errors.Is(err, memnote.ErrNoRoot) ||
		errors.Is(err, memnote.ErrNoHome) || errors.Is(err, memnote.ErrNotFound) {
		return &exitError{Code: 2, Msg: msg + " (nothing written)"}
	}
	return errors.New(msg)
}

// newMemoryCommand builds the `memory` verb.
func newMemoryCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "memory",
		Long: "Keep memory notes: short lessons an agent writes so a later session does not\n" +
			"relearn them. Personal notes live in the machine layer, `" + memnote.MachineDisplayPath + "`;\n" +
			"notes a person promoted into the project live in its `" + memnote.RepoRelPath + "`, beside each\n" +
			"rules.json. Both are read with the trust checks the rules layers get.\n\n" +
			"Bare `abcd memory` counts the personal notes tied to this repository and those\n" +
			"recalled everywhere, and the notes in the project's file, and names any layer\n" +
			"that could not be read and why. `abcd memory add` writes one note to the machine\n" +
			"layer and nothing else, after a gate refuses any note holding a secret, a\n" +
			"network address or a name this checkout's private banlist holds. `abcd memory\n" +
			"list` shows every note with its body, recall words, stamps and reach; `abcd\n" +
			"memory remove <mem-id>` removes any personal note. A note in the project's file\n" +
			"is removed by editing that file in a reviewed change, never by this verb.\n\n" +
			"Exit 2 on a refusal, with nothing written.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			home, repoRoot, err := memoryWhere(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			l, root, unreadable := readMemoryLayers(home, repoRoot)
			out := memoryBareOutput{Project: len(l.Project), Unreadable: unreadable}
			for _, n := range l.Machine {
				switch reachOf(n, root) {
				case reachThisRepository:
					out.ThisRepository++
				case reachEverywhere:
					out.Everywhere++
				default:
					out.OtherRepositories++
				}
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				fmt.Fprintf(w, "abcd memory — %d personal note(s) for this repository, %d recalled everywhere; %d in the project's file\n",
					out.ThisRepository, out.Everywhere, out.Project)
				if out.OtherRepositories > 0 {
					fmt.Fprintf(w, "  %d personal note(s) are tied to other repositories (`abcd memory list` shows them)\n", out.OtherRepositories)
				}
				for _, u := range out.Unreadable {
					fmt.Fprintf(w, "  unreadable: %s — %s\n", u.File, u.Reason)
				}
			})
		},
	}

	var title, by, session string
	var recallWords []string
	var everywhere, movedFromHost bool
	add := &cobra.Command{
		Use: "add --title <title> --recall <word> --by <vendor:model>",
		Long: "Write one note to the machine layer, the body read from stdin. The note is\n" +
			"stamped with the writer (--by, the Assisted-by form <Vendor>:<model-version>),\n" +
			"the clock's time and the host's session id (--session; without it the session\n" +
			"is recorded as unknown, and the write says so). It is tied to this repository\n" +
			"unless --everywhere recalls it in every repository; such a note has been\n" +
			"checked against this checkout's private names only, and the write says so.\n" +
			"--moved-from-host stamps a lesson the person pointed at in the host's own\n" +
			"memory files; the text still arrives on stdin, since add takes no path.\n\n" +
			"Exit 2 on a refusal — a field past its limit, a secret, a network address or\n" +
			"a private name, a session shaped like a link — with nothing written.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			home, repoRoot, err := memoryWhere(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			raw, err := readCappedStdin(cmd, maxMemoryStdinBytes)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd memory add: the body on stdin: " + err.Error() + " (nothing written)"}
			}
			n, err := memnote.Add(memnote.AddRequest{
				Home: home, RepoRoot: repoRoot, Title: title, Body: strings.TrimSpace(string(raw)),
				Recall: recallWords, By: by, Session: session, Everywhere: everywhere, MovedFromHost: movedFromHost,
			})
			if err != nil {
				return memoryRefusal("add", err)
			}
			out := memoryAddOutput{ID: n.ID, File: memnote.MachineDisplayPath, Notices: []string{}}
			if n.Session == nil {
				out.Notices = append(out.Notices, memorySessionUnknownNotice)
			}
			if everywhere {
				out.Notices = append(out.Notices, memoryEverywhereNotice)
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				fmt.Fprintf(w, "%s written to %s\n", out.ID, out.File)
				for _, s := range out.Notices {
					fmt.Fprintf(w, "  note: %s\n", s)
				}
			})
		},
	}
	add.Flags().StringVar(&title, "title", "", fmt.Sprintf("the note's title, one line of at most %d characters", memnote.MaxTitleChars))
	add.Flags().StringArrayVar(&recallWords, "recall", nil, fmt.Sprintf("a word or short phrase a later prompt would use (repeatable, %d to %d)", memnote.MinRecall, memnote.MaxRecall))
	add.Flags().StringVar(&by, "by", "", "the writer, in the Assisted-by form <Vendor>:<model-version>")
	add.Flags().StringVar(&session, "session", "", "the host's id for this session; without it the session is recorded as unknown")
	add.Flags().BoolVar(&everywhere, "everywhere", false, "recall the note in every repository, not only this one")
	add.Flags().BoolVar(&movedFromHost, "moved-from-host", false, "stamp a lesson the person pointed at in the host's own memory files")
	// The three the Use line names are refused by the core when missing (an
	// empty title, no recall word, a writer not in the trailer form), never
	// marked required on the flag set: the tree marks no flag required
	// (TestLiveTreeMarksNoFlagRequired).
	cmd.AddCommand(add)

	cmd.AddCommand(&cobra.Command{
		Use:  "list",
		Args: cobra.NoArgs,
		Long: "List every note in both layers: its id, title, body, recall words, who wrote\n" +
			"it, when and in which session, its origin, and its reach (this repository,\n" +
			"everywhere, another repository, or the project's file). Read-only.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			home, repoRoot, err := memoryWhere(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			l, root, unreadable := readMemoryLayers(home, repoRoot)
			out := memoryListOutput{Notes: []memoryNoteView{}, Unreadable: unreadable}
			for _, n := range l.Machine {
				out.Notes = append(out.Notes, noteView("machine", reachOf(n, root), n))
			}
			for _, n := range l.Project {
				out.Notes = append(out.Notes, noteView("project", reachProject, n))
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				if len(out.Notes) == 0 {
					fmt.Fprintln(w, "abcd memory list — no notes")
				}
				for _, v := range out.Notes {
					renderNoteView(w, v)
				}
				for _, u := range out.Unreadable {
					fmt.Fprintf(w, "unreadable: %s — %s\n", u.File, u.Reason)
				}
			})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:  "remove <mem-id>",
		Args: cobra.ExactArgs(1),
		Long: "Remove one personal note from the machine layer, whoever wrote it, run by a\n" +
			"person or by an agent at a person's request. A note in the project's file is\n" +
			"removed by editing that file in a reviewed change, and is refused here.\n\n" +
			"Exit 2 on a refusal, with nothing written.",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, repoRoot, err := memoryWhere(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			id := args[0]
			if _, err := memnote.Remove(home, id); err != nil {
				if errors.Is(err, memnote.ErrNotFound) {
					if f, _, rerr := memnote.ReadRepo(repoRoot); rerr == nil {
						for _, n := range f.Notes {
							if n.ID == id {
								return &exitError{Code: 2, Msg: "abcd memory remove: " + id + " is in the project's file " + memnote.RepoRelPath +
									"; a note there is removed by editing that file in a reviewed change, never by this verb (nothing written)"}
							}
						}
					}
				}
				return memoryRefusal("remove", err)
			}
			out := memoryRemoveOutput{Removed: id, File: memnote.MachineDisplayPath}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				fmt.Fprintf(w, "%s removed from %s\n", out.Removed, out.File)
			})
		},
	})
	return cmd
}

// noteView is a note as the list shows it.
func noteView(layer, reach string, n memnote.Note) memoryNoteView {
	return memoryNoteView{Layer: layer, Reach: reach, ID: n.ID, Title: n.Title, Body: n.Body, Recall: n.Recall,
		Origin: n.Origin, WrittenBy: n.WrittenBy, WrittenAt: n.WrittenAt, Session: n.Session}
}

// renderNoteView writes one note for a person to review it by.
func renderNoteView(w io.Writer, v memoryNoteView) {
	session := "session unknown"
	if v.Session != nil {
		session = "session " + *v.Session
	}
	fmt.Fprintf(w, "%s  %s\n", v.ID, termsafe.Sanitize(v.Title))
	for _, line := range strings.Split(v.Body, "\n") {
		fmt.Fprintf(w, "    %s\n", termsafe.Sanitize(line))
	}
	fmt.Fprintf(w, "    recall: %s\n", termsafe.Sanitize(strings.Join(v.Recall, ", ")))
	fmt.Fprintf(w, "    reach: %s\n", v.Reach)
	fmt.Fprintf(w, "    written by %s at %s, %s; origin %s\n", v.WrittenBy, v.WrittenAt, termsafe.Sanitize(session), v.Origin)
}
