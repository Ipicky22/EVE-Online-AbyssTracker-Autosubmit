//go:build windows

package main

import (
	"html"
	"log"
	"os"
	"os/exec"
	"syscall"
)

// Windows notification through PowerShell, to avoid any library dependency.
// The content goes through an environment variable: no text is interpolated into the script.
const toastScript = `
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($env:AUTOSUBMIT_TOAST)
$app = '{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe'
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($app).Show([Windows.UI.Notifications.ToastNotification]::new($xml))
`

// notify shows a notification; clicking it opens link when one is given.
func notify(msg, link string) {
	toast := `<toast>`
	if link != "" {
		toast = `<toast activationType="protocol" launch="` + html.EscapeString(link) + `">`
	}
	toast += `<visual><binding template="ToastGeneric"><text>AbyssTracker Autosubmit</text><text>` + html.EscapeString(msg) + `</text></binding></visual></toast>`

	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", toastScript)
	cmd.Env = append(os.Environ(), "AUTOSUBMIT_TOAST="+toast)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	go func() {
		if err := cmd.Run(); err != nil {
			log.Printf("notification: %v", err)
		}
	}()
}
