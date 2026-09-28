package guard

import (
	"strings"
	"testing"
)

// TestParameterExpansionIsAnUnknownWord — iss-2609251824244354. What a
// parameter expansion prints is not in the command line any more than a
// command substitution's output is, and bash builds the hazard from either: a
// dash glued to a variable (`--$X`, `-$F`) was read as the literal text `--$X`,
// which names no flag, so every blocker allowed it while its `--$(echo x)`
// twin blocked, and a variable standing as the program name (`$GIT`,
// `${GIT:-git}`) was compared as text. A parameter expansion — `$NAME`, `$1`,
// `$@`, `$*`, `${…}` — now leaves the unknown word's mark where its value
// goes, so each role reads it as it reads a substitution's output (unknown.go).
// A single-quoted or escaped `$` is text, as bash reads it.
func TestParameterExpansionIsAnUnknownWord(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		// Glued to a dash: a flag of unknown name.
		{`git push --$X origin main`, VerdictBlock, "git-push-force"},
		{`git push "--$X" origin main`, VerdictBlock, "git-push-force"},
		{`git push --${X} origin main`, VerdictBlock, "git-push-force"},
		{`git push --${X:-force} origin main`, VerdictBlock, "git-push-force"},
		{`git push "--${X#x}" origin main`, VerdictBlock, "git-push-force"},
		{`git push -$F origin main`, VerdictBlock, "git-push-force"},
		{`git push --$1 origin main`, VerdictBlock, "git-push-force"},
		{`git push --$@ origin main`, VerdictBlock, "git-push-force"},
		{`git commit -m x --$X`, VerdictBlock, "git-commit-no-verify"},
		{`git commit -m x --no-$X`, VerdictBlock, "git-commit-no-verify"},
		{`cd s && rm -$F *`, VerdictBlock, "rm-rf-after-cd-chain"},
		{`cd s && rm -r$F *`, VerdictBlock, "rm-rf-after-cd-chain"},

		// In command position: a program of unknown name.
		{`$GIT push --force origin main`, VerdictBlock, ""},
		{`"$GIT" push --force origin main`, VerdictBlock, ""},
		{`${GIT:-git} push --force origin main`, VerdictBlock, ""},
		{`"${GIT:-git}" push --force origin main`, VerdictBlock, ""},
		{`/usr/bin/$G push --force origin main`, VerdictBlock, ""},
		{`$1 push --force origin main`, VerdictBlock, ""},
		{`timeout 5 $GIT push --force origin main`, VerdictBlock, ""},
		{`$GH repo delete o/r --yes`, VerdictBlock, ""},

		// Inside a string the guard reads as a payload.
		{`sh -c 'git push --$X origin main'`, VerdictBlock, "git-push-force"},
		{`bash -c "git push --\$X origin main"`, VerdictBlock, "git-push-force"},
		{`sh -c '$GIT push --force origin main'`, VerdictBlock, ""},
		{`sh -c "git push --$X origin main"`, VerdictBlock, "git-push-force"},
		{`sh -c "$GIT push --force origin main"`, VerdictBlock, ""},
		{`eval "git push --$X origin main"`, VerdictBlock, "git-push-force"},

		// Quoted or escaped, a dollar is text.
		{`git push '--$X' origin main`, VerdictAllow, ""},
		{`git push --\$X origin main`, VerdictAllow, ""},
		{`git push "--\$X" origin main`, VerdictAllow, ""},
		{`echo '$GIT push --force origin main'`, VerdictAllow, ""},
	})
}

