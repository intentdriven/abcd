package cli

// start.go is `abcd start [--profile <name>] [--permission-mode <mode>] [prompt]`
// (itd-2611251725203320; spc-2611251757236507): the one command a person types, in a
// plain terminal in a checkout abcd manages, to work in a session on abcd alone. It makes
// the account's first profile and starts nothing, or it proves one short launch with the
// session's arguments and replaces this process with the host. The lists, the arguments
// and the judge are the core's (runner/session_start.go); the order and the words are here.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/refusal"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// startConventionsBound is the most a conventions file may hold to be handed over whole:
// one launch argument is capped at 131072 bytes on Linux, and the file is never cut.
const startConventionsBound = 100 << 10

// startLine opens every line and every refusal of this verb.
const startLine = "abcd start: "

// startNotSandbox is what every line that names a profile says of it.
const startNotSandbox = "a separate login and settings on this one account, not a sandbox"

// startSelf names the running abcd, and startExec replaces this process with the host,
// returning only on a failure. Both are seams the tests replace.
var (
	startSelf = runner.SelfBinary
	startExec = func(path string, argv, env []string) error {
		// #nosec G204 -- path is the host's file as runner.SessionHarness admitted it.
		return syscall.Exec(path, argv, env)
	}
)

// newStartCommand builds `abcd start`.
func newStartCommand() *cobra.Command {
	var profile, mode string
	cmd := &cobra.Command{
		Use: "start [prompt]",
		Long: "Start a session on abcd alone: the few tools the work needs, abcd's hooks and the\n" +
			"repository's conventions file, and nothing else of your set-up. You type it, in a\n" +
			"plain terminal, in a checkout abcd manages. A run has one of two outcomes.\n\n" +
			"On an account with no profile it makes one directory under\n" +
			abcdhome.Display("profiles") + "/, writes abcd's settings there and nothing of yours,\n" +
			"prints the host's own login command, and starts nothing. The login is yours, in\n" +
			"your own shell: abcd never performs it, and never reads, stores or forwards it.\n\n" +
			"On a profile that is logged in it runs one short launch with the session's\n" +
			"arguments on that profile, refuses naming anything the launch reports that the\n" +
			"printed list does not name, and otherwise opens the session in place of abcd. The\n" +
			"short launch spends a little of the subscription and is written to the launch\n" +
			"ledger. It is one launch, not the open session: what the host adds after the start\n" +
			"abcd cannot see.\n\n" +
			"A profile is a separate login and settings on this one account, not a sandbox: a\n" +
			"command the session runs can still read whatever the account can.\n\n" +
			"With a prompt the session opens with it; with none it opens waiting for you. With\n" +
			"several profiles, name one with --profile. `abcd session` is another thing: the\n" +
			"live brief sessions.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r := startRequest{profile: profile, mode: mode,
				profileNamed: cmd.Flags().Changed("profile"), modeNamed: cmd.Flags().Changed("permission-mode")}
			if len(args) == 1 {
				r.prompt = args[0]
			}
			return runStart(cmd, r)
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "the profile to start on; on an account with none, the name the one made is given")
	enumFlag(cmd.Flags(), &mode, "permission-mode", "", "the permission mode the session starts in, ", runner.SessionPermissionModes(),
		"; without it, the host's default")
	return cmd
}

// startRequest is what the person typed.
type startRequest struct {
	profile, mode, prompt   string
	profileNamed, modeNamed bool
}

// startText is a refusal's words: opened with the verb, fit for a terminal, and closed by
// saying that nothing opened.
func startText(format string, args ...any) string {
	return startLine + termsafe.Sanitize(fsutil.RedactHome(fmt.Sprintf(format, args...))) + "; no session was started"
}

