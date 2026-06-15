package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	minWidth           = 72
	sidebarWidth       = 28
	queuePaneWidth     = 34
	minPlayerPaneWidth = 54
	paneGapWidth       = 1
	compactAtWidth     = 92
	queueAtWidth       = sidebarWidth + minPlayerPaneWidth + queuePaneWidth + 2*paneGapWidth
	borderChrome       = 2
)

func (m Model) renderApp() string {
	width := m.width
	if width <= 0 {
		width = 100
	}
	if width < minWidth {
		width = minWidth
	}
	height := m.height
	if height <= 0 {
		height = 24
	}

	contentWidth := width - 2
	view := m.renderAppContent(contentWidth)
	rendered := baseStyle.Width(width).Height(height).Render(view)
	if supportsKittyGraphics() {
		return clearKittyGraphics() + rendered
	}
	return rendered
}

func (m Model) renderAppContent(width int) string {
	if m.compactLayout {
		return m.renderCompactAppContent(width)
	}
	if width < compactAtWidth {
		body := m.renderBody(width)
		footer := m.renderFooterRow(width)
		return lipgloss.JoinVertical(lipgloss.Left, body, footer)
	}

	left := m.renderRooms(sidebarWidth)
	rightWidth := width - sidebarWidth - paneGapWidth
	if m.showQueuePane(width) {
		rightWidth -= queuePaneWidth + paneGapWidth
	}
	rightWidth = max(1, rightWidth)
	right := m.renderRightPane(rightWidth)
	footer := m.renderFooterPane(rightWidth)
	spacer := lipgloss.NewStyle().Width(rightWidth).Height(1).Render("")
	center := lipgloss.JoinVertical(lipgloss.Left, right, spacer, footer)

	if m.showQueuePane(width) {
		queue := m.renderQueue(queuePaneWidth)
		separatorHeight := max(lipgloss.Height(left), max(lipgloss.Height(center), lipgloss.Height(queue)))
		return lipgloss.JoinHorizontal(lipgloss.Top, left, paneGap(separatorHeight), center, paneGap(separatorHeight), queue)
	}

	separatorHeight := max(lipgloss.Height(left), lipgloss.Height(center))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, paneGap(separatorHeight), center)
}

func (m Model) renderCompactAppContent(width int) string {
	contentWidth := max(1, min(width, 88))
	main := m.renderRightPane(contentWidth)
	spacer := lipgloss.NewStyle().Width(contentWidth).Height(1).Render("")
	footer := m.renderFooterPane(contentWidth)
	stack := lipgloss.JoinVertical(lipgloss.Left, main, spacer, footer)
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, stack)
}

func (m Model) renderBody(width int) string {
	if width < compactAtWidth {
		sections := []string{
			m.renderHeader(width),
			m.renderNowPlaying(width),
		}
		switch m.mode {
		case modeSearch:
			sections = append(sections, m.renderSearchPanel(width))
		case modePlaybackConfig:
			sections = append(sections, m.renderPlaybackConfigPanel(width))
		default:
			sections = append(sections, m.renderCompactRooms(width))
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			sections...,
		)
	}
	return m.renderDashboardBody(width)
}

func (m Model) renderDashboardBody(width int) string {
	left := m.renderRooms(sidebarWidth)
	if m.showQueuePane(width) {
		rightWidth := width - sidebarWidth - queuePaneWidth - 2*paneGapWidth
		right := m.renderRightPane(rightWidth)
		queue := m.renderQueue(queuePaneWidth)
		separatorHeight := max(lipgloss.Height(left), max(lipgloss.Height(right), lipgloss.Height(queue)))
		return lipgloss.JoinHorizontal(lipgloss.Top, left, paneGap(separatorHeight), right, paneGap(separatorHeight), queue)
	}

	rightWidth := width - sidebarWidth - paneGapWidth
	right := m.renderRightPane(rightWidth)
	separatorHeight := max(lipgloss.Height(left), lipgloss.Height(right))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, paneGap(separatorHeight), right)
}

