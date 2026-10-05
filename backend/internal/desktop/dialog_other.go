//go:build !windows

package desktop

import "log"

// ShowError logs the message; native dialogs exist only on Windows.
func ShowError(title, text string) {
	log.Printf("%s: %s", title, text)
}

// PickFolder is not supported outside Windows; it returns "".
func PickFolder(title string) (string, error) {
	return "", nil
}
