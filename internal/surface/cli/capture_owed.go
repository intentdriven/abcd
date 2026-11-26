package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/findings"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/refusal"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// capture_owed.go — `abcd capture owed` (step 1 of spc-2611252115253843,
// itd-2611251831138387): the list of findings abcd keeps under its own home,
// keyed on the repository's root commit. Bare, it lists what is owed and
// writes nothing; an adding flag puts one entry from standard input; handles
// and one clearing flag settle them, as the kind table allows. It writes only
// under abcd's home, so it never refuses a main checkout two sessions share.

const (
	// owedShown is how many open entries the bare list shows before it
	// counts the rest.
	owedShown = 100
	// owedSettledShown is how many settled entries --settled lists.
	owedSettledShown = 50
	// owedReadBudget is the command's own budget of files, past the hook's,
	// so it can still list the handles of a list too large for a hook.
	owedReadBudget = 20000
	// owedTextBytes bounds the text read from standard input; the store cuts
	// what it keeps to its own bound.
	owedTextBytes = 64 << 10
	// owedSettledListing is --settled given bare: list the settled entries,
	// or, beside handles, take the record as the one word that is not a
	// handle.
	owedSettledListing = "list"
	// owedClearing is the clearing act the list and a put name.
	owedClearing = "abcd capture owed <handle> --captured <iss-N> | --settled=<record> | --refuted \"<test>\" | --own \"<run>\" | --not-a-finding \"<reason>\" | --private"
)

// owedTimeout bounds one call: the git questions the records ask and the
// wait for the store's lock.
var owedTimeout = 60 * time.Second

// owedEntry is one entry as the command shows it. A private entry's text is
// blank everywhere but the person's terminal, and never in --json.
type owedEntry struct {
	Handle    string               `json:"handle"`
	Kind      findings.Kind        `json:"kind"`
	Checkout  string               `json:"checkout,omitempty"`
	Session   string               `json:"session,omitempty"`
	Agent     string               `json:"agent,omitempty"`
	AgentType string               `json:"agent_type,omitempty"`
	Run       string               `json:"run,omitempty"`
	Lane      string               `json:"lane,omitempty"`
	Text      string               `json:"text,omitempty"`
	Withheld  string               `json:"text_withheld,omitempty"`
	At        time.Time            `json:"at"`
	Settled   *findings.Settlement `json:"settled,omitempty"`
}

func owedEntryOf(e findings.Entry) owedEntry {
	out := owedEntry{
		Handle: e.Handle(), Kind: e.Kind, Checkout: fsutil.DisplayPath(e.Checkout),
		Session: e.Session, Agent: e.Agent, AgentType: e.AgentType, Run: e.Run, Lane: e.Lane,
		Text: e.Text, Withheld: e.WithheldLine(), At: e.At, Settled: e.Settled,
	}
	if out.Checkout == "." {
		out.Checkout = e.Checkout
	}
	return out
}

func owedEntries(es []findings.Entry) []owedEntry {
	out := make([]owedEntry, 0, len(es))
	for _, e := range es {
		out = append(out, owedEntryOf(e))
	}
	return out
}

// owedListOutput is the bare list in --json: the open entries, a hundred at
// most, the count of the rest, and the private list's count alone.
type owedListOutput struct {
	Open         []owedEntry           `json:"open"`
	OpenCount    int                   `json:"open_count"`
	More         int                   `json:"more"`
	PrivateCount int                   `json:"private_count"`
	Newer        []string              `json:"newer"`
	Unreadable   []findings.Unreadable `json:"unreadable"`
	Full         bool                  `json:"full"`
}

// owedSettledOutput is --settled in --json.
type owedSettledOutput struct {
	Settled    []owedEntry           `json:"settled"`
	Unreadable []findings.Unreadable `json:"unreadable"`
}

// owedPutOutput is a put in --json.
type owedPutOutput struct {
	Put        []owedEntry           `json:"put"`
	Unreadable []findings.Unreadable `json:"unreadable"`
}

// owedSettleOutput is a settle in --json.
type owedSettleOutput struct {
	Act     findings.Act `json:"act"`
	Settled []owedEntry  `json:"settled"`
}

