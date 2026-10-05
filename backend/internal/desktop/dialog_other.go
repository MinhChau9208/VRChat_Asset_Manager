//go:build !windows

package desktop

import (
	"log"
	"os"
	"strings"
)

// japaneseUI reports whether the locale environment asks for Japanese.
func japaneseUI() bool {
	return strings.HasPrefix(strings.ToLower(os.Getenv("LANG")), "ja")
}

// ShowError logs the message; native dialogs exist only on Windows.
func ShowError(title, text string) {
	log.Printf("%s: %s", title, text)
}

// PickFolder is not supported outside Windows; it returns "".
func PickFolder(title string) (string, error) {
	return "", nil
}
