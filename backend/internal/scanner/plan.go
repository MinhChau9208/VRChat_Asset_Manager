package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Config is the user-editable scanner configuration.
type Config struct {
	Roots       []string `json:"roots"`
	Ignore      []string `json:"ignore"`       // folder/file names never scanned or read (case-insensitive)
	ArchiveDirs []string `json:"archive_dirs"` // folders holding original zips, e.g. LEGACY
	// FolderMap maps a category folder name (lower-case) to a category name.
	FolderMap map[string]string `json:"folder_map"`
	// KnownDependencies are BOOTH item ids that readmes link to but that are
	// never the asset itself (shaders, tools).
	KnownDependencies []string `json:"known_dependencies"`
}

// DefaultConfig matches the folder layout the app was designed around.
func DefaultConfig() Config {
	return Config{
		Roots:       []string{},
		Ignore:      []string{"AvatarPass", "$RECYCLE.BIN", "System Volume Information"},
		ArchiveDirs: []string{"LEGACY", "Zips", "Archive", "Archives"},
		FolderMap: map[string]string{
			"models": "Avatar", "avatar": "Avatar", "avatars": "Avatar",
			"outfit": "Outfit", "outfits": "Outfit",
			"clothes": "Clothes", "clothing": "Clothes", "衣装": "Clothes",
			"shoes": "Shoes",
			"hair":  "Hair", "髪型": "Hair",
			"accessory": "Accessory", "accessories": "Accessory",
			"ears": "Ears & Tail", "tail": "Ears & Tail", "tails": "Ears & Tail",
			"eyes": "Eyes", "eye": "Eyes",
			"facials": "Expression", "facial": "Expression", "expression": "Expression", "expressions": "Expression",
			"makeups": "Makeup", "makeup": "Makeup",
			"gimmick": "Gimmick", "gimmicks": "Gimmick",
			"pianogimick": "Prop", "prop": "Prop", "props": "Prop",
			"animation": "Animation", "animations": "Animation", "motion": "Animation",
			"texture": "Texture & Material", "textures": "Texture & Material", "material": "Texture & Material", "materials": "Texture & Material",
			"shader": "Tool & Shader", "shaders": "Tool & Shader", "tools": "Tool & Shader", "community": "Tool & Shader",
			"world": "World", "worlds": "World",
			"audio": "Audio", "music": "Audio", "sound": "Audio",
			"unorganized": "Other", "other": "Other", "misc": "Other",
		},
		KnownDependencies: []string{"3087170", "5901276"}, // lilToon, BlendShare
	}
}

// Member is one folder or file that belongs to a group.
type Member struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"` // folder | archive | unitypackage | file
	Version string `json:"version"`
}

// BoothLink is a BOOTH item id found in a readme or .url file.
type BoothLink struct {
	ID   string `json:"id"`
	File string `json:"file"` // file name it was found in
}

// Group is everything on disk that the scanner believes is one asset.
type Group struct {
	Key         string
	Name        string
	Category    string // category name from the folder mapping, "" if unknown
	Members     []Member
	BoothID     string // from a folder/file name
	BoothLinks  []BoothLink
	PreviewPath string
}

// entry is one folder or file found by the walk, before grouping.
type entry struct {
	path, kind, category string
	isFile               bool
}

type planner struct {
	cfg     Config
	ignore  map[string]bool
	archive map[string]bool
	entries []entry
	images  map[string]string // matchKey -> image path found next to archives
	warns   []string
}

func lowerSet(names []string) map[string]bool {
	set := map[string]bool{}
	for _, n := range names {
		set[strings.ToLower(strings.TrimSpace(n))] = true
	}
	return set
}

// PlanOptions carries what the library already knows into a scan.
type PlanOptions struct{}

// Plan walks the configured roots and groups what it finds. It reads only
// directory listings, small text files and image headers; it never writes.
func Plan(cfg Config) ([]*Group, []string) { return PlanWith(cfg, PlanOptions{}) }