// owedAtTerminal says whether the caller is the person at a terminal: the
// input and the output are a terminal, no session marker is set and the
// process is in no stage. Only then is a private text printed, or a private
// entry settled.
func owedAtTerminal(cmd *cobra.Command) bool {
	if !connectIsTerminal(cmd.InOrStdin()) || !connectIsTerminal(cmd.OutOrStdout()) {
		return false
	}
	for _, k := range driveSessionMarkers {
		if os.Getenv(k) != "" {
			return false
		}
	}
	return os.Getenv(runner.StageWritesEnv) == "" && os.Getenv(runner.StageHomeEnv) == ""
}

// owedStoreFor is the list of the repository root is a checkout of, or
// ok false where it has no commit yet and so no key.
func owedStoreFor(ctx context.Context, root string) (findings.Store, bool, error) {
	sha, err := gitutil.RootCommitContext(ctx, root)
	if err != nil {
		return findings.Store{}, false, err
	}
	if sha == "" {
		return findings.Store{}, false, nil
	}
	home, err := abcdhome.MachineHome()
	if err != nil {
		return findings.Store{}, false, err
	}
	s, err := findings.NewStore(home, sha)
	return s, err == nil, err
}

// recordingAnswerer keeps the last answer the records gave, so a refused
// claim on an issue no ledger holds is told apart as capture/unknown-issue.
type recordingAnswerer struct {
	inner *findings.Records
	last  findings.Answer
}

func (a *recordingAnswerer) Answer(act findings.Act, id string) (findings.Answer, error) {
	ans, err := a.inner.Answer(act, id)
	a.last = ans
	return ans, err
}

// owedStoreRefusal is the store's refusal as the command's: a request it cannot
// answer is capture/invalid-input, and anything else a fault, with the home
// written as ~.
func owedStoreRefusal(err error) error {
	msg := "abcd capture owed: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))
	for _, e := range []error{findings.ErrNoEntry, findings.ErrAmbiguous, findings.ErrNotCleared,
		findings.ErrNewer, findings.ErrUnknownKind, findings.ErrFull, findings.ErrTooLarge} {
		if errors.Is(err, e) {
			return refused(2, refusal.CaptureInvalidInput, msg+" (nothing written)")
		}
	}
	return exitWith(1, refusal.None, msg)
}