func (m Model) showQueuePane(width int) bool {
	return width >= queueAtWidth
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
		statusView = lipgloss.JoinHorizontal(lipgloss.Center, spinnerStyle.Render(m.spinner()), paneSpace(1), statusView)
	}
	line := lipgloss.JoinHorizontal(
		lipgloss.Center,
		accentStyle.Render("sonosh"),
		paneSpace(2),
		subtitleStyle.Render(displayText(room, max(12, width/3))),
		paneSpace(2),
		statusView,
	)
	return lipgloss.NewStyle().Width(width).Padding(0, 1).Background(colorPanel).Render(line)
}

func (m Model) renderRooms(width int) string {
	contentWidth := max(1, width-borderChrome)
	rowWidth := max(1, contentWidth-sidebarStyle.GetHorizontalPadding())
	var lines []string
	lines = append(lines, labelStyle.Width(rowWidth).Render("Rooms"))
	if len(m.rooms) == 0 {
		lines = append(lines, subtitleStyle.Width(rowWidth).Render("No rooms found"))
		lines = append(lines, hintStyle.Width(rowWidth).Render("Press r to discover"))
		return sidebarStyle.Width(contentWidth).Render(strings.Join(lines, "\n"))
	}

	for i, room := range m.rooms {
		nameWidth := max(8, contentWidth-5)
		name := displayText(room.Name, nameWidth)
		members := displayText(strings.Join(room.GroupMembers, ", "), nameWidth)
		if members == "" {
			members = room.IP
		}
		if i == m.roomIndex {
			lines = append(lines, selectedRoomRow(name, members, rowWidth))
		} else {
			lines = append(lines, roomRow(name, members, rowWidth))
		}
	}

	return sidebarStyle.Width(contentWidth).Render(strings.Join(lines, "\n"))
}

func (m Model) renderCompactRooms(width int) string {
	contentWidth := max(1, width-borderChrome)
	rowWidth := max(1, contentWidth-2)
	var lines []string
	lines = append(lines, labelStyle.Width(rowWidth).Render("Rooms"))
	if len(m.rooms) == 0 {
		lines = append(lines, subtitleStyle.Width(rowWidth).Render("No rooms found"))
		lines = append(lines, hintStyle.Width(rowWidth).Render("Press r to discover"))
		return compactSidebarStyle().Width(contentWidth).Render(strings.Join(lines, "\n"))
	}

	for i, room := range m.rooms {
		lines = append(lines, compactRoomRow(displayText(room.Name, max(8, rowWidth-2)), rowWidth, i == m.roomIndex))
	}

	return compactSidebarStyle().Width(contentWidth).Render(strings.Join(lines, "\n"))
}

func (m Model) renderQueue(width int) string {
	contentWidth := max(1, width-borderChrome)
	style := panelStyle.Copy()
	if m.dashboardFocus == focusQueue {
		style = style.BorderForeground(colorSelected)
	}
	return style.Width(contentWidth).Render(m.renderQueueContent(contentWidth))
}

