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
