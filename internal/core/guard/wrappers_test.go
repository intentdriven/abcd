package guard

import (
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
)

// DERIVED FROM THE BINARY, NOT FROM A DOCUMENT.
//
// gh-299 is why. Its git value-flag list was taken from the bug report, and it
// was wrong three ways — it omitted `--shallow-file` (a live force-push bypass,
// in git since 1.9) and counted two non-value flags as value-taking. It totalled
// nine either way, so a size assertion certified the wrong list as complete.
//
// The wrapper table repeats that shape at a larger scale, and documentation is
// not a way out: `nsenter --help` renders `-S/--setuid` as optional-argument and
// `unshare --help` renders the same letter, same package, same version, as
// required-argument, while BOTH binaries consume the following token. nsenter's
// help contradicts its own parser.
//
// So the help text is used for exactly one thing — ENUMERATING which flags exist
// — and never for classifying them. Over-enumerating is free: a flag that turns
// out to be boolean is simply confirmed as one. Under-enumerating is the defect,
// and a hand-written candidate list is under-enumeration by construction. An
// earlier draft of this file hand-listed candidates and missed `nice
// --adjustment`, `runuser -s` and `runuser -w`, each of which quietly degraded a
// precise block to a warn on a real command line.
type wrapperProbe struct {
	// name is the program, and the key in the wrappers map.
	name string
	// operands is how many mandatory non-flag operands sit between the wrapper's
	// options and the command it launches (`chrt PRIORITY COMMAND`).
	operands int
	// operandArgs supplies those operands for the probe, and any flags the wrapper
	// needs before it will run at all. It doubles as the CONTROL probe: if the
	// wrapper cannot run this, nothing it says about a flag is evidence.
	operandArgs []string
	// extraCandidates are flags to classify beyond the ones --help enumerates, for
	// a binary whose help is incomplete.
	extraCandidates []string
	// values supplies a known-good value per flag; anything unlisted is probed
	// with genericProbeValues.
	values map[string]string
	// unprobeable names flags this environment cannot classify, each with the
	// reason. A flag may be unclassified ONLY by being written down here — silence
	// is what gh-299 shipped, and "we could not check it" is a different statement
	// from "we did not think about it". A reason is not a licence: when the probe
	// classifies the flag anyway, it is still asserted.
	unprobeable map[string]string
}

