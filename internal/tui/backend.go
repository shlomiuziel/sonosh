package tui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shlomiuziel/sonosh/internal/sonos"
)

type Room struct {
	Name          string
	IP            string
	CoordinatorIP string
	GroupMembers  []string
}

type Status struct {
	State            string
	Title            string
	Artist           string
	Album            string
	AlbumArt         string
	Position         string
	Duration         string
	Volume           int
	Muted            bool
	CrossfadeEnabled bool
	CrossfadeKnown   bool
	ShuffleEnabled   bool
	ShuffleKnown     bool
	RepeatMode       string
	RepeatKnown      bool
	QueuePosition    int
}

type QueueItem struct {
	Position int
	Title    string
	Artist   string
	Album    string
	URI      string
}

type QueuePage struct {
	Items          []QueueItem
	NumberReturned int
	TotalMatches   int
	UpdateID       int
}

type SearchResult struct {
	Item sonos.SMAPIItem
}

func (r SearchResult) Title() string {
	if strings.TrimSpace(r.Item.Title) != "" {
		return strings.TrimSpace(r.Item.Title)
	}
	if strings.TrimSpace(r.Item.Summary) != "" {
		return strings.TrimSpace(r.Item.Summary)
	}
	return strings.TrimSpace(r.Item.ID)
}

type Backend interface {
	Discover(context.Context) ([]Room, error)
	Status(context.Context, Room) (Status, error)
	Transport(context.Context, Room, string) error
	SetVolume(context.Context, Room, int) error
	ToggleMute(context.Context, Room) error
	ToggleCrossfade(context.Context, Room) (bool, error)
	ToggleShuffle(context.Context, Room) (bool, error)
	ToggleRepeat(context.Context, Room) (string, error)
	Scrub(context.Context, Room, string, string, int) (string, error)
	Queue(context.Context, Room, int, int) (QueuePage, error)
	PlayQueuePosition(context.Context, Room, int) error
	RemoveQueuePosition(context.Context, Room, int) error
	ClearQueue(context.Context, Room) error
	MoveQueuePosition(context.Context, Room, int, int) error
	Search(context.Context, Room, string, string, string, int) ([]SearchResult, error)
	BrowsePlaylist(context.Context, Room, string, SearchResult, int) ([]SearchResult, error)
	PlaySearchResult(context.Context, Room, string, SearchResult) error
}

type SonosBackend struct {
	Timeout time.Duration
}

func NewSonosBackend(timeout time.Duration) *SonosBackend {
	if timeout <= 0 {
		timeout = sonos.DefaultTimeout
	}
	return &SonosBackend{Timeout: timeout}
}

func (b *SonosBackend) Discover(ctx context.Context) ([]Room, error) {
	devices, err := sonos.Discover(ctx, sonos.DiscoverOptions{Timeout: b.Timeout})
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, errors.New("no Sonos speakers found")
	}

	top, err := sonos.NewClient(devices[0].IP, b.Timeout).GetTopology(ctx)
	if err != nil {
		return roomsFromDevices(devices), nil
	}

	var rooms []Room
	for _, g := range top.Groups {
		members := visibleMemberNames(g.Members)
		for _, m := range g.Members {
			if !m.IsVisible {
				continue
			}
			coordIP := g.Coordinator.IP
			if coordIP == "" {
				coordIP = m.IP
			}
			rooms = append(rooms, Room{
				Name:          fallback(m.Name, m.IP),
				IP:            m.IP,
				CoordinatorIP: coordIP,
				GroupMembers:  members,
			})
		}
	}
	sort.Slice(rooms, func(i, j int) bool {
		return strings.ToLower(rooms[i].Name) < strings.ToLower(rooms[j].Name)
	})
	if len(rooms) == 0 {
		return roomsFromDevices(devices), nil
	}
	return rooms, nil
}

