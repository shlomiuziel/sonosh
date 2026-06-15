package sonos

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCrossfadeModeCalls(t *testing.T) {
	t.Parallel()

	var sawGet, sawSetOn, sawSetOff bool
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		action := r.Header.Get("SOAPACTION")
		body, _ := io.ReadAll(r.Body)
		switch {
		case strings.Contains(action, "#GetCrossfadeMode"):
			sawGet = true
			return httpResponse(200, `<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:GetCrossfadeModeResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><CrossfadeMode>1</CrossfadeMode></u:GetCrossfadeModeResponse></s:Body></s:Envelope>`), nil
		case strings.Contains(action, "#SetCrossfadeMode") && strings.Contains(string(body), "<CrossfadeMode>1</CrossfadeMode>"):
			sawSetOn = true
			return httpResponse(200, `<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:SetCrossfadeModeResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:SetCrossfadeModeResponse></s:Body></s:Envelope>`), nil
		case strings.Contains(action, "#SetCrossfadeMode") && strings.Contains(string(body), "<CrossfadeMode>0</CrossfadeMode>"):
			sawSetOff = true
			return httpResponse(200, `<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:SetCrossfadeModeResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:SetCrossfadeModeResponse></s:Body></s:Envelope>`), nil
		default:
			t.Fatalf("unexpected SOAP request action=%q body=%s", action, string(body))
			return nil, nil
		}
	})

	c := &Client{
		IP: "192.0.2.1",
		HTTP: &http.Client{
			Timeout:   time.Second,
			Transport: rt,
		},
	}

	enabled, err := c.GetCrossfadeMode(context.Background())
	if err != nil {
		t.Fatalf("GetCrossfadeMode: %v", err)
	}
	if !enabled {
		t.Fatal("expected crossfade enabled")
	}
	if err := c.SetCrossfadeMode(context.Background(), true); err != nil {
		t.Fatalf("SetCrossfadeMode(true): %v", err)
	}
	if err := c.SetCrossfadeMode(context.Background(), false); err != nil {
		t.Fatalf("SetCrossfadeMode(false): %v", err)
	}
	if !sawGet || !sawSetOn || !sawSetOff {
		t.Fatalf("expected get/set calls, get=%v setOn=%v setOff=%v", sawGet, sawSetOn, sawSetOff)
	}
}