// runStart walks the refusals in their order, and then starts.
func runStart(cmd *cobra.Command, r startRequest) error {
	say := func(format string, args ...any) {
		fmt.Fprintln(cmd.OutOrStdout(), startLine+termsafe.Sanitize(fsutil.RedactHome(fmt.Sprintf(format, args...))))
	}

	for _, k := range driveSessionMarkers {
		if os.Getenv(k) != "" {
			return refused(2, refusal.StartOutsideATerminal, startText("%s is set, as a host sets it inside its own session; "+
				"type abcd start in a plain terminal", k))
		}
	}
	for _, s := range []struct {
		name   string
		stream any
	}{{"input", cmd.InOrStdin()}, {"output", cmd.OutOrStdout()}, {"error", cmd.ErrOrStderr()}} {
		if !connectIsTerminal(s.stream) {
			return refused(2, refusal.StartOutsideATerminal, startText("standard %s is not a terminal; "+
				"type abcd start in a plain terminal", s.name))
		}
	}
	modes := runner.SessionPermissionModes()
	if r.modeNamed && !slices.Contains(modes, r.mode) {
		return refused(2, refusal.UsageInvalidFlagValue, startText("the --permission-mode value is not accepted (it is not repeated here); "+
			"accepted values: %s", strings.Join(modes, ", ")))
	}
	if r.profileNamed && !runner.PlainProfileName(r.profile) {
		return refused(2, refusal.UsageInvalidFlagValue, startText("the --profile value is not a profile's name (it is not repeated here): "+
			"a letter or digit, then letters, digits, '.', '_' and '-'"))
	}
	cwd, err := os.Getwd()
	if err != nil {
		return refused(2, refusal.StartRepositoryUnready, startText("the working directory could not be read"))
	}
	at := cwd
	root, rootErr := gitutil.Toplevel(cwd)
	if rootErr == nil {
		at = root
	}
	roots, notes := layered.RootsFor(at)
	for _, n := range notes {
		fmt.Fprintln(cmd.ErrOrStderr(), noteLine(n))
	}
	home := roots.Home
	if !filepath.IsAbs(home) {
		return exitWith(1, refusal.None, startText("the person's home directory is not known as an absolute path"))
	}

	names, err := runner.ProfileNames(home)
	if err != nil {
		return refused(2, refusal.StartProfileUnusable, startText("%s", err.Error()))
	}
	if len(names) == 0 {
		return startMakesProfile(cmd, home, r, say)
	}
	// A name the account holds is a directory's, so a refusal may name it: withheld as the
	// caller's word, it would leave the refusals of that directory and the login command it
	// prints unusable.
	if slices.Contains(names, r.profile) {
		declareClosedSet(cmd.Flags(), "profile", r.profile)
	}
	p, err := runner.DriveProfile(home, r.profile)
	if err != nil {
		return refused(2, refusal.StartProfileUnusable, startText("%s", err.Error()))
	}
	shown := abcdhome.Display("profiles", p.Name())
	if entry := runner.ProfileLoads(p); entry != "" {
		way := "move it out by hand"
		if entry == "skills" {
			way = "start `abcd drain --drive` on this profile, which sets aside the copy of the account's skills the host syncs there, or move it out by hand"
		}
		return refused(2, refusal.StartProfileUnusable, startText("%s holds %s, which a launch would load; %s", shown, entry, way))
	}
	say("found the profile %s as it was; nothing in it is changed (%s)", shown, startNotSandbox)

	if rootErr != nil {
		return refused(2, refusal.StartRepositoryUnready, startText("this is not inside a git checkout"))
	}
	managed, err := ahoy.ManagedContext(cmd.Context(), root)
	switch {
	case err != nil:
		return refused(2, refusal.StartRepositoryUnready, startText("whether abcd manages this checkout could not be read: %s", err.Error()))
	case !managed:
		return refused(2, refusal.StartRepositoryUnready, startText("abcd does not manage this checkout"))
	}
	conventions, why := startConventions(root)
	if why != "" {
		return refused(2, refusal.StartRepositoryUnready, startText("%s at the root of this checkout %s, so it cannot be handed to the session whole",
			loop.ConventionsFile, why))
	}

	p.Repo = root
	harness, err := runner.SessionHarness(p, root)
	if err != nil {
		return refused(2, refusal.StartHostNotRunnable, startText("%s", err.Error()))
	}
	self := startSelf()
	if self == "" {
		return refused(2, refusal.StartHostNotRunnable, startText("abcd cannot name its own binary, which the session's hooks run"))
	}
	cfg, err := loadRunners(cmd, roots)
	if err != nil {
		return refused(2, refusal.StartHostNotRunnable, startText("the machine's runner configuration could not be read, so no launch ledger opens: %s", err.Error()))
	}

	// Held from the first launch: an interrupt ends the launch through the launcher's own
	// kill of its process group.
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runner.CheckLogin(ctx, p); err != nil {
		var none *runner.NoVerdictError
		if errors.As(err, &none) {
			return refused(2, refusal.StartNoVerdict, startText("the host gave no answer about the login on %s", shown))
		}
		return refused(2, refusal.StartNotLoggedIn, startLoggedOut(p, shown))
	}
	say("the host reports a login there; abcd asked for that one state and kept nothing else")

	kept, err := runner.KeepSessionBinary(home, self)
	if err != nil {
		return refused(2, refusal.StartHostNotRunnable, startText("the copy of abcd the session's hooks run could not be kept: %s", err.Error()))
	}
	hooks, left := startHooks(cmd.Root())
	s := runner.Session{Profile: p, Bin: kept, Hooks: hooks, ConventionsName: loop.ConventionsFile, Conventions: conventions,
		Mode: r.mode, Prompt: r.prompt}
	env, cleared := s.Env(os.Environ())
	say("tools: %s; no others; no server and no server's tool; no browser tool; no plugin but the host's own", strings.Join(runner.SessionTools(), ", "))
	say("hooks: abcd %s, a copy kept at %s, so an update, a rebuild or a removed worktree changes nothing this session runs", core.Version, kept)
	say("conventions: %s, %d bytes, handed over whole as a launch argument, as it stands at this moment", loop.ConventionsFile, len(conventions))
	if r.mode == "" {
		say("permission mode: the host's default (name one with --permission-mode)")
	} else {
		say("permission mode: %s, as you named it", r.mode)
	}
	if len(cleared) > 0 {
		say("left out of the session's environment: %s", strings.Join(cleared, ", "))
	}
	for _, event := range left {
		say("this binary has no hook for %s, so the session's settings leave that event out", event)
	}
	if why := startGuard(cmd, cwd); why != "" {
		return refused(2, refusal.StartWallNotHeld, startText("%s", why))
	}
	say("one short launch with the session's arguments is starting; it spends a little of the subscription and is written to the launch ledger")

	proof, err := runner.ProveSession(ctx, s, harness, root, cfg.Ledger())
	var limit *runner.LimitError
	var wall *runner.WallError
	switch {
	case errors.As(err, &limit):
		resets := ""
		if !limit.ResetsAt.IsZero() {
			resets = ", resetting at " + limit.ResetsAt.UTC().Format(time.RFC3339)
		}
		return refused(2, refusal.StartNoVerdict, startText("%s%s; nothing waits and nothing is tried again", limit.Error(), resets))
	case errors.As(err, &wall) && wall.Wall != runner.WallLaunch:
		return refused(2, refusal.StartWallNotHeld, startText("%s", err.Error()))
	case err != nil:
		return refused(2, refusal.StartNoVerdict, startText("the short launch came to no verdict: %s", err.Error()))
	case ctx.Err() != nil:
		return refused(2, refusal.StartNoVerdict, startText("the start was interrupted"))
	}
	// Read again: the launch took its time, and the open session reads it as it then stands.
	if why := startGuard(cmd, cwd); why != "" {
		return refused(2, refusal.StartWallNotHeld, startText("%s", why))
	}
	say("%s", startReported(proof))
	say("that was read from one launch with the session's arguments (in print mode, with abcd's own prompt and without a permission mode of yours), " +
		"not from the open session; what the host adds after the start abcd cannot see")
	say("opening the session; abcd says nothing after this line")

	// The host starts with default dispositions.
	stop()
	if err := startExec(harness, append([]string{harness}, s.Args()...), env); err != nil {
		return exitWith(1, refusal.None, startLine+termsafe.Sanitize(fsutil.RedactHome(fmt.Sprintf("the host could not be started (%s)", err.Error())))+"; no session is open")
	}
	return nil
}

