package cli

// source.go is the front door onto internal/core/source — the personal sources
// corpus and its provenance ledger (itd-76, spc-31).
//
// Exit codes: 0 done; 1 cite-check found a confidential source in the text; 2 a
// refusal (a bad operand, a failed gate, a corpus that disagrees with itself);
// 3 there is no corpus at the configured location — the distinct no-corpus code
// every verb but `init` returns, so a script can tell "nothing to check against"
// from "checked and clean". `sync-banlist --refresh` is the guard's mode: with no
// corpus it says so on one line and exits 0, because a commit on a machine that
// never made a corpus is not a failure.
//
// Nothing here prints a confidential title, alias or author. The core's refusals
// never carry one, and every report names sources by key.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/core/source"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// maxScanBytes caps a cite-check input (trust boundary).
const maxScanBytes = 64 << 20

// sourceCorpusDir resolves --corpus, or the default under the user-level home.
func sourceCorpusDir(flag string) (string, error) {
	if flag != "" {
		if !strings.HasPrefix(flag, "/") {
			return "", &exitError{Code: 2, Msg: "abcd source: --corpus must be an absolute path"}
		}
		return flag, nil
	}
	dir, err := source.DefaultDir()
	if err != nil {
		return "", &exitError{Code: 2, Msg: "abcd source: " + err.Error()}
	}
	return dir, nil
}

// sourceError maps a core error to the verb's exit code and one line.
func sourceError(verb, dir string, err error) error {
	if errors.Is(err, source.ErrNoCorpus) {
		return &exitError{Code: 3, Msg: fmt.Sprintf("abcd source %s: no sources corpus at %s — nothing read or written (create one with `abcd source init`)",
			verb, fsutil.RedactHome(dir))}
	}
	return &exitError{Code: 2, Msg: "abcd source " + verb + ": " + termsafe.Sanitize(scrubPaths(err))}
}

// newSourceCommand builds the `source` verb.
func newSourceCommand(asJSON *bool) *cobra.Command {
	var corpusFlag string
	cmd := &cobra.Command{
		Use:   "source",
		Short: "The personal sources corpus and its provenance ledger (bare renders its state, read-only)",
		Long: "The personal sources corpus: documents you may consult, a CSL-JSON bibliography, and one\n" +
			"append-only influence ledger per repository, in a local-only git repository with no\n" +
			"remote (~/.abcd/sources by default; --corpus names another). The folder a source sits\n" +
			"in — confidential/<key>/ or public/<key>/ — is its classification.\n\n" +
			"Consult freely, cite deliberately: confidential entries are projected into this\n" +
			"repository's untracked private banlist (sync-banlist), which the committed pre-commit\n" +
			"guard refreshes and enforces; cite-check clears text before it leaves the machine; and\n" +
			"a ledger line becomes a public citation only when the source permits it AND a person\n" +
			"flips the line (adr-41). No output names a confidential source except by key.\n\n" +
			"Bare `abcd source` is read-only. Exit 3 when there is no corpus, on every verb but init.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := sourceCorpusDir(corpusFlag)
			if err != nil {
				return err
			}
			st, err := source.Status(dir)
			if err != nil {
				return sourceError("status", dir, err)
			}
			if !st.Present {
				return sourceError("status", dir, source.ErrNoCorpus)
			}
			st.Dir = fsutil.RedactHome(st.Dir)
			return render(cmd.OutOrStdout(), *asJSON, st, func(w io.Writer) { renderSourceStatus(w, st) })
		},
	}
	cmd.PersistentFlags().StringVar(&corpusFlag, "corpus", "", "the corpus directory (absolute; default ~/.abcd/sources)")
	cmd.AddCommand(newSourceInitCommand(asJSON, &corpusFlag))
	cmd.AddCommand(newSourceAddCommand(asJSON, &corpusFlag))
	cmd.AddCommand(newSourceDeclassifyCommand(asJSON, &corpusFlag))
	cmd.AddCommand(newSourceLedgerCommand(asJSON, &corpusFlag))
	cmd.AddCommand(newSourceSyncBanlistCommand(asJSON, &corpusFlag))
	cmd.AddCommand(newSourceCiteCheckCommand(asJSON, &corpusFlag))
	return cmd
}

