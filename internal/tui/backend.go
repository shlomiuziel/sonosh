package tui

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
	State    string
	Title    string
	Artist   string
	Album    string
	AlbumArt string
	Position string
	Duration string
	Volume   int
	Muted    bool
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
	Search(context.Context, Room, string, string, string, int) ([]SearchResult, error)
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

	status := Status{
		State:    strings.TrimSpace(transport.State),
		Position: strings.TrimSpace(position.RelTime),
		Duration: strings.TrimSpace(position.TrackDuration),
		Volume:   volume,
		Muted:    muted,
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
