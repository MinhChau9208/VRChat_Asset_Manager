// Package desktop connects the release build to the user's desktop: the tray
// icon, message boxes, the native folder picker, and opening the browser or a
// folder. The dialogs are Windows-only; elsewhere they are no-ops.
package desktop

import (
	_ "embed"
	"os/exec"
	"runtime"

	"fyne.io/systray"
)

// Icon is the app icon (rendered by scripts/icon/render.ps1).
//
//go:embed icon.ico
var Icon []byte

// OpenBrowser opens url in the user's default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// OpenFolder shows dir in the file manager.
func OpenFolder(dir string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}

// TrayOptions describes the tray icon and what its menu does.
type TrayOptions struct {
	Tooltip    string
	Version    string
	OnOpen     func() // left click, and "Open" in the menu
	OnOpenData func()
}

// RunTray shows the tray icon and blocks until the user picks Quit or
// QuitTray is called. It must run on the main goroutine.
func RunTray(o TrayOptions) {
	systray.Run(func() {
		systray.SetIcon(Icon)
		systray.SetTooltip(o.Tooltip)
		systray.SetOnTapped(o.OnOpen)

		txt := Text()
		open := systray.AddMenuItem(txt.Open, txt.OpenTip)
		data := systray.AddMenuItem(txt.Data, txt.DataTip)
		systray.AddSeparator()
		version := systray.AddMenuItem(txt.Version+o.Version, "")
		version.Disable()
		quit := systray.AddMenuItem(txt.Quit, txt.QuitTip)

		go func() {
			for {
				select {
				case <-open.ClickedCh:
					o.OnOpen()
				case <-data.ClickedCh:
					o.OnOpenData()
				case <-quit.ClickedCh:
					systray.Quit()
					return
				}
			}
		}()
	}, nil)
}

// QuitTray removes the tray icon and makes RunTray return.
func QuitTray() {
	systray.Quit()
}