// startLoggedOut is the logged-out refusal's words: the login that is the person's, on a
// line of its own, the one the profile's making prints.
func startLoggedOut(p runner.Profile, shown string) string {
	return startLine + "the host reports no login on " + shown + "; abcd never performs the login and never reads, stores or forwards its token; " +
		"log the host in once, in your own shell (this opens no session):\n\n  " + runner.LoginCommand(p) + "\n\n" +
		startLine + "run abcd start again once the login is done; no session was started"
}

// startMakesProfile makes the account's first profile, prints the login that is the
// person's, and starts nothing.
func startMakesProfile(cmd *cobra.Command, home string, r startRequest, say func(string, ...any)) error {
	self := startSelf()
	if self == "" {
		return refused(2, refusal.StartHostNotRunnable, startText("abcd cannot name its own binary, which the profile's settings run"))
	}
	kept, err := runner.KeepSessionBinary(home, self)
	if err != nil {
		return refused(2, refusal.StartHostNotRunnable, startText("the copy of abcd the profile's settings run could not be kept: %s", err.Error()))
	}
	name := r.profile
	if name == "" {
		name = runner.DefaultProfileName
	}
	p, err := runner.MakeProfile(home, name, kept)
	if err != nil {
		return refused(2, refusal.StartProfileUnusable, startText("%s", err.Error()))
	}
	say("this account held no profile, so one was made: %s", abcdhome.Display("profiles", p.Name()))
	say("a profile is %s: a command a session runs can still read whatever the account can", startNotSandbox)
	say("abcd wrote its own settings there (settings.json, and an empty skills folder closed to writes) and nothing of yours")
	say("abcd never performs the login and never reads, stores or forwards its token; log the host in once, in your own shell (this opens no session):")
	fmt.Fprintf(cmd.OutOrStdout(), "\n  %s\n\n", runner.LoginCommand(p))
	say("no session was started; run abcd start again once the login is done (abcd drain --drive runs on the same profile)")
	return nil
}

