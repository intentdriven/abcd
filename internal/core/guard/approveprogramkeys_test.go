package guard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestApproveRefusesEachProgramSettingGitDocuments: each setting git 2.52's
// git-config(1) or Git LFS's git-lfs-config(5) documents as a command, a
// program path or a helper name, or as a switch that makes git start a
// program, is refused as the list's other keys are: a local `git config`
// write, in any case, and a git command that sets it for itself.
func TestApproveRefusesEachProgramSettingGitDocuments(t *testing.T) {
	refused := refusedAs(gitConfigWriteEntryID)
	for _, key := range []string{
		"trailer.sign.cmd",
		"trailer.sign.command",
		"hook.lint.command",
		"submodule.lib.update",
		"gc.recentObjectsHook",
		"guitool.lint.cmd",
		"help.browser",
		"man.viewer",
		"imap.tunnel",
		"instaweb.browser",
		"instaweb.httpd",
		"instaweb.modulePath",
		"interactive.diffFilter",
		"diff.tool",
		"diff.guitool",
		"merge.tool",
		"merge.guitool",
		"merge.lines.recursive",
		"tag.forceSignAnnotated",
		"init.templateDir",
		"protocol.allow",
		"protocol.ext.allow",
		"sendemail.headerCmd",
		"sendemail.work.toCmd",
		"sendemail.work.ccCmd",
		"sendemail.work.headerCmd",
		"sendemail.work.smtpServer",
		"sendemail.work.sendmailCmd",
		"lfs.customtransfer.tx.path",
		"lfs.customtransfer.tx.args",
		"lfs.extension.ext.clean",
		"lfs.extension.ext.smudge",
		"lfs.standalonetransferagent",
		"lfs.https://example.com/repo.standalonetransferagent",
	} {
		t.Run(key, func(t *testing.T) {
			r, _ := gitRig(t)
			r.run(t, []approveCase{
				{`git config ` + key + ` x`, refused},
				{`git config --local ` + key + ` x`, refused},
				{`git config set ` + strings.ToUpper(key) + ` x`, refused},
				{`git config --add ` + key + ` x`, refused},
				{`git -c ` + key + `=x status`, refused},
			})
		})
	}
}

// TestApproveLeavesSettingsThatNameNoProgram: the neighbours of the listed
// settings that name no program of their own are not refused.
func TestApproveLeavesSettingsThatNameNoProgram(t *testing.T) {
	r, _ := gitRig(t)
	var cases []approveCase
	for _, key := range []string{
		"format.pretty", "pretty.short", "merge.renormalize", "gpg.format", "ssh.variant",
		"protocol.file.allow", "submodule.lib.url", "submodule.lib.branch", "trailer.where",
		"trailer.sign.key", "lfs.url", "lfs.customtransfer.tx.concurrent", "lfs.extension.ext.priority",
		"sendemail.smtpServerPort", "help.format", "instaweb.port", "merge.lines.name", "gc.auto",
	} {
		cases = append(cases, approveCase{`git config ` + key + ` x`, nil}, approveCase{`git -c ` + key + `=x status`, nil})
	}
	r.run(t, cases)
}

