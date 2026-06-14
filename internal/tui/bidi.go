package tui

import (
	"strings"

	"golang.org/x/text/unicode/bidi"
)

func displayText(value string, width int) string {
	value = truncate(value, width)
	if !needsBidi(value) {
		return value
	}
	return bidiVisualOrder(value)
}

func bidiVisualOrder(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var paragraph bidi.Paragraph
	if _, err := paragraph.SetString(value, bidi.DefaultDirection(bidi.LeftToRight)); err != nil {
		return value
	}
	order, err := paragraph.Order()
	if err != nil {
		return value
	}
	var out strings.Builder
	for i := 0; i < order.NumRuns(); i++ {
		run := order.Run(i)
		text := run.String()
		if run.Direction() == bidi.RightToLeft {
			text = bidi.ReverseString(text)
		}
		out.WriteString(text)
	}
	return out.String()
}

func needsBidi(value string) bool {
	for _, r := range value {
		switch {
		case r >= 0x0590 && r <= 0x08FF:
			return true
		case r >= 0xFB1D && r <= 0xFDFF:
			return true
		case r >= 0xFE70 && r <= 0xFEFF:
			return true
		}
	}
	return false
}