func (m Model) renderQueueContent(width int) string {
	rowWidth := max(1, width-panelStyle.GetHorizontalPadding())
	lines := []string{labelStyle.Width(rowWidth).Render("Queue")}
	if m.queueTotal > 0 {
		lines = append(lines, subtitleStyle.Width(rowWidth).Render(fmt.Sprintf("%d tracks", m.queueTotal)))
	}
	if m.queueErr != nil {
		lines = append(lines, "")
		lines = append(lines, errorStyle.Width(rowWidth).Render(displayText(m.queueErr.Error(), rowWidth)))
		return paneBlock(rowWidth, lipgloss.Height(strings.Join(lines, "\n"))).Render(strings.Join(lines, "\n"))
	}
	if m.queueLoading && len(m.queueItems) == 0 {
		lines = append(lines, "")
		lines = append(lines, spinnerStyle.Width(rowWidth).Render(m.spinner()+" loading queue"))
		return paneBlock(rowWidth, lipgloss.Height(strings.Join(lines, "\n"))).Render(strings.Join(lines, "\n"))
	}
	if len(m.queueItems) == 0 {
		lines = append(lines, "")
		lines = append(lines, hintStyle.Width(rowWidth).Render("Queue is empty"))
		return paneBlock(rowWidth, lipgloss.Height(strings.Join(lines, "\n"))).Render(strings.Join(lines, "\n"))
	}

	visible := min(m.queueVisibleRows(), len(m.queueItems)-m.queueOffset)
	for i := 0; i < visible; i++ {
		index := m.queueOffset + i
		item := m.queueItems[index]
		lines = append(lines, queueRow(item, rowWidth, index == m.queueIndex, item.Position == m.status.QueuePosition))
	}
	if m.queueOffset+visible < len(m.queueItems) {
		lines = append(lines, subtitleStyle.Width(rowWidth).Render(fmt.Sprintf("+%d more", len(m.queueItems)-m.queueOffset-visible)))
	}
	if m.dashboardFocus == focusQueue {
		lines = append(lines, "")
		lines = append(lines, hintStyle.Width(rowWidth).Render("enter play  x remove  [] move"))
	}
	return paneBlock(rowWidth, lipgloss.Height(strings.Join(lines, "\n"))).Render(strings.Join(lines, "\n"))
}

func queueRow(item QueueItem, width int, selected, playing bool) string {
	marker := " "
	if playing {
		marker = "▶"
	} else if selected {
		marker = "▸"
	}
	detail := strings.TrimSpace(item.Artist)
	if detail == "" {
		detail = strings.TrimSpace(item.Album)
	}
	title := empty(item.Title, item.URI)
	if strings.TrimSpace(title) == "" {
		title = "Untitled"
	}
	text := fmt.Sprintf("%2d %s", item.Position, title)
	if detail != "" {
		text += " - " + detail
	}
	contentWidth := max(1, width-lipgloss.Width(marker))
	markerView := paneText(marker)
	if playing || selected {
		markerView = accentStyle.Render(marker)
	}
	if selected {
		contentWidth = max(1, contentWidth-selectedStyle.GetHorizontalPadding())
		content := selectedStyle.Width(width - lipgloss.Width(marker)).Render(displayText(text, contentWidth))
		return paneBlock(width, 1).Render(markerView + content)
	}
	content := subtitleStyle.Width(contentWidth).Render(displayText(text, contentWidth))
	if playing {
		content = titleStyle.Width(contentWidth).Render(displayText(text, contentWidth))
	}
	return paneBlock(width, 1).Render(markerView + content)
}

func (m Model) renderNowPlaying(width int) string {
	content := m.renderNowPlayingContent(width)
	return panelStyle.Width(max(1, width-borderChrome)).Render(content)
}

func (m Model) renderNowPlayingContent(width int) string {
	innerWidth := max(24, width-6)
	coverWidth := 22
	detailsWidth := innerWidth - coverWidth - 3
	if width < compactAtWidth || m.mode == modeSearch {
		coverWidth = min(28, innerWidth)
		detailsWidth = innerWidth
	}

	details := m.renderTrackDetails(detailsWidth)

	var content string
	if m.mode == modeSearch {
		content = lipgloss.JoinVertical(lipgloss.Center, details)
	} else if width < compactAtWidth {
		if m.compactLayout {
			coverWidth = min(24, innerWidth)
		}
		cover := m.renderCover(coverWidth)
		if m.compactLayout {
			cover = lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, cover, lipgloss.WithWhitespaceBackground(colorPanel))
		}
		content = lipgloss.JoinVertical(lipgloss.Center, cover, details)
	} else {
		cover := m.renderCover(coverWidth)
		rowHeight := max(lipgloss.Height(cover), lipgloss.Height(details))
		cover = paneBlock(lipgloss.Width(cover), rowHeight).Render(cover)
		details = paneBlock(lipgloss.Width(details), rowHeight).Render(details)
		content = lipgloss.JoinHorizontal(lipgloss.Top, cover, paneBlock(3, rowHeight).Render(""), details)
	}
	return content
}

