package scaffold

// Wiring to the managed repo's own CI check names (itd-93 AC1).
//
// A changelog-driven release is decided by merging the release pull request,
// so what gates that merge is what gates the release: the repository's own
// pull-request CI. The scaffold cannot configure branch protection — it writes
// files and holds no token — so it reads the repository's CI workflows, derives
// the check names its pull requests report, and writes them into the machinery
// it lays down: the runbook's merge-gate section, which tells the operator to
// require exactly these contexts on the default branch, and the release
// workflow's verify-job header, which names them as the merge gate the release
// job re-runs a subset of. Nothing is hard-coded to abcd's own check names.
//
// The trigger and the jobs come from the workflow's parsed YAML, read as a tree of
// nodes and walked by hand within named limits (readWorkflow), since a workflow file
// is text a hostile branch can choose. The derivation fails toward omission: a job whose check name GitHub computes at run
// time — a matrix job, a job named by an expression, a reusable-workflow call — is
// left out rather than guessed, and the runbook says a hand-added context may be
// needed. Every name that is kept
// is held to an allowlist before it is written anywhere, because the scaffold
// writes it into YAML and a hostile workflow file must not be able to inject.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// workflowsDir is where GitHub Actions discovers workflows.
const workflowsDir = ".github/workflows"

// maxWorkflowFiles bounds the directory walk; a repository with more workflow
// files than this is not one whose CI the scaffold can summarise faithfully.
const maxWorkflowFiles = 64

// checkNameRe is the allowlist for a derived check name written into the
// scaffolded files: letters, digits, space and a small punctuation set. It
// excludes every YAML and shell metacharacter (`:`, `#`, quotes, `$`, braces,
// backticks, newlines), so a name can only ever be inert text.
var checkNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._()/+-]{0,99}$`)

// jobIDRe is what GitHub accepts as a job id.
var jobIDRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

// DeriveCIChecks reads repoRoot's own CI workflows and returns the check names
// its pull requests report, sorted and de-duplicated. Only workflows triggered
// by `pull_request`, `pull_request_target` or `merge_group` gate a merge, and
// the scaffold's own workflows (the release pair and the dependency
// re-authoring) are never counted. An absent workflows directory yields nil, and
// the runbook then says no merge gate was found. What could not be read comes back in
// unread, names fit to write (writableWorkflowName), so the runbook names it rather
// than claim the list is whole or that no workflow exists.
func DeriveCIChecks(repoRoot string) (checks, unread []string) {
	seen := map[string]bool{}
	forEachWorkflow(repoRoot, func(name, src string) {
		found, err := workflowChecks(src)
		if err != nil {
			unread = append(unread, writableWorkflowName(name))
			return
		}
		for _, check := range found {
			seen[check] = true
		}
	}, func(name, _ string) {
		unread = append(unread, writableWorkflowName(name))
	}, nil)
	return sortedKeys(seen), unread
}

// workflowNameRe is a workflow file's name that is safe in the YAML and markdown the
// scaffold writes, with no metacharacter of either.
var workflowNameRe = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,100}$`)

// writableWorkflowName is a file's name as the scaffold may write it, since a file
// name can hold anything.
func writableWorkflowName(name string) string {
	if workflowNameRe.MatchString(name) {
		return name
	}
	return "a file with a name abcd does not write here"
}

// forEachWorkflow calls fn with the name and contents of each of repoRoot's
// own CI workflow files, in directory order, bounded by maxWorkflowFiles, and
// unread with what it could not read and why; an absent directory is neither. The
// scaffold's own workflows (the release pair and the dependency re-authoring)
// are never passed: the re-authoring workflow runs on pull requests but judges
// none, and counting it would change the release rendering the moment a
// repository opted in. reserved, when set, is called with each such file instead, or
// with why it could not be read. A file that cannot be read is skipped.
func forEachWorkflow(repoRoot string, fn func(name, src string), unread func(name, why string),
	reserved func(name, src string, err error)) {
	dir := filepath.Join(repoRoot, filepath.FromSlash(workflowsDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			unread(workflowsDir, "cannot be listed")
		}
		return
	}
	files := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !(strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")) {
			continue
		}
		if rel := workflowsDir + "/" + name; rel == ReleaseYMLPath || rel == AutoReleaseYMLPath || rel == ReauthorYMLPath {
			if reserved != nil {
				data, err := fsutil.ReadGuarded(filepath.Join(dir, name), maxWorkflowBytes)
				reserved(name, string(data), err)
			}
			continue
		}
		if files++; files > maxWorkflowFiles {
			unread(workflowsDir, fmt.Sprintf("holds more than the %d workflow files abcd reads", maxWorkflowFiles))
			return
		}
		data, err := fsutil.ReadGuarded(filepath.Join(dir, name), maxWorkflowBytes)
		if err != nil {
			unread(name, "cannot be read safely (a link, or larger than abcd reads)")
			continue
		}
		fn(name, string(data))
	}
}