// wrapperProbes covers every wrapper the guard steps whose grammar can be
// re-derived by running it. Platform: these are the Linux / util-linux /
// coreutils spellings. Several do not exist on macOS at all (`nsenter`,
// `unshare`, `chrt`, `taskset`, `ionice`, `setsid`, `flock`, `stdbuf`,
// `runuser`) and others carry BSD grammars there — macOS `su -c` means
// `-c class`, not a command string. A probe skips a binary it cannot find, so
// this file asserts on Linux and abstains elsewhere; the table itself is Linux's.
var wrapperProbes = []wrapperProbe{
	{name: "nice", values: map[string]string{"-n": "5", "--adjustment": "5"}},
	{name: "setsid", unprobeable: map[string]string{
		"-c": "--ctty needs a controlling terminal; a test process has none " +
			"(`setsid -c /bin/echo hi` answers \"failed to set the controlling terminal\"). It is a boolean.",
		"--ctty": "same as -c.",
	}},
	{name: "stdbuf", values: map[string]string{
		"-i": "0", "-o": "0", "-e": "0", "--input": "0", "--output": "0", "--error": "0"}},
	{name: "ionice", values: map[string]string{
		"-c": "3", "-n": "4", "--class": "3", "--classdata": "4"},
		unprobeable: map[string]string{
			"-p": "--pid, -P/--pgid and -u/--uid all replace the grammar: they retune a " +
				"RUNNING process and launch no command, so no probe of theirs can say anything " +
				"about the walk to command position.",
			"--pid": "same as -p.", "-P": "same as -p.", "--pgid": "same as -p.",
			"-u": "same as -p.", "--uid": "same as -p.",
		}},
	{name: "eatmydata"},
	{name: "chrt", operands: 1, operandArgs: []string{"-f", "1"},
		values: map[string]string{"-T": "100000", "-P": "1000000", "-D": "0"},
		unprobeable: map[string]string{
			"-T": "--sched-runtime is accepted only with SCHED_DEADLINE, which cannot be set here. " +
				"It consumes its value: `chrt -T -f 1 /bin/echo hi` answers \"invalid runtime argument: '-f'\".",
			"--sched-runtime":  "same as -T.",
			"-P":               "--sched-period, same restriction and same evidence as -T.",
			"--sched-period":   "same as -P.",
			"-D":               "--sched-deadline, same restriction as -T.",
			"--sched-deadline": "same as -D.",
			"-a":               "--all-tasks only applies with -p PID, a grammar that launches no command.",
			"--all-tasks":      "same as -a.",
			"-p":               "--pid operates on a running process and launches no command.",
			"--pid":            "same as -p.",
			"-R":               "--reset-on-fork modifies a policy flag rather than standing alone.",
			"--reset-on-fork":  "same as -R.",
			"-m":               "--max prints the priority range and exits without launching anything.",
			"--max":            "same as -m.",
		}},
	// The mask operand is "1", valid both as a bitmask and as a -c CPU list, so -c
	// can be classified without the operand confounding it.
	{name: "taskset", operands: 1, operandArgs: []string{"1"},
		unprobeable: map[string]string{
			"-p":    "--pid replaces the grammar entirely: it operates on a running process and launches no command.",
			"--pid": "same as -p.",
		}},
	{name: "flock", operands: 1, operandArgs: []string{"/tmp"},
		values: map[string]string{"-w": "1", "-E": "9", "--timeout": "1", "--wait": "1", "--conflict-exit-code": "9"},
		unprobeable: map[string]string{
			"-c":            "--command is the exec-string grammar, owned by execstring.go rather than the wrapper walk.",
			"--command":     "same as -c.",
			"-F":            "--no-fork changes how the command is executed, not how arguments are parsed; boolean.",
			"--no-fork":     "same as -F.",
			"-o":            "--close is a boolean this environment cannot distinguish.",
			"--close":       "same as -o.",
			"--fcntl":       "a locking mode; boolean.",
			"--no-own-file": "boolean.",
			"--verbose":     "diagnostic only.",
		}},
	{name: "chroot", operands: 1, operandArgs: []string{"/"},
		values: map[string]string{"--userspec": "0:0", "--groups": "0"}},
	{name: "unshare",
		values: map[string]string{"-S": "0", "-G": "0", "-w": "/tmp", "-R": "/",
			"--setuid": "0", "--setgid": "0", "--wd": "/tmp", "--root": "/",
			"--map-user": "0", "--map-group": "0", "--map-users": "0:0:1", "--map-groups": "0:0:1",
			"--propagation": "private", "--setgroups": "deny", "--monotonic": "0", "--boottime": "0"},
		unprobeable: map[string]string{
			"-S": "setuid needs privileges this process lacks (`unshare -S 0 /bin/echo hi` answers " +
				"\"setuid failed: Operation not permitted\"). Probed as root it CONSUMES its value.",
			"--setuid":      "same as -S.",
			"-G":            "setgid, same requirement as -S; consumes its value.",
			"--setgid":      "same as -G.",
			"-R":            "--root needs a chroot this process cannot perform; consumes its value.",
			"--root":        "same as -R.",
			"--setgroups":   "needs a user namespace this process cannot create; documented required-argument and listed on that basis.",
			"--map-user":    "classifiable only where user namespaces are unrestricted (this container as root: CONSUMES its value); an AppArmor-restricted runner cannot run either probe form.",
			"--map-group":   "same as --map-user.",
			"--map-users":   "needs a user namespace; consumes its value.",
			"--map-groups":  "needs a user namespace; consumes its value.",
			"--monotonic":   "needs a time namespace; consumes its value.",
			"--boottime":    "needs a time namespace; consumes its value.",
			"--propagation": "needs a mount namespace; consumes its value.",
			"--kill-child":  "optional-argument, and the child cannot be observed here.",
			"--keep-caps":   "needs a user namespace; boolean.",
			"-r":            "--map-root-user needs a user namespace; boolean.",
			"--map-auto":    "needs a user namespace; boolean.",
			// The namespace flags themselves. As root in a privileged container the
			// probe classifies every one of these as boolean; unprivileged — which is
			// what a CI runner is — unshare cannot create the namespace at all, so
			// neither probe form runs and the flag is inconclusive. NONE is listed,
			// and being wrong about one would over-step rather than under-step, which
			// is the direction that creates a miss.
			"-m":           "a namespace flag: unprivileged, unshare cannot create the namespace, so neither probe form runs. Optional-argument or boolean; not listed.",
			"--mount":      "same as -m.",
			"-u":           "see -m.",
			"--uts":        "see -m.",
			"-i":           "see -m.",
			"--ipc":        "see -m.",
			"-n":           "see -m.",
			"--net":        "see -m.",
			"-p":           "see -m.",
			"--pid":        "see -m.",
			"-C":           "see -m.",
			"--cgroup":     "see -m.",
			"-T":           "see -m.",
			"--time":       "see -m.",
			"--mount-proc": "see -m.",
		}},
	{name: "nsenter",
		values: map[string]string{"-t": "1", "-S": "0", "-G": "0", "-W": "/",
			"--target": "1", "--setuid": "0", "--setgid": "0", "--wd": "/"},
		unprobeable: map[string]string{
			"-S":       "setuid needs privileges this process lacks; probed as root it CONSUMES its value.",
			"--setuid": "same as -S.",
			"-G":       "setgid, same requirement; consumes its value.",
			"--setgid": "same as -G.",
			"-t":       "--target needs an enterable process; consumes its value.",
			"--target": "same as -t.",
			"-a":       "--all and the per-namespace flags (-m -u -i -n -p -U -C -T -e -r -w and their long forms) need a target namespace this process cannot enter, so neither form of the probe runs. Every one is optional-argument or boolean and NONE is listed; -W is the one that was verified, because its long form is where nsenter's help contradicts its parser.",
			"--all":    "same as -a.",
			"-m":       "see -a.", "--mount": "see -a.",
			"-u": "see -a.", "--uts": "see -a.",
			"-i": "see -a.", "--ipc": "see -a.",
			"-n": "see -a.", "--net": "see -a.",
			"-p": "see -a.", "--pid": "see -a.",
			"-U": "see -a.", "--user": "see -a.",
			"-C": "see -a.", "--cgroup": "see -a.",
			"-T": "see -a.", "--time": "see -a.",
			"-e": "see -a.", "--env": "see -a.",
			"-r": "see -a.", "--root": "see -a.",
			"-w": "see -a.", "--wd": "see -a.",
		}},
	{name: "runuser", operandArgs: []string{"-u", "root"},
		values: map[string]string{"-u": "root", "-g": "root", "-G": "root", "-s": "/bin/sh",
			"--user": "root", "--group": "root", "--supp-group": "root", "--shell": "/bin/sh",
			"-w": "PATH", "--whitelist-environment": "PATH"},
		unprobeable: map[string]string{
			"-c":                     "--command is the exec-string grammar, owned by execstring.go.",
			"--command":              "same as -c.",
			"--session-command":      "the exec-string grammar, owned by execstring.go.",
			"-l":                     "--login replaces the grammar: it starts a login shell rather than running a command.",
			"--login":                "same as -l.",
			"-p":                     "--preserve-environment is a boolean this environment cannot distinguish.",
			"--preserve-environment": "same as -p.",
			"-m":                     "same as -p.",
			"-P":                     "--pty needs a terminal this process has none of.",
			"--pty":                  "same as -P.",
			"-s": "--shell IS value-taking (it is listed, and execstring.go's own table names it for the same binary), " +
				"but both probe forms run here: with a value the shell is set, without one /bin/echo is accepted AS the shell " +
				"and still prints the marker. Listed on the documented grammar and on that agreement between two tables.",
			"--shell": "same as -s.",
			"-f":      "--fast is a boolean passed through to the shell.",
			"--fast":  "same as -f.",
		}},
}

