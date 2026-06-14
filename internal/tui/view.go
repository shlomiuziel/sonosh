package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	minWidth       = 72
	sidebarWidth   = 28
	compactAtWidth = 92
	borderChrome   = 2
)

func (m Model) renderApp() string {
	width := m.width
	if width <= 0 {
		width = 100
	}
	if width < minWidth {
		width = minWidth
	}

	contentWidth := width - 2
	body := m.renderBody(contentWidth)
	footer := m.renderFooter(contentWidth)

	view := lipgloss.JoinVertical(lipgloss.Left, body, footer)
	return baseStyle.Width(width).Render(view)
}

func (m Model) renderBody(width int) string {
	if width < compactAtWidth {
		sections := []string{
			m.renderHeader(width),
			m.renderNowPlaying(width),
			m.renderRooms(width),
		}
		if search := m.renderSearchIfActive(width); search != "" {
			sections = append(sections, search)
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			sections...,
		)
	}

	left := m.renderRooms(sidebarWidth)
	rightWidth := width - sidebarWidth - 1
	right := m.renderRightPane(rightWidth)
	separatorHeight := max(lipgloss.Height(left), lipgloss.Height(right))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, paneSeparator(separatorHeight), right)
}

func (m Model) renderHeader(width int) string {
	return m.renderHeaderContent(width)
}

func (m Model) renderHeaderContent(width int) string {
	state := normalizeState(m.status.State)
	room := "No room"
	if len(m.rooms) > 0 {
		room = m.selectedRoom().Name
	}

	status := state
	if m.loading {
		status = status + " / syncing"
	}
	statusView := statusPill(status)
	if m.loading {
		statusView = lipgloss.JoinHorizontal(lipgloss.Center, spinnerStyle.Render(m.spinner()), " ", statusView)
	}
	line := lipgloss.JoinHorizontal(
		lipgloss.Center,
		accentStyle.Render("sonosh"),
		"  ",
		subtitleStyle.Render(displayText(room, max(12, width/3))),
		"  ",
		statusView,
	)
	return lipgloss.NewStyle().Width(width).Padding(0, 1).Render(line)
}

func (m Model) renderRooms(width int) string {
	contentWidth := max(1, width-borderChrome)
	var lines []string
	lines = append(lines, labelStyle.Render("Rooms"))
	if len(m.rooms) == 0 {
		lines = append(lines, subtitleStyle.Render("No rooms found"))
		lines = append(lines, hintStyle.Render("Press r to discover"))
		return sidebarStyle.Width(contentWidth).Render(strings.Join(lines, "\n"))
	}

	for i, room := range m.rooms {
		nameWidth := max(8, contentWidth-4)
		name := displayText(room.Name, nameWidth)
		members := displayText(strings.Join(room.GroupMembers, ", "), nameWidth)
		if members == "" {
			members = room.IP
		}
		row := fmt.Sprintf("%s\n%s", titleStyle.Render(name), subtitleStyle.Render(members))
		if i == m.roomIndex {
			row = selectedStyle.Width(max(1, contentWidth-4)).Render(row)
		} else {
			row = lipgloss.NewStyle().Padding(0, 1).Render(row)
		}
		lines = append(lines, row)
	}

	return sidebarStyle.Width(contentWidth).Render(strings.Join(lines, "\n"))
}

func (m Model) renderNowPlaying(width int) string {
	content := m.renderNowPlayingContent(width)
	return panelStyle.Width(max(1, width-borderChrome)).Render(content)
}

func (m Model) renderNowPlayingContent(width int) string {
	innerWidth := max(24, width-6)
	coverWidth := 22
	detailsWidth := innerWidth - coverWidth - 3
	if width < compactAtWidth {
		coverWidth = min(28, innerWidth)
		detailsWidth = innerWidth
	}

	cover := m.renderCover(coverWidth)
	details := m.renderTrackDetails(detailsWidth)

	var content string
	if width < compactAtWidth {
		content = lipgloss.JoinVertical(lipgloss.Center, cover, details)
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, cover, "   ", details)
	}
	return content
}