// newCaptureOwedCommand builds `abcd capture owed`.
func newCaptureOwedCommand(asJSON *bool) *cobra.Command {
	var captured, settled, refuted, own, notAFinding string
	var untested, doubt, handedUp, private bool
	cmd := &cobra.Command{
		Use: "owed [<handle>...] [--captured <iss-N> | --settled <record> | --refuted <test> | --own <run> | --not-a-finding <reason> | --private] | owed --untested|--doubt|--handed-up|--private - | owed --settled",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := captureLedgerRoot(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(owedContext(cmd), owedTimeout)
			defer cancel()

			adding := map[string]findings.Kind{}
			for flag, on := range map[string]bool{"untested": untested, "doubt": doubt, "handed-up": handedUp} {
				if on {
					adding[flag] = findings.Kind(flag)
				}
			}
			handles, rest := owedHandles(args)
			// --private settles beside handles and puts beside "-".
			if private && len(handles) == 0 {
				adding["private"] = findings.KindPrivate
			}
			acts := map[string]bool{}
			for flag, v := range map[string]string{"captured": captured, "refuted": refuted, "own": own, "not-a-finding": notAFinding} {
				if v != "" || cmd.Flags().Changed(flag) {
					acts[flag] = true
				}
			}
			if private && len(handles) > 0 {
				acts["private"] = true
			}
			listingSettled := settled == owedSettledListing && len(handles) == 0
			if cmd.Flags().Changed("settled") && !listingSettled {
				acts["settled"] = true
			}
			named := append(owedKeys(adding), owedKeys(acts)...)
			if listingSettled {
				named = append(named, "settled")
			}
			sort.Strings(named)
			if len(named) > 1 {
				return refused(2, refusal.UsageFlagConflict, fmt.Sprintf(
					"abcd capture owed: --%s each choose what the call does; give one (nothing written)", strings.Join(named, ", --")))
			}

			switch {
			case len(adding) == 1:
				if len(args) != 1 || args[0] != "-" {
					return refused(2, refusal.UsageMissingArgument, fmt.Sprintf(
						"abcd capture owed: --%s reads its one line from standard input; give - as the one argument (nothing written)", named[0]))
				}
				return runOwedPut(ctx, cmd, *asJSON, root, adding[named[0]])
			case len(acts) == 1:
				if len(handles) == 0 {
					return refused(2, refusal.UsageMissingArgument, fmt.Sprintf(
						"abcd capture owed: --%s settles the entries its handles name; give at least one handle (f- and eight characters) (nothing written)", named[0]))
				}
				record := captured
				if acts["settled"] {
					record = settled
					if record == owedSettledListing {
						if len(rest) != 1 {
							return refused(2, refusal.UsageMissingArgument,
								"abcd capture owed: --settled names the record that settles the entries (an open issue, a wontfix, an ADR or a decision): --settled <record> (nothing written)")
						}
						record, rest = rest[0], nil
					}
				}
				if len(rest) > 0 {
					return refused(2, refusal.CaptureInvalidInput, fmt.Sprintf(
						"abcd capture owed: %q is not a handle (f- and eight characters) (nothing written)", termsafe.Sanitize(rest[0])))
				}
				reason := map[string]string{"refuted": refuted, "own": own, "not-a-finding": notAFinding}[named[0]]
				return runOwedSettle(ctx, cmd, *asJSON, root, handles, findings.Act(named[0]), record, reason)
			case listingSettled:
				if len(args) > 0 {
					return refused(2, refusal.CaptureInvalidInput, "abcd capture owed --settled: the listing takes no argument; to settle, give the handles and --settled <record> (nothing written)")
				}
				return runOwedListSettled(cmd, *asJSON, root)
			case len(args) > 0:
				return refused(2, refusal.UsageMissingRequiredFlag,
					"abcd capture owed: a handle is settled by one clearing flag: "+owedClearing+" (nothing written)")
			}
			return runOwedList(cmd, *asJSON, root)
		},
	}
	cmd.Flags().BoolVar(&untested, "untested", false, "put one finding not yet tested, its line read from standard input (-)")
	cmd.Flags().BoolVar(&doubt, "doubt", false, "put one finding in doubt on which side of the line it falls, read from standard input (-); cleared only by a record")
	cmd.Flags().BoolVar(&handedUp, "handed-up", false, "put one confirmed finding handed up by an agent that may not write, read from standard input (-)")
	cmd.Flags().BoolVar(&private, "private", false, "with -, put one finding kept on the private list, read from standard input; beside handles, move them there")
	cmd.Flags().StringVar(&captured, "captured", "", "settle the handles by the issue that captures them, fixed or not")
	cmd.Flags().StringVar(&settled, "settled", "", "bare, list the last fifty settled entries; beside handles, settle them by a record (an open issue, a wontfix, an ADR or a decision)")
	cmd.Flags().Lookup("settled").NoOptDefVal = owedSettledListing
	cmd.Flags().StringVar(&refuted, "refuted", "", "settle the handles as refuted, naming the test that did it")
	cmd.Flags().StringVar(&own, "own", "", "settle failing checks as the change's own, naming the run on the base")
	cmd.Flags().StringVar(&notAFinding, "not-a-finding", "", "settle lines the words put on the list, with one line of reason, which is kept")
	return cmd
}

// owedContext is cmd's context, or the background where it has none.
func owedContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

// owedHandles parts the positionals into handles and the rest.
func owedHandles(args []string) (handles, rest []string) {
	for _, a := range args {
		if strings.HasPrefix(a, "f-") {
			handles = append(handles, a)
		} else if a != "-" {
			rest = append(rest, a)
		}
	}
	return handles, rest
}

func owedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// runOwedPut puts one entry of kind from standard input.
func runOwedPut(ctx context.Context, cmd *cobra.Command, asJSON bool, root string, kind findings.Kind) error {
	raw, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), owedTextBytes+1))
	if err != nil {
		return exitWith(1, refusal.None, "abcd capture owed: reading standard input: "+termsafe.Sanitize(err.Error()))
	}
	if len(raw) > owedTextBytes {
		return refused(2, refusal.CaptureInvalidInput, fmt.Sprintf("abcd capture owed: the text is past %d KiB; give the finding in one line (nothing written)", owedTextBytes>>10))
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return refused(2, refusal.UsageMissingArgument, "abcd capture owed: standard input held no text; give the finding in one line (nothing written)")
	}
	s, ok, err := owedStoreFor(ctx, root)
	if err != nil {
		return owedStoreRefusal(err)
	}
	if !ok {
		return refused(2, refusal.CaptureInvalidInput, "abcd capture owed: the repository has no commit yet, so its list has no key; commit first (nothing written)")
	}
	session := os.Getenv(hostSessionIDEnv)
	if !autonomy.ValidSessionID(session) {
		session = ""
	}
	put, bad, err := s.Put(ctx, findings.Entry{Kind: kind, Session: session, Checkout: root, Text: text, At: time.Now().UTC()})
	if err != nil {
		return owedStoreRefusal(err)
	}
	out := owedPutOutput{Put: owedEntries(put), Unreadable: bad}
	return render(cmd.OutOrStdout(), asJSON, out, func(w io.Writer) {
		for _, e := range put {
			switch {
			case e.Settled != nil:
				fmt.Fprintf(w, "%s (%s) is already settled (%s); it stays settled\n", e.Handle(), e.Kind, e.Settled.Act)
			case e.Kind == findings.KindPrivate:
				fmt.Fprintf(w, "put %s on the private list; its text is shown only in the person's terminal\n", e.Handle())
			default:
				fmt.Fprintf(w, "put %s (%s)\n", e.Handle(), e.Kind)
				if l := e.WithheldLine(); l != "" {
					fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(l))
				}
			}
		}
		for _, u := range bad {
			fmt.Fprintf(w, "  unreadable %s — %s\n", u.Handle, termsafe.Sanitize(fsutil.RedactHome(u.Reason)))
		}
		if len(put) > 0 && put[0].Settled == nil {
			fmt.Fprintf(w, "clear it with: %s\n", owedClearing)
		}
	})
}

// runOwedSettle settles handles with one act.
func runOwedSettle(ctx context.Context, cmd *cobra.Command, asJSON bool, root string, handles []string, act findings.Act, record, reason string) error {
	s, ok, err := owedStoreFor(ctx, root)
	if err != nil {
		return owedStoreRefusal(err)
	}
	if !ok {
		return refused(2, refusal.CaptureInvalidInput, "abcd capture owed: the repository has no commit yet, so its list holds nothing to settle (nothing written)")
	}
	req := findings.SettleRequest{Handles: handles, Act: act, Reason: reason, Person: owedAtTerminal(cmd), At: time.Now().UTC()}
	var answers *recordingAnswerer
	if act == findings.ActCaptured || act == findings.ActSettled {
		if !recordid.CitedIDRe.MatchString(record) {
			return refused(2, refusal.CaptureInvalidInput, fmt.Sprintf(
				"abcd capture owed: --%s names a record id (iss-N, adr-N or dcn-N); %q is not one (nothing written)", act, termsafe.Sanitize(record)))
		}
		answers = &recordingAnswerer{inner: findings.NewRecords(ctx, root)}
		req.Record, req.Records = record, answers
	}
	got, err := s.Settle(ctx, req)
	if err != nil {
		if answers != nil && answers.last.State == "absent" && strings.HasPrefix(record, "iss-") {
			return refused(2, refusal.CaptureUnknownIssue, fmt.Sprintf(
				"abcd capture owed: no ledger the default branch or this checkout holds has %s (nothing written)", record))
		}
		return owedStoreRefusal(err)
	}
	return render(cmd.OutOrStdout(), asJSON, owedSettleOutput{Act: act, Settled: owedEntries(got)}, func(w io.Writer) {
		for _, e := range got {
			if act == findings.ActPrivate {
				fmt.Fprintf(w, "moved %s to the private list\n", e.Handle())
				continue
			}
			fmt.Fprintf(w, "settled %s (%s) — %s\n", e.Handle(), e.Kind, owedSettlementText(e.Settled))
		}
	})
}

// owedSettlementText is a settlement in one line: its act and its record, or its
// reason, test or run.
func owedSettlementText(st *findings.Settlement) string {
	if st == nil {
		return ""
	}
	out := string(st.Act)
	switch {
	case st.Record != "":
		out += " " + st.Record
	case st.ReasonWithheld:
		out += ": its reason was withheld because no full scan could clear it"
	case st.Reason != "":
		out += ": " + owedLine(st.Reason)
	}
	return termsafe.Sanitize(out)
}

