package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shlomiuziel/sonosh/internal/appconfig"
	"github.com/shlomiuziel/sonosh/internal/sonos"
	"github.com/shlomiuziel/sonosh/internal/tui"
	"github.com/spf13/cobra"
)

type rootFlags struct {
	IP             string
	Name           string
	Timeout        time.Duration
	Format         string
	JSON           bool // Deprecated: use --format json
	Debug          bool
	DebugLog       bool
	SearchService  string
	SearchCategory string
	SearchLimit    int
	MacHelperPath  string
}

func Execute() error {
	rootCmd, _, err := newRootCmd()
	if err != nil {
		return err
	}
	ctx := context.Background()
	rootCmd.SetContext(ctx)

	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}

var (
	newSonosClient = sonos.NewClient
	sonosDiscover  = sonos.Discover
	runTUIApp      = runTUI
)

var loadAppConfig = func() (appconfig.Config, error) {
	s, err := appconfig.NewDefaultStore()
	if err != nil {
		return appconfig.Config{}, err
	}
	return s.Load()
}

func newRootCmd() (*cobra.Command, *rootFlags, error) {
	flags := &rootFlags{}

	cfg, err := loadAppConfig()
	if err != nil {
		return nil, nil, err
	}
	cfg = cfg.Normalize()
	timeout := sonos.DefaultTimeout
	if cfg.DefaultTimeout != "" {
		if d, err := time.ParseDuration(cfg.DefaultTimeout); err == nil && d > 0 {
			timeout = d
		}
	}

	rootCmd := &cobra.Command{
		Use:     "sonosh",
		Short:   "Control Sonos speakers from the command line",
		Long:    "Control Sonos speakers over your local network (UPnP/SOAP): discover devices, show status, control playback, manage groups/queue, and play Spotify (plus Sonos-side SMAPI search).",
		Example: "  sonosh discover\n  sonosh status --name \"Kitchen\"\n  sonosh smapi search --service \"Spotify\" --category tracks \"miles davis\"\n  sonosh open --name \"Kitchen\" spotify:track:6NmXV4o6bmp704aPGyTVVG\n  sonosh volume set --name \"Kitchen\" 25",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUIApp(cmd.Context(), flags)
		},
		SilenceUsage: true,
		Version:      Version,
	}
	rootCmd.SetVersionTemplate("sonosh {{.Version}}\n")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if flags.Debug {
			enableDebugLogging()
		}
		if flags.DebugLog {
			if err := enableDebugFileLogging(); err != nil {
				return fmt.Errorf("debug log setup failed: %w", err)
			}
		}

		format := strings.TrimSpace(flags.Format)
		if format == "" {
			format = formatPlain
		}
		format = strings.ToLower(format)
		if flags.JSON && format == formatPlain {
			format = formatJSON
		}
		norm, err := normalizeFormat(format)
		if err != nil {
			return err
		}
		flags.Format = norm
		return nil
	}

	rootCmd.PersistentFlags().StringVar(&flags.IP, "ip", "", "Target speaker IP address")
	rootCmd.PersistentFlags().StringVar(&flags.Name, "name", cfg.DefaultRoom, "Target speaker name")
	rootCmd.PersistentFlags().DurationVar(&flags.Timeout, "timeout", timeout, "Timeout for discovery and network calls")
	rootCmd.PersistentFlags().StringVar(&flags.Format, "format", cfg.Format, "Output format: plain|json|tsv")
	rootCmd.PersistentFlags().BoolVar(&flags.JSON, "json", false, "Deprecated: use --format json")
	_ = rootCmd.PersistentFlags().MarkDeprecated("json", "use --format json")
	rootCmd.PersistentFlags().BoolVar(&flags.Debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&flags.DebugLog, "debug-log", false, "Append debug logs to ~/Library/Application Support/sonosh/debug.log")
	rootCmd.PersistentFlags().StringVar(&flags.SearchService, "service", "Spotify", "SMAPI service for TUI search")
	rootCmd.PersistentFlags().StringVar(&flags.SearchCategory, "category", "tracks", "SMAPI category for TUI search")
	rootCmd.PersistentFlags().IntVar(&flags.SearchLimit, "limit", 10, "SMAPI search result limit for the TUI")
	rootCmd.PersistentFlags().StringVar(&flags.MacHelperPath, "mac-helper-path", "", "path to sonosh-macos-helper (macOS only; defaults to SONOSH_MAC_HELPER or executable sibling)")

	if err := rootCmd.RegisterFlagCompletionFunc("name", nameFlagCompletion(flags)); err != nil {
		return nil, nil, err
	}

	rootCmd.AddCommand(newDiscoverCmd(flags))
	rootCmd.AddCommand(newConfigCmd(flags))
	rootCmd.AddCommand(newStatusCmd(flags))
	rootCmd.AddCommand(newPlayCmd(flags))
	rootCmd.AddCommand(newPauseCmd(flags))
	rootCmd.AddCommand(newStopCmd(flags))
	rootCmd.AddCommand(newNextCmd(flags))
	rootCmd.AddCommand(newPrevCmd(flags))
	rootCmd.AddCommand(newOpenCmd(flags))
	rootCmd.AddCommand(newEnqueueCmd(flags))
	rootCmd.AddCommand(newPlayURLCmd(flags))
	rootCmd.AddCommand(newSearchCmd(flags))
	rootCmd.AddCommand(newAuthCmd(flags))
	rootCmd.AddCommand(newSMAPICmd(flags))
	rootCmd.AddCommand(newGroupCmd(flags))
	rootCmd.AddCommand(newSceneCmd(flags))
	rootCmd.AddCommand(newFavoritesCmd(flags))
	rootCmd.AddCommand(newPlayURICmd(flags))
	rootCmd.AddCommand(newLineInCmd(flags))
	rootCmd.AddCommand(newTVCmd(flags))
	rootCmd.AddCommand(newQueueCmd(flags))
	rootCmd.AddCommand(newVolumeCmd(flags))
	rootCmd.AddCommand(newMuteCmd(flags))
	rootCmd.AddCommand(newWatchCmd(flags))
	rootCmd.AddCommand(newStreamDaemonCmd(flags))

	return rootCmd, flags, nil
}