// probeMarker is what the launched command prints.
const probeMarker = "abcd-guard-probe"

// genericProbeValues are tried for a flag with no known-good value: if the
// wrapper runs the command with one of them in place, the flag consumed it.
var genericProbeValues = []string{"1", "0", "/tmp", "root", "PATH"}

// helpFlagPattern matches a flag as `--help` renders it. Only the NAME is taken;
// whether help shows an argument is deliberately ignored, since that is precisely
// the claim nsenter's own help gets wrong.
var helpFlagPattern = regexp.MustCompile(`(^|[\s,])(--?[A-Za-z][A-Za-z0-9-]*)`)

// helpFlags enumerates a binary's flag names from its own --help. Enumeration is
// all the help text is trusted for.
func helpFlags(name string) []string {
	out, _ := exec.Command(name, "--help").CombinedOutput()
	seen := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") {
			continue // a description line, not an option line
		}
		for _, m := range helpFlagPattern.FindAllStringSubmatch(trimmed, -1) {
			switch flag := m[2]; flag {
			case "-", "--", "-h", "--help", "-V", "--version":
			default:
				seen[flag] = true
			}
		}
	}
	flags := make([]string, 0, len(seen))
	for f := range seen {
		flags = append(flags, f)
	}
	sort.Strings(flags)
	return flags
}

