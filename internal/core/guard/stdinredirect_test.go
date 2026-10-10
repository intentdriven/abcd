package guard

import "testing"

// A shell with no -c string and no script operand reads its script from
// standard input, and a redirection can point that at a file
// (adr-2610091150447054 decision 1, iss-2610090932257904). `cat f | bash`
// blocked as a stream while `bash < f` was an allow, because the stream record
// was set for a pipe, a here-document and a here-string, never for a `<`.

func TestShellStdinRedirectedFromAFileIsRead(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"s.sh":  hazardLine,
		"ok.sh": "echo hi",
	})
	runScriptCases(t, dir, []scriptCase{
		{`bash < {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash -s < {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash -s arg1 arg2 < {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`sh 0< {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash <{d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash < s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash <> {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`< {d}/s.sh bash`, VerdictBlock, scriptHazardEntryID},
		{`bash - < {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash /dev/stdin < {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		// A descriptor duplicated onto stdin from a file opened earlier.
		{`bash 3< {d}/s.sh 0<&3`, VerdictBlock, scriptHazardEntryID},
		{`exec 3< {d}/s.sh; bash <&3`, VerdictBlock, scriptHazardEntryID},
		// An exec that redirects the shell's own stdin.
		{`exec < {d}/s.sh; bash`, VerdictBlock, scriptHazardEntryID},
		{`exec 0< {d}/s.sh && sh`, VerdictBlock, scriptHazardEntryID},
		// What the shell running a string reads is what its commands read.
		{`sh -c 'bash' < {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		// Written then run.
		{`printf '%s\n' '` + hazardLine + `' > {d}/n.sh; bash < {d}/n.sh`, VerdictBlock, scriptWrittenEntryID},

		{`bash < {d}/ok.sh`, VerdictAllow, ""},
		{`bash < /dev/null`, VerdictAllow, ""},
		{`bash < {d}/absent`, VerdictAllow, ""},
		// The script is the operand or the -c string, and stdin is its data.
		{`bash -c 'cat' < {d}/s.sh`, VerdictAllow, ""},
		{`bash {d}/ok.sh < {d}/s.sh`, VerdictAllow, ""},
		{`cat < {d}/s.sh`, VerdictAllow, ""},
		{`bash 3< {d}/s.sh`, VerdictAllow, ""},
		{`bash 0<&3 3< {d}/s.sh`, VerdictAllow, ""},
		{`bash < "$IN"`, VerdictAllow, ""},
	})
}