func nameFlagCompletion(flags *rootFlags) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		timeout := completionTimeoutForFlags(flags)
		now := time.Now()

		names, ok := cachedNameCompletions(now)
		if !ok {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			devs, err := sonosDiscover(ctx, sonos.DiscoverOptions{Timeout: timeout})
			if err == nil && len(devs) > 0 {
				names = extractDeviceNames(devs)
				_ = storeNameCompletions(now, names)
			} else {
				// Best-effort fallback: if discovery fails, return stale cache rather than nothing.
				if cache, ok := readNameCompletionCacheFile(); ok {
					names = cache.Names
				}
			}
		}

		needle := strings.ToLower(strings.TrimSpace(toComplete))
		seen := map[string]struct{}{}
		filtered := make([]string, 0, len(names))
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if needle != "" && !strings.HasPrefix(strings.ToLower(name), needle) {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			filtered = append(filtered, name)
		}
		sort.Strings(filtered)
		out := make([]string, 0, len(filtered))
		for _, name := range filtered {
			out = append(out, escapeBashCompletionValue(name))
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}

func extractDeviceNames(devs []sonos.Device) []string {
	out := make([]string, 0, len(devs))
	for _, d := range devs {
		name := strings.TrimSpace(d.Name)
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	return out
}

func escapeBashCompletionValue(value string) string {
	// Cobra's bash completion uses `compgen -W`, which is whitespace-delimited.
	// Escaping spaces keeps multi-word speaker names intact (e.g. "Living Room").
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, " ", `\ `)
	value = strings.ReplaceAll(value, "\t", `\	`)
	return value
}

func completionTimeoutForFlags(flags *rootFlags) time.Duration {
	const maxCompletionTimeout = 1 * time.Second

	if flags == nil || flags.Timeout <= 0 {
		return maxCompletionTimeout
	}
	if flags.Timeout < maxCompletionTimeout {
		return flags.Timeout
	}
	return maxCompletionTimeout
}