// probeDir is a throwaway working directory for the wrapper probes. Some probe
// values land in an operand position that the wrapper opens as a file — flock
// treats its FILE operand as a lock file and creates it — so without a cwd
// outside the source tree, `go test` would write junk files (`0`, `1`, `PATH`,
// `root`) into internal/core/guard and they would be committed. It is created
// once per test process; the OS reclaims it.
var probeDir = sync.OnceValue(func() string {
	d, err := os.MkdirTemp("", "guard-wrapper-probe-")
	if err != nil {
		return os.TempDir()
	}
	return d
})

// runsCommand reports whether the wrapper launched the command, which is the only
// discriminator that does not depend on reading a message.
func runsCommand(name string, args ...string) bool {
	cmd := exec.Command(name, args...)
	cmd.Dir = probeDir()
	out, err := cmd.CombinedOutput()
	return err == nil && strings.Contains(string(out), probeMarker)
}

// consumesNextToken reports whether the wrapper reads the token after flag as
// that flag's value:
//
//	<wrapper> <flag> <value> <operandArgs...> /bin/echo MARKER   ran => value-taking
//	<wrapper> <flag> <operandArgs...> /bin/echo MARKER           ran => NOT value-taking
//
// The flag goes BEFORE the mandatory operands, because these wrappers stop
// parsing options at the first operand: `chrt -f 1 -T 100000 /bin/echo hi`
// answers "failed to execute -T", reading the flag as the command.
//
// The two forms must disagree. If the wrapper refuses both — a permission it
// lacks, a value it dislikes — the probe learned nothing and says so rather than
// guessing, because guessing is what gh-299 cost.
func consumesNextToken(p wrapperProbe, flag string) (consumes, learned bool) {
	form := func(head ...string) []string {
		args := append([]string{}, head...)
		args = append(args, p.operandArgs...)
		return append(args, "/bin/echo", probeMarker)
	}

	ranWithout := runsCommand(p.name, form(flag)...)

	values := genericProbeValues
	if v, ok := p.values[flag]; ok {
		values = append([]string{v}, genericProbeValues...)
	}
	ranWith := false
	for _, v := range values {
		if runsCommand(p.name, form(flag, v)...) {
			ranWith = true
			break
		}
	}

	switch {
	case ranWith && !ranWithout:
		return true, true // the value was needed: the flag consumed it
	case ranWithout && !ranWith:
		return false, true // the extra token broke it: the flag took nothing
	default:
		return false, false // both or neither: inconclusive
	}
}

