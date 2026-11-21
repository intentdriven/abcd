package guard

import "testing"

// TestGuardReadsTheCodeAZshGlobQualifierRuns: zsh runs an `e` glob
// qualifier's string, and a `+name` qualifier's command, for every file the
// pattern matches, so the guard reads that code as it reads the string an
// eval runs: a refused command in it is refused, whatever delimiter the
// qualifier takes, behind `(#q…)` too. Qualifiers that run nothing, an `e`
// that is a modifier (`:e`), and code that is harmless are allowed, and the
// headless approver still refuses any glued `(`.
func TestGuardReadsTheCodeAZshGlobQualifierRuns(t *testing.T) {
	r := Defaults()
	for _, line := range []string{
		`ls *(e:'rm -rf ~':)`,
		`ls **/*(e:'rm -rf ~':)`,
		`print -l *(e:'rm -rf /':)`,
		`echo *(e{rm -rf ~})`,
		`echo *(.e|rm -rf ~|)`,
		`echo *(^e[rm -rf ~])`,
		`echo *(oe:'rm -rf ~':)`,
		`echo *(#qe:'rm -rf ~':)`,
		`echo *(u:root:e:'rm -rf ~':)`,
		`echo *(mh-1e:'rm -rf ~':)`,
		`ls *(N,e:'rm -rf ~':)`,
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != "rm-rf-root-or-home" {
			t.Errorf("%q: verdict %q under %q, want a block under rm-rf-root-or-home", line, d.Verdict, d.EntryID)
		}
		if a := approveShell(clean(), NewFiles(t.TempDir()), line); a.Allow {
			t.Errorf("%q: the approver allowed it", line)
		}
	}
	if d, err := r.Check(`ls *(e:'git push --force origin main':)`); err != nil || d.Verdict != VerdictBlock {
		t.Errorf("a force push in a qualifier: verdict %q under %q (%v), want a block", d.Verdict, d.EntryID, err)
	}
	for _, line := range []string{
		`ls *(.)`,
		`ls *(N)`,
		`ls *(/)`,
		`ls *(om[1])`,
		`ls *(.:e)`,
		`ls x(:A)`,
		`ls *(u:eve:)`,
		`ls *(e:'[[ -d $REPLY ]]':)`,
		`ls *(+isdir)`,
		`ls *(f+644)`,
		`ls *(Lk+10)`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}

// TestZshQualifierCodeReadsEachQualifierThatRunsCode pins what the reading
// takes from a qualifier list: the code of each `e` and `+` qualifier, and
// nothing from the qualifiers whose arguments are not code.
func TestZshQualifierCodeReadsEachQualifierThatRunsCode(t *testing.T) {
	for in, want := range map[string][]string{
		`*(e:'rm -rf ~':)`:   {"rm -rf ~"},
		`*(e{rm -rf ~})`:     {"rm -rf ~"},
		`*(+fn)`:             {"fn"},
		`*(oe:a:O+b)`:        {"a", "b"},
		`*(#qe:a:)`:          {"a"},
		`*(u:e:g:e:P:e:)`:    nil,
		`*(f+644)`:           nil,
		`*(a+5m-2c+1L+3l-2)`: nil,
		`*(.:e)`:             nil,
		`*([1,2])`:           nil,
		`*(#i)x`:             nil,
		`*(e:unclosed`:       nil,
		`*(e:"a \"b\"":)`:    {`a "b"`},
		`*(e:a:,e:b:)`:       {"a", "b"},
	} {
		got, _ := zshQualifierCode(in, 1)
		if len(got) != len(want) {
			t.Errorf("%s: got %q, want %q", in, got, want)
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%s: got %q, want %q", in, got, want)
			}
		}
	}
}

// TestGuardRefusesAZshGlobQualifierListItCannotRead: a qualifier list that
// holds an expansion — a variable's value, a substitution's output, unquoted
// or in double quotes — is text zsh reads as qualifiers only once it has
// expanded the word, so the code it runs is not on the line and the command
// is refused. A positional or special parameter is refused only where the
// line can set it: a `set` with operands, a function's call, or a shell's
// arguments. A `$` inside single quotes, or escaped, is text, and stays
// allowed.
func TestGuardRefusesAZshGlobQualifierListItCannotRead(t *testing.T) {
	r := Defaults()
	for _, line := range []string{
		`q='e:echo RAN:'; : *($q)`,
		`: *(${~q})`,
		`ls *($(cat q))`,
		"ls *(`cat q`)",
		`ls *("$q")`,
		`ls *(e:"$q":)`,
		`ls *(.$q)`,
		`sh -c "ls *($q)"`,
		`set -- 'e:echo RAN:'; : *($1)`,
		`sh -c ': *($1)' _ 'e:echo RAN:'`,
		`g() { : *($1); }; g 'e:echo RAN:'`,
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != syntheticEntryID {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, syntheticEntryID)
		}
	}
	for _, line := range []string{
		`ls *(.)`,
		`ls *(e:'[[ -d $REPLY ]]':)`,
		`ls *(e:"[[ -d \$REPLY ]]":)`,
		`ls *(e:'$q':)`,
		`echo $(date)`,
		`f() { echo "$1"; }; f x`,
		`arr=($a $b); echo "${arr[@]}"`,
		`( cd "$d" && ls )`,
		`time ( cd "$d" && ls )`,
		`[[ $x =~ ^($y)$ ]] && echo ok`,
		`(( n = ($a + 1) * 2 ))`,
		`x=$(( ($a + 1) * 2 ))`,
		`index($0, root) == 1 {`,
		`json_view($0)`,
		`set -e; ls *($1)`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}