func (b *SonosBackend) Status(ctx context.Context, room Room) (Status, error) {
	roomClient := sonos.NewClient(room.IP, b.Timeout)
	transportClient := sonos.NewClient(room.transportIP(), b.Timeout)

	transport, err := transportClient.GetTransportInfo(ctx)
	if err != nil {
		return Status{}, err
	}
	position, err := transportClient.GetPositionInfo(ctx)
	if err != nil {
		return Status{}, err
	}
	volume, err := roomClient.GetVolume(ctx)
	if err != nil {
		return Status{}, err
	}
	muted, err := roomClient.GetMute(ctx)
	if err != nil {
		return Status{}, err
	}
	crossfade, crossfadeErr := transportClient.GetCrossfadeMode(ctx)
	playMode, playModeErr := transportClient.GetPlayMode(ctx)
	repeatMode := repeatModeFromPlayMode(playMode)
	queuePosition := 0
	if n, err := strconv.Atoi(strings.TrimSpace(position.Track)); err == nil && n > 0 {
		queuePosition = n
	}

	status := Status{
		State:            strings.TrimSpace(transport.State),
		Position:         strings.TrimSpace(position.RelTime),
		Duration:         strings.TrimSpace(position.TrackDuration),
		Volume:           volume,
		Muted:            muted,
		CrossfadeEnabled: crossfade,
		CrossfadeKnown:   crossfadeErr == nil,
		ShuffleEnabled:   sonosShuffleEnabled(playMode),
		ShuffleKnown:     playModeErr == nil,
		RepeatMode:       repeatMode,
		RepeatKnown:      playModeErr == nil,
		QueuePosition:    queuePosition,
	}
	if item, ok := sonos.ParseNowPlaying(position.TrackMeta); ok {
		status.Title = strings.TrimSpace(item.Title)
		status.Artist = strings.TrimSpace(item.Artist)
		status.Album = strings.TrimSpace(item.Album)
		status.AlbumArt = sonos.AlbumArtURL(room.transportIP(), item.AlbumArtURI)
	}
	if status.Title == "" {
		status.Title = strings.TrimSpace(position.TrackURI)
	}
	return status, nil
}

func (b *SonosBackend) Transport(ctx context.Context, room Room, action string) error {
	c := sonos.NewClient(room.transportIP(), b.Timeout)
	switch action {
	case "play":
		return c.Play(ctx)
	case "pause":
		return c.Pause(ctx)
	case "stop":
		return c.StopOrNoop(ctx)
	case "next":
		return c.Next(ctx)
	case "previous":
		return c.PreviousOrRestart(ctx)
	default:
		return fmt.Errorf("unknown transport action %q", action)
	}
}

func (b *SonosBackend) SetVolume(ctx context.Context, room Room, volume int) error {
	return sonos.NewClient(room.IP, b.Timeout).SetVolume(ctx, volume)
}

func (b *SonosBackend) ToggleMute(ctx context.Context, room Room) error {
	c := sonos.NewClient(room.IP, b.Timeout)
	muted, err := c.GetMute(ctx)
	if err != nil {
		return err
	}
	return c.SetMute(ctx, !muted)
}

func (b *SonosBackend) ToggleCrossfade(ctx context.Context, room Room) (bool, error) {
	c := sonos.NewClient(room.transportIP(), b.Timeout)
	enabled, err := c.GetCrossfadeMode(ctx)
	if err != nil {
		return false, err
	}
	next := !enabled
	if err := c.SetCrossfadeMode(ctx, next); err != nil {
		return false, err
	}
	return next, nil
}

func (b *SonosBackend) ToggleShuffle(ctx context.Context, room Room) (bool, error) {
	c := sonos.NewClient(room.transportIP(), b.Timeout)
	mode, err := c.GetPlayMode(ctx)
	if err != nil {
		return false, err
	}
	nextMode := setShufflePlayMode(mode, !sonosShuffleEnabled(mode))
	if err := c.SetPlayMode(ctx, nextMode); err != nil {
		return false, err
	}
	return sonosShuffleEnabled(nextMode), nil
}

func (b *SonosBackend) ToggleRepeat(ctx context.Context, room Room) (string, error) {
	c := sonos.NewClient(room.transportIP(), b.Timeout)
	mode, err := c.GetPlayMode(ctx)
	if err != nil {
		return "", err
	}
	nextRepeat := nextRepeatMode(repeatModeFromPlayMode(mode))
	nextMode := setRepeatPlayMode(mode, nextRepeat)
	if err := c.SetPlayMode(ctx, nextMode); err != nil {
		return "", err
	}
	return nextRepeat, nil
}

func (b *SonosBackend) Scrub(ctx context.Context, room Room, position, duration string, deltaSeconds int) (string, error) {
	target, err := scrubTarget(position, duration, deltaSeconds)
	if err != nil {
		return "", err
	}
	if err := sonos.NewClient(room.transportIP(), b.Timeout).SeekRelTime(ctx, target); err != nil {
		return "", err
	}
	return target, nil
}

