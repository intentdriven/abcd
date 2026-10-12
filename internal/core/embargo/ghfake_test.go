package embargo

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// ghFake is a stand-in `gh` at the front of PATH. Every remote call the
// embargo verbs make goes through it, so no test reaches GitHub. It records
// each call's argv and stdin, and answers from fixture JSON keyed on the
// request's method, path and state parameter; a key given several numbered
// answers serves them in order, then repeats the last.
type ghFake struct {
	t   *testing.T
	dir string
}

func newGHFake(t *testing.T) *ghFake {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	dir := t.TempDir()
	for _, sub := range []string{"resp", "calls", "seq"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	script := `#!/usr/bin/env bash
set -eu
d='` + dir + `'
n=$(( $(cat "$d/count" 2>/dev/null || echo 0) + 1 ))
printf '%s' "$n" >"$d/count"
printf '%s\n' "$@" >"$d/calls/$n.argv"
cat >"$d/calls/$n.stdin"
[ "${1:-}" = api ] && shift
method=GET; path=; state=
while [ $# -gt 0 ]; do
  case "$1" in
    --hostname|-H) shift 2 ;;
    -X|--method) method="$2"; shift 2 ;;
    -f|-F) case "$2" in state=*) state="${2#state=}" ;; esac; shift 2 ;;
    --input) shift 2 ;;
    --paginate|--slurp) shift ;;
    *) path="$1"; shift ;;
  esac
done
key=$(printf '%s' "$method|$path|$state" | tr -c 'A-Za-z0-9._-' '_')
k=$(( $(cat "$d/seq/$key" 2>/dev/null || echo 0) + 1 ))
printf '%s' "$k" >"$d/seq/$key"
if [ -f "$d/resp/$key.err" ]; then cat "$d/resp/$key.err" >&2; exit 1; fi
if [ -f "$d/resp/$key.$k" ]; then cat "$d/resp/$key.$k"; exit 0; fi
last=$(ls "$d/resp" | grep -E "^$key\.[0-9]+$" | sed 's/.*\.//' | sort -n | tail -1 || true)
if [ -n "$last" ]; then cat "$d/resp/$key.$last"; exit 0; fi
if [ -f "$d/resp/$key" ]; then cat "$d/resp/$key"; exit 0; fi
echo "fake gh: no answer for $method $path $state" >&2
exit 97
`
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return &ghFake{t: t, dir: dir}
}

func ghKey(method, path, state string) string {
	b := []byte(method + "|" + path + "|" + state)
	for i, c := range b {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			b[i] = '_'
		}
	}
	return string(b)
}

// answer sets the response to a request; n > 0 makes it the n-th answer.
func (g *ghFake) answer(method, path, state string, n int, body string) {
	g.t.Helper()
	name := ghKey(method, path, state)
	if n > 0 {
		name += "." + strconv.Itoa(n)
	}
	if err := os.WriteFile(filepath.Join(g.dir, "resp", name), []byte(body), 0o600); err != nil {
		g.t.Fatal(err)
	}
}

// fail makes a request exit non-zero with stderr.
func (g *ghFake) fail(method, path, state, stderr string) {
	g.t.Helper()
	if err := os.WriteFile(filepath.Join(g.dir, "resp", ghKey(method, path, state)+".err"), []byte(stderr), 0o600); err != nil {
		g.t.Fatal(err)
	}
}

type ghCall struct {
	argv  []string
	stdin string
}

func (c ghCall) method() string {
	for i, a := range c.argv {
		if (a == "-X" || a == "--method") && i+1 < len(c.argv) {
			return c.argv[i+1]
		}
	}
	return "GET"
}

func (c ghCall) has(arg string) bool {
	for _, a := range c.argv {
		if a == arg {
			return true
		}
	}
	return false
}

// calls returns every call made, in order.
func (g *ghFake) calls() []ghCall {
	g.t.Helper()
	entries, err := os.ReadDir(filepath.Join(g.dir, "calls"))
	if err != nil {
		g.t.Fatal(err)
	}
	var nums []int
	for _, e := range entries {
		if n, ok := strings.CutSuffix(e.Name(), ".argv"); ok {
			i, err := strconv.Atoi(n)
			if err != nil {
				g.t.Fatal(err)
			}
			nums = append(nums, i)
		}
	}
	sort.Ints(nums)
	var out []ghCall
	for _, n := range nums {
		argv, err := os.ReadFile(filepath.Join(g.dir, "calls", strconv.Itoa(n)+".argv"))
		if err != nil {
			g.t.Fatal(err)
		}
		stdin, _ := os.ReadFile(filepath.Join(g.dir, "calls", strconv.Itoa(n)+".stdin"))
		out = append(out, ghCall{argv: strings.Split(strings.TrimRight(string(argv), "\n"), "\n"), stdin: string(stdin)})
	}
	return out
}

// writes returns every call that is not a GET.
func (g *ghFake) writes() []ghCall {
	var out []ghCall
	for _, c := range g.calls() {
		if c.method() != "GET" {
			out = append(out, c)
		}
	}
	return out
}
