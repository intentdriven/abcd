package guard

import "testing"

// TestArithmeticWithAnUnfixedOperandIsRefused: every place bash evaluates
// text as arithmetic is refused when its operand is anything but a literal
// number, quoted or not, since a name there can carry an index holding a
// command substitution the line never names.
func TestArithmeticWithAnUnfixedOperandIsRefused(t *testing.T) {
	for _, cmd := range []string{
		`[[ 'a[$(touch x)]' -eq 1 ]]`,
		`n=$(cat VERSION); [[ $n -gt 0 ]] && echo yes`,
		`[[ n -lt 3 ]]`,
		`[[ -v 'a[$(touch x)]' ]]`,
		`for ((i=0; i<n; i++)); do echo $i; done`,
		`for ((i=0; i<3; i++)); do echo $i; done`,
		`(( n > 0 )) && echo yes`,
		`(( 'a[$(touch x)]' ))`,
		`echo $(( n + 1 ))`,
		`echo "$(( n + 1 ))"`,
		`echo $(( (1+(2)) + n ))`,
		`echo $[ n + 1 ]`,
		`echo $(( 1 + 2 )`,
		`let n=n+1`,
		`x=1; let x++`,
		`declare -i n=$v`,
		`local -ai n`,
		`typeset -n ref=$v`,
		`export -n FOO; declare -x -i n=1`,
		`a[$i]=1`,
		`echo "${a[$i]}"`,
		`echo ${a["$(touch x)"]}`,
		`a=([$i]=1)`,
		`a=( [1]=x [$i]=y )`,
		`echo ${v:$o}`,
		`echo ${v:0:n}`,
		`echo ${!n}`,
		`printf -v "$n" '%s' x`,
		`printf -v 'a[$(touch x)]' '%s' x`,
		`read "$n"`,
		`unset "$n"`,
		`local "$n=1"`,
		`wait -p "$n" 1`,
		`[ -v "$n" ]`,
		`test -v 'a[$(touch x)]'`,
		"cat > s.sh <<'EOF'\nfor ((i=0; i<n; i++)); do :; done\nEOF",
		"[[ $n -e\\\nq 1 ]]",
		"(\\\n( n ))",
		"echo $(\\\n( n ))",
		`echo "${@:$n}"`,
		`echo ${*:$n} ${-:$n}`,
		`echo ${a[@]:$n}`,
		`echo ${a[0]:$n}`,
		`echo ${a[$i]}`,
		`"let" "x=$n"`,
		`l''et "x=$n"`,
		`\let "x=$n"`,
		`"declare" -i "x=$n"`,
		`"read" "$n" <<< 1`,
		`"printf" -v "$n" x`,
		`if true; then "let" x=1; fi`,
		`command "let" x=1`,
		"shopt -s expand_aliases\nalias c=rm\nc -rf x",
		"set -o posix\nalias c=rm\nc -rf x",
		`alias c='python3 -c'`,
		`x=1; alias c=rm`,
		"sh -c 'alias c=rm\nc -rf x'",
		`bash -c 'shopt -s expand_aliases; alias c=rm'`,
		`POSIXLY_CORRECT=1 bash -c 'echo'`,
		`eval $'\x28\x28 n \x29\x29'`,
		`bash -c $'\050\050n))'`,
		`unset 'a[n]'`,
		`declare 'a[n]=1'`,
		`awk '{ c[$1]+=1 }'`,
		`"test" -v 'a[$(touch x)]'`,
		`\test -v 'a[$(touch x)]'`,
		`"[" -v 'a[$(touch x)]' ]`,
		`\[ -v 'a[$(touch x)]' ]`,
		"echo \\\\\n\"let\" \"w=$n\"",
		`grep -q 'a$' f; eval $'\x28\x28 n \x29\x29'`,
		`$'let' x=$n`,
		`$'alias' c=rm`,
	} {
		if why := arithmeticUnfixed(cmd); why == "" {
			t.Errorf("arithmeticUnfixed(%q) = \"\"; want it refused", cmd)
		}
	}
}