// TestParameterExpansionInOperandPositionStaysAnOperand pins the other half of
// the rule: a variable that is a whole word is read as one operand of unknown
// value, never as every flag, exactly as a whole-word substitution is (allow
// (1) of DECISIONS 2026-09-25). That is how everyday commands spell a branch, a
// message, an API path and a file, and none of them is a new block.
func TestParameterExpansionInOperandPositionStaysAnOperand(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`git push origin "$branch"`, VerdictAllow, ""},
		{`git push origin $BRANCH`, VerdictAllow, ""},
		{`git push -u origin "${BRANCH}"`, VerdictAllow, ""},
		{`git push origin "HEAD:refs/heads/$branch"`, VerdictAllow, ""},
		{`git commit -m "$msg"`, VerdictAllow, ""},
		{`git commit -m "fix: $subject"`, VerdictAllow, ""},
		{`git log --oneline "$base..HEAD"`, VerdictAllow, ""},
		{`git -C "$repo" status --porcelain`, VerdictAllow, ""},
		{`git worktree add "$WT" -b "$BR" origin/main`, VerdictAllow, ""},
		{`gh api "repos/$OWNER/$REPO/pulls/$PR/comments"`, VerdictAllow, ""},
		{`gh api repos/$OWNER/$REPO/actions/runs --jq '.workflow_runs[0].id'`, VerdictAllow, ""},
		{`gh pr view "$PR" --json state`, VerdictAllow, ""},
		{`cd "$DIR" && ls`, VerdictAllow, ""},
		{`rm -f "$tmp"`, VerdictAllow, ""},
		{`kill "$pid"`, VerdictAllow, ""},
		{`kill -- -"$pg"`, VerdictAllow, ""},
		{`echo "${1:-default}"`, VerdictAllow, ""},
		{`printf '%s\n' "$@"`, VerdictAllow, ""},
		{`GOOS=$os GOARCH=$arch go build -o "bin/abcd-$os-$arch" ./cmd/abcd`, VerdictAllow, ""},
		{`export PATH="$HOME/go/bin:$PATH"`, VerdictAllow, ""},
		{`sed -n "${start},${end}p" "$file"`, VerdictAllow, ""},
		{`bash "$script"`, VerdictAllow, ""},
		{`kill -9 $$`, VerdictAllow, ""},

		// A variable's value as a script is not a stream, and as a program
		// name it is not read as a pkill or killall, whose entries name
		// nothing but the program and an operand (variableCarried).
		{`bash "$script"`, VerdictAllow, ""},
		{`. "$ENV_FILE"`, VerdictAllow, ""},
		{`"$GO" build ./...`, VerdictAllow, ""},
		{`$EDITOR notes.md`, VerdictAllow, ""},
		{`"${EDITOR:-vi}" notes.md`, VerdictAllow, ""},
		{`"$SHELL" -c 'echo hi'`, VerdictAllow, ""},

		// A string handed to a shell carries the expansion for that shell to
		// read, as a bare $VAR always read there: no new warn.
		{`bash -c "echo $HOME"`, VerdictAllow, ""},
		{`sh -c "cd $DIR && make test"`, VerdictAllow, ""},
		{`sh -c "$GO build ./..."`, VerdictAllow, ""},
		{`eval "$cmd"`, VerdictAllow, ""},
		{`wait $!; echo $?`, VerdictAllow, ""},
	})
}

// TestParameterExpansionReadingStaysLinear pins the cost of reading parameter
// expansions: each `${` is scanned to its own `}` once, the scans for `${`s
// with no close spend the line's closing-scan budget and no more, and a word
// spelled back for a string is written once.
func TestParameterExpansionReadingStaysLinear(t *testing.T) {
	shapes := map[string]func(int) string{
		"many unclosed ${": func(n int) string {
			return "echo " + strings.Repeat("${a ", n)
		},
		"many unclosed ${ in double quotes": func(n int) string {
			return `echo "` + strings.Repeat("${a ", n) + `"`
		},
		"many closed ${…} and $X": func(n int) string {
			return "echo " + strings.Repeat("${a:-b}$X-${c} ", n)
		},
		"nested ${…}": func(n int) string {
			return "echo " + strings.Repeat("${a:-", n) + "x" + strings.Repeat("}", n)
		},
		"a string holding many variables": func(n int) string {
			return `sh -c "echo ` + strings.Repeat("$X ${Y} ", n) + `"`
		},
	}
	for name, build := range shapes {
		build := build
		t.Run(name, func(t *testing.T) {
			assertWorkGrowth(t, build, 1<<9, "each parameter expansion is read once")
		})
	}
}
