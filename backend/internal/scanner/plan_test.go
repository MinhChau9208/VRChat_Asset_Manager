package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// tree creates folders (paths ending in "/") and files under a temp root.
func tree(t *testing.T, paths ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, p := range paths {
		full := filepath.Join(root, filepath.FromSlash(p))
		if strings.HasSuffix(p, "/") {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// planned returns "category: member, member" per group, with paths relative to root.
func planned(t *testing.T, root string, opts PlanOptions) []string {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Roots = []string{root}
	groups, _ := PlanWith(cfg, opts)
	var out []string
	for _, g := range groups {
		var members []string
		for _, m := range g.Members {
			rel, _ := filepath.Rel(root, m.Path)
			members = append(members, filepath.ToSlash(rel))
		}
		out = append(out, g.Category+": "+strings.Join(members, ", "))
	}
	sort.Strings(out)
	return out
}

func expectPlan(t *testing.T, got []string, want ...string) {
	t.Helper()
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("plan mismatch\n got:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

func TestSameNameInDifferentCategoriesStaysApart(t *testing.T) {
	root := tree(t,
		"Models/Kipfel/Kipfel.unitypackage",
		"Models/Zips/Kipfel.zip",
		"Clothes/Kipfel/Outfit.unitypackage",
		"Clothes/Boots/Boots.unitypackage",
		"Shoes/Boots/Boots.unitypackage",
		"Shoes/LEGACY/Boots.zip",
		"Clothes/LEGACY/Sneakers.zip",
		"Shoes/Sneakers/Sneakers.unitypackage",
	)
	expectPlan(t, planned(t, root, PlanOptions{}),
		"Avatar: Models/Kipfel, Models/Zips/Kipfel.zip",
		"Clothes: Clothes/Kipfel",
		"Clothes: Clothes/Boots",
		"Shoes: Shoes/Boots, Shoes/LEGACY/Boots.zip",
		// A zip stored in another category's LEGACY folder still finds its only folder.
		"Shoes: Shoes/Sneakers, Clothes/LEGACY/Sneakers.zip",
	)
}

func TestNestedAssetsAreFoundWhereTheirPartsAre(t *testing.T) {
	root := tree(t,
		// A shop folder holding two items: each item is an asset.
		"Clothes/ShopA/DressX/Prefab/DressX.prefab",
		"Clothes/ShopA/DressX/Textures/",
		"Clothes/ShopA/BootsY/BootsY.unitypackage",
		// A wrapper folder around a single item is the asset.
		"Facials/Cute_Facial/だるラボ_表情セット/set.unitypackage",
		// Numbered part folders and per-version / per-avatar children belong to the parent.
		"Models/Savarum/01Unitypackage/Savarum.unitypackage",
		"Models/Savarum/02FBX/Body.fbx",
		"Models/Kipfel/Kipfel.unitypackage",
		"Clothes/Ribbon_Dress/Kipfel/Dress.prefab",
		"Clothes/Ribbon_Dress/for Savarum/Dress.prefab",
		"Hair/TwinTail/1.0.0/Hair.unitypackage",
		"Hair/TwinTail/1.1.0/Hair.unitypackage",
		// A folder with only a readme is still an asset.
		"Gimmick/4460917 Piano/readme.txt",
	)
	expectPlan(t, planned(t, root, PlanOptions{}),
		"Clothes: Clothes/ShopA/DressX",
		"Clothes: Clothes/ShopA/BootsY",
		"Expression: Facials/Cute_Facial",
		"Avatar: Models/Savarum",
		"Avatar: Models/Kipfel",
		"Clothes: Clothes/Ribbon_Dress",
		"Hair: Hair/TwinTail",
		"Gimmick: Gimmick/4460917 Piano",
	)
}

func TestUnityProjectOnlyAssetsFolderIsScanned(t *testing.T) {
	root := tree(t,
		"Projects/MyAvatar/Assets/ShopA/DressX/Prefab/DressX.prefab",
		"Projects/MyAvatar/Assets/ShopA/BootsY/Prefab/BootsY.prefab",
		"Projects/MyAvatar/Assets/Kipfel/Prefab/Kipfel.prefab",
		"Projects/MyAvatar/Assets/Scenes/Main.unity",
		"Projects/MyAvatar/Assets/Editor/Tool.cs",
		"Projects/MyAvatar/Packages/com.vrchat.avatars/package.json",
		"Projects/MyAvatar/Library/PackageCache/x/",
		"Projects/MyAvatar/ProjectSettings/ProjectSettings.asset",
	)
	expectPlan(t, planned(t, root, PlanOptions{}),
		": Projects/MyAvatar/Assets/ShopA/DressX",
		": Projects/MyAvatar/Assets/ShopA/BootsY",
		": Projects/MyAvatar/Assets/Kipfel",
	)

	// A root that is itself a Unity project works the same way.
	project := filepath.Join(root, "Projects", "MyAvatar")
	got := planned(t, project, PlanOptions{})
	if len(got) != 3 {
		t.Errorf("project as root: expected 3 drafts, got %v", got)
	}
}
