package rules

import (
	"regexp"
	"strings"
)

// LeavingDomain is the bundled domain that has a session set the away mode
// on the person's own word that they are leaving (iss-2611211733065807). Its
// rule leads to `abcd mode autonomous`, which no check ties to the person, so
// two things narrow when it is recalled beyond the first-person phrases it
// declares: it is matched against the person's own words alone (OwnWords),
// and a drive's stage never loads it (ForStage). Both answer the security and
// code reviews of away-blockers-1123.
const LeavingDomain = "LEAVING"

// ownWordsDomains are matched against OwnWords(prompt) rather than the whole
// prompt, by Match and by Hits alike.
var ownWordsDomains = map[string]bool{LeavingDomain: true}

// stageOffDomains are never loaded on a drive's stage: a lane's prompt is
// abcd's composition around an issue body, never the person's word.
var stageOffDomains = []string{LeavingDomain}

var (
	// fencedRe is a fenced block, ``` or ~~~, closed by the same fence; an
	// unclosed fence matches nothing.
	fencedRe = regexp.MustCompile("(?s)```.*?```|~~~.*?~~~")
	// quotedLineRe is a block-quoted line.
	quotedLineRe = regexp.MustCompile(`(?m)^[ \t]*>.*$`)
	// quotedRe is text between straight or curly double quotes on one line.
	quotedRe = regexp.MustCompile(`"[^"\n]*"|“[^”\n]*”`)
)

// OwnWords is the prompt with what reads as someone else's text taken out:
// fenced blocks, block-quoted lines, and text between double quotes on one
// line. Text pasted with no such mark stays, and so does single-quoted text;
// the LEAVING rule tells the session to set nothing on either. An unclosed
// fence takes nothing out, and neither does a lone double quote with no other
// on its line, so the person's own sentence after one still counts. A stray
// double quote (an inch mark, say) does pair with the next one on its line,
// and what lies between is taken out: that fails towards LEAVING not being
// recalled, never towards a mode set on words that are not the person's.
func OwnWords(prompt string) string {
	prompt = fencedRe.ReplaceAllString(prompt, " ")
	prompt = quotedLineRe.ReplaceAllString(prompt, " ")
	return quotedRe.ReplaceAllString(prompt, " ")
}

// ForStage is the set a hook on a drive's stage injects: rs without the
// domains no stage carries, so neither recall nor a star command reaches
// them there. rs itself is left as it was.
func (rs RuleSet) ForStage() RuleSet {
	domains := make(map[string]Domain, len(rs.Domains))
	for name, d := range rs.Domains {
		domains[name] = d
	}
	for _, name := range stageOffDomains {
		delete(domains, name)
	}
	rs.Domains = domains
	return rs
}

// ownWordsPrompt is prompt as domain name is matched against it, its recall
// and its star command alike.
func ownWordsPrompt(name, prompt string) string {
	if ownWordsDomains[strings.ToUpper(name)] {
		return OwnWords(prompt)
	}
	return prompt
}