func validateTarget(flags *rootFlags) error {
	if flags.IP == "" && flags.Name == "" {
		return errors.New("provide --ip or --name (or run `sonosh discover`)")
	}
	return nil
}

func resolveTargetCoordinatorIP(ctx context.Context, flags *rootFlags) (string, error) {
	if err := validateTarget(flags); err != nil {
		return "", err
	}

	// If IP is provided, attempt to resolve to coordinator, but fall back.
	if flags.IP != "" {
		c := newSonosClient(flags.IP, flags.Timeout)
		top, err := c.GetTopology(ctx)
		if err != nil {
			return flags.IP, nil //nolint:nilerr // best-effort coordinator lookup; explicit IP remains a valid fallback.
		}
		if coordIP, ok := top.CoordinatorIPFor(flags.IP); ok {
			return coordIP, nil
		}
		return flags.IP, nil
	}

	// Name-based selection: discover a speaker, then use topology.
	devs, err := sonosDiscover(ctx, sonos.DiscoverOptions{Timeout: flags.Timeout})
	if err != nil {
		return "", err
	}
	if len(devs) == 0 {
		return "", errors.New("no speakers found")
	}

	c := newSonosClient(devs[0].IP, flags.Timeout)
	top, err := c.GetTopology(ctx)
	if err != nil {
		return "", err
	}
	coordIP, ok := top.CoordinatorIPForName(flags.Name)
	if !ok {
		return "", errors.New("speaker name not found in topology: " + flags.Name)
	}
	return coordIP, nil
}

func runTUI(ctx context.Context, flags *rootFlags) error {
	lastRoomConfigPath, err := tui.DefaultLastRoomConfigPath()
	if err != nil {
		lastRoomConfigPath = ""
	}
	themeConfigPath, err := tui.DefaultThemeConfigPath()
	if err != nil {
		themeConfigPath = ""
	}
	layoutConfigPath, err := tui.DefaultLayoutConfigPath()
	if err != nil {
		layoutConfigPath = ""
	}
	helperHUDConfigPath, err := tui.DefaultHelperHUDConfigPath()
	if err != nil {
		helperHUDConfigPath = ""
	}
	carouselPath, err := tui.DefaultPlaylistCarouselConfigPath()
	if err != nil {
		carouselPath = ""
	}

	storedTheme, err := tui.LoadThemeName(themeConfigPath)
	if err != nil {
		storedTheme = ""
	}
	storedHelperHUD, err := tui.LoadHelperHUDConfig(helperHUDConfigPath)
	if err != nil {
		storedHelperHUD = tui.DefaultHelperHUDConfig()
	}

	cfg := tui.Config{
		Timeout:             normalizeTUITimeout(flags.Timeout),
		SearchService:       flags.SearchService,
		SearchCategory:      flags.SearchCategory,
		SearchLimit:         flags.SearchLimit,
		MacHelperPath:       flags.MacHelperPath,
		HelperHUDEnabled:    storedHelperHUD.Enabled,
		HelperHUDPosition:   storedHelperHUD.Position,
		HelperHUDConfigPath: helperHUDConfigPath,
		Theme:               storedTheme,
		ThemeConfigPath:     themeConfigPath,
		LayoutConfigPath:    layoutConfigPath,
		CarouselPath:        carouselPath,
		LastRoomConfigPath:  lastRoomConfigPath,
	}

	model := tui.NewModel(tui.NewSonosBackend(cfg.Timeout), cfg)
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))
	finalModel, err := program.Run()
	if m, ok := finalModel.(tui.Model); ok {
		_ = m.Close()
	}
	return err
}

func normalizeTUITimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return sonos.DefaultTimeout
	}
	return timeout
}

func enableDebugFileLogging() error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "sonosh", "debug.log")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // path is created under the app's config directory.
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug})))
	slog.Debug("sonosh: debug logging enabled", "path", path)
	return nil
}

func coordinatorClient(ctx context.Context, flags *rootFlags) (*sonos.Client, error) {
	ip, err := resolveTargetCoordinatorIP(ctx, flags)
	if err != nil {
		return nil, err
	}
	return newSonosClient(ip, flags.Timeout), nil
}