func (m Model) renderRightPane(width int) string {
	contentWidth := max(1, width-borderChrome)
	switch m.mode {
	case modeSearch:
		return panelStyle.
			BorderForeground(colorSelected).
			Width(contentWidth).
			Render(m.renderSearchSpotlight(contentWidth))
	case modePlaybackConfig:
		return panelStyle.
			BorderForeground(colorSelected).
			Width(contentWidth).
			Render(m.renderPlaybackConfigSpotlight(contentWidth))
	}

	var parts []string
	parts = append(parts, m.renderHeaderContent(contentWidth))
	parts = append(parts, m.renderNowPlayingContent(contentWidth))
	return panelStyle.Width(contentWidth).Render(strings.Join(parts, "\n"))
}

func (m Model) renderCover(width int) string {
	contentWidth := max(1, width-borderChrome)
	innerWidth := max(1, contentWidth-coverStyle.GetHorizontalPadding())
	title := empty(m.status.Title, "No Track")
	artist := empty(m.status.Artist, "sonosh")
	initials := coverInitials(title, artist)
	albumLine := displayText(empty(m.status.Album, "Sonos"), max(8, innerWidth))
	coverArt := lipgloss.NewStyle().Foreground(colorAccent2).Background(colorPanel).Bold(true).Render(initials)
	centerArt := true
	if strings.TrimSpace(m.artView) != "" && m.artURL == strings.TrimSpace(m.status.AlbumArt) {
		coverArt = renderCoverArt(innerWidth, m.artView)
		centerArt = !m.compactLayout
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
	contentWidth := max(1, width-trackStyle.GetHorizontalPadding())
	progressLabel := fmt.Sprintf("%s / %s", empty(m.status.Position, "--:--"), empty(m.status.Duration, "--:--"))
	progress := renderBar(progressRatio(m.status.Position, m.status.Duration), max(8, contentWidth-lipgloss.Width(progressLabel)-3), colorAccent)
	volumeLabel := fmt.Sprintf("%3d%%  %s", clamp(m.status.Volume, 0, 100), muteLabel(m.status.Muted))
	volume := renderBar(float64(clamp(m.status.Volume, 0, 100))/100, max(8, contentWidth-lipgloss.Width(volumeLabel)-3), colorAccent2)

	title := titleStyle.
		Foreground(colorInk).
		Bold(true).
		Render(displayText(empty(m.status.Title, "Nothing playing"), contentWidth))
	artist := subtitleStyle.Render(displayText(empty(m.status.Artist, "Unknown artist"), contentWidth))
	album := subtitleStyle.Render(displayText(empty(m.status.Album, "Unknown album"), contentWidth))

	rows := []string{
		labelStyle.Render("Track"),
		title,
		artist,
		album,
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, progress, paneSpace(2), paneText(progressLabel)),
		lipgloss.JoinHorizontal(lipgloss.Top, volume, paneSpace(2), paneText(volumeLabel)),
		"",
		m.renderTransport(contentWidth),
	}
	return trackStyle.Width(width).Render(strings.Join(rows, "\n"))
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
		keycap("o", "Options"),
		keycap("/", "Search"),
	}
	line := strings.Join(controls, paneSpace(2))
	return truncate(line, width)
}

func (m Model) renderPlaybackConfigPanel(width int) string {
	content := m.renderPlaybackConfigContent(max(1, width-borderChrome))
	return panelStyle.BorderForeground(colorSelected).Width(max(1, width-borderChrome)).Render(content)
}