// startConventions reads the conventions file at the checkout's root, whole, or says why it
// cannot be handed over. A link there is refused, never followed: it would hand a file from
// outside the checkout to the host as its instructions.
func startConventions(root string) (raw []byte, why string) {
	dir, err := os.OpenRoot(root)
	if err != nil {
		return nil, "could not be read"
	}
	defer dir.Close()
	raw, err = fsutil.ReadGuardedInRoot(dir, loop.ConventionsFile, startConventionsBound)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, "is absent"
	case errors.Is(err, fsutil.ErrNotRegular):
		return nil, "is a link or not a regular file"
	case errors.Is(err, fsutil.ErrTooBig):
		return nil, fmt.Sprintf("holds more than %d bytes", startConventionsBound)
	case err != nil:
		return nil, "could not be read"
	case bytes.IndexByte(raw, 0) >= 0:
		return nil, "holds a NUL byte"
	}
	return raw, ""
}

// startGuard is why the guard of a session opened in cwd would not refuse the command the
// short launch's guard was seen to refuse, "" when it would. The short launch runs in a
// repository of its own, and the open session under the registry its checkout holds: it is
// loaded and asked here as the guard hook does.
func startGuard(cmd *cobra.Command, cwd string) string {
	ld := guard.LoadRepo(rulesRoot(cwd, cmd.ErrOrStderr()))
	switch {
	case ld.Posture == guard.LoadUnavailable:
		return "the guard in this checkout has no hazard registry to load"
	case ld.Registry.Disabled:
		return "the guard in this checkout is switched off in " + guard.RepoRelPath
	}
	dec, err := ld.Registry.ReadingFrom(guard.NewFiles(cwd)).Check(runner.GuardProofCommand())
	if err != nil || dec.Verdict != guard.VerdictBlock {
		return "the guard in this checkout does not refuse the command it is meant to refuse (" + runner.GuardProofCommand() + "), as " + guard.RepoRelPath + " sets it"
	}
	return ""
}

// startHooks is the session's hooks whose every verb root holds, and what each event left
// out is called: a hook is never wired to a verb that would fail.
func startHooks(root *cobra.Command) (hooks []runner.SessionHook, left []string) {
	for _, h := range runner.SessionHooks() {
		held := true
		for _, v := range h.Verbs {
			held = held && findByPath(root, strings.Fields(v)) != nil
		}
		switch {
		case held:
			hooks = append(hooks, h)
		case h.Event == "SubagentStart":
			left = append(left, "a sub-agent's start")
		default:
			left = append(left, "the host's "+h.Event+" event")
		}
	}
	return hooks, left
}

// startReported is the line on what the short launch showed.
func startReported(p runner.SessionProof) string {
	list := func(names []string) string {
		if len(names) == 0 {
			return "none"
		}
		return strings.Join(names, ", ")
	}
	version := p.HarnessVersion
	if version == "" {
		version = "not reported"
	}
	return fmt.Sprintf("the launch reported: tools %s; named and not reported: %s; no server; no key in use; "+
		"the guard refused the command it is meant to refuse; plugins, the host's own: %s; built-in skills (%d): %s; built-in agents (%d): %s; host %s",
		list(p.Reported), list(p.Unreported), list(p.Plugins), len(p.Skills), list(p.Skills), len(p.Agents), list(p.Agents), version)
}
