package desktop

import "sync"

// Texts are the words shown by the tray menu and native dialogs. They follow
// the Windows display language; the web UI has its own language switch.
type Texts struct {
	Open, OpenTip string
	Data, DataTip string
	Version       string // followed by the version number
	Quit, QuitTip string
	PickFolder    string
}

var english = Texts{
	Open:       "Open VRChat Asset Manager",
	OpenTip:    "Show the app in your browser",
	Data:       "Open data folder",
	DataTip:    "Your library, previews and backups",
	Version:    "Version ",
	Quit:       "Quit",
	QuitTip:    "Stop the app",
	PickFolder: "Select VRChat Asset Folder",
}

var japanese = Texts{
	Open:       "VRChat Asset Manager を開く",
	OpenTip:    "ブラウザでアプリを表示します",
	Data:       "データフォルダを開く",
	DataTip:    "ライブラリ、画像、バックアップ",
	Version:    "バージョン ",
	Quit:       "終了",
	QuitTip:    "アプリを終了します",
	PickFolder: "VRChat アセットフォルダを選択",
}

var (
	textOnce sync.Once
	text     Texts
)

// Text returns the texts for the user's display language (read once).
func Text() Texts {
	textOnce.Do(func() {
		text = english
		if japaneseUI() {
			text = japanese
		}
	})
	return text
}
