package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/checklist"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/spf13/cobra"
)

// checklist.go — the `abcd checklist` verbs, the front door onto
// internal/core/checklist (itd-2610170638184486, spc-2610230805429429 step 2).
// Every write goes through checklist.Update, under the checklist's lock, and
// so through checklist.Write, whose schema and secret refusals
// reach the caller as they are: the field and the rule or pattern kind, never
// the value, at exit 2.
//
// The spec names a `show` verb for the whole list; the naming rule forbids a
// sub-verb that renames the bare state (brief 02-constraints/04-naming.md), so
// bare `abcd checklist` is the list and `show` takes the one id it shows.

// hostSessionIDEnv is where the host exports the session id of the session
// running a command. updated_by records it; a host that exports none, or a
// value not shaped like a session id, falls back to the away record's writer
// name (checklistWriter).
const hostSessionIDEnv = "CLAUDE_CODE_SESSION_ID"

// checklistStoreName names the store in a checkout-root refusal.
const checklistStoreName = "checklist"

// checklistIDCap caps the slug part of a minted id, short enough to type and
// leaving room for a -<n> suffix within the schema's id cap.
const checklistIDCap = 32

// checklistItemOutput is the --json envelope of add, set and show: the item's
// id and the item. Notice says the write started from something other than a
// clean checklist.json, and is absent otherwise.
type checklistItemOutput struct {
	ID     string         `json:"id"`
	Item   checklist.Item `json:"item"`
	Notice string         `json:"notice,omitempty"`
}

// checklistListOutput is the --json envelope of bare `abcd checklist`: the
// file the reader used, why, and the checklist, absent when there is none.
type checklistListOutput struct {
	Source    checklist.Source     `json:"source"`
	Reason    string               `json:"reason"`
	Checklist *checklist.Checklist `json:"checklist,omitempty"`
}

// checklistRenderOutput is the --json envelope of `abcd checklist render`.
type checklistRenderOutput struct {
	Source checklist.Source `json:"source"`
	Text   string           `json:"text"`
}

// newChecklistCommand builds `abcd checklist` and its sub-verbs.
func newChecklistCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "checklist",
		Long: "The working copy's checklist: what the session is working on, what it waits on\n" +
			"and what it has promised, kept in .abcd/.work.local/checklist.json so it\n" +
			"survives compaction and crashes. One file per working copy.\n\n" +
			"Bare `abcd checklist` lists every item with every field and writes nothing.\n" +
			"The reader validates and scans the file; one that fails is passed over for the\n" +
			"last copy a checked write produced, and the first line says why. `add`, `set`\n" +
			"and the bare list are the session's verbs; edit the file by hand and the reader\n" +
			"refuses what it cannot vouch for.\n\n" +
			"Every write validates the whole checklist against its schema and scans every\n" +
			"field for secrets. A refusal names the field and the rule or the pattern kind,\n" +
			"never the value, exits 2 and writes nothing. updated_by is the host's session\n" +
			"id where the host exports one, otherwise the name the away record would give\n" +
			"the writer. No verb makes a network request.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := checklistRoot("")
			if err != nil {
				return err
			}
			res := checklist.Read(root)
			out := checklistListOutput{Source: res.Source, Reason: res.Reason}
			if res.Source != checklist.FromNothing {
				out.Checklist = &res.Checklist
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				writeChecklistList(w, res)
			})
		},
	}
	cmd.AddCommand(newChecklistAddCommand(asJSON), newChecklistSetCommand(asJSON),
		newChecklistShowCommand(asJSON), newChecklistRenderCommand(asJSON))
	return cmd
}