// sortedKeys returns a set's members sorted, or nil for an empty set.
func sortedKeys(set map[string]bool) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// RequiredChecks returns the check names a ruleset may require on
// defaultBranch without blocking a merge (itd-2610290746077716): the checks
// EVERY pull request into it reports. Stricter than DeriveCIChecks, which
// lists what a pull request may report, because a required check that never
// reports holds the merge for ever. A workflow is counted only when it runs on
// `pull_request` or `pull_request_target` with no `paths`/`paths-ignore`
// filter, no `branches-ignore` filter, and a `branches` filter (if any) that
// matches defaultBranch; and, when mergeQueue is set, also on `merge_group`,
// since a queue entry waits on the same contexts. A matrix job a run may skip is
// left out as well (skippableJobs): skipped, it reports one check under its
// unexpanded name, so the contexts its values name never arrive. A job with no
// matrix still counts, since skipped it reports its one name, which a ruleset
// reads as passed. Each workflow or job left out is named in notes, so a person can
// see why a check is not required.
// A trigger with a key abcd does not read is left out, never read as unfiltered.
// Fails toward omission, like DeriveCIChecks: an empty answer requires nothing.
//
// A shorter list reads as an answer, so a workflow the read cannot take whole is
// no omission: a file that does not parse or cannot be read, one past the read's
// limits (readWorkflow), too many files, or a workflow whose jobs or needs GitHub
// would refuse return an error naming each, and no checks at all. A file name and
// a parser's words reach a JSON surface as well as the terminal, so hidden runes in
// either are encoded here.
func RequiredChecks(repoRoot, defaultBranch string, mergeQueue bool) (checks, notes []string, err error) {
	seen := map[string]bool{}
	var refused []string
	refuse := func(name, why string) {
		refused = append(refused, termsafe.EncodeHiddenRunes(name+" "+why))
	}
	forEachWorkflow(repoRoot, func(file, src string) {
		wf, err := readWorkflow(src)
		if err != nil {
			refuse(file, "does not read as a workflow: "+boundedError(err))
			return
		}
		name := termsafe.EncodeHiddenRunes(file)
		if !wf.on.present {
			notes = append(notes, name+" names no `on:` key, so GitHub runs nothing from it and its checks are not required")
			return
		}
		pr, ok := wf.on.events["pull_request"]
		if !ok {
			pr, ok = wf.on.events["pull_request_target"]
		}
		if !ok && !wf.on.unread {
			return
		}
		why := ""
		switch {
		case wf.on.unread || pr.unread:
			why = "writes its trigger in a form abcd does not read (a merge key, a key that is not plain or not one " +
				"of GitHub's filters, or a filter that is not a list)"
		case pr.has("paths") || pr.has("paths-ignore"):
			why = "runs only when some paths change"
		case pr.has("branches-ignore"):
			why = "has a branches-ignore filter on its pull-request trigger"
		case pr.has("branches") && !branchListMatches(pr.filters["branches"], defaultBranch):
			why = "its pull-request trigger leaves out the default branch"
		case pr.has("types") && !coversPushes(pr.filters["types"]):
			why = "runs only on some pull-request activities (its types filter leaves out opened, synchronize or reopened)"
		case mergeQueue && !wf.on.names("merge_group"):
			why = "does not run on the merge queue, which is on"
		}
		if why != "" {
			notes = append(notes, name+" "+why+", so its checks are not required")
			return
		}
		if len(wf.jobs) == 0 {
			refuse(file, "names no jobs")
			return
		}
		skippable, err := skippableJobs(wf.jobs)
		if err != nil {
			refuse(file, "does not read as a workflow: "+boundedError(err))
			return
		}
		for _, job := range wf.jobs {
			if job.matrix && skippable[job.id] {
				notes = append(notes, name+": job "+job.id+" is a matrix job abcd cannot tell every run reports: its "+
					"`if:` or a job it needs may skip it, and a skipped matrix job reports only its unexpanded name, so "+
					"it is not required. If every pull request does report it, require it by hand or through "+
					"github.required_checks")
				continue
			}
			if cs, ok := job.checkNames(); ok {
				for _, c := range cs {
					seen[c] = true
				}
				continue
			}
			notes = append(notes, name+": job "+job.id+" reports a name only the run knows "+
				"(a matrix, an expression or a reusable workflow), so it is not required")
		}
	}, refuse, func(file, src string, err error) {
		// One that runs on no pull request has nothing to require.
		if err == nil {
			if wf, err := readWorkflow(src); err == nil && !wf.on.gatesAMerge() && !wf.on.unread {
				return
			}
		}
		notes = append(notes, termsafe.EncodeHiddenRunes(file)+" has the name of a file `abcd launch scaffold` writes, "+
			"so abcd counts no check from it. If it is this repository's own CI, rename it or name its checks in "+
			"github.required_checks")
	})
	if len(refused) > 0 {
		return nil, nil, errors.New("abcd cannot read every workflow, so it proposes no required checks: " +
			strings.Join(refused, "; ") + ". Fix the file, or name the checks in github.required_checks")
	}
	return sortedKeys(seen), notes, nil
}

