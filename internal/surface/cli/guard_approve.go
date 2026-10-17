package cli

// guard_approve.go is `guard approve`: the permission prompt answerer of an
// agent abcd started without a terminal (iss-2610161612203555,
// dcn-2610161624025493). The claude runner names it on a launch whose role's
// contract grants the shell (internal/core/runner/claude.go): the harness
// starts it as an MCP server on stdio and, wherever it would ask a person,
// calls its one tool, `approve`, with the tool's name and input. The decision
// is core's (guard.ApprovePrompt); this file is the transport.
//
// The protocol is the subset the harness's prompt tool uses, spoken with the
// standard library: newline-delimited JSON-RPC 2.0 with initialize, ping,
// tools/list and tools/call answered, notifications read and left unanswered,
// and every other method refused as not found. The answer is one text block
// holding {"behavior":"allow","updatedInput":…} or
// {"behavior":"deny","message":…}, the shape the harness reads.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/spf13/cobra"
)

// maxApproveLine caps one JSON-RPC message the approver reads. A shell call's
// input is the command, which the guard caps at 64 KiB, and a few short
// fields; a line past this is refused, never truncated.
const maxApproveLine = 1 << 20

// approveProtocol is the MCP revision the approver answers with when the
// client asks for one it cannot read as a revision date.
const approveProtocol = "2025-06-18"

var protocolRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func newGuardApproveCommand() *cobra.Command {
	var dir, base string
	var runDirs []string
	cmd := &cobra.Command{
		Use: "approve --dir <working-tree> --base <commit> [--run-dir <dir>]...",
		Long: "Answers the permission prompts of an agent abcd started without a terminal,\n" +
			"over MCP on stdio: the claude runner starts it, as the one server of a launch\n" +
			"whose role's contract grants the shell, and names its tool, approve, as the\n" +
			"harness's permission prompt tool. Each prompt is judged against the hazard\n" +
			"registry of the working tree --dir names, read afresh for every prompt, and\n" +
			"in force only where it is at least as strong as the registry at --base, the\n" +
			"commit the run started on: any weakening since then, of a warn or a blocker,\n" +
			"committed or not, refuses every command.\n\n" +
			"A shell command is approved only when the guard reads it whole and clears\n" +
			"it: no registry entry and no reading signal, a warn included; no note about\n" +
			"the check, such as a script that does not exist; and every program named by\n" +
			"the line rather than by a variable, a substitution or a glob. Beyond the\n" +
			"registry, it refuses:\n\n" +
			"  - inline code for python -c, node -e or -p, perl -e, ruby -e, php -r,\n" +
			"    osascript, lua or Rscript -e, deno eval and PowerShell -Command; a flag\n" +
			"    after the script's name, or after python's -m module, is the script's;\n" +
			"  - code on the standard input (a pipe, a here-document, a here-string or a\n" +
			"    redirected file) of one of those interpreters, or of ex, when no script\n" +
			"    file is named;\n" +
			"  - an awk program that calls system(, pipes to or from a command, or is a\n" +
			"    variable; a sed script with the e command or the e flag of s; tar's\n" +
			"    options that run a program (--checkpoint-action, --to-command,\n" +
			"    --use-compress-program, -I and the like); a vim or ex command (-c, --cmd,\n" +
			"    +) holding a !;\n" +
			"  - a write the guard's write model reads (redirections and a list of known\n" +
			"    writers, so a program outside the list writes unread), or a removal,\n" +
			"    whose target the line does not spell out (a variable,\n" +
			"    a substitution, a glob, or a name after an unknown change of directory),\n" +
			"    except a glob removal in a directory inside the working tree, followed\n" +
			"    through symlinks, that no protected name could match, when no earlier\n" +
			"    part of the line writes or removes a file and no wildcard stands in a\n" +
			"    directory part of the glob, nor does it end in a /;\n" +
			"  - such a write, or a removal, reaching a protected path, with its directories\n" +
			"    resolved through symlinks and case folded where the filesystem folds it:\n" +
			"    the git common directory and its hooks, each checkout's .git, the home\n" +
			"    shell start-up files and git configuration, the host's settings, .mcp.json,\n" +
			"    .abcd/guard.json, abcd's own folder in the home directory (its\n" +
			"    trusted-roots, rules.json, config.json and the rest, but for the run's\n" +
			"    own worktree kept there), and the names the host protects in any\n" +
			"    working tree;\n" +
			"  - a removal whose target lies outside the working tree, the run's own\n" +
			"    directories (each --run-dir) and the temporary directory; a write\n" +
			"    outside the working tree is judged by the protected paths alone;\n" +
			"  - git config with --global, --system or --file on a write, setting a key\n" +
			"    that starts a program (core.hooksPath, core.pager, an alias and the like)\n" +
			"    or core.worktree, or opening the editor, and any git command setting such\n" +
			"    a key for itself, but for -c core.pager or core.sshCommand set to a\n" +
			"    command the line spells out, which is judged as that command;\n" +
			"  - removing, moving or recursively changing the root, a top-level directory,\n" +
			"    the home directory or a directory directly in it, the working tree or a\n" +
			"    directory above it, or another checkout of the repository; a find that\n" +
			"    deletes from home or above; an rsync --delete into such a directory;\n" +
			"  - a call asking to run outside the host's sandbox.\n\n" +
			"An approved call runs with the command judged and its description, and\n" +
			"nothing else it carried, from the working tree it was judged for: the\n" +
			"command comes back as cd -- '<dir>' && { <command> }, the command on lines\n" +
			"of its own inside the braces, since the host keeps the shell's directory\n" +
			"from one call to the next; when the directory cannot be entered, no part of\n" +
			"the command runs. That text is read again with the guard's own parse, and a\n" +
			"command that would close the braces early or leave them unparsable (a } or\n" +
			"a ) with nothing open, a fi, done or esac with nothing open, a } after a\n" +
			"command's words, or a backslash at its end) is refused, since part of it\n" +
			"could run outside the working tree. A { or a keyword counts only where the\n" +
			"line leaves it unquoted: a quoted one is a command's name to the shell, so\n" +
			"\"{\" on a line, then }, then a command, is refused for the same reason.\n" +
			"Everything else is refused with the guard's reason\n" +
			"and what to run instead, which the harness hands the agent, so the agent can\n" +
			"rewrite the command. A prompt for any tool but the shell is refused, since\n" +
			"nobody is there to approve it.\n\n" +
			"The file tools the role is granted (--allowedTools on the launch) write\n" +
			"without asking the approver, and the abcd binary that answers is not among\n" +
			"the paths it protects; the OS sandbox for launched agents\n" +
			"(spc-2610161751056096) is what bounds those writes.\n\n" +
			"The guard is a mistake filter, not a security boundary (see `abcd guard\n" +
			"check --help` for what an allow does not see), and this adds no sandbox.\n" +
			"It never touches the person's host settings, and interactive sessions never\n" +
			"start it. It exits 0 when its input ends.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
				return &exitError{Code: 2, Msg: "guard approve: --dir must name the working tree as a clean absolute path"}
			}
			if !gitutil.IsFullSHA(base) {
				return &exitError{Code: 2, Msg: "guard approve: --base must name the commit the run started on by its full object name"}
			}
			for _, d := range runDirs {
				if !filepath.IsAbs(d) || filepath.Clean(d) != d {
					return &exitError{Code: 2, Msg: "guard approve: --run-dir must name a directory of the run as a clean absolute path"}
				}
			}
			return serveApprover(cmd.InOrStdin(), cmd.OutOrStdout(), func(tool string, input json.RawMessage) (guard.Approval, json.RawMessage) {
				// Each prompt reads the registry afresh, as the hook does, so
				// a run judges against the repository as it stands, and only
				// where that is at least as strong as it was at the base.
				ld := guard.LoadRepoAt(rulesRoot(dir, cmd.ErrOrStderr()), base)
				return guard.ApprovePrompt(ld, guard.NewFiles(dir).WithRunDirs(runDirs...), tool, input)
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "the working tree the agent runs in, as an absolute path")
	cmd.Flags().StringVar(&base, "base", "", "the commit the run started on, by its full object name")
	cmd.Flags().StringArrayVar(&runDirs, "run-dir", nil, "a directory the run keeps its own files in (its brief, its receipt), as an absolute path; repeatable")
	return cmd
}

// rpcRequest is one JSON-RPC message the client sent.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// rpcError is a JSON-RPC error object.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// approveToolSchema is the approve tool's input, as the harness calls it.
var approveToolSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"tool_name":   map[string]any{"type": "string", "description": "the tool the agent asked to run"},
		"input":       map[string]any{"type": "object", "description": "the tool's input"},
		"tool_use_id": map[string]any{"type": "string", "description": "the tool call's id"},
	},
	"required": []string{"tool_name", "input"},
}