func renderSourceStatus(w io.Writer, st source.StatusReport) {
	fmt.Fprintf(w, "abcd source — corpus at %s\n", st.Dir)
	fmt.Fprintf(w, "  sources:  %d confidential, %d public\n", st.Confidential, st.Public)
	if len(st.Ledgers) == 0 {
		fmt.Fprintln(w, "  ledgers:  none")
	}
	for _, l := range st.Ledgers {
		fmt.Fprintf(w, "  ledger:   %s (%d line%s)\n", l.Repo, l.Lines, pluralS(l.Lines))
	}
	if len(st.Problems) == 0 {
		fmt.Fprintln(w, "  problems: none")
	}
	for _, p := range st.Problems {
		fmt.Fprintf(w, "  problem:  %s — %s\n", termsafe.Sanitize(p.Key), termsafe.Sanitize(p.Reason))
	}
	if st.Remotes > 0 {
		fmt.Fprintf(w, "  WARNING:  the corpus repository has %d remote(s); documents and ledgers never leave this machine (adr-41) — remove it\n", st.Remotes)
	}
}

func newSourceInitCommand(asJSON *bool, corpusFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create an empty corpus: a no-remote git repository with an empty bibliography",
		Long: "Create the corpus at its location (0700): a git repository with no remote, an empty\n" +
			"sources.json and a README, in one commit. Refuses an existing corpus, a non-empty\n" +
			"directory, and a location inside another repository's working tree.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := sourceCorpusDir(*corpusFlag)
			if err != nil {
				return err
			}
			st, err := source.Init(dir)
			if err != nil {
				return sourceError("init", dir, err)
			}
			st.Dir = fsutil.RedactHome(st.Dir)
			return render(cmd.OutOrStdout(), *asJSON, st, func(w io.Writer) {
				fmt.Fprintf(w, "abcd source init — corpus created at %s (a git repository with no remote)\n", st.Dir)
			})
		},
	}
}

// sourceMeta is --meta's JSON: the identifying fields, kept out of argv.
type sourceMeta struct {
	Title    string        `json:"title"`
	Aliases  []string      `json:"aliases"`
	Authors  []source.Name `json:"author"`
	Keywords []string      `json:"keywords"`
}

// parseAuthor reads "Family, Given" as a CSL name, anything else as a literal.
func parseAuthor(s string) source.Name {
	if fam, giv, ok := strings.Cut(s, ","); ok {
		return source.Name{Family: strings.TrimSpace(fam), Given: strings.TrimSpace(giv)}
	}
	return source.Name{Literal: strings.TrimSpace(s)}
}