// boundedError is an error from reading a workflow file, short enough for a note.
func boundedError(err error) string {
	const limit = 200
	if s := err.Error(); len(s) <= limit {
		return s
	}
	return strings.ToValidUTF8(err.Error()[:limit], "") + "..."
}

// prFilterKeys are the filters GitHub reads under a pull-request trigger.
var prFilterKeys = map[string]bool{
	"branches": true, "branches-ignore": true, "paths": true, "paths-ignore": true, "types": true,
}

// coversPushes reports whether a pull-request `types:` filter keeps the three
// activities GitHub runs on by default, so the workflow still reports on every
// pull request and on every push to one.
func coversPushes(types []string) bool {
	have := map[string]bool{}
	for _, t := range types {
		have[t] = true
	}
	return have["opened"] && have["synchronize"] && have["reopened"]
}

// branchListMatches reports whether a `branches` filter covers branch, by
// GitHub's filter globs: `**` matches any run of characters, `*` any run but
// `/`, and `?` one character. A negated pattern (`!x`) is not interpreted, so a
// list holding one is treated as not matching, the direction that requires
// nothing, and so is a pattern longer than maxGlobBytes.
func branchListMatches(patterns []string, branch string) bool {
	matched := false
	for _, p := range patterns {
		if strings.HasPrefix(p, "!") {
			return false
		}
		if len(p) <= maxGlobBytes && branchGlob(p).MatchString(branch) {
			matched = true
		}
	}
	return matched
}

