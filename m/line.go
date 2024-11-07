package m

import (
	"regexp"
	"sync"

	"github.com/walles/moar/m/linenumbers"
	"github.com/walles/moar/m/textstyles"
	"github.com/walles/moar/twin"
)

// A Line represents a line of text that can / will be paged
type Line struct {
	raw   string
	plain *string
	lock  sync.Mutex
}

// NewLine creates a new Line from a (potentially ANSI / man page formatted) string
func NewLine(raw string) Line {
	return Line{
		raw:   raw,
		plain: nil,
		lock:  sync.Mutex{},
	}
}

// Returns a representation of the string split into styled tokens. Any regexp
// matches are highlighted. A nil regexp means no highlighting.
func (line *Line) HighlightedTokens(linePrefix string, search *regexp.Regexp, lineNumber *linenumbers.LineNumber) textstyles.StyledRunesWithTrailer {
	plain := line.Plain(lineNumber)
	matchRanges := getMatchRanges(&plain, search)

	fromString := textstyles.StyledRunesFromString(linePrefix, line.raw, lineNumber)
	returnRunes := make([]twin.StyledRune, 0, len(fromString.StyledRunes))
	for _, token := range fromString.StyledRunes {
		style := token.Style
		if matchRanges.InRange(len(returnRunes)) {
			if standoutStyle != nil {
				style = *standoutStyle
			} else {
				style = style.WithAttr(twin.AttrReverse)
				style = style.WithBackground(twin.ColorDefault)
				style = style.WithForeground(twin.ColorDefault)
			}
		}

		returnRunes = append(returnRunes, twin.StyledRune{
			Rune:  token.Rune,
			Style: style,
		})
	}

	return textstyles.StyledRunesWithTrailer{
		StyledRunes: returnRunes,
		Trailer:     fromString.Trailer,
	}
}

// Plain returns a plain text representation of the initial string
func (line *Line) Plain(lineNumber *linenumbers.LineNumber) string {
	line.lock.Lock()
	defer line.lock.Unlock()

	if line.plain == nil {
		line.lock.Unlock()
		// The computation doesn't need the lock
		plain := textstyles.WithoutFormatting(line.raw, lineNumber)
		line.lock.Lock()
		line.plain = &plain
	}
	return *line.plain
}
