package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Folder names that hold the parts of an asset rather than an asset: a folder
// with a "Prefab" or "Textures" child is the asset itself. Matched on the first
// word of the normalized name, so "Prefab_Kipfel" and "Textures (PSD)" count.
var partNames = lowerSet([]string{
	"prefab", "prefabs", "texture", "textures", "tex", "material", "materials", "mat", "mats",
	"animation", "animations", "anim", "anims", "animator", "animators", "controller", "controllers",
	"fx", "model", "models", "mesh", "meshes", "fbx", "shader", "shaders", "script", "scripts",
	"sound", "sounds", "sfx", "menu", "menus", "expression", "expressions", "param", "params", "parameters",
	"psd", "blend", "source", "sources", "unitypackage",
	"プレハブ", "テクスチャ", "マテリアル", "アニメーション", "モデル",
})

// Folders of a Unity project's Assets/ that never hold purchased items.
var unityProjectSkip = lowerSet([]string{
	"editor", "plugins", "scenes", "settings", "gizmos", "textmesh pro", "xr", "streamingassets",
	"vrcsdk", "vrchat examples", "samples",
})

// "1.2.0", "v2", "ver1.02": a child folder that is only a version.
var pureVersionRe = regexp.MustCompile(`(?i)^(?:ver(?:sion)?|v)?[\s_.\-]*\d+(?:[._]\d+)*$`)

// maxNestDepth is how many levels below a category folder the scanner looks
// for assets inside collection folders (shop folders, Unity projects).
const maxNestDepth = 4

// normPath is the case-insensitive form used to compare paths.
func normPath(p string) string {
	return strings.ToLower(filepath.Clean(strings.TrimSpace(p)))
}

// Leading numbering of part folders: "01Unitypackage", "02_FBX".
var leadingNumberRe = regexp.MustCompile(`^\d+[\s_.\-]*`)

func isPartName(name string) bool {
	first, _, _ := strings.Cut(matchKey(leadingNumberRe.ReplaceAllString(name, "")), " ")
	return partNames[first]
}

// list reads a directory once; detection looks at the same folders many times.
func (p *planner) list(dir string) []os.DirEntry {
	if entries, ok := p.dirs[dir]; ok {
		return entries
	}
	entries := p.readDir(dir)
	p.dirs[dir] = entries
	return entries
}

// subdirs returns the child folders worth looking at as assets.
func (p *planner) subdirs(dir string) []string {
	var out []string
	for _, e := range p.list(dir) {
		name := e.Name()
		if e.IsDir() && !p.skip(name) && !p.archive[strings.ToLower(name)] {
			out = append(out, filepath.Join(dir, name))
		}
	}
	return out
}

// hasParts reports whether a folder directly holds Unity/3D files or a parts
// folder ("Prefab", "Textures"…), which makes it an asset.
func (p *planner) hasParts(dir string) bool {
	for _, e := range p.list(dir) {
		if p.skip(e.Name()) {
			continue
		}
		if e.IsDir() {
			if isPartName(e.Name()) {
				return true
			}
		} else if assetMarkerExts[strings.ToLower(filepath.Ext(e.Name()))] {
			return true
		}
	}
	return false
}

// isUnityProject reports whether dir is a Unity project: Assets/ next to one
// of Unity's own folders (ProjectSettings, Packages, Library).
func (p *planner) isUnityProject(dir string) bool {
	var assets, unity bool
	for _, e := range p.list(dir) {
		if !e.IsDir() {
			continue
		}
		switch strings.ToLower(e.Name()) {
		case "assets":
			assets = true
		case "projectsettings", "packages", "library":
			unity = true
		}
	}
	return assets && unity
}

// containsAssets reports whether some folder below dir looks like an asset.
func (p *planner) containsAssets(dir string, depth int) bool {
	if depth > maxNestDepth {
		return false
	}
	for _, c := range p.subdirs(dir) {
		if p.boundary[normPath(c)] || p.hasParts(c) || p.isUnityProject(c) || p.containsAssets(c, depth+1) {
			return true
		}
	}
	return false
}

// Words around an avatar name in per-avatar variant folders.
var variantFillers = lowerSet([]string{"for", "ver", "version", "only", "対応", "用", "向け", "専用"})

// isAvatarVariant reports whether a normalized folder name is just an avatar
// name ("Kipfel", "for Kipfel", "キプフェル対応"): such a folder is a per-avatar
// variant of its parent, not an asset of its own. "Small Lady Kipfel" is not.
func (p *planner) isAvatarVariant(key string) bool {
	if len(p.avatarKeys) == 0 {
		return false
	}
	// "キプフェル対応" -> "キプフェル 対応"
	words := strings.Fields(key)
	for i, word := range words {
		for _, suffix := range []string{"対応", "専用", "向け", "用"} {
			if trimmed := strings.TrimSuffix(word, suffix); trimmed != word && trimmed != "" {
				words[i] = trimmed + " " + suffix
				break
			}
		}
	}
	rest := " " + strings.Join(words, " ") + " "
	found := false
	for avatar := range p.avatarKeys {
		if strings.Contains(rest, " "+avatar+" ") {
			rest = strings.ReplaceAll(rest, " "+avatar+" ", " ")
			found = true
		}
	}
	if !found {
		return false
	}
	for _, word := range strings.Fields(rest) {
		if !variantFillers[word] {
			return false
		}
	}
	return true
}

// isCollection decides whether a folder groups several assets (a shop folder,
// "Assets/ShopA") instead of being one asset. It is one when it holds no parts
// itself and at least one child folder looks like an asset, unless the children
// are versions or per-avatar variants of the folder ("Kipfel_1.2.0", "Kipfel/",
// "for Manuka/").
func (p *planner) isCollection(dir string, depth int) bool {
	if p.ancestors[normPath(dir)] {
		return true // a registered or ignored asset lives below: look inside
	}
	if depth >= maxNestDepth || p.hasParts(dir) {
		return false
	}
	parentKey := matchKey(filepath.Base(dir))
	assetKeys := map[string]bool{}
	assetChildren := 0
	for _, c := range p.subdirs(dir) {
		name := filepath.Base(c)
		key := matchKey(name)
		if pureVersionRe.MatchString(name) || key == parentKey || p.isAvatarVariant(key) {
			return false
		}
		if p.boundary[normPath(c)] || p.hasParts(c) || p.isUnityProject(c) || p.containsAssets(c, depth+1) {
			assetKeys[key] = true
			assetChildren++
		}
	}
	// One asset inside is a wrapper folder ("Cute_Facial/だるラボ_表情セット/"):
	// the outer folder is the asset.
	if assetChildren < 2 {
		return false
	}
	// "Outfit/Dress_1.1/" and "Outfit/Dress_1.2/" are versions of one asset.
	return !(assetChildren > 1 && len(assetKeys) == 1)
}