func (m Model) renderPlaybackConfigSpotlight(width int) string {
	modalWidth := min(width-6, 58)
	if modalWidth < 36 {
		modalWidth = max(1, width-2)
	}
	content := m.renderPlaybackConfigContent(modalWidth)
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSelected).
		Padding(1, 2).
		Width(modalWidth).
		Background(colorPanel).
		Render(content)
	modalHeight := lipgloss.Height(modal)
	height := m.height
	if height <= 0 {
		height = 24
	}
	targetHeight := max(modalHeight, height-6)
	return lipgloss.Place(width, targetHeight, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) renderPlaybackConfigContent(width int) string {
	contentWidth := max(1, width-borderChrome)
	state := "unknown"
	if m.status.CrossfadeKnown {
		state = onOff(m.status.CrossfadeEnabled)
	}
	rows := []string{
		labelStyle.Width(contentWidth).Render("Playback"),
		playbackSettingRow("Crossfade", state, m.status.CrossfadeKnown, contentWidth),
		"",
		hintStyle.Width(contentWidth).Render("space toggle  esc close"),
	}
	return paneBlock(width, lipgloss.Height(strings.Join(rows, "\n"))).Render(strings.Join(rows, "\n"))
}

func playbackSettingRow(label, state string, known bool, width int) string {
	pill := togglePill(state, known)
	labelWidth := max(1, width-lipgloss.Width(pill)-2)
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		titleStyle.Width(labelWidth).Render(displayText(label, labelWidth)),
		paneSpace(2),
		pill,
	)
}

func togglePill(value string, known bool) string {
	style := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true).
		Foreground(colorPanel)
	if !known {
		return style.Foreground(colorMuted).Background(colorPanelHi).Render("unknown")
	}
	if value == "on" {
		return style.Background(colorAccent).Render("on")
	}
	return style.Background(colorSubtle).Foreground(colorInk).Render("off")
}

func (m Model) renderSearchPanel(width int) string {
	content := m.renderSearchContent(width)
	return panelStyle.BorderForeground(colorSelected).Width(max(1, width-borderChrome)).Render(content)
}

func (m Model) renderSearchSpotlight(width int) string {
	modalWidth := min(width-6, 78)
	if modalWidth < 44 {
		modalWidth = max(1, width-2)
	}
	content := m.renderSearchContent(modalWidth)
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSelected).
		Padding(1, 2).
		Width(modalWidth).
		Render(content)
	modalHeight := lipgloss.Height(modal)
	height := m.height
	if height <= 0 {
		height = 24
	}
	targetHeight := max(modalHeight, height-6)
	return lipgloss.Place(width, targetHeight, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) renderSearchContent(width int) string {
	contentWidth := max(1, width-borderChrome)
	var lines []string
	query := m.searchQuery
	if query == "" {
		query = "type to search..."
	}
	lines = append(lines, labelStyle.Render(m.config.SearchService+" / "+m.searchCategory))
	lines = append(lines, renderSearchField(query, contentWidth))

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
				row = selectedLine(row, max(1, contentWidth-4))
			} else {
				row = lipgloss.NewStyle().PaddingLeft(1).Render(row)
			}
			lines = append(lines, row)
		}
		if len(m.searchItems) > limit {
			lines = append(lines, subtitleStyle.Render(fmt.Sprintf("+%d more", len(m.searchItems)-limit)))
		}
	}

	if preview := m.renderPlaylistPreview(contentWidth); preview != "" {
		lines = append(lines, "")
		lines = append(lines, preview)
	}

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("enter play  ctrl+t tracks  ctrl+p playlists  esc close"))

	return strings.Join(lines, "\n")
}

func renderSearchField(query string, width int) string {
	innerWidth := max(8, width-8)
	display := displayText(query, innerWidth)
	if strings.TrimSpace(query) == "" {
		display = hintStyle.Render(display)
	} else {
		display = titleStyle.Foreground(colorInk).Render(display)
	}
	field := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSelected).
		Foreground(colorInk).
		Width(max(16, width-4)).
		Render("> " + display)
	return field
}

