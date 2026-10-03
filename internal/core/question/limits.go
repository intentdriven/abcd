// Package question holds what a question abcd puts to a person may contain:
// the field limits, written once (Limits, Default), the field view every front
// door finally shows (Fields), and the limits check (CheckLimits) that refuses
// a badly built question naming the part, the value, and the limit
// (spc-2610030944505997; itd-2610030810350727, itd-201).
//
// Nothing else in the tree states a limit. The question check in the guard
// hook, the GRILL rule text and the interview pages read Default, so changing a
// limit here changes every statement of it (itd-2610030810350727 criterion A7).
//
// The package is a library: it returns values and never prints. It measures
// text through internal/textwidth, a pure leaf, and imports nothing that does
// terminal I/O.
package question

// Limits is every bound a question's fields are held to. question.Default is
// the one value the check, the GRILL domain, and the interview pages read
// (itd-2610030810350727 decision 14, criterion A7).
type Limits struct {
	HeaderColumns     int      // 12: the host's header chip
	ChipRoles         []string // "Product", "Tech", "Setup": the chip's first word
	QuestionsPerCall  [2]int   // 1..4: one question, or up to four tabs
	OptionsPerQ       [2]int   // 2..4, the decide-later option included
	LabelWords        int      // 5
	MeaningSentences  int      // 2: an option's description
	LaterLabels       []string // "Decide later", "None of these"
	Columns           int      // 80: the narrow window promised
	Rows              int      // 24: one question, or one tab, at Columns
	HostTextColumns   int      // the text measure inside the host's frame at Columns
	HostChromeRows    int      // rows the frame draws around a question
	NowPrefix         string   // "Now:"
	ChangeLaterPrefix string   // "Change later:"
	NotApplicable     string   // "not applicable"
}

// ProductRole is the chip role word that names the product thinker. Where no
// mode names the addressee, a chip carrying it stands in for the mode (rule 9).
const ProductRole = "Product"

// Default is the one statement of every limit.
var Default = Limits{
	HeaderColumns:    12,
	ChipRoles:        []string{ProductRole, "Tech", "Setup"},
	QuestionsPerCall: [2]int{1, 4},
	OptionsPerQ:      [2]int{2, 4},
	LabelWords:       5,
	MeaningSentences: 2,
	LaterLabels:      []string{"Decide later", "None of these"},
	Columns:          80,
	Rows:             24,
	// HostTextColumns and HostChromeRows are PROVISIONAL: they are measured,
	// not chosen, and step 5 of spc-2610030944505997 calibrates them against
	// the dated screenshot of the host's question view at 80 by 24. Until then
	// they carry these estimates.
	HostTextColumns:   76,
	HostChromeRows:    6,
	NowPrefix:         "Now:",
	ChangeLaterPrefix: "Change later:",
	NotApplicable:     "not applicable",
}