func newChecklistAddCommand(asJSON *bool) *cobra.Command {
	var waitsOn, nextLook, promise string
	cmd := &cobra.Command{
		Use: `add "<title>"`,
		Long: "Add one item under a new id: a short slug of the title, with -2, -3 and so on\n" +
			"when the slug is taken. The id never changes after; `set` and `show` take it.\n\n" +
			"The item is working, or waiting when --waits-on names what it waits on: a record\n" +
			"id, pr:<number>, person:<role> or slot:<kind>. --next-look is when to look again,\n" +
			"an RFC 3339 timestamp. --promise is \"<trigger> => <act>\", split at the first =>.\n" +
			"The title is one line of at most 200 characters.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			it := checklist.Item{Title: args[0], State: checklist.Working, WaitsOn: waitsOn, NextLook: nextLook}
			if waitsOn != "" {
				it.State = checklist.Waiting
			}
			if promise != "" {
				trigger, act, ok := strings.Cut(promise, "=>")
				if !ok {
					return &exitError{Code: 2, Msg: `abcd checklist add: --promise must be "<trigger> => <act>" (nothing written)`}
				}
				it.Promise = &checklist.Promise{Trigger: strings.TrimSpace(trigger), Act: strings.TrimSpace(act)}
			}
			return checklistWrite(cmd, *asJSON, "add", func(c *checklist.Checklist) (int, error) {
				it.ID = mintChecklistID(c.Items, it.Title)
				c.Items = append(c.Items, it)
				return len(c.Items) - 1, nil
			})
		},
	}
	cmd.Flags().StringVar(&waitsOn, "waits-on", "", "what the item waits on: a record id, pr:<number>, person:<role> or slot:<kind>")
	cmd.Flags().StringVar(&nextLook, "next-look", "", "when to look at the item again, an RFC 3339 timestamp")
	cmd.Flags().StringVar(&promise, "promise", "", `a trigger and the act it promises, as "<trigger> => <act>"`)
	return cmd
}

func newChecklistSetCommand(asJSON *bool) *cobra.Command {
	var state string
	cmd := &cobra.Command{
		Use: "set <id> --state <state>",
		Long: "Set one item's state: " + checklistStateWords() + ". The id is the one `add`\n" +
			"minted, and it stays the same. An unknown id or state is refused (exit 2).",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if state == "" {
				return &exitError{Code: 2, Msg: "abcd checklist set: --state is required: one of " + checklistStateWords() + " (nothing written)"}
			}
			return checklistWrite(cmd, *asJSON, "set", func(c *checklist.Checklist) (int, error) {
				i := checklistIndex(c.Items, args[0])
				if i < 0 {
					return 0, errChecklistNoItem
				}
				c.Items[i].State = checklist.State(state)
				return i, nil
			})
		},
	}
	cmd.Flags().StringVar(&state, "state", "", "the item's new state: "+checklistStateWords())
	return cmd
}

func newChecklistShowCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "show <id>",
		Long: "Show one item, every field, by the id `add` minted. Bare `abcd checklist`\n" +
			"lists them all. Writes nothing.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := checklistRoot("show")
			if err != nil {
				return err
			}
			res := checklist.Read(root)
			i := checklistIndex(res.Checklist.Items, args[0])
			if i < 0 {
				return checklistRefusal("show", errChecklistNoItem)
			}
			out := checklistItemOutput{ID: res.Checklist.Items[i].ID, Item: res.Checklist.Items[i]}
			if res.Source != checklist.FromFile {
				out.Notice = res.Reason
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				if out.Notice != "" {
					fmt.Fprintln(w, out.Notice)
				}
				writeChecklistItem(w, out.Item)
			})
		},
	}
}

func newChecklistRenderCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "render",
		Long: "Print the checklist as the text a session's context is given: the open items\n" +
			"first, then the done and dropped ones, each on one line. A checklist that fails\n" +
			"the reader's checks is passed over for the last good copy, with one line saying\n" +
			"why. Writes nothing.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := checklistRoot("render")
			if err != nil {
				return err
			}
			res := checklist.Read(root)
			text := renderChecklist(res)
			return render(cmd.OutOrStdout(), *asJSON, checklistRenderOutput{Source: res.Source, Text: text}, func(w io.Writer) {
				io.WriteString(w, text)
			})
		},
	}
}

// errChecklistNoItem is the refusal for an id the checklist does not hold. It
// does not repeat the id: an operand is as able to carry a pasted secret as a
// field is.
var errChecklistNoItem = errors.New("no item has that id; `abcd checklist` lists them")

// checklistRoot resolves the checkout root the store lives under, refusing at
// exit 2 outside any checkout.
func checklistRoot(verb string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := gitutil.CheckoutRoot(cwd, checklistStoreName)
	if err != nil {
		return "", checklistRefusal(verb, err)
	}
	return root, nil
}

