package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type visualTheme struct {
	Name     string
	Base     lipgloss.Color
	Ink      lipgloss.Color
	Muted    lipgloss.Color
	Subtle   lipgloss.Color
	Panel    lipgloss.Color
	PanelHi  lipgloss.Color
	Accent   lipgloss.Color
	Accent2  lipgloss.Color
	Warn     lipgloss.Color
	Error    lipgloss.Color
	Selected lipgloss.Color
	Cover    lipgloss.Color
}

var visualThemes = []visualTheme{
	{
		Name:     "aurora",
		Base:     lipgloss.Color("#141C29"),
		Ink:      lipgloss.Color("#F5F7FB"),
		Muted:    lipgloss.Color("#91A0B6"),
		Subtle:   lipgloss.Color("#56657A"),
		Panel:    lipgloss.Color("#222B3D"),
		PanelHi:  lipgloss.Color("#35435C"),
		Accent:   lipgloss.Color("#78E08F"),
		Accent2:  lipgloss.Color("#5ED6FF"),
		Warn:     lipgloss.Color("#FFB15C"),
		Error:    lipgloss.Color("#FF758F"),
		Selected: lipgloss.Color("#FFD166"),
		Cover:    lipgloss.Color("#1A2433"),
	},
	{
		Name:     "sunset",
		Base:     lipgloss.Color("#1B1620"),
		Ink:      lipgloss.Color("#FFF6F2"),
		Muted:    lipgloss.Color("#B8A7AE"),
		Subtle:   lipgloss.Color("#5F4E5A"),
		Panel:    lipgloss.Color("#2A2030"),
		PanelHi:  lipgloss.Color("#43324C"),
		Accent:   lipgloss.Color("#FF9F6E"),
		Accent2:  lipgloss.Color("#FF7A90"),
		Warn:     lipgloss.Color("#FFD166"),
		Error:    lipgloss.Color("#FF6B6B"),
		Selected: lipgloss.Color("#FFC857"),
		Cover:    lipgloss.Color("#261D2B"),
	},
	{
		Name:     "electric",
		Base:     lipgloss.Color("#111826"),
		Ink:      lipgloss.Color("#F2FAFF"),
		Muted:    lipgloss.Color("#8EA1BC"),
		Subtle:   lipgloss.Color("#4B5D7A"),
		Panel:    lipgloss.Color("#1D2737"),
		PanelHi:  lipgloss.Color("#31415F"),
		Accent:   lipgloss.Color("#66F0C2"),
		Accent2:  lipgloss.Color("#79B8FF"),
		Warn:     lipgloss.Color("#FFC857"),
		Error:    lipgloss.Color("#FF6F91"),
		Selected: lipgloss.Color("#F9D65C"),
		Cover:    lipgloss.Color("#182133"),
	},
	{
		Name:     "midnight",
		Base:     lipgloss.Color("#101722"),
		Ink:      lipgloss.Color("#F4F7FB"),
		Muted:    lipgloss.Color("#92A0B6"),
		Subtle:   lipgloss.Color("#556274"),
		Panel:    lipgloss.Color("#1F2733"),
		PanelHi:  lipgloss.Color("#2E394B"),
		Accent:   lipgloss.Color("#6EE7B7"),
		Accent2:  lipgloss.Color("#7DD3FC"),
		Warn:     lipgloss.Color("#FBBF24"),
		Error:    lipgloss.Color("#F87171"),
		Selected: lipgloss.Color("#A7F3D0"),
		Cover:    lipgloss.Color("#182130"),
	},
}

var (
	colorInk      lipgloss.Color
	colorMuted    lipgloss.Color
	colorSubtle   lipgloss.Color
	colorPanel    lipgloss.Color
	colorPanelHi  lipgloss.Color
	colorAccent   lipgloss.Color
	colorAccent2  lipgloss.Color
	colorWarn     lipgloss.Color
	colorError    lipgloss.Color
	colorSelected lipgloss.Color

	baseStyle    lipgloss.Style
	panelStyle   lipgloss.Style
	sidebarStyle lipgloss.Style
	titleStyle   lipgloss.Style

	subtitleStyle lipgloss.Style
	labelStyle    lipgloss.Style
	accentStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	coverStyle    lipgloss.Style
	trackStyle    lipgloss.Style
	hintStyle     lipgloss.Style
	errorStyle    lipgloss.Style
	messageStyle  lipgloss.Style
	spinnerStyle  lipgloss.Style
)

var activeThemeIndex int
var activeThemeName string

func init() {
	applyTheme(visualThemes[0].Name)
}

func applyTheme(name string) string {
	index := themeIndex(name)
	if index < 0 {
		index = 0
	}
	activeThemeIndex = index
	activeThemeName = visualThemes[index].Name
	theme := visualThemes[index]

	colorInk = theme.Ink
	colorMuted = theme.Muted
	colorSubtle = theme.Subtle
	colorPanel = theme.Panel
	colorPanelHi = theme.PanelHi
	colorAccent = theme.Accent
	colorAccent2 = theme.Accent2
	colorWarn = theme.Warn
	colorError = theme.Error
	colorSelected = theme.Selected

	baseStyle = lipgloss.NewStyle().
		Foreground(colorInk)

	panelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle).
		Padding(0, 2)

	sidebarStyle = panelStyle.Copy().
		BorderForeground(colorPanelHi).
		Padding(1, 1)

	titleStyle = lipgloss.NewStyle().
		Foreground(colorInk).
		Bold(true)

	subtitleStyle = lipgloss.NewStyle().
		Foreground(colorMuted)

	labelStyle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Transform(strings.ToUpper)

	accentStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	selectedStyle = lipgloss.NewStyle().
		Foreground(colorSelected).
		Bold(true).
		Padding(0, 1)

	coverStyle = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(colorAccent2).
		Foreground(colorInk).
		Padding(0, 2)

	trackStyle = lipgloss.NewStyle().
		Foreground(colorInk).
		Padding(1, 2)

	hintStyle = lipgloss.NewStyle().
		Foreground(colorMuted)

	errorStyle = lipgloss.NewStyle().
		Foreground(colorError).
		Bold(true)

	messageStyle = lipgloss.NewStyle().
		Foreground(colorWarn)

	spinnerStyle = lipgloss.NewStyle().
		Foreground(colorAccent2).
		Bold(true)

	return activeThemeName
}

func cycleTheme() string {
	if len(visualThemes) == 0 {
		return ""
	}
	return applyTheme(visualThemes[(activeThemeIndex+1)%len(visualThemes)].Name)
}

func themeIndex(name string) int {
	name = strings.ToLower(strings.TrimSpace(name))
	for i, theme := range visualThemes {
		if strings.ToLower(theme.Name) == name {
			return i
		}
	}
	return -1
}