func (m Model) renderRightPane(width int) string {
	contentWidth := max(1, width-borderChrome)
	var parts []string
	parts = append(parts, m.renderHeaderContent(contentWidth))
	parts = append(parts, m.renderNowPlayingContent(contentWidth))
	if search := m.renderSearchContent(contentWidth); search != "" {
		parts = append(parts, search)
	}
	return panelStyle.Width(contentWidth).Render(strings.Join(parts, "\n"))
}

func (m Model) renderCover(width int) string {
	contentWidth := max(1, width-borderChrome)
	innerWidth := max(1, contentWidth-coverStyle.GetHorizontalPadding())
	title := empty(m.status.Title, "No Track")
	artist := empty(m.status.Artist, "sonosh")
	initials := coverInitials(title, artist)
	albumLine := displayText(empty(m.status.Album, "Sonos"), max(8, innerWidth))
	coverArt := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render(initials)
	centerArt := true
	if strings.TrimSpace(m.artView) != "" && m.artURL == strings.TrimSpace(m.status.AlbumArt) {
		coverArt = renderCoverArt(innerWidth, m.artView)
		centerArt = false
	}
	if centerArt {
		coverArt = centerLine(innerWidth, coverArt)
	}

	coverText := strings.Join([]string{
		centerLine(innerWidth, labelStyle.Render("Now Playing")),
		coverArt,
		centerLine(innerWidth, subtitleStyle.Render(albumLine)),
	}, "\n")

	return coverStyle.Width(contentWidth).Height(12).Render(coverText)
}

func renderCoverArt(width int, art string) string {
	artWidth := albumArtColumns
	if width > 0 && width < artWidth {
		artWidth = max(1, width)
	}
	leftPad := max(0, (width-artWidth)/2)
	prefix := strings.Repeat(" ", leftPad)
	blank := prefix + strings.Repeat(" ", artWidth)

	lines := []string{""}
	if strings.Contains(art, "\x1b_G") {
		lines = append(lines, prefix+art+strings.Repeat(" ", artWidth))
		for i := 1; i < albumArtRows; i++ {
			lines = append(lines, blank)
		}
		return strings.Join(lines, "\n")
	}

	artLines := strings.Split(art, "\n")
	for i := 0; i < albumArtRows; i++ {
		if i < len(artLines) {
			lines = append(lines, prefix+artLines[i])
		} else {
			lines = append(lines, blank)
		}
	}
	return strings.Join(lines, "\n")
}

func centerLine(width int, value string) string {
	if width <= 0 {
		return value
	}
	lineWidth := lipgloss.Width(value)
	if lineWidth >= width {
		return value
	}
	return strings.Repeat(" ", (width-lineWidth)/2) + value
}

func (m Model) renderTrackDetails(width int) string {
	progressLabel := fmt.Sprintf("%s / %s", empty(m.status.Position, "--:--"), empty(m.status.Duration, "--:--"))
	progress := renderBar(progressRatio(m.status.Position, m.status.Duration), max(8, width-lipgloss.Width(progressLabel)-3), colorAccent)
	volumeLabel := fmt.Sprintf("%3d%%  %s", clamp(m.status.Volume, 0, 100), muteLabel(m.status.Muted))
	volume := renderBar(float64(clamp(m.status.Volume, 0, 100))/100, max(8, width-lipgloss.Width(volumeLabel)-3), colorAccent2)

	title := titleStyle.
		Foreground(colorInk).
		Bold(true).
		Render(displayText(empty(m.status.Title, "Nothing playing"), width))
	artist := subtitleStyle.Render(displayText(empty(m.status.Artist, "Unknown artist"), width))
	album := subtitleStyle.Render(displayText(empty(m.status.Album, "Unknown album"), width))

	rows := []string{
		labelStyle.Render("Track"),
		title,
		artist,
		album,
		"",
		fmt.Sprintf("%s  %s", progress, progressLabel),
		fmt.Sprintf("%s  %s", volume, volumeLabel),
		"",
		m.renderTransport(width),
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(rows, "\n"))
}