// branchGlob compiles one filter glob into an anchored regular expression.
func branchGlob(p string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(p); i++ {
		switch {
		case strings.HasPrefix(p[i:], "**"):
			b.WriteString(".*")
			i++
		case p[i] == '*':
			b.WriteString("[^/]*")
		case p[i] == '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(p[i : i+1]))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

// workflowChecks returns the check names one workflow file reports, none when the
// workflow does not gate a pull request, or why it could not be read.
func workflowChecks(src string) ([]string, error) {
	wf, err := readWorkflow(src)
	if err != nil {
		return nil, err
	}
	if wf.on.unread {
		return nil, errors.New("its trigger holds a form abcd does not read")
	}
	if !wf.on.gatesAMerge() {
		return nil, nil
	}
	var out []string
	for _, job := range wf.jobs {
		if name, ok := job.checkName(); ok {
			out = append(out, name)
		}
	}
	return out, nil
}

// The bounds on reading one workflow file. A parsed tree's memory grows with the file,
// but the parse's time does not always, and the library offers no limit of its own, so
// before it runs a scan of the raw bytes holds the file to limits on its `#` characters,
// its byte order marks, its directive lines and how each line opens (prescanWorkflow).
// Nothing after the parse decodes into a Go value, where the library's duplicate check is
// quadratic and allocates for each pair: the read walks nodes by hand, follows an
// alias by pointer and never copies what it names. A file past a limit is refused
// rather than read in part.
const (
	// maxWorkflowNodes bounds the nodes a file's tree may hold, and separately what the
	// read may visit: each node, each byte of text it reaches through an alias, and each
	// byte of a glob it will compile. abcd's own largest workflow, at 68 KiB, holds under
	// 700 nodes, so a full 1 MiB of ordinary workflow holds about a tenth of this.
	maxWorkflowNodes = 100_000
	// maxWorkflowDepth bounds how deeply a file's tree may nest. abcd's own workflows
	// nest 7 levels.
	maxWorkflowDepth = 64
	// maxMatrixValues is GitHub's limit of jobs one matrix may start. A longer list
	// is not one abcd expands.
	maxMatrixValues = 256
	// maxGlobBytes is as long as a branch name can be, so a longer pattern is not
	// compiled and matches nothing.
	maxGlobBytes = 255
	// maxHashes bounds the `#` characters in a file, since any of them could open a
	// comment. abcd's own most commented workflow holds 627 in 68 KiB, so a full 1 MiB
	// of the same holds about half of this.
	maxHashes = 20_000
	// maxDirectiveLines bounds the lines that open with `%`. A workflow has no use for
	// more than a `%YAML` line.
	maxDirectiveLines = 8
	// maxLineLead bounds a line's opening run of spaces and block indicators, which is
	// the most block levels the line can stand at. abcd's own workflows reach 14, and a
	// script in one would have to indent past 256 spaces to meet it.
	maxLineLead = 256
)

// prescanWorkflow holds a file's raw bytes to the limits the parser has no setting
// for, in one pass, reading a line's end as YAML does. A UTF-16 file is refused, since
// the parser would read what this scan cannot, and so is a byte order mark anywhere but
// at the start, since the scanner skips one where this scan sees a line's first byte.
func prescanWorkflow(src string) error {
	if strings.HasPrefix(src, "\xff\xfe") || strings.HasPrefix(src, "\xfe\xff") {
		return errors.New("it is not UTF-8")
	}
	src = strings.TrimPrefix(src, "\ufeff")
	if strings.Contains(src, "\ufeff") {
		return errors.New("it holds a byte order mark past its start")
	}
	if strings.Count(src, "#") > maxHashes {
		return fmt.Errorf("it holds more than the %d `#` characters abcd reads", maxHashes)
	}
	directives := 0
	for line := 1; len(src) > 0; line++ {
		lead, at := lineLead(src)
		if lead > maxLineLead {
			return fmt.Errorf("line %d opens with more than %d spaces and block indicators, which abcd takes for that many "+
				"YAML levels, even in a script", line, maxLineLead)
		}
		if src[0] == '%' {
			if directives++; directives > maxDirectiveLines {
				return fmt.Errorf("it holds more than the %d directive lines abcd reads", maxDirectiveLines)
			}
		}
		next := strings.IndexAny(src[at:], "\n\r\u0085\u2028\u2029")
		if next < 0 {
			return nil
		}
		_, width := utf8.DecodeRuneInString(src[at+next:])
		if strings.HasPrefix(src[at+next:], "\r\n") {
			width = 2
		}
		src = src[at+next+width:]
	}
	return nil
}

// lineLead counts the spaces and block indicators a line opens with, and says where
// they end.
func lineLead(line string) (lead, end int) {
	for end < len(line) {
		switch {
		case line[end] == ' ':
			end++
		case strings.IndexByte("-?:", line[end]) >= 0 && end+1 < len(line) && line[end+1] == ' ':
			end += 2
		default:
			return lead, end
		}
		lead++
	}
	return lead, end
}

// mergeTag is the tag the parser gives a merge key.
const mergeTag = "!!merge"

// trigger is a workflow's `on:` as far as a merge gate cares.
type trigger struct {
	present bool
	// events holds each event named, with its filters read for the two pull-request
	// events alone.
	events map[string]eventFilter
	// unread says the events hold a shape abcd does not read, which could name more.
	unread bool
}

// eventFilter is one event's filters.
type eventFilter struct {
	filters map[string][]string
	// unread says the event's value holds a shape abcd does not read, which could
	// hold a filter.
	unread bool
}

func (f eventFilter) has(key string) bool {
	_, ok := f.filters[key]
	return ok
}

// names reports whether the trigger names event.
func (t trigger) names(event string) bool {
	_, ok := t.events[event]
	return ok
}

// gatesAMerge reports whether the trigger runs the workflow against a pull request
// or its merge-queue entry.
func (t trigger) gatesAMerge() bool {
	return t.names("pull_request") || t.names("pull_request_target") || t.names("merge_group")
}

// workflowFile is what the read takes from one workflow file.
type workflowFile struct {
	on   trigger
	jobs []ciJob
}

// readWorkflow reads a workflow's trigger and jobs from its parsed tree, refusing a
// file that does not parse, one past the read's limits, one defining a key twice, and
// a jobs block GitHub would refuse. A panic in the parser reads as a file it could not
// parse, since the input is a file abcd did not write.
func readWorkflow(src string) (wf workflowFile, err error) {
	defer func() {
		if r := recover(); r != nil {
			wf, err = workflowFile{}, errors.New("the YAML parser could not read it")
		}
	}()
	if err := prescanWorkflow(src); err != nil {
		return workflowFile{}, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		return workflowFile{}, err
	}
	if doc.IsZero() || len(doc.Content) == 0 {
		return workflowFile{}, nil
	}
	nodes := 0
	if err := checkTree(&doc, 0, &nodes); err != nil {
		return workflowFile{}, err
	}
	r := &workflowRead{}
	root, err := r.follow(doc.Content[0])
	if err != nil {
		return workflowFile{}, err
	}
	if isNull(root) {
		return workflowFile{}, nil
	}
	if root.Kind != yaml.MappingNode {
		return workflowFile{}, errors.New("it is not a mapping")
	}
	if err := r.visit(len(root.Content)); err != nil {
		return workflowFile{}, err
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, value := root.Content[i], root.Content[i+1]
		if key.Tag == mergeTag {
			return workflowFile{}, fmt.Errorf("line %d holds a merge key at the top, which abcd does not read", key.Line)
		}
		if key.Kind != yaml.ScalarNode {
			continue
		}
		switch key.Value {
		case "on":
			if wf.on, err = r.readOn(value); err != nil {
				return workflowFile{}, err
			}
		case "jobs":
			if wf.jobs, err = r.readJobs(value); err != nil {
				return workflowFile{}, err
			}
		}
	}
	return wf, nil
}

// checkTree holds a parsed tree to the limits on its size and depth, and refuses a key
// defined twice in any mapping, as GitHub does. It visits each node once and never
// follows an alias, so its work is the tree's size.
func checkTree(n *yaml.Node, depth int, nodes *int) error {
	if *nodes++; *nodes > maxWorkflowNodes {
		return fmt.Errorf("it holds more than the %d YAML nodes abcd reads", maxWorkflowNodes)
	}
	if depth > maxWorkflowDepth {
		return fmt.Errorf("it nests deeper than the %d levels abcd reads", maxWorkflowDepth)
	}
	if n.Kind == yaml.MappingNode {
		seen := make(map[string]bool, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i]
			if key.Kind != yaml.ScalarNode {
				continue
			}
			if seen[key.Value] {
				return fmt.Errorf("line %d defines a key its mapping already holds", key.Line)
			}
			seen[key.Value] = true
		}
	}
	for _, child := range n.Content {
		if err := checkTree(child, depth+1, nodes); err != nil {
			return err
		}
	}
	return nil
}

// workflowRead counts the walk over one parsed file, which reads only the keys it asks
// for and never what is under a job's steps.
type workflowRead struct {
	visited int
}

// visit counts n things the walk is about to read.
func (r *workflowRead) visit(n int) error {
	if r.visited += n; r.visited > maxWorkflowNodes {
		return fmt.Errorf("reading it visits more than the %d YAML nodes abcd reads", maxWorkflowNodes)
	}
	return nil
}

// follow is a node, or the node its alias names, with any text there counted, since
// an alias can repeat long text at the price of a few bytes. An anchor cannot sit on an
// alias, so one step arrives.
func (r *workflowRead) follow(n *yaml.Node) (*yaml.Node, error) {
	if n.Kind != yaml.AliasNode {
		return n, nil
	}
	if err := r.visit(1 + len(n.Alias.Value)); err != nil {
		return nil, err
	}
	return n.Alias, nil
}

// isNull reports a key given no value.
func isNull(n *yaml.Node) bool { return n.Kind == yaml.ScalarNode && n.Tag == "!!null" }

// readOn reads `on:`: one event, a list of them, or a mapping of them to their filters.
func (r *workflowRead) readOn(n *yaml.Node) (trigger, error) {
	t := trigger{present: true, events: map[string]eventFilter{}}
	n, err := r.follow(n)
	if err != nil {
		return t, err
	}
	switch {
	case isNull(n):
	case n.Kind == yaml.ScalarNode:
		t.events[n.Value] = eventFilter{}
	case n.Kind == yaml.SequenceNode:
		events, ok, err := r.readList(n)
		if err != nil {
			return t, err
		}
		t.unread = !ok
		for _, e := range events {
			t.events[e] = eventFilter{}
		}
	case n.Kind == yaml.MappingNode:
		if err := r.visit(len(n.Content)); err != nil {
			return t, err
		}
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag == mergeTag {
				t.unread = true
				continue
			}
			f := eventFilter{}
			if key.Value == "pull_request" || key.Value == "pull_request_target" {
				if f, err = r.readEventFilter(n.Content[i+1]); err != nil {
					return t, err
				}
			}
			t.events[key.Value] = f
		}
	default:
		t.unread = true
	}
	return t, nil
}

// readEventFilter reads a pull-request event's value: nothing, or a mapping of GitHub's
// filters to their lists.
func (r *workflowRead) readEventFilter(n *yaml.Node) (eventFilter, error) {
	f := eventFilter{filters: map[string][]string{}}
	n, err := r.follow(n)
	if err != nil {
		return f, err
	}
	if isNull(n) {
		return f, nil
	}
	if n.Kind != yaml.MappingNode {
		f.unread = true
		return f, nil
	}
	if err := r.visit(len(n.Content)); err != nil {
		return f, err
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := n.Content[i]
		if key.Kind != yaml.ScalarNode || key.Tag == mergeTag || !prFilterKeys[key.Value] {
			f.unread = true
			continue
		}
		list, ok, err := r.readList(n.Content[i+1])
		if err != nil {
			return f, err
		}
		f.unread = f.unread || !ok
		f.filters[key.Value] = list
		if key.Value == "branches" {
			// Each of these is compiled later, at a cost that follows its length.
			for _, glob := range list {
				if len(glob) <= maxGlobBytes {
					if err := r.visit(len(glob)); err != nil {
						return f, err
					}
				}
			}
		}
	}
	return f, nil
}

// readList reads a value that is absent, one word, or a list of words; ok is false
// when it holds anything else.
func (r *workflowRead) readList(n *yaml.Node) (list []string, ok bool, err error) {
	n, err = r.follow(n)
	if err != nil {
		return nil, false, err
	}
	switch {
	case isNull(n):
		return nil, true, nil
	case n.Kind == yaml.ScalarNode:
		return []string{n.Value}, true, nil
	case n.Kind != yaml.SequenceNode:
		return nil, false, nil
	}
	if err := r.visit(len(n.Content)); err != nil {
		return nil, false, err
	}
	ok = true
	for _, item := range n.Content {
		if item, err = r.follow(item); err != nil {
			return nil, false, err
		}
		switch {
		case isNull(item):
		case item.Kind == yaml.ScalarNode:
			list = append(list, item.Value)
		default:
			ok = false
		}
	}
	return list, ok, nil
}

// jobCondition is what a job's `if:` says about whether a run going ahead runs it.
type jobCondition int

const (
	// condFollowsNeeds is no `if:`, or one that adds nothing to the implicit
	// success(): the job runs unless an ancestor was skipped or failed.
	condFollowsNeeds jobCondition = iota
	// condAlwaysRuns reads only the run's own state, so the job runs whatever
	// happened to its ancestors.
	condAlwaysRuns
	// condCannotTell is any other condition, which this reader does not evaluate.
	condCannotTell
)

// ciJob is one job as its workflow's parsed YAML gives it.
type ciJob struct {
	id    string
	name  string
	named bool
	uses  bool
	// matrix says the job has a matrix, or a strategy too computed to look into.
	matrix bool
	// matrixValues are the values of a matrix with one key, listed, with no
	// include or exclude, each of which GitHub reports as it is written. It is nil
	// for any other matrix.
	matrixValues []string
	cond         jobCondition
	needs        []string
}

// readJobs reads the jobs block in file order, refusing one GitHub would refuse.
func (r *workflowRead) readJobs(n *yaml.Node) ([]ciJob, error) {
	block, err := r.follow(n)
	if err != nil {
		return nil, err
	}
	if isNull(block) {
		return nil, nil
	}
	if block.Kind != yaml.MappingNode {
		return nil, errors.New("its jobs key is not a mapping")
	}
	if err := r.visit(len(block.Content)); err != nil {
		return nil, err
	}
	jobs := make([]ciJob, 0, len(block.Content)/2)
	for i := 0; i+1 < len(block.Content); i += 2 {
		key := block.Content[i]
		if key.Kind != yaml.ScalarNode || !jobIDRe.MatchString(key.Value) {
			return nil, fmt.Errorf("line %d holds a job id GitHub would refuse", key.Line)
		}
		job, err := r.readJob(key.Value, block.Content[i+1])
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// readJob reads one job's own keys; steps and everything under them are not its
// keys, so nothing written there is read as one.
func (r *workflowRead) readJob(id string, n *yaml.Node) (ciJob, error) {
	job := ciJob{id: id}
	n, err := r.follow(n)
	if err != nil {
		return job, err
	}
	if n.Kind != yaml.MappingNode {
		return job, fmt.Errorf("job %s is not a mapping", id)
	}
	if err := r.visit(len(n.Content)); err != nil {
		return job, err
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		key, value := n.Content[i], n.Content[i+1]
		if key.Kind != yaml.ScalarNode || key.Tag == mergeTag {
			return job, fmt.Errorf("job %s holds a key abcd does not read, on line %d", id, key.Line)
		}
		switch key.Value {
		case "name":
			v, err := r.follow(value)
			if err != nil {
				return job, err
			}
			job.named, job.name = true, v.Value
		case "uses":
			job.uses = true
		case "if":
			if job.cond, err = r.readCondition(value); err != nil {
				return job, err
			}
		case "needs":
			needs, ok, err := r.readList(value)
			if err != nil {
				return job, err
			}
			if !ok {
				return job, fmt.Errorf("job %s has a needs key that is neither a job id nor a list of them", id)
			}
			job.needs = needs
		case "strategy":
			if job.matrix, job.matrixValues, err = r.readMatrix(value); err != nil {
				return job, err
			}
		}
	}
	return job, nil
}

// readCondition judges a job's `if:`, written bare or as one expression. It
// evaluates nothing: only the conditions whose answer needs no context are known.
func (r *workflowRead) readCondition(n *yaml.Node) (jobCondition, error) {
	n, err := r.follow(n)
	if err != nil {
		return condCannotTell, err
	}
	if n.Kind != yaml.ScalarNode || (n.Tag != "!!str" && n.Tag != "!!bool") {
		return condCannotTell, nil
	}
	cond := strings.TrimSpace(n.Value)
	if inner, ok := strings.CutPrefix(cond, "${{"); ok {
		inner, closed := strings.CutSuffix(inner, "}}")
		if !closed || strings.Contains(inner, "${{") {
			return condCannotTell, nil
		}
		cond = strings.TrimSpace(inner)
	}
	switch cond {
	case "always()", "!cancelled()":
		return condAlwaysRuns, nil
	case "success()", "true":
		return condFollowsNeeds, nil
	}
	return condCannotTell, nil
}

// readMatrix reads a job's `strategy:`: whether it has a matrix, and the values of
// the one shape whose check names the file settles. A computed strategy or matrix, or
// a key it does not read in either, reads as a matrix only the run can name.
func (r *workflowRead) readMatrix(n *yaml.Node) (matrix bool, values []string, err error) {
	if n, err = r.follow(n); err != nil {
		return false, nil, err
	}
	if isNull(n) {
		return false, nil, nil
	}
	if n.Kind != yaml.MappingNode {
		return true, nil, nil
	}
	if err := r.visit(len(n.Content)); err != nil {
		return false, nil, err
	}
	var m *yaml.Node
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := n.Content[i]
		if key.Kind != yaml.ScalarNode || key.Tag == mergeTag {
			return true, nil, nil
		}
		if key.Value == "matrix" {
			m = n.Content[i+1]
		}
	}
	if m == nil {
		return false, nil, nil
	}
	if m, err = r.follow(m); err != nil {
		return false, nil, err
	}
	if m.Kind != yaml.MappingNode || len(m.Content) != 2 {
		return true, nil, nil
	}
	key := m.Content[0]
	if key.Kind != yaml.ScalarNode || key.Tag == mergeTag || key.Value == "include" || key.Value == "exclude" {
		return true, nil, nil
	}
	list, err := r.follow(m.Content[1])
	if err != nil {
		return false, nil, err
	}
	if list.Kind != yaml.SequenceNode || len(list.Content) > maxMatrixValues {
		return true, nil, nil
	}
	if err := r.visit(len(list.Content)); err != nil {
		return false, nil, err
	}
	for _, item := range list.Content {
		if item, err = r.follow(item); err != nil {
			return false, nil, err
		}
		if item.Kind != yaml.ScalarNode || !reportsAsWritten(item) {
			return true, nil, nil
		}
		values = append(values, item.Value)
	}
	return true, values, nil
}

// plainIntRe is an integer written the one way GitHub reports it.
var plainIntRe = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)

// reportsAsWritten reports whether GitHub names a matrix value as its text: a
// string with no expression in it, a plain integer or a boolean. It reads a float
// as a number, so an unquoted 1.10 reports as 1.1.
func reportsAsWritten(item *yaml.Node) bool {
	switch item.Tag {
	case "!!str":
		return item.Value != "" && !strings.Contains(item.Value, "${{")
	case "!!int":
		return plainIntRe.MatchString(item.Value)
	case "!!bool":
		return item.Value == "true" || item.Value == "false"
	}
	return false
}

// skippableJobs reports, by job id, the jobs a run going ahead may skip. A job with
// a condition this reader cannot judge may be skipped. So may one following the
// implicit success() with such a job anywhere among its ancestors: success() reads
// every ancestor, including through a job that ran under always()
// (actions/runner#491). A job whose own condition is always() or !cancelled() runs.
// A need that names no job, and a cycle, are refused, as GitHub refuses the file.
func skippableJobs(jobs []ciJob) (map[string]bool, error) {
	byID := make(map[string]ciJob, len(jobs))
	for _, j := range jobs {
		byID[j.id] = j
	}
	const walking, walked = 1, 2
	state := make(map[string]int, len(jobs))
	behind := make(map[string]bool, len(jobs))
	var walk func(j ciJob) error
	walk = func(j ciJob) error {
		state[j.id] = walking
		for _, id := range j.needs {
			need, ok := byID[id]
			if !ok {
				return fmt.Errorf("job %s needs %q, which is no job in the file", j.id, id)
			}
			switch state[id] {
			case walking:
				return fmt.Errorf("job %s needs itself through %s", j.id, id)
			case 0:
				if err := walk(need); err != nil {
					return err
				}
			}
			if need.cond == condCannotTell || behind[id] {
				behind[j.id] = true
			}
		}
		state[j.id] = walked
		return nil
	}
	skippable := make(map[string]bool, len(jobs))
	for _, j := range jobs {
		if state[j.id] == 0 {
			if err := walk(j); err != nil {
				return nil, err
			}
		}
		skippable[j.id] = j.cond == condCannotTell || (j.cond == condFollowsNeeds && behind[j.id])
	}
	return skippable, nil
}

// checkName is the check context GitHub reports for the job, when it is
// knowable from the file: the job's `name:`, else its id. A matrix job, a
// reusable-workflow call and an expression-named job report names only the run
// knows, so they are omitted, never guessed.
func (j ciJob) checkName() (string, bool) {
	if j.matrix || j.uses {
		return "", false
	}
	name := j.id
	if j.named {
		name = j.name
	}
	if !checkNameRe.MatchString(name) {
		return "", false
	}
	return name, true
}

// checkNames is checkName with one matrix shape expanded (ciJob.matrixValues),
// which GitHub reports as one check per value, `<name> (<value>)`. Any other matrix
// still reports names only the run knows.
func (j ciJob) checkNames() ([]string, bool) {
	if !j.matrix {
		if n, ok := j.checkName(); ok {
			return []string{n}, true
		}
		return nil, false
	}
	if j.uses {
		return nil, false
	}
	base := j.id
	if j.named {
		base = j.name
	}
	var out []string
	for _, value := range j.matrixValues {
		name := base + " (" + value + ")"
		if !checkNameRe.MatchString(name) {
			return nil, false
		}
		out = append(out, name)
	}
	return out, len(out) > 0
}
