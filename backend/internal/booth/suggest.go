package booth

import (
	"regexp"
	"strconv"
	"strings"

	"vrchat-asset-manager/backend/internal/asset"
	"vrchat-asset-manager/backend/internal/scanner"
)

// Suggestion is what the app proposes to fill in from a BOOTH item.
type Suggestion struct {
	ItemID            string               `json:"item_id"`
	Name              string               `json:"name"`
	Author            string               `json:"author"`
	BoothURL          string               `json:"booth_url"`
	CategoryID        *int64               `json:"category_id"`
	CategoryName      string               `json:"category_name"`
	BoothCategory     string               `json:"booth_category"` // e.g. "3Dモデル > 3D衣装"
	Tags              []string             `json:"tags"`
	CompatibleAvatars []asset.CompatAvatar `json:"compatible_avatars"`
	Images            []string             `json:"images"`
	IsAdult           bool                 `json:"is_adult"`
	Price             string               `json:"price"`
}

// Avatar is an avatar asset already in the library.
type Avatar struct {
	ID      int64
	Name    string
	BoothID string
	Aliases []string // names the avatar may be called by, see AvatarAliases
}

// boothCategoryMap maps BOOTH's 3D model subcategories to app categories.
var boothCategoryMap = map[string]string{
	"3Dキャラクター":   "Avatar",
	"3D髪型":       "Hair",
	"3D衣装":       "Clothes",
	"3D装飾品":      "Accessory",
	"3D小道具":      "Prop",
	"3Dテクスチャ":    "Texture & Material",
	"3Dツール・システム": "Tool & Shader",
	"3Dモーション・アニメーション": "Animation",
	"3D環境・ワールド":       "World",
	"3Dモデル（その他）":      "Other",
}

// Keyword refinements: a hair item is usually listed under 3D装飾品 or 3D衣装.
var refinements = []struct {
	from     []string // BOOTH categories the rule applies to
	keywords []string // matched case-insensitively in name and tags
	unless   []string // ...unless the name also has one of these
	to       string
}{
	{[]string{"3D装飾品", "3D衣装"}, []string{"髪型", "ヘア", "hair", "ウィッグ"}, []string{"衣装", "outfit", "セット"}, "Hair"},
	{[]string{"3D装飾品", "3D衣装"}, []string{"靴", "シューズ", "ブーツ", "shoes", "sneaker"}, []string{"衣装", "outfit", "セット", "ワンピ"}, "Shoes"},
	{[]string{"3D装飾品"}, []string{"耳", "しっぽ", "尻尾", "ear", "tail"}, nil, "Ears & Tail"},
	{[]string{"3Dテクスチャ"}, []string{"瞳", "アイ", "eye", "目"}, nil, "Eyes"},
	{[]string{"3Dテクスチャ"}, []string{"メイク", "makeup", "make up", "チーク"}, nil, "Makeup"},
	{[]string{"3Dテクスチャ", "3Dツール・システム"}, []string{"表情", "facial", "expression", "シェイプキー"}, nil, "Expression"},
	{[]string{"3Dツール・システム", "3D小道具"}, []string{"ギミック", "gimmick"}, nil, "Gimmick"},
}

// Tags too generic to be useful in a personal library.
var genericTags = map[string]bool{
	"3d": true, "3dcg": true, "3dモデル": true, "3dキャラクター": true, "3d衣装": true, "3d装飾品": true,
	"vrchat": true, "vrc": true, "vrc想定モデル": true, "vrchat想定": true, "blender": true, "fbx": true,
	"unity": true, "アバター": true, "衣装": true, "オリジナル3dモデル": true, "オリジナル": true,
	"modularavatar": true, "modular avatar": true, "ma": true, "ma対応": true, "vrchat向け": true,
}

const maxTags = 10

var (
	compatSectionRe = regexp.MustCompile(`対応アバター|対応モデル|Supported Avatars?|Compatible Avatars?`)
	urlRe           = regexp.MustCompile(`https?://\S+`)
)

func containsAny(text string, words []string) bool {
	text = strings.ToLower(text)
	for _, w := range words {
		if strings.Contains(text, strings.ToLower(w)) {
			return true
		}
	}
	return false
}

// Suggest builds suggestions from an item. categoryIDs maps lower-case
// category names to ids; avatars are the library's avatar assets.
func Suggest(item *Item, categoryIDs map[string]int64, avatars []Avatar) Suggestion {
	id := itoa(item.ID)
	s := Suggestion{
		ItemID:   id,
		Name:     strings.TrimSpace(item.Name),
		Author:   strings.TrimSpace(item.Shop.Name),
		BoothURL: scanner.BoothItemURL(id),
		IsAdult:  item.IsAdult,
		Price:    item.Price,
		Tags:     []string{},
		Images:   []string{},
	}
	if item.Category.Parent.Name != "" {
		s.BoothCategory = item.Category.Parent.Name + " > " + item.Category.Name
	} else {
		s.BoothCategory = item.Category.Name
	}

	// Category
	var tagText strings.Builder
	for _, t := range item.Tags {
		tagText.WriteString(t.Name + " ")
	}
	category := boothCategoryMap[item.Category.Name]
	for _, r := range refinements {
		if !containsAny(item.Category.Name, r.from) {
			continue
		}
		if containsAny(item.Name+" "+tagText.String(), r.keywords) && !containsAny(item.Name, r.unless) {
			category = r.to
			break
		}
	}
	if catID, ok := categoryIDs[strings.ToLower(category)]; ok {
		s.CategoryID = &catID
		s.CategoryName = category
	}

	// Tags
	for _, t := range item.Tags {
		name := strings.TrimSpace(t.Name)
		if name == "" || genericTags[strings.ToLower(name)] {
			continue
		}
		s.Tags = append(s.Tags, name)
		if len(s.Tags) == maxTags {
			break
		}
	}

	// Images
	for _, img := range item.Images {
		if img.Original != "" {
			s.Images = append(s.Images, img.Original)
		}
	}

	s.CompatibleAvatars = suggestCompat(item, avatars)
	return s
}