// PlanWith is Plan with knowledge from the library.
func PlanWith(cfg Config, opts PlanOptions) ([]*Group, []string) {
	p := &planner{
		cfg:     cfg,
		ignore:  lowerSet(cfg.Ignore),
		archive: lowerSet(cfg.ArchiveDirs),
		images:  map[string]string{},
	}
	for _, root := range cfg.Roots {
		p.scanRoot(root)
	}

	groups := p.group()
	for _, g := range groups {
		p.finish(g)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Members[0].Path < groups[j].Members[0].Path })
	return groups, p.warns
}

// group turns entries into groups. Folders only merge with folders of the same
// category, so two different items that share a name ("Boots" in Shoes and in
// Clothes, the Kipfel avatar and an outfit folder called "Kipfel") stay apart.
// Archives and packages join a same-named folder, preferring one in their own
// category; a stray zip in another category's LEGACY folder joins the only
// folder with its name, but never across the Avatar / non-Avatar line.
func (p *planner) group() []*Group {
	var groups []*Group
	byKey := map[string]*Group{}     // key + category -> group
	folders := map[string][]*Group{} // key -> folder groups
	get := func(key, category string) *Group {
		id := key + "\x00" + strings.ToLower(category)
		g := byKey[id]
		if g == nil {
			g = &Group{Key: key, Category: category}
			byKey[id] = g
			groups = append(groups, g)
		}
		return g
	}
	addMember := func(g *Group, e entry) {
		_, version := splitVersion(filepath.Base(e.path))
		g.Members = append(g.Members, Member{Path: e.path, Kind: e.kind, Version: version})
	}

	for _, e := range p.entries {
		if e.isFile {
			continue
		}
		key := matchKey(filepath.Base(e.path))
		g := get(key, e.category)
		if len(g.Members) == 0 {
			folders[key] = append(folders[key], g)
		}
		addMember(g, e)
	}
	for _, e := range p.entries {
		if !e.isFile {
			continue
		}
		key := matchKey(filepath.Base(e.path))
		var target *Group
		var crossing []*Group
		for _, g := range folders[key] {
			if strings.EqualFold(g.Category, e.category) {
				target = g
				break
			}
			if !crossesAvatarLine(g.Category, e.category) {
				crossing = append(crossing, g)
			}
		}
		if target == nil && len(crossing) == 1 {
			target = crossing[0]
		}
		if target == nil {
			target = get(key, e.category)
		}
		addMember(target, e)
	}
	return groups
}

// crossesAvatarLine reports whether one category is Avatar and the other is a
// known non-Avatar category: an avatar and an outfit never share files.
func crossesAvatarLine(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(a, "Avatar") != strings.EqualFold(b, "Avatar")
}

func (p *planner) skip(name string) bool {
	return strings.HasPrefix(name, ".") || p.ignore[strings.ToLower(name)]
}

func (p *planner) readDir(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		p.warns = append(p.warns, "cannot read "+dir+": "+err.Error())
	}
	return entries
}

func (p *planner) scanRoot(root string) {
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		p.warns = append(p.warns, "library root not found: "+root)
		return
	}
	for _, e := range p.readDir(root) {
		name, path := e.Name(), filepath.Join(root, e.Name())
		if p.skip(name) {
			continue
		}
		switch {
		case e.IsDir() && p.archive[strings.ToLower(name)]:
			p.scanArchiveDir(path, "")
		case e.IsDir():
			category, mapped := p.cfg.FolderMap[strings.ToLower(name)]
			if !mapped && looksLikeAsset(path) {
				p.add(path, "folder", "", false) // loose asset at the root
			} else {
				p.scanCategory(path, category)
			}
		default:
			p.addFile(path, "")
		}
	}
}

func (p *planner) scanCategory(dir, category string) {
	for _, e := range p.readDir(dir) {
		name, path := e.Name(), filepath.Join(dir, e.Name())
		if p.skip(name) {
			continue
		}
		switch {
		case e.IsDir() && p.archive[strings.ToLower(name)]:
			p.scanArchiveDir(path, category)
		case e.IsDir():
			p.add(path, "folder", category, false)
		default:
			p.addFile(path, category)
		}
	}
}

// scanArchiveDir collects original downloads kept in LEGACY-style folders.
func (p *planner) scanArchiveDir(dir, category string) {
	for _, e := range p.readDir(dir) {
		if e.IsDir() || p.skip(e.Name()) {
			continue
		}
		p.addFile(filepath.Join(dir, e.Name()), category)
	}
}