// serveApprover answers JSON-RPC lines from in on out until in ends, judging
// each prompt with judge. A line it cannot read is answered with a parse
// error when it is too long or not JSON; nothing it reads ends the session
// but the end of its input or a reply it cannot write.
func serveApprover(in io.Reader, out io.Writer, judge func(string, json.RawMessage) (guard.Approval, json.RawMessage)) error {
	r := bufio.NewReaderSize(in, 64<<10)
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	reply := func(id json.RawMessage, result any, e *rpcError) error {
		msg := map[string]any{"jsonrpc": "2.0", "id": id}
		if e != nil {
			msg["error"] = e
		} else {
			msg["result"] = result
		}
		return enc.Encode(msg)
	}
	for {
		line, tooLong, err := readApproveLine(r)
		if errors.Is(err, io.EOF) && len(line) == 0 && !tooLong {
			return nil
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("guard approve: reading the client: %w", err)
		}
		last := errors.Is(err, io.EOF)
		line = bytes.TrimSpace(line)
		switch {
		case tooLong:
			if werr := reply(json.RawMessage("null"), nil, &rpcError{Code: -32700, Message: "message too long"}); werr != nil {
				return werr
			}
		case len(line) == 0:
		default:
			if werr := answerApprove(line, reply, judge); werr != nil {
				return werr
			}
		}
		if last {
			return nil
		}
	}
}

// readApproveLine reads one line of at most maxApproveLine bytes; a longer one
// is read to its end and reported too long, with nothing of it returned.
func readApproveLine(r *bufio.Reader) (line []byte, tooLong bool, err error) {
	for {
		chunk, isPrefix, rerr := r.ReadLine()
		if !tooLong {
			line = append(line, chunk...)
			if len(line) > maxApproveLine {
				tooLong, line = true, nil
			}
		}
		if rerr != nil {
			return line, tooLong, rerr
		}
		if !isPrefix {
			return line, tooLong, nil
		}
	}
}

// answerApprove answers one JSON-RPC message.
func answerApprove(line []byte, reply func(json.RawMessage, any, *rpcError) error,
	judge func(string, json.RawMessage) (guard.Approval, json.RawMessage)) error {
	var req rpcRequest
	if json.Unmarshal(line, &req) != nil {
		return reply(json.RawMessage("null"), nil, &rpcError{Code: -32700, Message: "not a JSON-RPC message"})
	}
	if len(req.ID) == 0 || string(req.ID) == "null" {
		// A notification: read, never answered.
		return nil
	}
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		version := approveProtocol
		if protocolRe.MatchString(p.ProtocolVersion) {
			version = p.ProtocolVersion
		}
		return reply(req.ID, map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "abcd", "version": "1"},
		}, nil)
	case "ping":
		return reply(req.ID, map[string]any{}, nil)
	case "tools/list":
		return reply(req.ID, map[string]any{"tools": []any{map[string]any{
			"name":        "approve",
			"description": "abcd's guard answers a permission prompt: a shell command it reads whole and clears is approved, anything else refused with its reason.",
			"inputSchema": approveToolSchema,
		}}}, nil)
	case "tools/call":
		var p struct {
			Name      string `json:"name"`
			Arguments struct {
				ToolName string          `json:"tool_name"`
				Input    json.RawMessage `json:"input"`
			} `json:"arguments"`
		}
		if json.Unmarshal(req.Params, &p) != nil || p.Name != "approve" {
			return reply(req.ID, nil, &rpcError{Code: -32602, Message: "the one tool served is approve"})
		}
		a, updated := judge(p.Arguments.ToolName, p.Arguments.Input)
		answer := map[string]any{"behavior": "deny", "message": a.Reason}
		if a.Allow {
			answer = map[string]any{"behavior": "allow", "updatedInput": updated}
		}
		text, err := json.Marshal(answer)
		if err != nil {
			return reply(req.ID, nil, &rpcError{Code: -32603, Message: "the answer could not be written"})
		}
		return reply(req.ID, map[string]any{"content": []any{map[string]any{"type": "text", "text": string(text)}}}, nil)
	}
	return reply(req.ID, nil, &rpcError{Code: -32601, Message: "method not found"})
}