func (b *SonosBackend) Queue(ctx context.Context, room Room, start, count int) (QueuePage, error) {
	page, err := sonos.NewClient(room.transportIP(), b.Timeout).ListQueue(ctx, start, count)
	if err != nil {
		return QueuePage{}, err
	}
	items := make([]QueueItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, QueueItem{
			Position: item.Position,
			Title:    strings.TrimSpace(item.Item.Title),
			Artist:   strings.TrimSpace(item.Item.Artist),
			Album:    strings.TrimSpace(item.Item.Album),
			URI:      strings.TrimSpace(item.Item.URI),
		})
	}
	return QueuePage{
		Items:          items,
		NumberReturned: page.NumberReturned,
		TotalMatches:   page.TotalMatches,
		UpdateID:       page.UpdateID,
	}, nil
}

func (b *SonosBackend) PlayQueuePosition(ctx context.Context, room Room, position int) error {
	return sonos.NewClient(room.transportIP(), b.Timeout).PlayQueuePosition(ctx, position)
}

func (b *SonosBackend) RemoveQueuePosition(ctx context.Context, room Room, position int) error {
	return sonos.NewClient(room.transportIP(), b.Timeout).RemoveQueuePosition(ctx, position)
}

func (b *SonosBackend) ClearQueue(ctx context.Context, room Room) error {
	return sonos.NewClient(room.transportIP(), b.Timeout).ClearQueue(ctx)
}

func (b *SonosBackend) MoveQueuePosition(ctx context.Context, room Room, fromPosition, toPosition int) error {
	return sonos.NewClient(room.transportIP(), b.Timeout).MoveQueuePosition(ctx, fromPosition, toPosition)
}

func sonosShuffleEnabled(mode string) bool {
	return strings.Contains(strings.ToUpper(strings.TrimSpace(mode)), "SHUFFLE")
}

func setShufflePlayMode(mode string, enabled bool) string {
	normalized := strings.ToUpper(strings.TrimSpace(mode))
	if normalized == "" {
		normalized = "NORMAL"
	}
	if enabled {
		switch repeatModeFromPlayMode(normalized) {
		case "all":
			return "SHUFFLE"
		case "once":
			return "SHUFFLE_REPEAT_ONE"
		default:
			return "SHUFFLE_NOREPEAT"
		}
	}
	if !sonosShuffleEnabled(normalized) {
		return normalized
	}
	switch repeatModeFromPlayMode(normalized) {
	case "all":
		return "REPEAT_ALL"
	case "once":
		return "REPEAT_ONE"
	default:
		return "NORMAL"
	}
}

func repeatModeFromPlayMode(mode string) string {
	normalized := strings.ToUpper(strings.TrimSpace(mode))
	switch {
	case strings.Contains(normalized, "REPEAT_ONE"):
		return "once"
	case strings.Contains(normalized, "REPEAT_ALL"), normalized == "SHUFFLE":
		return "all"
	default:
		return "off"
	}
}

func nextRepeatMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "all":
		return "once"
	case "once":
		return "off"
	default:
		return "all"
	}
}

func setRepeatPlayMode(mode, repeat string) string {
	normalized := strings.ToUpper(strings.TrimSpace(mode))
	if normalized == "" {
		normalized = "NORMAL"
	}
	shuffle := sonosShuffleEnabled(normalized)
	switch strings.ToLower(strings.TrimSpace(repeat)) {
	case "once":
		if shuffle {
			return "SHUFFLE_REPEAT_ONE"
		}
		return "REPEAT_ONE"
	case "all":
		if shuffle {
			return "SHUFFLE"
		}
		return "REPEAT_ALL"
	default:
		if shuffle {
			return "SHUFFLE_NOREPEAT"
		}
		return "NORMAL"
	}
}

func scrubTarget(position, duration string, deltaSeconds int) (string, error) {
	cur, err := parseTimecode(position)
	if err != nil {
		return "", err
	}
	total, err := parseTimecode(duration)
	if err != nil {
		total = 0
	}
	next := cur + time.Duration(deltaSeconds)*time.Second
	if next < 0 {
		next = 0
	}
	if total > 0 && next > total {
		next = total
	}
	return formatTimecode(next), nil
}

func parseTimecode(value string) (time.Duration, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid timecode %q", value)
	}
	var values [3]int
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid timecode %q", value)
		}
		values[i] = n
	}
	return time.Duration(values[0])*time.Hour + time.Duration(values[1])*time.Minute + time.Duration(values[2])*time.Second, nil
}

func formatTimecode(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int(d.Round(time.Second) / time.Second)
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}