// owedLine is s with its whitespace runs as single spaces.
func owedLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// runOwedList is the bare list.
func runOwedList(cmd *cobra.Command, asJSON bool, root string) error {
	out := owedListOutput{Open: []owedEntry{}, Newer: []string{}, Unreadable: []findings.Unreadable{}}
	var view findings.View
	var s findings.Store
	s, ok, err := owedStoreFor(owedContext(cmd), root)
	if err != nil {
		return owedStoreRefusal(err)
	}
	if ok {
		if view, err = s.ReadWithin(owedReadBudget); err != nil {
			return owedStoreRefusal(err)
		}
	}
	ordered := view.Ordered(root)
	out.OpenCount, out.PrivateCount, out.Full = len(ordered), len(view.Private), view.Full
	if len(ordered) > owedShown {
		out.More = len(ordered) - owedShown
		ordered = ordered[:owedShown]
	}
	out.Open = owedEntries(ordered)
	out.Newer = append(out.Newer, view.Newer...)
	out.Unreadable = append(out.Unreadable, view.Unreadable...)
	person := !asJSON && owedAtTerminal(cmd)
	return render(cmd.OutOrStdout(), asJSON, out, func(w io.Writer) {
		head := fmt.Sprintf("abcd capture owed — %d open · %d private", out.OpenCount, out.PrivateCount)
		if len(out.Newer) > 0 {
			head += fmt.Sprintf(" · %d a newer abcd's", len(out.Newer))
		}
		if out.Full {
			head += " · the list is full"
		}
		fmt.Fprintln(w, head)
		if out.OpenCount == 0 && out.PrivateCount == 0 && len(out.Newer) == 0 && len(out.Unreadable) == 0 {
			fmt.Fprintln(w, "nothing owed")
			return
		}
		for _, e := range ordered {
			text := owedLine(e.Text)
			if l := e.WithheldLine(); l != "" {
				text = l
			}
			fmt.Fprintf(w, "  %s  %-9s  %s  %s\n", e.Handle(), e.Kind, termsafe.Sanitize(owedEntryOf(e).Checkout), termsafe.Sanitize(text))
		}
		if out.More > 0 {
			fmt.Fprintf(w, "  and %d more\n", out.More)
		}
		if out.PrivateCount > 0 {
			if person {
				fmt.Fprintln(w, "private (shown here only, at the person's terminal):")
				for _, e := range view.Private {
					text, withheld, err := s.PrivateText(e.Handle(), true)
					switch {
					case err != nil:
						text = "unreadable: " + fsutil.RedactHome(err.Error())
					case withheld:
						text = e.WithheldLine()
					}
					fmt.Fprintf(w, "  %s  %s\n", e.Handle(), termsafe.Sanitize(owedLine(text)))
				}
			} else {
				fmt.Fprintf(w, "  %d private: the text is shown only at the person's terminal, with no session marker and no stage\n", out.PrivateCount)
			}
		}
		for _, h := range out.Newer {
			fmt.Fprintf(w, "  %s is a newer abcd's: no command of this build settles it\n", h)
		}
		for _, u := range out.Unreadable {
			fmt.Fprintf(w, "  unreadable %s — %s\n", u.Handle, termsafe.Sanitize(fsutil.RedactHome(u.Reason)))
		}
		fmt.Fprintf(w, "clear one: %s\n", owedClearing)
	})
}

// runOwedListSettled is --settled: the last fifty settled entries, each with
// its act and the record, reason or run given.
func runOwedListSettled(cmd *cobra.Command, asJSON bool, root string) error {
	out := owedSettledOutput{Settled: []owedEntry{}, Unreadable: []findings.Unreadable{}}
	s, ok, err := owedStoreFor(owedContext(cmd), root)
	if err != nil {
		return owedStoreRefusal(err)
	}
	var got []findings.Entry
	if ok {
		var bad []findings.Unreadable
		if got, bad, err = s.SettledView(owedSettledShown); err != nil {
			return owedStoreRefusal(err)
		}
		out.Settled, out.Unreadable = owedEntries(got), append(out.Unreadable, bad...)
	}
	return render(cmd.OutOrStdout(), asJSON, out, func(w io.Writer) {
		fmt.Fprintf(w, "abcd capture owed --settled — the last %d settled\n", len(got))
		if len(got) == 0 {
			fmt.Fprintln(w, "nothing settled")
		}
		for _, e := range got {
			fmt.Fprintf(w, "  %s  %-9s  %s\n", e.Handle(), e.Kind, owedSettlementText(e.Settled))
		}
		for _, u := range out.Unreadable {
			fmt.Fprintf(w, "  unreadable %s — %s\n", u.Handle, termsafe.Sanitize(fsutil.RedactHome(u.Reason)))
		}
	})
}
