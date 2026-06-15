package tui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	viewModel := m
	if m.mode == modeSearch || m.mode == modePlaybackConfig {
		viewModel.mode = modeDashboard
		if strings.Contains(viewModel.artView, "\x1b_G") && !strings.Contains(viewModel.artView, "z=-") {
			viewModel.artView = viewModel.artFallbackView
		}
	}
	view := viewModel.renderAppContent(contentWidth)
	rendered := baseStyle.Width(width).Height(height).Render(view)
	if overlay := m.renderOverlay(height, contentWidth); overlay.content != "" {
		rendered = overlayFrame(rendered, overlay.x, overlay.y, overlay.content)
	}
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
	spacer := appSpace(rightWidth, 1)
	center := lipgloss.JoinVertical(lipgloss.Left, right, spacer, footer)

	if m.showQueuePane(width) {
		queue := m.renderQueue(queuePaneWidth)
		separatorHeight := max(lipgloss.Height(left), max(lipgloss.Height(center), lipgloss.Height(queue)))
		left = padColumn(left, sidebarWidth, separatorHeight)
		center = padColumn(center, rightWidth, separatorHeight)
		queue = padColumn(queue, queuePaneWidth, separatorHeight)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, paneGap(separatorHeight), center, paneGap(separatorHeight), queue)
	}

	separatorHeight := max(lipgloss.Height(left), lipgloss.Height(center))
	left = padColumn(left, sidebarWidth, separatorHeight)
	center = padColumn(center, rightWidth, separatorHeight)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, paneGap(separatorHeight), center)
}

func (m Model) renderCompactAppContent(width int) string {
	contentWidth := max(1, min(width, 88))
	main := m.renderRightPane(contentWidth)
	spacer := appSpace(contentWidth, 1)
	footer := m.renderFooterPane(contentWidth)
	stack := lipgloss.JoinVertical(lipgloss.Left, main, spacer, footer)
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, stack, lipgloss.WithWhitespaceBackground(colorBase))
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
		left = padColumn(left, sidebarWidth, separatorHeight)
		right = padColumn(right, rightWidth, separatorHeight)
		queue = padColumn(queue, queuePaneWidth, separatorHeight)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, paneGap(separatorHeight), right, paneGap(separatorHeight), queue)
	}

	rightWidth := width - sidebarWidth - paneGapWidth
	right := m.renderRightPane(rightWidth)
	separatorHeight := max(lipgloss.Height(left), lipgloss.Height(right))
	left = padColumn(left, sidebarWidth, separatorHeight)
	right = padColumn(right, rightWidth, separatorHeight)
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
		cover = lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, cover, lipgloss.WithWhitespaceBackground(colorPanel))
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
	return paneSpace((width-lineWidth)/2) + value
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
		keycap("left/right", "5s"),
		keycap("+/-", "Vol"),
		keycap("m", "Mute"),
		keycap("o", "Options"),
		keycap("/", "Search"),
	}
	line := strings.Join(controls, paneSpace(2))
	return truncatePane(line, width)
}

func (m Model) renderPlaybackConfigPanel(width int) string {
	content := m.renderPlaybackConfigContent(max(1, width-borderChrome))
	return panelStyle.BorderForeground(colorSelected).Width(max(1, width-borderChrome)).Render(content)
}

func (m Model) renderPlaybackConfigSpotlight(width int) string {
	modal := m.renderPlaybackConfigModal(width)
	modalHeight := lipgloss.Height(modal)
	height := m.height
	if height <= 0 {
		height = 24
	}
	targetHeight := max(modalHeight, height-6)
	return lipgloss.Place(width, targetHeight, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) renderPlaybackConfigModal(width int) string {
	modalWidth := min(width-6, 58)
	if modalWidth < 36 {
		modalWidth = max(1, width-2)
	}
	content := m.renderPlaybackConfigContent(modalWidth)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSelected).
		Padding(1, 2).
		Width(modalWidth).
		Background(colorPanel).
		Render(content)
}

func (m Model) renderPlaybackConfigContent(width int) string {
	contentWidth := max(1, width-borderChrome)
	crossfadeState := "unknown"
	if m.status.CrossfadeKnown {
		crossfadeState = onOff(m.status.CrossfadeEnabled)
	}
	shuffleState := "unknown"
	if m.status.ShuffleKnown {
		shuffleState = onOff(m.status.ShuffleEnabled)
	}
	repeatState := "unknown"
	if m.status.RepeatKnown {
		repeatState = m.status.RepeatMode
		if repeatState == "" {
			repeatState = "off"
		}
	}
	rows := []string{
		labelStyle.Width(contentWidth).Render("Playback"),
		playbackSettingRow("Crossfade", crossfadeState, m.status.CrossfadeKnown, contentWidth, m.playbackConfigIndex == 0),
		playbackSettingRow("Shuffle", shuffleState, m.status.ShuffleKnown, contentWidth, m.playbackConfigIndex == 1),
		playbackSettingRow("Repeat", repeatState, m.status.RepeatKnown, contentWidth, m.playbackConfigIndex == 2),
		"",
		hintStyle.Width(contentWidth).Render("up/down move  space toggle  esc close"),
	}
	return paneBlock(width, lipgloss.Height(strings.Join(rows, "\n"))).Render(strings.Join(rows, "\n"))
}

