package guard

import "testing"

// TestAnAppendOrSubscriptPrefixIsAnAssignment — iss-2610121214249290. bash
// and zsh read NAME+=VALUE, NAME[SUB]=VALUE and NAME[SUB]+=VALUE before a
// command as assignments, and run the command after them (bash 5 reports a
// subscripted prefix as not a valid identifier and still runs it), so the
// command is judged and the variable it sets is read as one. Verified
// against /bin/bash 3.2, bash 5 and /bin/zsh 5.9.
func TestAnAppendOrSubscriptPrefixIsAnAssignment(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`FOO+=x rm -rf /`, VerdictBlock, "rm-rf-root-or-home"},
		{`FOO=x BAR+=y rm -rf /`, VerdictBlock, "rm-rf-root-or-home"},
		{`A[1]=x rm -rf /`, VerdictBlock, "rm-rf-root-or-home"},
		{`A[1]+=x rm -rf /`, VerdictBlock, "rm-rf-root-or-home"},
		{`A[$i]=x rm -rf /`, VerdictBlock, "rm-rf-root-or-home"},
		{`PS4+='$(rm -rf /)' bash -x -c true`, VerdictBlock, "rm-rf-root-or-home"},
		{`GIT_CONFIG_GLOBAL+=/tmp/x git status`, VerdictWarn, "git-config-rewrite-unread"},

		{`FOO+=x echo hi`, VerdictAllow, ""},
		{`FOO+=x`, VerdictAllow, ""},
		{`A[1]=x echo hi`, VerdictAllow, ""},
		{`echo FOO+=x A[1]=y`, VerdictAllow, ""},
		{`PATH+=:/opt/bin go version`, VerdictAllow, ""},
	})
}