func newSourceAddCommand(asJSON *bool, corpusFlag *string) *cobra.Command {
	var (
		key, title, typ, perm, venue, url, text, meta string
		authors, keywords, aliases                    []string
		year                                          int
		confidential, public, banAuthors              bool
	)
	cmd := &cobra.Command{
		Use:   "add [document]",
		Short: "Register a source: its entry, the document and its text under its class folder",
		Long: "Register a source: write its CSL-JSON entry (with the custom block), store the document\n" +
			"as original.<ext> and its extracted text as text.md under confidential/<key>/ or\n" +
			"public/<key>/, and commit the corpus. The class is declared here, once: exactly one\n" +
			"of --confidential or --public is required. abcd converts nothing and fetches nothing:\n" +
			"a Markdown or text document is its own text, any other needs --text, and a URL\n" +
			"alone registers a metadata stub.\n\n" +
			"A confidential entry's title, aliases and (under --ban-authors) authors become banned\n" +
			"phrases, so each must hold at least three letters or digits, and its key must not\n" +
			"contain any of them — the key is what every refusal and scan prints. Pass those\n" +
			"strings with --meta FILE (or --meta - on stdin) to keep them out of argv and shell\n" +
			"history.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := sourceCorpusDir(*corpusFlag)
			if err != nil {
				return err
			}
			class := ""
			switch {
			case confidential && public:
				return &exitError{Code: 2, Msg: "abcd source add: --confidential and --public are exclusive"}
			case confidential:
				class = source.ClassConfidential
			case public:
				class = source.ClassPublic
			}
			req := source.AddRequest{Corpus: dir, Key: key, Title: title, Type: typ, Class: class, Permission: perm,
				Year: year, Venue: venue, URL: url, Keywords: splitList(keywords), Aliases: aliases, BanAuthors: banAuthors, Text: text}
			for _, a := range authors {
				req.Authors = append(req.Authors, parseAuthor(a))
			}
			if len(args) == 1 {
				req.Original = args[0]
			}
			if meta != "" {
				m, err := readSourceMeta(cmd, meta)
				if err != nil {
					return &exitError{Code: 2, Msg: "abcd source add: " + err.Error()}
				}
				if m.Title != "" {
					req.Title = m.Title
				}
				req.Aliases = append(req.Aliases, m.Aliases...)
				req.Authors = append(req.Authors, m.Authors...)
				req.Keywords = append(req.Keywords, m.Keywords...)
			}
			res, err := source.Add(req)
			if err != nil {
				return sourceError("add", dir, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd source add — %s registered %s (%s), committed in the corpus\n", res.Key, res.Class, res.Permission)
				for _, f := range res.Files {
					fmt.Fprintf(w, "  %s\n", f)
				}
				if res.Class == source.ClassConfidential {
					fmt.Fprintln(w, "  next: abcd source sync-banlist, in every repository you work in")
				}
			})
		},
	}
	f := cmd.Flags()
	f.StringVar(&key, "key", "", "the source key: lowercase, opaque for a confidential source (e.g. conf2026a)")
	f.StringVar(&title, "title", "", "the exact title (for a confidential source prefer --meta)")
	f.StringVar(&typ, "type", "", "the CSL item type (default document)")
	f.BoolVar(&confidential, "confidential", false, "file the source under confidential/ (exclusive with --public)")
	f.BoolVar(&public, "public", false, "file the source under public/ (exclusive with --confidential)")
	f.StringVar(&perm, "permission", "", "permission_status: "+enumHelp(source.Permissions)+" (default by class)")
	f.StringArrayVar(&authors, "author", nil, `an author, "Family, Given" or a literal name (repeatable)`)
	f.IntVar(&year, "year", 0, "the year of issue")
	f.StringVar(&venue, "venue", "", "the container title (journal, site, publisher)")
	f.StringVar(&url, "url", "", "the canonical URL (recorded, never fetched)")
	f.StringArrayVar(&keywords, "keywords", nil, "retrieval keywords, comma-separated (repeatable)")
	f.StringArrayVar(&aliases, "alias", nil, "another identifying name for a confidential source (repeatable)")
	f.BoolVar(&banAuthors, "ban-authors", false, "also ban the authors' names (a confidential source whose authorship is itself identifying)")
	f.StringVar(&text, "text", "", "the extracted text of a non-text document")
	f.StringVar(&meta, "meta", "", `a JSON file (or - for stdin) with "title", "aliases", "author" and "keywords"`)
	return cmd
}