func (b *SonosBackend) Search(ctx context.Context, room Room, serviceName, category, query string, limit int) ([]SearchResult, error) {
	speaker := sonos.NewClient(room.IP, b.Timeout)
	svc, err := b.findService(ctx, speaker, serviceName)
	if err != nil {
		return nil, err
	}
	store, err := sonos.NewDefaultSMAPITokenStore()
	if err != nil {
		return nil, err
	}
	smapi, err := sonos.NewSMAPIClient(ctx, speaker, svc, store)
	if err != nil {
		return nil, err
	}
	res, err := smapi.Search(ctx, category, query, 0, limit)
	if err != nil {
		return nil, err
	}
	items := append([]sonos.SMAPIItem{}, res.MediaMetadata...)
	items = append(items, res.MediaCollection...)
	out := make([]SearchResult, 0, len(items))
	for _, item := range items {
		out = append(out, SearchResult{Item: item})
	}
	return out, nil
}

func (b *SonosBackend) BrowsePlaylist(ctx context.Context, room Room, serviceName string, result SearchResult, limit int) ([]SearchResult, error) {
	if !strings.EqualFold(strings.TrimSpace(result.Item.ItemType), "playlist") {
		return nil, nil
	}

	speaker := sonos.NewClient(room.IP, b.Timeout)
	svc, err := b.findService(ctx, speaker, serviceName)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 6
	}
	store, err := sonos.NewDefaultSMAPITokenStore()
	if err != nil {
		return nil, err
	}
	smapi, err := sonos.NewSMAPIClient(ctx, speaker, svc, store)
	if err != nil {
		return nil, err
	}
	res, err := smapi.GetMetadata(ctx, result.Item.ID, 0, limit, true)
	if err != nil {
		return nil, err
	}
	items := append([]sonos.SMAPIItem{}, res.MediaMetadata...)
	if len(items) == 0 {
		items = append(items, res.MediaCollection...)
	}
	out := make([]SearchResult, 0, len(items))
	for _, item := range items {
		out = append(out, SearchResult{Item: item})
	}
	return out, nil
}

func (b *SonosBackend) PlaySearchResult(ctx context.Context, room Room, serviceName string, result SearchResult) error {
	speaker := sonos.NewClient(room.IP, b.Timeout)
	svc, err := b.findService(ctx, speaker, serviceName)
	if err != nil {
		return err
	}
	transport := sonos.NewClient(room.transportIP(), b.Timeout)

	ref := strings.TrimSpace(result.Item.ID)
	if _, ok := sonos.ParseSpotifyRef(ref); ok {
		_, err = transport.EnqueueSpotify(ctx, ref, sonos.EnqueueOptions{
			Position: 0,
			AsNext:   true,
			PlayNow:  true,
			Title:    result.Title(),
		})
		return err
	}

	_, err = transport.EnqueueSMAPIItem(ctx, svc, result.Item, sonos.EnqueueOptions{
		Position: 0,
		AsNext:   true,
		PlayNow:  true,
		Title:    result.Title(),
	})
	return err
}

func (b *SonosBackend) findService(ctx context.Context, speaker *sonos.Client, name string) (sonos.MusicServiceDescriptor, error) {
	services, err := speaker.ListAvailableServices(ctx)
	if err != nil {
		return sonos.MusicServiceDescriptor{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Spotify"
	}
	var matches []sonos.MusicServiceDescriptor
	for _, svc := range services {
		if strings.EqualFold(strings.TrimSpace(svc.Name), name) {
			return svc, nil
		}
		if strings.Contains(strings.ToLower(svc.Name), strings.ToLower(name)) {
			matches = append(matches, svc)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		names := make([]string, 0, len(matches))
		for _, svc := range matches {
			names = append(names, svc.Name)
		}
		sort.Strings(names)
		return sonos.MusicServiceDescriptor{}, fmt.Errorf("ambiguous service %q: %s", name, strings.Join(names, ", "))
	}
	return sonos.MusicServiceDescriptor{}, fmt.Errorf("service not found: %s", name)
}

func (r Room) transportIP() string {
	if strings.TrimSpace(r.CoordinatorIP) != "" {
		return r.CoordinatorIP
	}
	return r.IP
}

func roomsFromDevices(devices []sonos.Device) []Room {
	out := make([]Room, 0, len(devices))
	for _, d := range devices {
		out = append(out, Room{
			Name:          fallback(d.Name, d.IP),
			IP:            d.IP,
			CoordinatorIP: d.IP,
			GroupMembers:  []string{fallback(d.Name, d.IP)},
		})
	}
	return out
}

func visibleMemberNames(members []sonos.Member) []string {
	var names []string
	for _, m := range members {
		if m.IsVisible {
			names = append(names, fallback(m.Name, m.IP))
		}
	}
	sort.Strings(names)
	return names
}

func fallback(value, fallbackValue string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallbackValue)
}