func playbackSettingRow(label, state string, known bool, width int, selected bool) string {
	pill := settingPill(state, known)
	contentWidth := max(1, width-lipgloss.Width(pill)-2)
	marker := paneSpace(2)
	labelWidth := contentWidth
	if selected {
		marker = accentStyle.Render("▸") + paneSpace(1)
		labelWidth = max(1, width-lipgloss.Width(pill)-lipgloss.Width(marker)-2)
	}
	row := lipgloss.JoinHorizontal(
		lipgloss.Center,
		marker,
		titleStyle.Width(labelWidth).Render(displayText(label, labelWidth)),
		paneSpace(2),
		pill,
	)
	if selected {
		return selectedStyle.Width(width).Render(row)
	}
	return row
}

func settingPill(value string, known bool) string {
	style := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true).
		Foreground(colorPanel)
	if !known {
		return style.Foreground(colorMuted).Background(colorPanelHi).Render("unknown")
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "all", "once":
		return style.Background(colorAccent).Render(strings.ToLower(strings.TrimSpace(value)))
	case "off":
		return style.Background(colorSubtle).Foreground(colorInk).Render("off")
	}
	return style.Background(colorSubtle).Foreground(colorInk).Render(displayText(value, 4))
}

func (m Model) renderSearchPanel(width int) string {
	content := m.renderSearchContent(width)
	return panelStyle.BorderForeground(colorSelected).Width(max(1, width-borderChrome)).Render(content)
}

func (m Model) renderSearchSpotlight(width int) string {
	modal := m.renderSearchModal(width)
	modalHeight := lipgloss.Height(modal)
	height := m.height
	if height <= 0 {
		height = 24
	}
	targetHeight := max(modalHeight, height-6)
	return lipgloss.Place(width, targetHeight, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) renderSearchModal(width int) string {
	modalWidth := min(width-6, 78)
	if modalWidth < 44 {
		modalWidth = max(1, width-2)
	}
	content := m.renderSearchContent(modalWidth)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSelected).
		Padding(1, 2).
		Width(modalWidth).
		Background(colorPanel).
		Render(content)
}

type overlayContent struct {
	x       int
	y       int
	content string
}

func (m Model) renderOverlay(height, contentWidth int) overlayContent {
	paneX, paneWidth := m.overlayPaneBounds(contentWidth)
	switch m.mode {
	case modeSearch:
		return m.renderModalOverlay(paneX, paneWidth, height, m.renderSearchModal(paneWidth))
	case modePlaybackConfig:
		return m.renderModalOverlay(paneX, paneWidth, height, m.renderPlaybackConfigModal(paneWidth))
	default:
		return overlayContent{}
	}
}

func (m Model) overlayPaneBounds(contentWidth int) (int, int) {
	if m.compactLayout {
		paneWidth := max(1, min(contentWidth, 88))
		return max(1, (contentWidth-paneWidth)/2+1), paneWidth
	}
	if contentWidth < compactAtWidth {
		return 1, max(1, contentWidth)
	}
	paneWidth := contentWidth - sidebarWidth - paneGapWidth
	if m.showQueuePane(contentWidth) {
		paneWidth -= queuePaneWidth + paneGapWidth
	}
	return sidebarWidth + paneGapWidth + 1, max(1, paneWidth)
}

func (m Model) renderModalOverlay(paneX, paneWidth, height int, modal string) overlayContent {
	modalWidth := lipgloss.Width(modal)
	modalHeight := lipgloss.Height(modal)
	if modalWidth <= 0 || modalHeight <= 0 {
		return overlayContent{}
	}
	x := paneX + max(0, (paneWidth-modalWidth)/2)
	y := max(1, (height-modalHeight)/2+1)
	return overlayContent{x: x, y: y, content: modal}
}

func overlayFrame(base string, x, y int, content string) string {
	if x <= 0 || y <= 0 || content == "" {
		return base
	}
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(content, "\n")
	start := y - 1
	leftWidth := x - 1
	for i, overlayLine := range overlayLines {
		index := start + i
		if index < 0 || index >= len(baseLines) {
			continue
		}
		line := baseLines[index]
		overlayWidth := lipgloss.Width(overlayLine)
		prefix := ansi.Truncate(line, leftWidth, "")
		suffix := styledVisibleSuffix(line, leftWidth+overlayWidth)
		baseLines[index] = prefix + overlayLine + suffix
	}
	return strings.Join(baseLines, "\n")
}

