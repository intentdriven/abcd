package landing

import (
	"errors"
	"fmt"
	"strings"
)

// The review brief (iss-2611220942565038): the return a reviewer writes for a head,
// in the shape --record-review reads one. It is built from the constants the
// reading holds a return to, so the brief and the reading cannot drift apart.

const (
	// reviewedHeadLabel opens a full pair's return (statesHead, reviewedHead).
	reviewedHeadLabel = "Reviewed head: "
	// reviewedDeltaLabel opens a delta's return (statesDelta, reviewedDelta).
	reviewedDeltaLabel = "Reviewed delta: "
	// verdictHeadingLine is the Verdict heading as a brief writes it; verdictHeading
	// reads it at any depth.
	verdictHeadingLine = "### Verdict"
	// briefFindings stands where the reviewer writes its findings. It states no
	// head, delta or verdict, so a brief left unfilled is refused at its verdict.
	briefFindings = "<the findings, each with its severity: critical, major, minor or nitpick>"
)

// briefVerdict stands where the reviewer writes its verdict word: each word a
// validator writes, the ones that pass the stop rule first.
var briefVerdict = func() string {
	var pass, fail []string
	for _, w := range knownVerdicts {
		if passingVerdicts[w] {
			pass = append(pass, w)
		} else {
			fail = append(fail, w)
		}
	}
	return "<" + orList(pass) + " when no finding is critical or major; otherwise " + orList(fail) + ">"
}()

// orList joins words as a sentence lists them: "a, b or c".
func orList(words []string) string {
	if len(words) < 2 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " or " + words[len(words)-1]
}

// ReviewBrief returns the return a reviewer writes for head, the commit's full
// object name, with the findings and the verdict word left for the reviewer: its
// first line states the head it read, or with base, the full pair's head a delta
// rests on, the delta base..head, and the line under its one Verdict heading names
// the head in full. It writes nothing. A head or a base that is not a commit the
// repository holds is refused, and so is a head resting on itself; whether a delta
// stands is judged when its pair is recorded (RecordDeltaReview).
func ReviewBrief(repoRoot, head, base string) (string, error) {
	if err := holdsCommit(repoRoot, head); err != nil {
		return "", err
	}
	first := reviewedHeadLabel + head
	if base != "" {
		if !fullSHA.MatchString(base) {
			return "", fmt.Errorf("the base head %s is not a commit's full object name", clip(base))
		}
		if base == head {
			return "", errors.New(restsOnItself)
		}
		if err := holdsCommit(repoRoot, base); err != nil {
			return "", err
		}
		first = reviewedDeltaLabel + base + ".." + head
	}
	return first + "\n\n" + briefFindings + "\n\n" + verdictHeadingLine + "\n\n**" + briefVerdict + "** " + head + "\n", nil
}