// addFile handles a loose file: packages and archives are assets, audio/video
// files are assets of kind "file", images are kept as preview candidates.
func (p *planner) addFile(path, category string) {
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case archiveExts[ext]:
		p.add(path, "archive", category, true)
	case ext == ".unitypackage":
		p.add(path, "unitypackage", category, true)
	case mediaExts[ext]:
		p.add(path, "file", category, true)
	case imageExts[ext]:
		if key := matchKey(filepath.Base(path)); key != "" {
			p.images[key] = path
		}
	}
}

func (p *planner) add(path, kind, category string, isFile bool) {
	if matchKey(filepath.Base(path)) == "" {
		return
	}
	p.entries = append(p.entries, entry{path: path, kind: kind, category: category, isFile: isFile})
}

// looksLikeAsset reports whether a folder directly contains Unity/3D files.
func looksLikeAsset(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && assetMarkerExts[strings.ToLower(filepath.Ext(e.Name()))] {
			return true
		}
	}
	return false
}

// compareVersions compares dotted numeric versions ("1.10" > "1.9").
func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

var kindRank = map[string]int{"folder": 3, "unitypackage": 2, "archive": 1, "file": 0}

// finish orders members (primary first), names the group and collects hints.
func (p *planner) finish(g *Group) {
	sort.SliceStable(g.Members, func(i, j int) bool {
		a, b := g.Members[i], g.Members[j]
		if kindRank[a.Kind] != kindRank[b.Kind] {
			return kindRank[a.Kind] > kindRank[b.Kind]
		}
		if c := compareVersions(a.Version, b.Version); c != 0 {
			return c > 0
		}
		return a.Path < b.Path
	})
	g.Name = displayName(filepath.Base(g.Members[0].Path))

	for _, m := range g.Members {
		if id := boothIDFromName(filepath.Base(m.Path)); id != "" && g.BoothID == "" {
			g.BoothID = id
			// "4460917 avatargimmick ..." -> "avatargimmick ..."
			g.Name = strings.TrimPrefix(g.Name, id+" ")
		}
		if m.Kind == "folder" {
			g.BoothLinks = append(g.BoothLinks, p.findBoothLinks(m.Path)...)
			if g.PreviewPath == "" {
				g.PreviewPath = findPreview(m.Path)
			}
		}
	}
	if g.PreviewPath == "" {
		g.PreviewPath = p.images[g.Key]
	}
}

const (
	maxTextFileSize  = 200 << 10
	maxTextFilesRead = 40
	maxLinkDepth     = 3
)

// findBoothLinks reads small .url/.txt/.md/.html files (up to 3 levels deep)
// inside an asset folder and returns the BOOTH items they link to.
func (p *planner) findBoothLinks(dir string) []BoothLink {
	var links []BoothLink
	read := 0
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path != dir && p.skip(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			if rel != "." && strings.Count(rel, string(filepath.Separator)) >= maxLinkDepth-1 {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".url", ".txt", ".md", ".html", ".htm":
		default:
			return nil
		}
		if read >= maxTextFilesRead {
			return filepath.SkipAll
		}
		if info, err := d.Info(); err != nil || info.Size() > maxTextFileSize {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		read++
		for _, id := range boothIDsInText(string(data)) {
			links = append(links, BoothLink{ID: id, File: d.Name()})
		}
		return nil
	})
	return links
}

var previewNameHints = []string{"main", "thumb", "preview", "cover", "sample", "top", "icon"}

// findPreview picks an image at the top of an asset folder: one whose name
// suggests a cover image, or the only image there.
func findPreview(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var images []string
	for _, e := range entries {
		if !e.IsDir() && imageExts[strings.ToLower(filepath.Ext(e.Name()))] {
			images = append(images, e.Name())
		}
	}
	for _, hint := range previewNameHints {
		for _, name := range images {
			if strings.Contains(strings.ToLower(name), hint) {
				return filepath.Join(dir, name)
			}
		}
	}
	if len(images) == 1 {
		return filepath.Join(dir, images[0])
	}
	return ""
}