func (m Model) renderTransport(width int) string {
	play := "Play"
	if strings.EqualFold(m.status.State, "PLAYING") {
		play = "Pause"
	}
	controls := []string{
		keycap("space", play),
		keycap("p", "Prev"),
		keycap("n", "Next"),
		keycap("s", "Stop"),
		keycap("+/-", "Vol"),
		keycap("m", "Mute"),
		keycap("/", "Search"),
	}
	line := strings.Join(controls, "  ")
	return truncate(line, width)
}

func (m Model) renderSearchIfActive(width int) string {
	if m.mode != modeSearch {
		return ""
	}
	return m.renderSearchPanel(width)
}

func (m Model) renderSearchPanel(width int) string {
	content := m.renderSearchContent(width)
	return panelStyle.BorderForeground(colorSelected).Width(max(1, width-borderChrome)).Render(content)
}

func (m Model) renderSearchContent(width int) string {
	contentWidth := max(1, width-borderChrome)
	var lines []string
	query := m.searchQuery
	if query == "" {
		query = "type to search..."
	}
	lines = append(lines, lipgloss.JoinHorizontal(
		lipgloss.Center,
		labelStyle.Render(m.config.SearchService+" / "+m.searchCategory),
		"  ",
		accentStyle.Render("> "+displayText(query, max(10, width-24))),
	))

	if m.searchPreviewQuery != "" && m.searchPreviewQuery != m.searchQuery {
		lines = append(lines, subtitleStyle.Render("updating results for "+displayText(m.searchQuery, max(10, contentWidth-22))))
	} else if m.searchPreviewQuery != "" {
		lines = append(lines, subtitleStyle.Render("results for "+displayText(m.searchPreviewQuery, max(10, contentWidth-12))))
	}

	if len(m.searchItems) == 0 {
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("Search results will appear here"))
	} else {
		limit := min(len(m.searchItems), 8)
		for i := 0; i < limit; i++ {
			item := m.searchItems[i]
			row := fmt.Sprintf("%2d  %-9s  %s", i+1, truncate(item.Item.ItemType, 9), displayText(item.Title(), max(8, contentWidth-18)))
			if i == m.searchIndex {
				row = selectedStyle.Width(max(1, contentWidth-4)).Render(row)
			} else {
				row = lipgloss.NewStyle().PaddingLeft(1).Render(row)
			}
			lines = append(lines, row)
		}
		if len(m.searchItems) > limit {
			lines = append(lines, subtitleStyle.Render(fmt.Sprintf("+%d more", len(m.searchItems)-limit)))
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderFooter(width int) string {
	var status string
	if m.err != nil {
		status = errorStyle.Render("Error: " + displayText(m.err.Error(), max(10, width-8)))
	} else if m.loading {
		status = accentStyle.Render(m.spinner() + " loading")
	} else if m.message != "" {
		status = messageStyle.Render(displayText(m.message, max(10, width-8)))
	} else {
		status = hintStyle.Render("q quit  tab switch  r refresh  ctrl+v theme")
	}

	theme := themePill(m.themeName)
	if theme == "" {
		theme = themePill(activeThemeName)
	}
	themeHint := hintStyle.Render("ctrl+v")
	keys := "arrows/jk move  enter play  / search"
	if m.mode == modeSearch {
		keys = "enter play  ctrl+t tracks  ctrl+p playlists  esc close"
	}
	themeWidth := 0
	if theme != "" {
		themeWidth = lipgloss.Width(theme) + lipgloss.Width(themeHint) + 1
	}
	available := max(0, width-lipgloss.Width(status)-themeWidth-4)
	keys = footerKeys(keys, available)
	segments := []string{status}
	if keys != "" {
		segments = append(segments, "    ", hintStyle.Render(keys))
	}
	if theme != "" && width-lipgloss.Width(status)-lipgloss.Width(keys)-4 > themeWidth {
		segments = append(segments, "  ", theme, " ", themeHint)
	}
	return lipgloss.NewStyle().Width(width).Padding(0, 1).Render(lipgloss.JoinHorizontal(lipgloss.Top, segments...))
}

func footerKeys(value string, width int) string {
	if width <= 0 {
		return ""
	}
	choices := []string{value}
	switch value {
	case "enter play  ctrl+t tracks  ctrl+p playlists  esc close":
		choices = append(choices,
			"enter play  ctrl+t  ctrl+p  esc",
			"enter  esc",
		)
	default:
		choices = append(choices,
			"move  enter  /",
			"/",
		)
	}
	for _, choice := range choices {
		if lipgloss.Width(choice) <= width {
			return choice
		}
	}
	return ""
}

func paneSeparator(height int) string {
	return lipgloss.NewStyle().
		Foreground(colorSubtle).
		Background(colorPanel).
		Width(1).
		Height(max(1, height)).
		Render("│")
}

func statusPill(state string) string {
	state = truncate(empty(state, "idle"), 18)
	style := lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(lipgloss.Color("#101820"))
	switch strings.ToLower(state) {
	case "playing":
		return style.Background(colorAccent).Render(state)
	case "paused", "paused playback":
		return style.Background(colorWarn).Render(state)
	default:
		return style.Background(colorSubtle).Foreground(colorInk).Render(state)
	}
}

func (m Model) spinner() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return frames[m.spinnerFrame%len(frames)]
}

func keycap(key, label string) string {
	keyStyle := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true)
	return keyStyle.Render(key) + " " + subtitleStyle.Render(label)
}