// checklistWrite is add's and set's shared path: read the checklist, apply
// change, stamp it and write it, all under the lock checklist.Update holds,
// then print the item change answers the index of.
//
// A write starts from what the reader returns. When checklist.json fails its
// checks the reader returns the last good copy, and the write builds on that;
// when both fail the write starts a new list. Either way the notice says so,
// on stderr in text form and as a field with --json.
func checklistWrite(cmd *cobra.Command, asJSON bool, verb string, change func(*checklist.Checklist) (int, error)) error {
	root, err := checklistRoot(verb)
	if err != nil {
		return err
	}
	// The whole cycle runs under the checklist's lock (checklist.Update), so a
	// concurrent writer's item is never lost to this one's write.
	var (
		c      checklist.Checklist
		i      int
		notice string
	)
	err = checklist.Update(root, func(res checklist.Result) (checklist.Checklist, error) {
		c, notice = res.Checklist, ""
		switch {
		case res.Source == checklist.FromLastGood:
			notice = res.Reason + " The write builds on that copy."
		case res.Source == checklist.FromNothing && checklistFilesExist(root):
			notice = res.Reason + " The write starts a new list."
		}
		if res.Source == checklist.FromNothing {
			c = checklist.Checklist{}
		}
		var cerr error
		if i, cerr = change(&c); cerr != nil {
			return c, cerr
		}
		c.Schema = checklist.Schema
		c.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		c.UpdatedBy = checklistWriter(root)
		return c, nil
	})
	if err != nil {
		return checklistRefusal(verb, err)
	}
	out := checklistItemOutput{ID: c.Items[i].ID, Item: c.Items[i], Notice: notice}
	if !asJSON && notice != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "abcd checklist "+verb+": "+notice)
	}
	return render(cmd.OutOrStdout(), asJSON, out, func(w io.Writer) {
		fmt.Fprintf(w, "%s %s\n", out.ID, out.Item.State)
	})
}

// checklistFilesExist reports whether either checklist file is present, so a
// reader's "nothing" over a file that failed is told from one never written.
func checklistFilesExist(root string) bool {
	for _, rel := range []string{checklist.FileRelPath, checklist.LastGoodRelPath} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
			return true
		}
	}
	return false
}