// suggestCompat finds avatars the item is made for: library avatars linked or
// named in the description/variations, plus avatar names listed under a
// "対応アバター" section that are not in the library.
func suggestCompat(item *Item, avatars []Avatar) []asset.CompatAvatar {
	result := []asset.CompatAvatar{}
	seen := map[string]bool{}
	add := func(c asset.CompatAvatar) {
		key := strings.ToLower(c.AvatarName)
		if c.AvatarName == "" || seen[key] {
			return
		}
		seen[key] = true
		result = append(result, c)
	}

	// Supported avatars appear in the name, variations, description or tags.
	var text strings.Builder
	text.WriteString(item.Name + "\n" + item.Description + "\n")
	for _, v := range item.Variations {
		text.WriteString(v.Name + "\n")
	}
	for _, t := range item.Tags {
		text.WriteString(t.Name + "\n")
	}
	fullText := text.String()
	linked := map[string]bool{}
	for _, id := range scanner.BoothIDsInText(item.Description) {
		linked[id] = true
	}

	matchesLibrary := func(text string) bool {
		for _, a := range avatars {
			if mentions(text, a) {
				return true
			}
		}
		return false
	}

	for _, a := range avatars {
		if itoa(item.ID) == a.BoothID {
			continue // the item is this avatar
		}
		if (a.BoothID != "" && linked[a.BoothID]) || mentions(fullText, a) {
			id := a.ID
			add(asset.CompatAvatar{AvatarAssetID: &id, AvatarName: a.Name})
		}
	}

	// Names listed in a "対応アバター" section that are not library avatars.
	if loc := compatSectionRe.FindStringIndex(item.Description); loc != nil {
		section := item.Description[loc[1]:]
		for i, line := range strings.Split(section, "\n") {
			if i > 25 {
				break
			}
			line = strings.TrimSpace(line)
			if i > 0 && (strings.HasPrefix(line, "■") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "【")) {
				break
			}
			if !strings.HasPrefix(line, "・") && !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "•") && !strings.HasPrefix(line, "◆") {
				continue
			}
			lineIDs := scanner.BoothIDsInText(line)
			isLibrary := false
			for _, a := range avatars {
				for _, lid := range lineIDs {
					if a.BoothID != "" && a.BoothID == lid {
						isLibrary = true
					}
				}
			}
			name := urlRe.ReplaceAllString(line, "")
			name = strings.Trim(name, " ・-•◆−–—:：　")
			if isLibrary || matchesLibrary(name) || name == "" || len([]rune(name)) > 40 {
				continue
			}
			add(asset.CompatAvatar{AvatarName: name})
		}
	}
	return result
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

var (
	aliasSplitRe = regexp.MustCompile(`[\s/／|｜()（）\[\]【】「」『』・,、:：~〜_]+`)
	versionLike  = regexp.MustCompile(`^v?[\d.]+$`)
	// Words in avatar product names that are not part of the avatar's name.
	aliasStopWords = map[string]bool{
		"オリジナル3dモデル": true, "オリジナル": true, "3dモデル": true, "3d": true, "avatar": true,
		"アバター": true, "original": true, "model": true, "vrchat": true, "vrc": true, "quest対応": true,
		// Common English words that would match unrelated descriptions.
		"new": true, "test": true, "base": true, "basic": true, "body": true, "version": true,
		"quest": true, "mobile": true, "free": true, "full": true, "pack": true, "package": true,
	}
)

// AvatarAliases splits avatar names ("Kipfel_1.2.0", "キプフェル Kipfel / オリジナル3Dモデル")
// into the names an item description might use ("Kipfel", "キプフェル").
func AvatarAliases(names ...string) []string {
	var aliases []string
	seen := map[string]bool{}
	for _, name := range names {
		for _, part := range aliasSplitRe.Split(name, -1) {
			part = strings.Trim(part, "-−–—.")
			lower := strings.ToLower(part)
			// Latin names need 4+ letters ("Kipfel", "Siska"); Japanese 3+ ("マヌカ").
			minLen := 3
			if isASCII(part) {
				minLen = 4
			}
			if len([]rune(part)) < minLen || aliasStopWords[lower] || versionLike.MatchString(lower) || seen[lower] {
				continue
			}
			seen[lower] = true
			aliases = append(aliases, part)
		}
	}
	return aliases
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

// mentions reports whether text names the avatar. Latin names must match whole
// words ("Kipfel" but not "Kipfelsuit"); Japanese names have no word breaks,
// so they match as substrings.
func mentions(text string, a Avatar) bool {
	key := scanner.MatchKey(text)
	for _, alias := range a.Aliases {
		if isASCII(alias) {
			if scanner.ContainsWord(key, scanner.MatchKey(alias)) {
				return true
			}
		} else if strings.Contains(text, alias) {
			return true
		}
	}
	return false
}