func themePill(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true).
		Foreground(colorPanel).
		Background(colorSelected).
		Render("theme " + name)
}

func muteLabel(muted bool) string {
	if muted {
		return "muted"
	}
	return "live"
}

func renderBar(ratio float64, width int, color lipgloss.Color) string {
	width = max(4, width)
	ratio = maxFloat(0, minFloat(1, ratio))
	filled := int(ratio * float64(width))
	if ratio > 0 && filled == 0 {
		filled = 1
	}
	return lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("━", filled)) +
		lipgloss.NewStyle().Foreground(colorSubtle).Render(strings.Repeat("─", width-filled))
}

func progressRatio(position, duration string) float64 {
	pos, okPos := parseClock(position)
	dur, okDur := parseClock(duration)
	if !okPos || !okDur || dur <= 0 {
		return 0
	}
	return float64(pos) / float64(dur)
}

func parseClock(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "NOT_IMPLEMENTED" {
		return 0, false
	}
	parts := strings.Split(value, ":")
	total := 0
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return 0, false
		}
		total = total*60 + n
	}
	return total, true
}

func coverInitials(title, artist string) string {
	var initials []string
	for _, word := range strings.Fields(title + " " + artist) {
		r := []rune(word)
		if len(r) == 0 {
			continue
		}
		initials = append(initials, strings.ToUpper(string(r[0])))
		if len(initials) == 2 {
			break
		}
	}
	if len(initials) == 0 {
		return "SO"
	}
	return strings.Join(initials, " ")
}

func truncate(value string, width int) string {
	value = strings.TrimSpace(value)
	if width <= 0 || lipgloss.Width(value) <= width {
		return value
	}
	if width <= 1 {
		return "…"
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func normalizeState(state string) string {
	state = strings.TrimSpace(strings.ToLower(state))
	state = strings.ReplaceAll(state, "_", " ")
	if state == "" {
		return "idle"
	}
	return state
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
