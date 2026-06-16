package tui

import "testing"

func TestOuterCanvasColorIsStableAcrossThemes(t *testing.T) {
	startTheme := activeThemeName
	t.Cleanup(func() {
		applyTheme(startTheme)
	})

	for _, theme := range visualThemes {
		t.Run(theme.Name, func(t *testing.T) {
			applyTheme(theme.Name)

			if colorBase != terminalCanvasColor {
				t.Fatalf("colorBase = %v, want fixed canvas %v", colorBase, terminalCanvasColor)
			}
			if got := baseStyle.GetBackground(); got != terminalCanvasColor {
				t.Fatalf("baseStyle background = %v, want fixed canvas %v", got, terminalCanvasColor)
			}
			if got := footerHintStyle().GetBackground(); got != terminalCanvasColor {
				t.Fatalf("footer hint background = %v, want fixed canvas %v", got, terminalCanvasColor)
			}
			if got := footerMessageStyle().GetBackground(); got != terminalCanvasColor {
				t.Fatalf("footer message background = %v, want fixed canvas %v", got, terminalCanvasColor)
			}
		})
	}
}

func TestPaneBackgroundsStayThemeDriven(t *testing.T) {
	startTheme := activeThemeName
	t.Cleanup(func() {
		applyTheme(startTheme)
	})

	for _, theme := range visualThemes {
		t.Run(theme.Name, func(t *testing.T) {
			applyTheme(theme.Name)

			if colorPanel != theme.Panel {
				t.Fatalf("colorPanel = %v, want theme panel %v", colorPanel, theme.Panel)
			}
			if colorPanel == colorBase {
				t.Fatalf("panel color should stay separate from fixed canvas color %v", colorBase)
			}
			if got := panelStyle.GetBackground(); got != theme.Panel {
				t.Fatalf("panelStyle background = %v, want theme panel %v", got, theme.Panel)
			}
			if got := paneBlock(1, 1).GetBackground(); got != theme.Panel {
				t.Fatalf("pane block background = %v, want theme panel %v", got, theme.Panel)
			}
			if got := hintStyle.GetBackground(); got != theme.Panel {
				t.Fatalf("hintStyle background = %v, want theme panel %v", got, theme.Panel)
			}
		})
	}
}