// TestLiteralArithmeticAndPlainCommandsPass: arithmetic on literal numbers,
// and the commands that only look like it, are left to the rest of the check.
func TestLiteralArithmeticAndPlainCommandsPass(t *testing.T) {
	for _, cmd := range []string{
		`echo $((1 + 2))`,
		`echo "$(( (3 * 4) % 5 ))"`,
		`(( 1 < 2 )) && echo yes`,
		`[[ 1 -eq 1 ]]`,
		`[[ -n "$x" ]] && echo ok`,
		`[[ $a == "$b" ]]`,
		`[ "$n" -gt 0 ] && echo yes`,
		`test "$n" -eq 1`,
		`[ -f x ] && rm x`,
		`echo ${v:-default} ${v:=x} ${v:+y} ${v:?z}`,
		`echo ${v:2:3} ${v: -1}`,
		`echo ${#v} ${a[@]} ${a[*]} ${a[0]}`,
		`a[1]=x; a=( [2]=y )`,
		`ls log[0-9].txt`,
		`while read -r line; do echo "$line"; done < "$f"`,
		`read -r x <<< "$y"`,
		`mapfile -t lines < "$f"`,
		`local x="$a $b"; export PATH="$HOME/bin:$PATH"`,
		`unset FOO BAR`,
		`printf -v out '%s' "$x"`,
		`wait "$pid"`,
		`git read-tree HEAD`,
		`echo $(echo hi)`,
		`( (cd sub && make) )`,
		`grep -n 'func[A-Z]' f`,
		`rg 'Test[A-Z]\w+'`,
		`git log --grep 'fix[a-z]'`,
		`ls case[a-z].txt`,
		`awk '{ c[$1]++ }'`,
		`go test -run 'Test[A-Z]+' ./...`,
		`git commit -m "fix: map[string]int in a[b]c"`,
		`grep "read" f; echo "set"`,
		`printf '%s\n' $'a\tb\n'`,
		`echo ${line:0:20} "${arr[@]}" "${arr[1]}"`,
		`unset 'a[1]'`,
		"echo a \\\n  b",
		`git commit -m 'fix: alias resolution for users'`,
		`grep -rn alias internal/`,
		`rg 'alias ' .`,
		`git commit -m "chore: unset the [deprecated] flag"`,
		"git commit -F - <<'EOF'\nread [x] first\nEOF",
		`read -r -p "Continue [y/n]? " ans`,
		"cat > a.ts <<'EOF'\nexport const xs = [1, 2];\nEOF",
		"cat > a.py <<'EOF'\np.add_argument(\"command\")\np.add(\"set\")\nEOF",
	} {
		if why := arithmeticUnfixed(cmd); why != "" {
			t.Errorf("arithmeticUnfixed(%q) = %q; want it left alone", cmd, why)
		}
	}
}

// TestApproveRefusesArithmeticOnAValueTheLineDoesNotFix: the lines a review
// ran end to end, where bash ran a command the line never names, are refused
// by the approver, and a comparison through `[` stays approved.
func TestApproveRefusesArithmeticOnAValueTheLineDoesNotFix(t *testing.T) {
	r := plainRig(t)
	arith := refusedAs(arithmeticEntryID)
	r.run(t, []approveCase{
		{`[[ 'a[$(touch x)]' -eq 1 ]]`, arith},
		{`n=$(cat VERSION); [[ $n -gt 0 ]] && echo yes`, arith},
		{`n=$(cat VERSION); for ((i=0; i<n; i++)); do echo $i; done`, arith},
		{`n=$(cat VERSION); (( n > 0 )) && echo yes`, arith},
		{`(( 'a[$(touch x)]' ))`, arith},
		{`n=$(cat VERSION); [ "$n" -gt 0 ] && echo yes`, nil},
		{`echo $((1 + 2))`, nil},
	})
}
