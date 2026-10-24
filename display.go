package main

// This file turns AniList's raw values into what our API shows: titles,
// subtitles, cleaned-up synopses, scores, statuses, and formats.
// Every function here is "pure": it only looks at its inputs and returns
// a result, with no network or HTTP involved. That makes the display rules
// easy to test and easy to change in one place.

import (
	"html"
	"regexp"
	"strings"
)

// Regular expressions are compiled once, when the program starts, and
// reused for every request. (?i) means "ignore upper/lower case".
var (
	// brTag matches line-break tags: <br>, <br/>, <br />, <BR>, ...
	brTag = regexp.MustCompile(`(?i)<br\s*/?>`)

	// anyTag matches any other HTML tag, opening or closing, e.g. <i>,
	// </i>, <a href="...">. We remove the tag but keep the text around it.
	anyTag = regexp.MustCompile(`<[^>]*>`)

	// whitespaceRun matches one or more spaces, tabs, or line breaks.
	whitespaceRun = regexp.MustCompile(`\s+`)

	// sourceCredit matches a paragraph that is only a credit like
	// "(Source: VIZ Media)". We drop these (see Attribution in CLAUDE.md).
	sourceCredit = regexp.MustCompile(`(?i)^\(source:.*\)$`)
)

// untitled is shown in the (so far never seen) case that AniList has no
// title at all, because our API promises the title is never null.
const untitled = "Untitled"

// trimmed returns the text a pointer points to, without surrounding
// whitespace, or "" if the pointer is nil. AniList sometimes sends stray
// whitespace, e.g. "Attack on Titan Volume 0\r\n".
func trimmed(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

// displayTitle picks the main title: romaji first, then English, then native.
func displayTitle(t anilistTitle) string {
	for _, candidate := range []*string{t.Romaji, t.English, t.Native} {
		if s := trimmed(candidate); s != "" {
			return s
		}
	}
	return untitled
}

// displaySubtitle picks the line shown under the title: the English title
// if it differs from the main title, otherwise the native title if that
// differs, otherwise nil (no subtitle).
//
// "Differs" ignores upper/lower case, so "ONE PIECE" with English title
// "One Piece" doesn't get a subtitle that just repeats it.
func displaySubtitle(t anilistTitle, title string) *string {
	for _, candidate := range []*string{t.English, t.Native} {
		s := trimmed(candidate)
		if s != "" && !strings.EqualFold(s, title) {
			return &s
		}
	}
	return nil
}

// cleanSynopsis turns AniList's HTML description into a list of plain-text
// paragraphs. It always returns a list (never nil), so the JSON is [] and
// not null when there's nothing to show.
func cleanSynopsis(description *string) []string {
	paragraphs := []string{}
	if description == nil {
		return paragraphs
	}

	// Every <br> starts a new paragraph.
	for _, piece := range brTag.Split(*description, -1) {
		// Remove the remaining tags first, and only then decode entities
		// like &hearts; or &lt;. In the other order, text that is meant to
		// show "<" (written as &lt;) would become "<" and then be mistaken
		// for the start of a tag and deleted.
		text := anyTag.ReplaceAllString(piece, "")
		text = html.UnescapeString(text)

		// Collapse line breaks and repeated spaces into one space. AniList
		// shows descriptions as HTML, where a bare line break is just a space.
		text = whitespaceRun.ReplaceAllString(text, " ")
		text = strings.TrimSpace(text)

		if text == "" || sourceCredit.MatchString(text) {
			continue
		}
		paragraphs = append(paragraphs, text)
	}
	return paragraphs
}

// displayScore converts AniList's 0-100 score to our 0-10 score: 84 -> 8.4.
// Dividing a whole number by 10 always leaves at most one decimal place,
// so no extra rounding is needed.
func displayScore(score *int) *float64 {
	if score == nil {
		return nil
	}
	s := float64(*score) / 10
	return &s
}

// statusText maps AniList's status codes to the text we show.
var statusText = map[string]string{
	"FINISHED":         "Finished",
	"RELEASING":        "Releasing",
	"NOT_YET_RELEASED": "Not yet released",
	"CANCELLED":        "Cancelled",
	"HIATUS":           "On hiatus",
}

// displayStatus returns the text for a status, or nil if the status is
// missing or one we don't know (e.g. if AniList adds a new one).
func displayStatus(status *string) *string {
	if status == nil {
		return nil
	}
	text, ok := statusText[*status]
	if !ok {
		return nil
	}
	return &text
}

// displayFormat returns the text for a format. Our AniList queries only
// ask for MANGA and ONE_SHOT, so anything else is passed through unchanged
// rather than hidden.
func displayFormat(format string) string {
	switch format {
	case "MANGA":
		return "Manga"
	case "ONE_SHOT":
		return "One Shot"
	default:
		return format
	}
}