// splitList flattens comma-separated values.
func splitList(in []string) []string {
	var out []string
	for _, v := range in {
		for _, p := range strings.Split(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

// readSourceMeta reads --meta from a file or stdin.
func readSourceMeta(cmd *cobra.Command, from string) (sourceMeta, error) {
	var data []byte
	var err error
	if from == "-" {
		data, err = io.ReadAll(io.LimitReader(cmd.InOrStdin(), 1<<20))
	} else {
		data, err = fsutil.ReadGuarded(from, 1<<20)
	}
	if err != nil {
		return sourceMeta{}, errors.New("--meta cannot be read (absent, oversize, or not a regular file)")
	}
	var m sourceMeta
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return sourceMeta{}, errors.New(`--meta is not a JSON object of "title", "aliases", "author" and "keywords" (its content is withheld)`)
	}
	return m, nil
}

func newSourceDeclassifyCommand(asJSON *bool, corpusFlag *string) *cobra.Command {
	var perm string
	cmd := &cobra.Command{
		Use:   "declassify <key>",
		Short: "Move a published confidential source to public/ — a visible, committed move",
		Long: "Declassify a confidential source once it is published: `git mv` its folder from\n" +
			"confidential/ to public/ and set the entry's confidential flag and permission_status\n" +
			"(citable unless --permission says otherwise), in one corpus commit. The next\n" +
			"sync-banlist drops its strings, and its ledger lines become flippable.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := sourceCorpusDir(*corpusFlag)
			if err != nil {
				return err
			}
			res, err := source.Declassify(dir, args[0], perm)
			if err != nil {
				return sourceError("declassify", dir, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd source declassify — %s moved to %s (%s), committed in the corpus\n", res.Key, res.Folder, res.Permission)
				fmt.Fprintln(w, "  next: abcd source sync-banlist, in every repository you work in")
			})
		},
	}
	cmd.Flags().StringVar(&perm, "permission", "", "permission_status after the move: "+enumHelp(source.Permissions)+" (default citable)")
	return cmd
}

// ledgerRepo is the ledger's repository handle: --repo, or the first twelve hex
// digits of this checkout's root commit, which every worktree and clone of one
// repository shares whatever its directory is called.
func ledgerRepo(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := gitutil.CheckoutRoot(cwd, "the ledger's repository")
	if err != nil {
		return "", &exitError{Code: 2, Msg: "abcd source ledger: " + err.Error() + " (or name the ledger with --repo)"}
	}
	sha := gitutil.RootCommit(root)
	if !gitutil.IsFullSHA(sha) {
		return "", &exitError{Code: 2, Msg: "abcd source ledger: this checkout has no commit to name its ledger by; name it with --repo"}
	}
	return sha[:12], nil
}

func newSourceLedgerCommand(asJSON *bool, corpusFlag *string) *cobra.Command {
	var (
		repo, decision, claim, key, locator, influence string
		usedIn                                         []string
		corrects, flip                                 int
		list                                           bool
	)
	cmd := &cobra.Command{
		Use:   "ledger",
		Short: "Append an influence record to this repository's ledger; --flip cites one; --list reads it",
		Long: "Append one influence record — {ts, repo, decision_ref, claim, source_key, locator,\n" +
			"influence, cited_publicly: false} — to this repository's ledger in the corpus, and\n" +
			"commit it. The ledger is append-only: a correction is a new line (--corrects N).\n\n" +
			"--flip N is the person's act of citing line N publicly. It checks the source first\n" +
			"(adr-41 gate 1: the folder is public/ and permission_status is citable), refuses\n" +
			"naming the failing gate, and on success appends a NEW line with cited_publicly true.\n" +
			"An agent never runs it. --list prints the ledger, numbered.\n\n" +
			"The repository is named by its root commit's first twelve hex digits unless --repo\n" +
			"names it.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := sourceCorpusDir(*corpusFlag)
			if err != nil {
				return err
			}
			r, err := ledgerRepo(repo)
			if err != nil {
				return err
			}
			switch {
			case list:
				lines, err := source.List(dir, r)
				if err != nil {
					return sourceError("ledger", dir, err)
				}
				if lines == nil {
					lines = []source.AppendResult{}
				}
				return render(cmd.OutOrStdout(), *asJSON, lines, func(w io.Writer) {
					fmt.Fprintf(w, "abcd source ledger — %s, %d line%s\n", r, len(lines), pluralS(len(lines)))
					for _, l := range lines {
						mark := ""
						switch {
						case l.Record.Flips != 0:
							mark = fmt.Sprintf(" [cites line %d]", l.Record.Flips)
						case l.Record.Corrects != 0:
							mark = fmt.Sprintf(" [corrects line %d]", l.Record.Corrects)
						}
						fmt.Fprintf(w, "  %3d  %s  %s  %s — %s%s\n", l.Line, termsafe.Sanitize(l.Record.SourceKey), l.Record.Influence,
							termsafe.Sanitize(l.Record.DecisionRef), termsafe.Sanitize(l.Record.Claim), mark)
					}
				})
			case flip != 0:
				res, err := source.Flip(dir, r, flip, time.Now())
				if err != nil {
					return sourceError("ledger", dir, err)
				}
				return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
					fmt.Fprintf(w, "abcd source ledger — line %d cites line %d publicly (%s), committed in the corpus\n", res.Line, flip, res.Record.SourceKey)
				})
			}
			res, err := source.Append(source.AppendRequest{Corpus: dir, Repo: r, DecisionRef: decision, Claim: claim,
				SourceKey: key, Locator: locator, Influence: influence, UsedIn: usedIn, Corrects: corrects, Now: time.Now()})
			if err != nil {
				return sourceError("ledger", dir, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd source ledger — line %d: %s → %s (%s), committed in the corpus\n",
					res.Line, res.Record.SourceKey, termsafe.Sanitize(res.Record.DecisionRef), res.Record.Influence)
				fmt.Fprintln(w, "  cited_publicly: false — citing it is the person's call (abcd source ledger --flip)")
			})
		},
	}
	f := cmd.Flags()
	f.StringVar(&repo, "repo", "", "the ledger's repository handle (default: this checkout's root commit, 12 hex digits)")
	f.StringVar(&decision, "decision", "", "the decision influenced: a DECISIONS.md date, an ADR or intent id, or free text")
	f.StringVar(&claim, "claim", "", "what was decided or claimed")
	f.StringVar(&key, "source", "", "the source key")
	f.StringVar(&locator, "locator", "", "where in the source (pp., §)")
	f.StringVar(&influence, "influence", "", "the influence: "+enumHelp(source.Influences))
	f.StringArrayVar(&usedIn, "used-in", nil, "a repository-relative path the influence landed in (repeatable)")
	f.IntVar(&corrects, "corrects", 0, "the line this record corrects")
	f.IntVar(&flip, "flip", 0, "cite line N publicly (the person's act; checks the source's permission first)")
	f.BoolVar(&list, "list", false, "print the ledger, numbered (read-only)")
	cmd.MarkFlagsMutuallyExclusive("flip", "list")
	return cmd
}

