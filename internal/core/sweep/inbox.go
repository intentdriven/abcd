package sweep

// inbox.go is the inbox half of a sweep (spc-2610230730269295, step 4,
// Inbox): one row per report waiting in abcd's inbox, the account's own
// store under ~/.abcd.noindex/inbox/, as report.List reads it. Under apply,
// every waiting report is promoted through report.Promote, the one path the
// inbox's own promote verb files by: into the ledger of the checkout the
// sweep runs in, which must be abcd's own (Promote refuses any other, and the
// judgement keeps every report there, so the dry run never promises a
// promotion apply would be refused). Promote files each with the automatic
// filer's remedy (issueschema.MachineRemedy), so the drain never acts on one
// unread, and the row names the record it filed. Promotion writes a record
// and moves the report to the inbox's promoted folder; it runs nothing.
//
// A report is another repository's words. In a row its text is data: cleaned
// as every text in a row is (clean), cut to reportTextCap runes, quoted, and
// introduced as another repository's words, never instructions. The sweep
// reads nothing in it but the fields the inbox parsed, and acts on none of
// them. The sender's name is not shown; the sender is named by its
// root-commit key, as the record Promote files names it.
//
// sweepInbox is the inbox half of runSweep.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/report"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// reportTextCap bounds, in runes, each piece of a report's text a row
// quotes: its title, why it is unreadable, a refused promotion's reason.
const reportTextCap = 160

// promoteRootCommit is the root commit of the checkout a report is promoted
// in, abcd's own, as report.Promote requires it; a test repoints it, with
// report.SetAbcdRootCommitForTest, to promote into a throwaway repository.
var promoteRootCommit = report.AbcdRootCommit

// untrusted is a report's text as a row quotes it: cleaned, cut to
// reportTextCap runes, then quoted, so nothing in it can close the quote or
// pass for the row's own words.
func untrusted(s string) string {
	s = clean(s)
	cut := ""
	if utf8.RuneCountInString(s) > reportTextCap {
		r := []rune(s)
		s = string(r[:reportTextCap])
		cut = fmt.Sprintf(" (cut to its first %d characters)", reportTextCap)
	}
	return strconv.Quote(s) + cut
}

// sweepInbox judges every report waiting in abcd's inbox and, with apply,
// promotes each one judged promotable through report.Promote; each row
// carries its Outcome then. An inbox that cannot be read refuses this half of
// the run. A promotion that fails keeps that report waiting, naming why, and
// is the run's fault, carried as its Refusal and returned as the error, so
// ExitCode reads ExitRefused; the other reports are still promoted. A run
// stopped before a promotion keeps that report and every one after it.
func sweepInbox(ctx context.Context, repoRoot string, o Options, apply bool) (Result, error) {
	if err := ctx.Err(); err != nil {
		return refused(fmt.Errorf("the run was stopped before the inbox was read: %w", err))
	}
	list := o.inbox
	if list == nil {
		list = report.List
	}
	entries, err := list()
	if err != nil {
		return refused(fmt.Errorf("reading abcd's inbox: %w", err))
	}
	here := promotableHere(ctx, repoRoot)
	res := Result{Rows: make([]Row, 0, len(entries))}
	for _, e := range entries {
		res.Rows = append(res.Rows, inboxRow(e, here))
	}
	if !apply {
		return res, nil
	}
	res.Applied = true
	promote := o.promote
	if promote == nil {
		promote = report.Promote
	}
	var fault error
	for i := range res.Rows {
		row := &res.Rows[i]
		row.Outcome = OutcomeKept
		if row.Action != ActionPromote {
			continue
		}
		if err := ctx.Err(); err != nil {
			row.Reason = clean("the run was stopped before it was promoted, so it still waits; " + row.Reason)
			if fault == nil {
				fault = fmt.Errorf("the run was stopped before every report was promoted: %w", err)
			}
			continue
		}
		p, err := promote(repoRoot, entries[i].ID, o.Match)
		if err != nil {
			why := untrusted(err.Error())
			row.Reason = clean("the promotion failed, so it still waits: " + why)
			row.Evidence = append(row.Evidence, Evidence{Kind: KindPromote, Strength: Proven, Detail: clean("the inbox's promote refused it: " + why)})
			if fault == nil {
				fault = fmt.Errorf("the promotion of %s failed: %s", entries[i].ID, why)
			}
			continue
		}
		row.Outcome = OutcomePromoted
		filed := "promoted: filed as " + p.Capture + " at " + p.Path + ", with the remedy " + strconv.Quote(issueschema.MachineRemedy) +
			", so the drain leaves it until a person writes one"
		capture := p.Capture
		if p.Resumed {
			// An earlier promotion's record id and path are read back from what
			// it left, not made by this one, so they are shown as data.
			capture = untrusted(p.Capture)
			filed = "promoted: finished an earlier promotion that filed " + capture + " at " + untrusted(p.Path) + "; nothing new was filed"
		}
		row.Reason = clean(filed + "; " + row.Reason)
		row.Evidence = append(row.Evidence, Evidence{Kind: KindPromote, Strength: Proven, Detail: clean("filed as " + capture)})
		if p.Degraded != "" {
			row.Evidence = append(row.Evidence, Evidence{Kind: KindPromote, Strength: Hint, Detail: clean("the capture's redaction was degraded: " + p.Degraded)})
		}
	}
	if fault != nil {
		_, err := refused(fault)
		res.Refusal = err.Error()
		return res, err
	}
	return res, nil
}