// checklistRefusal maps the store's refusals to exit 2 with nothing written:
// a schema violation, a secret, no local tier, a scanner that cannot vouch
// for the text, no checkout, an unknown id, and a lock another writer held past the wait. The store's wording names the
// field and the rule or pattern kind and never the value. Anything else is an
// I/O failure at exit 1.
func checklistRefusal(verb string, err error) error {
	prefix := "abcd checklist"
	if verb != "" {
		prefix += " " + verb
	}
	var fe *checklist.FieldError
	var se *checklist.SecretError
	switch {
	case errors.As(err, &fe), errors.As(err, &se),
		errors.Is(err, checklist.ErrNoLocalTier), errors.Is(err, checklist.ErrScannerUnavailable),
		errors.Is(err, gitutil.ErrNoCheckoutRoot), errors.Is(err, errChecklistNoItem),
		errors.Is(err, checklist.ErrLockHeld):
		msg := strings.TrimPrefix(err.Error(), "checklist: ")
		if !strings.Contains(msg, "nothing was written") {
			msg += " (nothing written)"
		}
		return &exitError{Code: 2, Msg: prefix + ": " + msg}
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

// checklistWriter is updated_by: the host's session id where the host exports
// one shaped like a session id, otherwise the away record's writer name (the
// holder name reservations resolve without a network read, else git's
// user.name), otherwise "unknown".
func checklistWriter(root string) string {
	if id := os.Getenv(hostSessionIDEnv); autonomy.ValidSessionID(id) {
		return id
	}
	if h := claim.ReaderHolder(root, claim.Options{}); h != "" {
		return h
	}
	if id, err := identity.EffectiveIdentity(root); err == nil && strings.TrimSpace(id.Name) != "" {
		return id.Name
	}
	return "unknown"
}

// mintChecklistID is a new item's id: the title's slug, or "item" for a title
// with nothing slug-able, suffixed -2, -3 and so on past the ids already held.
func mintChecklistID(items []checklist.Item, title string) string {
	base := recordid.Slug(title, checklistIDCap)
	if base == "" {
		base = "item"
	}
	id := base
	for n := 2; checklistIndex(items, id) >= 0; n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}

func checklistIndex(items []checklist.Item, id string) int {
	for i, it := range items {
		if it.ID == id {
			return i
		}
	}
	return -1
}

func checklistStateWords() string {
	words := make([]string, 0, len(checklist.States()))
	for _, s := range checklist.States() {
		words = append(words, string(s))
	}
	return strings.Join(words, ", ")
}

// checklistOpen reports whether an item is still to be acted on.
func checklistOpen(s checklist.State) bool { return s == checklist.Working || s == checklist.Waiting }

// writeChecklistList is the bare list: the reader's reason when it did not
// read the file clean, then every item with every field. Every field the
// reader passed is one line with no control character, so it prints as it is.
func writeChecklistList(w io.Writer, res checklist.Result) {
	if res.Source != checklist.FromFile {
		fmt.Fprintln(w, res.Reason)
	}
	if res.Source == checklist.FromNothing {
		return
	}
	c := res.Checklist
	fmt.Fprintf(w, "Checklist: %d items, updated %s by %s\n", len(c.Items), c.UpdatedAt, c.UpdatedBy)
	for _, it := range c.Items {
		writeChecklistItem(w, it)
	}
}

// writeChecklistItem prints one item: its id, state and title, then one
// indented line per optional field it carries.
func writeChecklistItem(w io.Writer, it checklist.Item) {
	fmt.Fprintf(w, "- %s [%s] %s\n", it.ID, it.State, it.Title)
	if it.WaitsOn != "" {
		fmt.Fprintf(w, "    waits on: %s\n", it.WaitsOn)
	}
	if it.NextLook != "" {
		fmt.Fprintf(w, "    next look: %s\n", it.NextLook)
	}
	if it.Promise != nil {
		fmt.Fprintf(w, "    promise: %s => %s\n", it.Promise.Trigger, it.Promise.Act)
	}
	if len(it.OwnerRefs) > 0 {
		fmt.Fprintf(w, "    about: %s\n", strings.Join(it.OwnerRefs, ", "))
	}
	if cr := it.Credential; cr != nil {
		scopes := make([]string, len(cr.Scope))
		for i, s := range cr.Scope {
			scopes[i] = string(s)
		}
		fmt.Fprintf(w, "    credential: %s, scope %s, expiry %s\n", cr.Kind, strings.Join(scopes, ", "), cr.Expiry)
	}
}

// renderChecklist is the injection text, kept minimal: the reader's reason
// when it did not read the file clean, then the open items, then the closed
// ones, one line each. It has no budget; step 3 of the spec moves the render
// into the core with the hook's character budget, newest first and a count of
// what it left out.
func renderChecklist(res checklist.Result) string {
	var b strings.Builder
	if res.Source != checklist.FromFile {
		fmt.Fprintln(&b, res.Reason)
	}
	if res.Source == checklist.FromNothing {
		return b.String()
	}
	var open, closed []checklist.Item
	for _, it := range res.Checklist.Items {
		if checklistOpen(it.State) {
			open = append(open, it)
		} else {
			closed = append(closed, it)
		}
	}
	fmt.Fprintln(&b, "This working copy's checklist (`abcd checklist` lists every field):")
	section := func(title string, items []checklist.Item) {
		if len(items) == 0 {
			return
		}
		fmt.Fprintln(&b, title)
		for _, it := range items {
			line := fmt.Sprintf("- [%s] %s: %s", it.State, it.ID, it.Title)
			if it.WaitsOn != "" {
				line += " (waits on " + it.WaitsOn + ")"
			}
			fmt.Fprintln(&b, line)
		}
	}
	section("Open:", open)
	section("Done or dropped:", closed)
	if len(res.Checklist.Items) == 0 {
		fmt.Fprintln(&b, "No items.")
	}
	return b.String()
}