func newSourceSyncBanlistCommand(asJSON *bool, corpusFlag *string) *cobra.Command {
	var refresh bool
	cmd := &cobra.Command{
		Use:   "sync-banlist",
		Short: "Project confidential titles and aliases into this repository's untracked private banlist",
		Long: "Regenerate the corpus's block in this repository's untracked private banlist\n" +
			"(.abcd/.work.local/private-names.txt, the banlist verb's private layer): every\n" +
			"confidential source's title and aliases, and its authors under ban_authors, as\n" +
			"whitespace-flexible, case-insensitive phrases. Lines outside the block survive.\n" +
			"A corpus whose folders and entries disagree is refused and nothing is written.\n\n" +
			"--refresh is the pre-commit guard's mode: it updates a private store that already\n" +
			"exists and declares the keyed format, and never creates one. With no corpus, no store\n" +
			"or a legacy store (migrate it with `abcd banlist migrate`) it says so on one line and\n" +
			"exits 0.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := sourceCorpusDir(*corpusFlag)
			if err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, "the private banlist")
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd source sync-banlist: " + err.Error() + " (nothing written)"}
			}
			res, err := source.SyncBanlist(dir, root, source.SyncOptions{Refresh: refresh})
			if refresh && errors.Is(err, source.ErrNoCorpus) {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd source sync-banlist: no sources corpus at %s — generated banlist block not refreshed (skipped)\n",
					fsutil.RedactHome(dir))
				return nil
			}
			if refresh && errors.Is(err, banlist.ErrNoStore) {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd source sync-banlist: no private store at %s — the refresh never creates one; run `abcd source sync-banlist` to opt this repository in (skipped)\n",
					banlist.PrivateRelPath)
				return nil
			}
			if refresh && errors.Is(err, banlist.ErrLegacyStore) {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd source sync-banlist: %s predates the keyed format — migrate it once with `abcd banlist migrate` (not refreshed)\n",
					banlist.PrivateRelPath)
				return nil
			}
			if err != nil {
				return sourceError("sync-banlist", dir, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				state := "unchanged"
				switch {
				case res.Block.Created:
					state = "created the store"
				case res.Block.Changed:
					state = "rewrote the block"
				}
				fmt.Fprintf(w, "abcd source sync-banlist — %d confidential source%s, %d pattern%s in %s (%s)\n",
					res.Sources, pluralS(res.Sources), res.Block.Entries, pluralS(res.Block.Entries), res.Block.Path, state)
			})
		},
	}
	cmd.Flags().BoolVar(&refresh, "refresh", false, "the guard's mode: update an existing store only; an absent corpus or store is a one-line notice and exit 0")
	return cmd
}