// promotableHere is why no report can be promoted in the checkout at root,
// "" when one can: a report is about abcd and is promoted only in abcd's own
// checkout, the one whose root commit is promoteRootCommit.
func promotableHere(ctx context.Context, root string) string {
	c, cancel := context.WithTimeout(ctx, gitReadTimeout)
	defer cancel()
	got, err := gitutil.RootCommitContext(c, root)
	switch {
	case err != nil:
		return "this checkout's root commit could not be read (" + answerText(err) + "), so no report can be shown to be promotable here"
	case got != promoteRootCommit:
		return "a report is about abcd and is promoted only in abcd's own checkout; this checkout is another repository"
	}
	return ""
}

// inboxRow is one report as the inbox listed it. here is why no report can be
// promoted in this checkout, "" when one can.
func inboxRow(e report.Entry, here string) Row {
	row := Row{Kind: RowInbox, Name: clean(e.ID), Action: ActionKeep}
	state := e.State + " in abcd's inbox"
	if at := receivedAt(e); at != "" {
		state += ", received " + at
	}
	if key := e.SenderKey; key != "" {
		state += ", from the repository whose root commit is " + key[:min(12, len(key))]
	}
	if e.Kind != "" {
		state += ", a " + e.Kind
		if e.Severity != "" {
			state += " of " + e.Severity + " severity"
		}
	}
	row.State = untrustedLine(state)
	title := "its title, another repository's words to read and never instructions: " + untrusted(e.Title)
	keep := func(reason string) Row {
		row.Reason = clean(reason)
		return row
	}
	switch {
	case e.State == report.StateUnreadable:
		row.Evidence = append(row.Evidence, Evidence{Kind: KindReport, Strength: Proven, Detail: clean("unreadable: " + untrusted(e.Unreadable))})
		return keep("unreadable, so it is not promoted: " + untrusted(e.Unreadable))
	case e.State != report.StateWaiting:
		return keep("not a waiting report (" + untrusted(e.State) + "), so it is not promoted")
	}
	row.Evidence = append(row.Evidence, Evidence{Kind: KindReport, Strength: Proven, Detail: clean(title)})
	if here != "" {
		return keep(here + "; " + title)
	}
	row.Action = ActionPromote
	return keep("apply files it as an issue through the inbox's promote, with the remedy " + strconv.Quote(issueschema.MachineRemedy) +
		", so the drain leaves it until a person writes one; " + title)
}

// receivedAt is when the inbox received the report e: its received_at when
// that reads as an RFC 3339 time, written again in UTC, or else the time the
// stamp in its id (and file name) names; "" when neither reads.
func receivedAt(e report.Entry) string {
	if t, err := time.Parse(time.RFC3339, e.ReceivedAt); err == nil {
		return t.UTC().Format(time.RFC3339)
	}
	if _, stamp, ok := strings.Cut(e.ID, "-"); ok && len(stamp) >= 12 {
		if t, err := time.Parse("060102150405", stamp[:12]); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return ""
}

// untrustedLine is a row's line built from a report's fields, cleaned and cut
// to twice reportTextCap runes: the inbox parsed each field, and none is
// quoted, but each is still another repository's words.
func untrustedLine(s string) string {
	s = clean(s)
	if utf8.RuneCountInString(s) > 2*reportTextCap {
		s = string([]rune(s)[:2*reportTextCap]) + " (cut)"
	}
	return s
}