// heldProgramRows judges each alternative of the settings a drive holds
// across its lanes (driveHeldConfig in internal/core/implement/loop/remoteid.go,
// and watchedConfig in configwatch.go beside it) against the approver's list:
// each sample key is one the alternative matches, and program says whether the
// approver refuses setting it. A sample that is not a program says why.
var heldProgramRows = map[string][]struct {
	key     string
	program bool
	why     string
}{
	// watchedConfig (configwatch.go).
	`filter\..+`: {{"filter.lfs.clean", true, ""}, {"filter.lfs.smudge", true, ""}, {"filter.lfs.process", true, ""}},
	`core\.(hookspath|sshcommand|fsmonitor|pager|editor|worktree|askpass|gitproxy)`: {
		{"core.hookspath", true, ""}, {"core.sshcommand", true, ""}, {"core.fsmonitor", true, ""}, {"core.pager", true, ""},
		{"core.editor", true, ""}, {"core.askpass", true, ""}, {"core.gitproxy", true, ""},
		{"core.worktree", false, "moves the working tree; refused as treeKeys, not as a program"},
	},
	`extensions\.worktreeconfig`: {{"extensions.worktreeconfig", false, "says where git reads a worktree's settings; each program setting there is judged by its own key"}},
	`credential\..+`: {
		{"credential.helper", true, ""}, {"credential.https://example.com.helper", true, ""},
		{"credential.username", false, "a value sent, naming no program"},
	},
	`remote\..+\.(uploadpack|receivepack|proxy|vcs|url|pushurl)`: {
		{"remote.origin.uploadpack", true, ""}, {"remote.origin.receivepack", true, ""}, {"remote.origin.vcs", true, ""},
		{"remote.origin.proxy", false, "an address; core.gitProxy is the program"},
		{"remote.origin.url", false, "an address; the ext transport that runs a command needs protocol.ext.allow or protocol.allow, which are listed"},
		{"remote.origin.pushurl", false, "an address, as remote.<name>.url"},
	},
	`url\..+\.(insteadof|pushinsteadof)`: {{"url.https://example.com/.insteadof", false, "rewrites an address, as remote.<name>.url"}},
	`protocol\.(.+\.)?allow`: {
		{"protocol.allow", true, ""}, {"protocol.ext.allow", true, ""},
		{"protocol.file.allow", false, "allows a transport that starts no command of the address's choosing"},
	},
	`diff\..+\.(command|textconv)`: {{"diff.pdf.command", true, ""}, {"diff.pdf.textconv", true, ""}},
	`merge\..+\.driver`:            {{"merge.lines.driver", true, ""}},
	`gpg\.(.+\.)?program`:          {{"gpg.program", true, ""}, {"gpg.x509.program", true, ""}},
	`gpg\.ssh\.defaultkeycommand`:  {{"gpg.ssh.defaultkeycommand", true, ""}},
	`include\.path`:                {{"include.path", true, ""}},
	`includeif\..+\.path`:          {{"includeif.onbranch:main.path", true, ""}},

	// driveHeldConfig (remoteid.go).
	`http\.(.+\.)?(proxy|proxyauthmethod|proxysslcainfo|proxysslcert|proxysslkey|proxysslcertpasswordprotected|sslverify|sslversion|sslcipherlist|sslcainfo|sslcapath|pinnedpubkey|sslbackend|schannelcheckrevoke|schannelusesslcainfo|ssltry|sslcert|sslcerttype|sslkey|sslkeytype|sslcertpasswordprotected|curloptresolve|followredirects|cookiefile|savecookies|extraheader|useragent|delegation|emptyauth|proactiveauth)`: {
		{"http.proxy", false, "where an https fetch connects, whom it trusts and what it sends; no program"},
		{"http.sslverify", false, "trust, naming no program"},
	},
	`remote\..+\.(proxyauthmethod|serveroption|promisor|partialclonefilter|token|mirror)`: {
		{"remote.origin.serveroption", false, "a value sent, naming no program"},
		{"remote.origin.promisor", false, "where a missing object is fetched from, naming no program"},
	},
	`extensions\.partialclone`: {{"extensions.partialclone", false, "names a promisor remote, not a program"}},
	`promisor\..+`:             {{"promisor.acceptfromserver", false, "which remotes a fetch accepts, naming no program"}},
	`push\.pushoption`:         {{"push.pushoption", false, "a value sent, naming no program"}},
	`transfer\.advertisesid`:   {{"transfer.advertisesid", false, "a value sent, naming no program"}},
	`fetch\.bundleuri`:         {{"fetch.bundleuri", false, "an address, naming no program"}},
	`core\.(alternaterefscommand|attributesfile)`: {
		{"core.alternaterefscommand", true, ""},
		{"core.attributesfile", false, "says which filters and drivers apply to a path; each filter and driver is listed by its own key"},
	},
	`diff\.external`: {{"diff.external", true, ""}},
	`merge\.(renormalize|verifysignatures)`: {
		{"merge.verifysignatures", true, ""},
		{"merge.renormalize", false, "makes a merge run the content filters, which are listed by their own keys"},
	},
	`format\.pretty`: {{"format.pretty", false, "a format; a signature it checks starts gpg.program, which is listed"}},
	`pretty\..+`:     {{"pretty.short", false, "a format, as format.pretty"}},
	`gpg\..+`: {
		{"gpg.program", true, ""}, {"gpg.ssh.program", true, ""}, {"gpg.ssh.defaultkeycommand", true, ""},
		{"gpg.format", false, "picks which of the gpg programs a signature uses; each is listed as gpg.<format>.program"},
		{"gpg.ssh.allowedsignersfile", false, "a file read, naming no program"},
	},
	`(commit|tag|push)\.gpgsign`:  {{"commit.gpgsign", true, ""}, {"tag.gpgsign", true, ""}, {"push.gpgsign", true, ""}},
	`tag\.forcesignannotated`:     {{"tag.forcesignannotated", true, ""}},
	`log\.showsignature`:          {{"log.showsignature", true, ""}},
	`ssh\.variant`:                {{"ssh.variant", false, "how git passes arguments to the ssh program core.sshCommand names, which is listed"}},
	`uploadpack\.packobjectshook`: {{"uploadpack.packobjectshook", true, ""}},
	`submodule\.recurse`:          {{"submodule.recurse", false, "makes a command reach into submodules, naming no program"}},
	`push\.recursesubmodules`:     {{"push.recursesubmodules", false, "as submodule.recurse"}},
	`status\.submodulesummary`:    {{"status.submodulesummary", false, "what a status prints, naming no program"}},
	`lfs\.(customtransfer\..+|extension\..+|(.+\.)?standalonetransferagent)`: {
		{"lfs.customtransfer.tx.path", true, ""}, {"lfs.customtransfer.tx.args", true, ""},
		{"lfs.extension.ext.clean", true, ""}, {"lfs.extension.ext.smudge", true, ""},
		{"lfs.standalonetransferagent", true, ""}, {"lfs.https://example.com/repo.standalonetransferagent", true, ""},
		{"lfs.customtransfer.tx.concurrent", false, "a switch of the transfer program, which .path names"},
	},
	`hook\..+`:              {{"hook.lint.command", true, ""}},
	`gc\.recentobjectshook`: {{"gc.recentobjectshook", true, ""}},
	`init\.templatedir`:     {{"init.templatedir", true, ""}},
}