func newSourceCiteCheckCommand(asJSON *bool, corpusFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "cite-check <file|->",
		Short: "Scan text for confidential sources; report offenders by key only (exit 1 on a hit)",
		Long: "Scan a file, or stdin with -, for every confidential source's title, aliases and\n" +
			"opted-in authors, through the private banlist's matcher — the engine the pre-commit\n" +
			"guard runs. Offenders are reported by key, field, line and byte offset, never by the\n" +
			"text matched, so the report is safe to relay. Exit 1 when anything is found.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := sourceCorpusDir(*corpusFlag)
			if err != nil {
				return err
			}
			var text []byte
			if args[0] == "-" {
				text, err = io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxScanBytes+1))
				if err == nil && len(text) > maxScanBytes {
					err = fmt.Errorf("stdin is over %d bytes", maxScanBytes)
				}
			} else {
				text, err = fsutil.ReadGuarded(args[0], maxScanBytes)
			}
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd source cite-check: the text cannot be read (absent, oversize, or not a regular file)"}
			}
			rep, err := source.CiteCheck(dir, text)
			if err != nil {
				return sourceError("cite-check", dir, err)
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, rep, func(w io.Writer) {
				if rep.Clean() {
					fmt.Fprintf(w, "abcd source cite-check — clean against %d confidential source%s\n", rep.Sources, pluralS(rep.Sources))
					return
				}
				fmt.Fprintf(w, "abcd source cite-check — %d finding%s (the matched text is withheld)\n", len(rep.Findings), pluralS(len(rep.Findings)))
				for _, f := range rep.Findings {
					fmt.Fprintf(w, "  %s  %s  line %d, byte %d\n", f.Source, f.Field, f.Line, f.Offset)
				}
			}); rerr != nil {
				return rerr
			}
			if !rep.Clean() {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
}

// applySourceFlagErrors withholds the offending token from a flag-parse failure on
// the source verbs, as the banlist verbs do: an argument read as a flag may be a
// confidential title or alias, and cobra's own message quotes it. It runs after the
// tree-wide usage tagging, which would otherwise replace it.
func applySourceFlagErrors(root *cobra.Command) {
	for _, cmd := range root.Commands() {
		if cmd.Name() != "source" {
			continue
		}
		cmd.SetFlagErrorFunc(sourceFlagError)
		for _, sub := range cmd.Commands() {
			sub.SetFlagErrorFunc(sourceFlagError)
		}
	}
}

func sourceFlagError(cmd *cobra.Command, err error) error {
	_ = err // deliberately discarded: it may quote a confidential string
	return &exitError{Code: 2, Msg: cmd.CommandPath() + ": a flag or its value was not understood (the text is withheld — it may be " +
		"a confidential name); see --help, and pass identifying strings with --meta"}
}