func selectedLine(value string, width int) string {
	marker := accentStyle.Render("▸")
	contentWidth := max(1, width-lipgloss.Width("▸"))
	textWidth := max(1, contentWidth-selectedStyle.GetHorizontalPadding())
	content := selectedStyle.Width(contentWidth).Render(displayText(value, textWidth))
	return paneBlock(width, 1).Render(marker + content)
}

func roomRow(name, members string, width int) string {
	contentWidth := max(1, width-2)
	nameLine := titleStyle.Width(contentWidth).Render(displayText(name, contentWidth))
	memberLine := subtitleStyle.Width(contentWidth).Render(displayText(members, contentWidth))
	return lipgloss.NewStyle().
		Width(width).
		Background(colorPanel).
		Padding(0, 1).
		Render(nameLine + "\n" + memberLine)
}

func selectedRoomRow(name, members string, width int) string {
	contentWidth := max(1, width-lipgloss.Width("▸"))
	textWidth := max(1, contentWidth-selectedStyle.GetHorizontalPadding())
	nameLine := accentStyle.Render("▸") + selectedStyle.Width(contentWidth).Render(displayText(name, textWidth))
	memberLine := paneSpace(1) + subtitleStyle.Foreground(colorSelected).Background(colorPanel).Bold(true).Width(contentWidth).Render(displayText(members, contentWidth))
	return paneBlock(width, 2).Render(nameLine + "\n" + memberLine)
}

func compactRoomRow(name string, width int, selected bool) string {
	marker := " "
	if selected {
		marker = "▸"
	}
	contentWidth := max(1, width-lipgloss.Width(marker))
	textWidth := contentWidth
	markerView := paneText(marker)
	content := titleStyle.Width(contentWidth).Render(displayText(name, textWidth))
	if selected {
		markerView = accentStyle.Render(marker)
		textWidth = max(1, contentWidth-selectedStyle.GetHorizontalPadding())
		content = selectedStyle.Width(contentWidth).Render(displayText(name, textWidth))
	}
	return paneBlock(width, 1).Render(markerView + content)
}

func compactSidebarStyle() lipgloss.Style {
	return panelStyle.Copy().
		BorderForeground(colorPanelHi).
		Padding(0, 1)
}