func styledVisibleSuffix(line string, start int) string {
	if start <= 0 {
		return line
	}
	var styles strings.Builder
	var suffix strings.Builder
	width := 0
	for i := 0; i < len(line); {
		if line[i] == '\x1b' {
			seq, next, sgr := scanANSISequence(line, i)
			if next <= i {
				i++
				continue
			}
			if width < start && sgr {
				styles.WriteString(seq)
			}
			if width >= start {
				suffix.WriteString(seq)
			}
			i = next
			continue
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		if r == utf8.RuneError && size == 0 {
			break
		}
		value := line[i : i+size]
		nextWidth := width + lipgloss.Width(value)
		if nextWidth > start {
			suffix.WriteString(value)
		}
		width = nextWidth
		i += size
	}
	if suffix.Len() == 0 {
		return ""
	}
	return styles.String() + suffix.String()
}

func scanANSISequence(value string, start int) (string, int, bool) {
	if start+1 >= len(value) || value[start] != '\x1b' {
		return "", start, false
	}
	if value[start+1] == '[' {
		for i := start + 2; i < len(value); i++ {
			if value[i] >= 0x40 && value[i] <= 0x7e {
				return value[start : i+1], i + 1, value[i] == 'm'
			}
		}
		return value[start:], len(value), false
	}
	if strings.ContainsRune("]_P^", rune(value[start+1])) {
		for i := start + 2; i < len(value); i++ {
			if value[i] == '\a' {
				return value[start : i+1], i + 1, false
			}
			if value[i] == '\x1b' && i+1 < len(value) && value[i+1] == '\\' {
				return value[start : i+2], i + 2, false
			}
		}
		return value[start:], len(value), false
	}
	return value[start : start+2], start + 2, false
}

func (m Model) renderSearchContent(width int) string {
	contentWidth := max(1, width-borderChrome)
	var lines []string
	query := m.searchQuery
	placeholder := query == ""
	if query == "" {
		query = "type to search..."
	}
	lines = append(lines, labelStyle.Render(m.config.SearchService+" / "+m.searchCategory))
	lines = append(lines, renderSearchField(query, contentWidth, placeholder, m.mode == modeSearch))

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

func renderSearchField(query string, width int, placeholder bool, cursorVisible bool) string {
	innerWidth := max(8, width-8)
	display := displayText(query, innerWidth)
	if placeholder {
		display = lipgloss.NewStyle().Foreground(colorMuted).Render(display)
	} else {
		display = lipgloss.NewStyle().Foreground(colorInk).Bold(true).Render(display)
	}
	if cursorVisible {
		cursor := lipgloss.NewStyle().
			Foreground(colorSelected).
			Background(colorPanel).
			Bold(true).
			Render("█")
		display += cursor
	}
	field := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSelected).
		Background(colorPanel).
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
	keys := layoutHint + "  left/right 5s  arrows/jk move  enter play  o options  / search"
	if m.compactLayout {
		keys = layoutHint + "  left/right 5s  o options  / search"
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
		segments = append(segments, footerSpace(4), footerHintStyle().Render(keys))
	}
	if theme != "" && width-lipgloss.Width(status)-lipgloss.Width(keys)-4 > themeWidth {
		segments = append(segments, footerSpace(2), theme, footerSpace(1), themeHint)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, segments...)
}

func (m Model) renderFooter(width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Background(colorBase).
		Render(m.renderFooterContent(width))
}

func (m Model) renderFooterPane(width int) string {
	footer := m.renderFooterContent(width)
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, footer, lipgloss.WithWhitespaceBackground(colorBase))
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
		Background(colorBase).
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
	case "left/right 5s  arrows/jk move  enter play  o options  / search":
		choices = append(choices,
			"left/right 5s  enter play  o options  / search",
			"left/right 5s  o options  / search",
		)
	case "left/right 5s  o options  / search":
		choices = append(choices,
			"left/right 5s  o options",
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
		Background(colorBase).
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

func footerSpace(width int) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Background(colorBase).Render(strings.Repeat(" ", width))
}

func appSpace(width, height int) string {
	return lipgloss.NewStyle().
		Width(max(1, width)).
		Height(max(1, height)).
		Background(colorBase).
		Render("")
}

func padColumn(value string, width, height int) string {
	return lipgloss.NewStyle().
		Width(max(1, width)).
		Height(max(1, height)).
		Background(colorBase).
		Render(value)
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
	return lipgloss.NewStyle().Foreground(colorMuted).Background(colorBase)
}

func footerMessageStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorWarn).Background(colorBase)
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
	return "unmuted"
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
	return ansi.Truncate(value, width, "…")
}

func truncatePane(value string, width int) string {
	value = strings.TrimSpace(value)
	if width <= 0 || lipgloss.Width(value) <= width {
		return value
	}
	return ansi.Truncate(value, width, hintStyle.Render("…"))
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
