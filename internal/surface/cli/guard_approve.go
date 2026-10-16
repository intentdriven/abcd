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
	var dir string
	cmd := &cobra.Command{
		Use: "approve --dir=<working tree>",
		Long: "Answers the permission prompts of an agent abcd started without a terminal,\n" +
			"over MCP on stdio: the claude runner starts it, as the one server of a launch\n" +
			"whose role's contract grants the shell, and names its tool, approve, as the\n" +
			"harness's permission prompt tool. Each prompt is judged against the hazard\n" +
			"registry of the working tree --dir names, read afresh for every prompt.\n\n" +
			"A shell command is approved only when the guard reads it whole and clears\n" +
			"it: no registry entry and no reading signal, a warn included; no note about\n" +
			"the check, such as a script that does not exist; the registry the repository\n" +
			"declares, loaded and switched on; every program named by the line rather\n" +
			"than by a variable, a substitution or a glob; and no inline code handed to a\n" +
			"non-shell interpreter (python -c, node -e, perl -e and the like). It runs\n" +
			"with exactly the input judged. Everything else is refused with the guard's\n" +
			"reason and what to run instead, which the harness hands the agent, so the\n" +
			"agent can rewrite the command. A prompt for any tool but the shell is\n" +
			"refused, since nobody is there to approve it.\n\n" +
			"The guard is a mistake filter, not a security boundary (see `abcd guard\n" +
			"check --help` for what an allow does not see), and this adds no sandbox.\n" +
			"It never touches the person's host settings, and interactive sessions never\n" +
			"start it. It exits 0 when its input ends.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
				return &exitError{Code: 2, Msg: "guard approve: --dir must name the working tree as a clean absolute path"}
			}
			return serveApprover(cmd.InOrStdin(), cmd.OutOrStdout(), func(tool string, input json.RawMessage) (guard.Approval, json.RawMessage) {
				// Each prompt reads the registry afresh, as the hook does, so
				// a run judges against the repository as it stands.
				ld := guard.LoadRepo(rulesRoot(dir, cmd.ErrOrStderr()))
				return guard.ApprovePrompt(ld, guard.NewFiles(dir), tool, input)
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "the working tree the agent runs in, as an absolute path")
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
