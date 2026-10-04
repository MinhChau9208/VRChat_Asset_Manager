package scanner

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	// Download-duplicate suffixes such as "(1)" or "__1_".
	dupSuffixRe = regexp.MustCompile(`(?:\s*\(\d+\)|_{1,2}\d+_?)$`)
	// "v2", "_v1.2", "ver1.02", "Ver_1.3", "LuminousWandv1.4"
	verPrefixedRe = regexp.MustCompile(`(?i)[\s_\-.]*(?:ver(?:sion)?|v)[\s_\-.]*(\d+(?:[._]\d+)*)$`)
	// "_1.2.0", "Kipfel1.0" (needs at least one dot so "Piano 2" is not a version)
	verDottedRe = regexp.MustCompile(`[\s_\-]*(\d+(?:\.\d+)+)$`)
	// BOOTH item ids are currently 7–8 digits; 6-digit numbers are usually dates.
	boothIDInNameRe = regexp.MustCompile(`(?:^|\D)(\d{7,8})(?:\D|$)`)
	boothLinkRe     = regexp.MustCompile(`https?://(?:[\w-]+\.)?booth\.pm/(?:[a-z]{2}/)?items/(\d+)`)
)

var archiveExts = map[string]bool{".zip": true, ".7z": true, ".rar": true}
var mediaExts = map[string]bool{".mp3": true, ".wav": true, ".ogg": true, ".flac": true, ".m4a": true, ".mp4": true}
var imageExts = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true}

// Files that mark a folder as an asset (rather than a category folder).
var assetMarkerExts = map[string]bool{".unitypackage": true, ".fbx": true, ".blend": true, ".psd": true, ".prefab": true}

// stripExt removes the extension of archive / package file names.
func stripExt(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if archiveExts[ext] || ext == ".unitypackage" || mediaExts[ext] || imageExts[ext] {
		return name[:len(name)-len(ext)]
	}
	return name
}

// splitVersion separates a trailing version from a name:
// "Kipfel_1.2.0" -> ("Kipfel", "1.2.0"), "sipisipi_Cosset_ver1.02" -> ("sipisipi_Cosset", "1.02").
func splitVersion(name string) (base, version string) {
	base = strings.TrimSpace(stripExt(name))
	base = dupSuffixRe.ReplaceAllString(base, "")
	base = strings.TrimRight(base, " _-")
	for _, re := range []*regexp.Regexp{verPrefixedRe, verDottedRe} {
		if m := re.FindStringSubmatchIndex(base); m != nil && m[0] > 0 {
			version = strings.ReplaceAll(base[m[2]:m[3]], "_", ".")
			base = strings.TrimRight(base[:m[0]], " _-.")
			return base, version
		}
	}
	return base, ""
}

// toHalfWidth maps full-width ASCII (＆, Ａ, １) and the ideographic space to ASCII.
func toHalfWidth(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 0xFF01 && r <= 0xFF5E:
			return r - 0xFEE0
		case r == 0x3000:
			return ' '
		}
		return r
	}, s)
}

// matchKey is the normalized name used to group folders, archives and versions
// of the same asset: case, width, separators, brackets and version are ignored.
func matchKey(name string) string {
	base, _ := splitVersion(name)
	base = strings.ToLower(toHalfWidth(base))
	var b strings.Builder
	space := false
	for _, r := range base {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteRune(r)
			space = false
		} else {
			space = true
		}
	}
	return b.String()
}

// displayName turns a folder/file name into a readable asset name.
func displayName(name string) string {
	base, _ := splitVersion(name)
	base = strings.ReplaceAll(base, "_", " ")
	return strings.Join(strings.Fields(base), " ")
}

// boothIDFromName finds a BOOTH item id embedded in a folder or file name.
func boothIDFromName(name string) string {
	if m := boothIDInNameRe.FindStringSubmatch(stripExt(name)); m != nil {
		return m[1]
	}
	return ""
}

// boothIDsInText returns the BOOTH item ids linked in a text (readme, .url file).
func boothIDsInText(text string) []string {
	var ids []string
	for _, m := range boothLinkRe.FindAllStringSubmatch(text, -1) {
		ids = append(ids, m[1])
	}
	return ids
}

// boothIDFromURL extracts the item id from a stored BOOTH URL.
func boothIDFromURL(url string) string {
	if m := boothLinkRe.FindStringSubmatch(url); m != nil {
		return m[1]
	}
	return ""
}

// boothItemURL is the canonical URL stored for a BOOTH item id.
func boothItemURL(id string) string {
	return "https://booth.pm/ja/items/" + id
}

// containsWord reports whether key contains word as a whole space-separated token sequence.
func containsWord(key, word string) bool {
	if word == "" {
		return false
	}
	return strings.Contains(" "+key+" ", " "+word+" ")
}

// Exported helpers shared with the BOOTH importer.

// MatchKey is the normalized name used to compare asset and avatar names.
func MatchKey(name string) string { return matchKey(name) }

// ContainsWord reports whether key contains word as whole tokens.
func ContainsWord(key, word string) bool { return containsWord(key, word) }

// BoothIDFromURL extracts the item id from a BOOTH URL ("" if none).
func BoothIDFromURL(url string) string { return boothIDFromURL(url) }

// BoothIDsInText returns the BOOTH item ids linked in a text.
func BoothIDsInText(text string) []string { return boothIDsInText(text) }

// BoothItemURL is the canonical URL stored for a BOOTH item id.
func BoothItemURL(id string) string { return boothItemURL(id) }