// TestWrapperValueFlagsMatchTheInstalledBinaries is the gh-299 lesson applied
// before the bug rather than after it. Every flag the table claims consumes a
// value must actually consume one, and — the direction gh-299's first fix left
// inert — every flag the probe finds value-taking must be in the table.
func TestWrapperValueFlagsMatchTheInstalledBinaries(t *testing.T) {
	// The table encodes the LINUX grammar, and a same-named binary elsewhere is
	// not evidence about it: macOS ships `nice` and `chroot` with BSD grammars
	// (no --adjustment, no --userspec), so probing them there reports the Linux
	// table wrong for being the Linux table. LookPath alone cannot draw this
	// line, because the trap is precisely the binaries that DO exist.
	if runtime.GOOS != "linux" {
		t.Skipf("the wrapper table encodes the Linux grammar; %s binaries that share a name carry different grammars", runtime.GOOS)
	}
	for _, p := range wrapperProbes {
		p := p
		t.Run(p.name, func(t *testing.T) {
			if _, err := exec.LookPath(p.name); err != nil {
				t.Skipf("%s is not installed here; the table cannot be re-derived (it encodes the Linux grammar)", p.name)
			}
			if !wrappers[p.name] {
				t.Fatalf("%s is probed but absent from the wrappers map: the probe and the table have diverged", p.name)
			}
			if got := wrapperOperands[p.name]; got != p.operands {
				t.Errorf("wrapperOperands[%q] = %d, probe expects %d mandatory operands", p.name, got, p.operands)
			}

			// The control probe, DERIVED rather than declared. If the wrapper cannot
			// run its own baseline here, nothing it says about a flag is evidence — and
			// a privilege the runner lacks must not be mistaken for a classification.
			// `chrt -f 1` needs CAP_SYS_NICE, which a GitHub-hosted runner does not
			// have; an earlier draft asserted straight through that and would have
			// failed on every non-root Linux, which is every CI leg this test exists for.
			control := append(append([]string{}, p.operandArgs...), "/bin/echo", probeMarker)
			if !runsCommand(p.name, control...) {
				t.Skipf("%s cannot run its own baseline here (`%s %s`); nothing it says about a flag is evidence",
					p.name, p.name, strings.Join(control, " "))
			}

			listed := wrapperValueFlags[p.name]
			// Every flag the binary documents, plus anything already in the table, so
			// a listed flag the help omits is still checked.
			candidates := append(helpFlags(p.name), p.extraCandidates...)
			candidates = append(candidates, listed...)

			var probed int
			seen := map[string]bool{}
			for _, flag := range candidates {
				if seen[flag] {
					continue
				}
				seen[flag] = true

				consumes, learned := consumesNextToken(p, flag)
				inTable := containsString(listed, flag)
				switch {
				case !learned:
					why, ok := p.unprobeable[flag]
					switch {
					case ok:
						t.Logf("%s %s: not probed here — %s (table claims value-taking=%v)", p.name, flag, why, inTable)
					case inTable:
						// The table asserts this flag consumes a value and nothing can
						// verify or excuse that here: a claim with no evidence is the
						// gh-299 shape, so it fails.
						t.Errorf("%s %s: LISTED as value-taking but the probe could not classify it and no reason is recorded.\n"+
							"Give it a known-good value, or record WHY it cannot be probed here.", p.name, flag)
					default:
						// An unlisted flag this environment cannot classify. Not a
						// failure, deliberately: --help's flag set varies by version
						// (the Ubuntu runner's unshare documents --map-current-user;
						// this container's does not), so erroring here couples CI to
						// one util-linux release. The exposure is bounded by adr-42's
						// own design — an unlisted value flag displaces the command,
						// no entry matches, and Tier 2 warns instead of allowing — and
						// the probe still fails loudly whenever it CAN classify an
						// unlisted flag as value-taking.
						t.Logf("%s %s: unclassifiable here and unlisted — a miss degrades to a Tier 2 warn, not a silent allow; classify or excuse it when adding it to the table", p.name, flag)
					}
				case consumes && !inTable:
					probed++
					t.Errorf("%s %s consumes the following token but is NOT in wrapperValueFlags.\n"+
						"Its value is read as the command, so every entry misses: `%s %s <value> <hazard>` is a silent allow.",
						p.name, flag, p.name, flag)
				case !consumes && inTable:
					probed++
					t.Errorf("%s %s takes NO value but IS in wrapperValueFlags.\n"+
						"The walk steps over the command itself, which is a false negative — listing a flag is not the safe direction.",
						p.name, flag)
				default:
					probed++
				}
			}
			t.Logf("%s: %d flags classified against the installed binary", p.name, probed)
		})
	}
}

// TestEveryProbedWrapperIsInTheTable holds the other half on every platform,
// including the macOS leg where nothing can be probed: a wrapper this file knows
// about must be one the matcher steps, and a wrapper the matcher steps must have
// a probe or a stated exemption.
func TestEveryProbedWrapperIsInTheTable(t *testing.T) {
	for _, p := range wrapperProbes {
		if !wrappers[p.name] {
			t.Errorf("%s is probed but missing from the wrappers map", p.name)
		}
	}
	for name := range wrappers {
		if containsString(wrappersWithoutProbes, name) {
			continue
		}
		found := false
		for _, p := range wrapperProbes {
			if p.name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s is stepped by the matcher but has no probe and no exemption: add one, or say why it needs none", name)
		}
	}
}

// wrappersWithoutProbes are the wrappers whose grammar this file does not
// re-derive, each for a stated reason rather than by omission.
var wrappersWithoutProbes = []string{
	// Running sudo/doas in a test either prompts, escalates, or both. Their flag
	// lists predate this file and come from their own documented grammars.
	"sudo", "doas",
	// Shell builtins, not programs: `command` and `exec` are executed by the
	// shell, and there is no binary to probe.
	"command", "exec",
	// `env`'s -S is owned by the payload pre-pass, not by the wrapper walk, and
	// its other flags are probed there.
	"env",
	// POSIX-stable and value-flag-free (`nohup`) or already covered by their own
	// entries in the shipped table.
	"nohup", "time", "xargs", "timeout",
	// A multiplexer, not a wrapper with flags: its first operand is the applet, so
	// stepping it leaves `sh -c …` in command position for the interpreter path.
	// Probing it would classify busybox's own applets, not a grammar.
	"busybox",
	// A loader shim with no options of its own.
	"proxychains",
	// zsh precommand modifiers, not programs: `noglob`/`nocorrect` are reserved
	// words the shell interprets, there is no binary to probe, and each takes no
	// options of its own — the token after it is command position (like `command`
	// and `exec` above). iss-2608270655497992.
	"noglob", "nocorrect",
	// A bash builtin, not a program: `builtin <name>` runs the shell builtin of
	// that name, takes no options, and there is no binary to probe.
	"builtin",
}
