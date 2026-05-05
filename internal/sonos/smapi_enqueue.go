package sonos

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
)

func (c *Client) EnqueueSMAPIItem(ctx context.Context, svc MusicServiceDescriptor, item SMAPIItem, opts EnqueueOptions) (int, error) {
	itemID := strings.TrimSpace(item.ID)
	if itemID == "" {
		return 0, errors.New("smapi item id is required")
	}
	serviceID := strings.TrimSpace(svc.ID)
	if serviceID == "" {
		return 0, errors.New("smapi service id is required")
	}

	desiredPos := opts.Position
	if desiredPos < 0 {
		desiredPos = 0
	}

	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = strings.TrimSpace(item.Title)
	}
	if title == "" {
		title = itemID
	}

	enqueuedURI := fmt.Sprintf("soco://%s?sid=%s&sn=0", escapeSMAPIItemID(itemID), url.QueryEscape(serviceID))
	meta := buildSMAPIDIDL(itemID, title, smapiItemClass(item.ItemType), smapiServiceDesc(svc))
	slog.Debug("smapi: AddURIToQueue", "service", svc.Name, "uri", enqueuedURI, "metadata", meta)

	first, err := c.AddURIToQueue(ctx, enqueuedURI, meta, desiredPos, opts.AsNext)
	if err != nil {
		return 0, err
	}
	if opts.PlayNow && first > 0 {
		if err := c.playFromQueueTrack(ctx, first); err != nil {
			return first, err
		}
	} else if opts.PlayNow {
		_ = c.Play(ctx)
	}
	return first, nil
}

func escapeSMAPIItemID(itemID string) string {
	return strings.ReplaceAll(url.QueryEscape(itemID), "+", "%20")
}

func smapiItemClass(itemType string) string {
	switch strings.ToLower(strings.TrimSpace(itemType)) {
	case "album":
		return "object.container.album.musicAlbum"
	case "artist", "container", "playlist":
		return "object.container.playlistContainer"
	case "program", "show", "station":
		return "object.item.audioItem.audioBroadcast"
	default:
		return "object.item.audioItem.musicTrack"
	}
}

func smapiServiceDesc(svc MusicServiceDescriptor) string {
	serviceType := strings.TrimSpace(svc.ServiceType)
	if serviceType == "" {
		if n, err := strconv.Atoi(strings.TrimSpace(svc.ID)); err == nil {
			serviceType = strconv.Itoa(n*256 + 7)
		}
	}
	if serviceType == "" {
		serviceType = strings.TrimSpace(svc.ID)
	}
	if svc.Auth == MusicServiceAuthDeviceLink || svc.Auth == MusicServiceAuthAppLink {
		return fmt.Sprintf("SA_RINCON%s_X_#Svc%s-0-Token", serviceType, serviceType)
	}
	return fmt.Sprintf("SA_RINCON%s_", serviceType)
}

func buildSMAPIDIDL(itemID, title, itemClass, desc string) string {
	return fmt.Sprintf(
		`<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns:r="urn:schemas-rinconnetworks-com:metadata-1-0/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/"><item id="%s" parentID="-1" restricted="true"><dc:title>%s</dc:title><upnp:class>%s</upnp:class><desc id="cdudn" nameSpace="urn:schemas-rinconnetworks-com:metadata-1-0/">%s</desc></item></DIDL-Lite>`,
		xmlEscapeText(itemID),
		xmlEscapeText(title),
		xmlEscapeText(itemClass),
		xmlEscapeText(desc),
	)
}