func (m Model) renderPlaylistPreview(width int) string {
	if m.searchPreviewItemID == "" && !m.searchPreviewLoading && len(m.searchPreviewItems) == 0 {
		return ""
	}
	var lines []string
	lines = append(lines, labelStyle.Render("Playlist preview"))
	if m.searchPreviewLoading && len(m.searchPreviewItems) == 0 {
		lines = append(lines, subtitleStyle.Render("Loading tracks..."))
		return strings.Join(lines, "\n")
	}
	if len(m.searchPreviewItems) == 0 {
		lines = append(lines, subtitleStyle.Render("No tracks found"))
		return strings.Join(lines, "\n")
	}
	limit := min(len(m.searchPreviewItems), 6)
	for i := 0; i < limit; i++ {
		item := m.searchPreviewItems[i]
		row := fmt.Sprintf("%2d  %s", i+1, displayText(item.Title(), max(8, width-6)))
		lines = append(lines, row)
	}
	if len(m.searchPreviewItems) > limit {
		lines = append(lines, subtitleStyle.Render(fmt.Sprintf("+%d more", len(m.searchPreviewItems)-limit)))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderFooterContent(width int) string {
	var status string
	if m.err != nil {
		status = errorStyle.Render("Error: " + displayText(m.err.Error(), max(10, width-8)))
	} else if m.loading {
		status = footerMessageStyle().Render(m.spinner() + " loading")
	} else if m.message != "" {
		status = footerMessageStyle().Render(displayText(m.message, max(10, width-8)))
	} else {
		hint := "q quit  tab queue  r refresh  ctrl+v theme"
		if m.compactLayout {
			hint = "q quit  ctrl+l full  r refresh  ctrl+v theme"
		}
		status = footerHintStyle().Render(hint)
	}

	theme := themePill(m.themeName)
	if theme == "" {
		theme = themePill(activeThemeName)
	}
	themeHint := footerHintStyle().Render("ctrl+v")
	layoutHint := "ctrl+l compact"
	if m.compactLayout {
		layoutHint = "ctrl+l full"
	}
	keys := layoutHint + "  arrows/jk move  enter play  o options  / search"
	if m.compactLayout {
		keys = layoutHint + "  o options  / search"
	}
	if m.mode == modeSearch {
		keys = "enter play  ctrl+t tracks  ctrl+p playlists  esc close"
	} else if m.mode == modePlaybackConfig {
		keys = "space toggle  esc close"
	} else if m.dashboardFocus == focusQueue {
		keys = "enter play  x remove  [] move  X clear  esc  " + layoutHint
	}
	themeWidth := 0
	if theme != "" {
		themeWidth = lipgloss.Width(theme) + lipgloss.Width(themeHint) + 1
	}
	available := max(0, width-lipgloss.Width(status)-themeWidth-4)
	keys = footerKeys(keys, available)
	segments := []string{status}
	if keys != "" {
		segments = append(segments, "    ", footerHintStyle().Render(keys))
	}
	if theme != "" && width-lipgloss.Width(status)-lipgloss.Width(keys)-4 > themeWidth {
		segments = append(segments, "  ", theme, " ", themeHint)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, segments...)
}

func (m Model) renderFooter(width int) string {
	return lipgloss.NewStyle().Width(width).Padding(0, 1).Render(m.renderFooterContent(width))
}

func (m Model) renderFooterPane(width int) string {
	footer := m.renderFooterContent(width)
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, footer)
}

func (m Model) renderFooterRow(width int) string {
	if width < compactAtWidth {
		return m.renderFooter(width)
	}
	rightWidth := width - sidebarWidth - paneGapWidth
	if m.showQueuePane(width) {
		rightWidth -= queuePaneWidth + paneGapWidth
	}
	rightWidth = max(1, rightWidth)
	gutter := lipgloss.NewStyle().
		Width(sidebarWidth + paneGapWidth).
		Render("")
	return lipgloss.JoinHorizontal(lipgloss.Top, gutter, m.renderFooterPane(rightWidth))
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
	case "space toggle  esc close":
		choices = append(choices,
			"space toggle  esc",
			"toggle  esc",
		)
	case "enter play  x remove  [] move  X clear  esc":
		choices = append(choices,
			"enter play  x remove  [] move  esc",
			"enter  x  []  esc",
		)
	default:
		choices = append(choices,
			"move  enter  o  /",
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

func paneGap(height int) string {
	return lipgloss.NewStyle().
		Width(paneGapWidth).
		Height(max(1, height)).
		Render("")
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
		return lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(colorMuted).
			Background(colorPanel).
			Render(state)
	}
}

func (m Model) spinner() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return frames[m.spinnerFrame%len(frames)]
}

func keycap(key, label string) string {
	keyStyle := lipgloss.NewStyle().Foreground(colorAccent2).Background(colorPanel).Bold(true)
	return keyStyle.Render(key) + paneSpace(1) + subtitleStyle.Render(label)
}

func paneSpace(width int) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Background(colorPanel).Render(strings.Repeat(" ", width))
}

func paneBlock(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(max(1, width)).
		Height(max(1, height)).
		Background(colorPanel)
}

func paneText(value string) string {
	return lipgloss.NewStyle().
		Foreground(colorInk).
		Background(colorPanel).
		Render(value)
}

func footerHintStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorMuted)
}

func footerMessageStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorWarn)
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
	return lipgloss.NewStyle().Foreground(color).Background(colorPanel).Render(strings.Repeat("━", filled)) +
		lipgloss.NewStyle().Foreground(colorSubtle).Background(colorPanel).Render(strings.Repeat("─", width-filled))
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