// TestApproverListHoldsEachProgramClassTheDriveHolds: each alternative of the
// settings a drive holds across its lanes has a row, each row's samples match
// it, and each sample judged a program is one the approver refuses setting.
// A new alternative in the drive's table fails here until it is judged.
func TestApproverListHoldsEachProgramClassTheDriveHolds(t *testing.T) {
	loop := filepath.Join("..", "implement", "loop")
	alts := append(patternAlternatives(t, stringConst(t, filepath.Join(loop, "configwatch.go"), "watchedConfig")),
		patternAlternatives(t, stringConst(t, filepath.Join(loop, "remoteid.go"), "driveHeldConfig"))...)
	seen := map[string]bool{}
	for _, alt := range alts {
		seen[alt] = true
		rows, ok := heldProgramRows[alt]
		if !ok {
			t.Errorf("the drive holds %q, and no row judges it against the approver's list", alt)
			continue
		}
		re := regexp.MustCompile(`^(` + alt + `)$`)
		for _, row := range rows {
			switch {
			case !re.MatchString(row.key):
				t.Errorf("%q: sample %q does not match it", alt, row.key)
			case row.program && !startsProgram(row.key):
				t.Errorf("the drive holds %q as a program setting, and the approver does not refuse setting it", row.key)
			case !row.program && row.why == "":
				t.Errorf("%q: judged no program with no reason", row.key)
			case !row.program && startsProgram(row.key) && !movesTree(row.key):
				t.Errorf("%q is judged no program (%s), and the approver refuses it as one", row.key, row.why)
			}
		}
	}
	for alt := range heldProgramRows {
		if !seen[alt] {
			t.Errorf("row %q names no alternative the drive holds", alt)
		}
	}
}

// stringConst reads the string constant name from the Go file at path, a
// concatenation of string literals.
func stringConst(t *testing.T, path, name string) string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	var eval func(ast.Expr) string
	eval = func(e ast.Expr) string {
		switch e := e.(type) {
		case *ast.BasicLit:
			s, err := strconv.Unquote(e.Value)
			if err != nil || e.Kind != token.STRING {
				t.Fatalf("%s: %s: not a string literal", path, name)
			}
			return s
		case *ast.BinaryExpr:
			if e.Op != token.ADD {
				t.Fatalf("%s: %s: not a concatenation", path, name)
			}
			return eval(e.X) + eval(e.Y)
		case *ast.ParenExpr:
			return eval(e.X)
		}
		t.Fatalf("%s: %s: not a constant string expression", path, name)
		return ""
	}
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, s := range g.Specs {
			v := s.(*ast.ValueSpec)
			for i, n := range v.Names {
				if n.Name == name && i < len(v.Values) {
					return eval(v.Values[i])
				}
			}
		}
	}
	t.Fatalf("%s: no constant %s", path, name)
	return ""
}

// patternAlternatives splits a `^(a|b|...)$` pattern into its top-level
// alternatives.
func patternAlternatives(t *testing.T, p string) []string {
	t.Helper()
	body, ok := strings.CutPrefix(p, "^(")
	if ok {
		body, ok = strings.CutSuffix(body, ")$")
	}
	if !ok {
		t.Fatalf("%q is not ^(...)$", p)
	}
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '\\':
			i++
		case '(':
			depth++
		case ')':
			depth--
		case '|':
			if depth == 0 {
				out = append(out, body[start:i])
				start = i + 1
			}
		}
	}
	return append(out, body[start:])
}
