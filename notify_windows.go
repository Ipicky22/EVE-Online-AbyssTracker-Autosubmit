//go:build windows

package main

import (
	"html"
	"log"
	"os"
	"os/exec"
	"syscall"
)

// Notification Windows via PowerShell, pour ne dépendre d'aucune bibliothèque.
// Le contenu passe par une variable d'environnement : aucun texte n'est interpolé dans le script.
const toastScript = `
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($env:ABYSSUP_TOAST)
$app = '{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe'
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($app).Show([Windows.UI.Notifications.ToastNotification]::new($xml))
`

// notify affiche une notification ; un clic ouvre link s'il est fourni.
func notify(msg, link string) {
	toast := `<toast>`
	if link != "" {
		toast = `<toast activationType="protocol" launch="` + html.EscapeString(link) + `">`
	}
	toast += `<visual><binding template="ToastGeneric"><text>abyssup</text><text>` + html.EscapeString(msg) + `</text></binding></visual></toast>`

	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", toastScript)
	cmd.Env = append(os.Environ(), "ABYSSUP_TOAST="+toast)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	go func() {
		if err := cmd.Run(); err != nil {
			log.Printf("notification: %v", err)
		}
	}()
}
