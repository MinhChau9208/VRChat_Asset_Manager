package scanner

import "testing"

// Real names from the user's library.
func TestSplitVersion(t *testing.T) {
	cases := []struct{ in, base, version string }{
		{"Kipfel_1.2.0", "Kipfel", "1.2.0"},
		{"Kipfel_1.2.0.zip", "Kipfel", "1.2.0"},
		{"Kipfel v1.1.1.zip", "Kipfel", "1.1.1"},
		{"sipisipi_Cosset_ver1.02", "sipisipi_Cosset", "1.02"},
		{"S_AyakashiMensKimono_Ver_1.3", "S_AyakashiMensKimono", "1.3"},
		{"LuminousWandv1.4.zip", "LuminousWand", "1.4"},
		{"IndustrialStripe_v2", "IndustrialStripe", "2"},
		{"GaeaRhythm-Medjed-v1", "GaeaRhythm-Medjed", "1"},
		{"BIG_OKUTI_Kipfel1.0", "BIG_OKUTI_Kipfel", "1.0"},
		{"定事務所代表_v1.3__1_", "定事務所代表", "1.3"},
		{"オフショルワンピ＆ゆるみつあみv1.1.1", "オフショルワンピ＆ゆるみつあみ", "1.1.1"},
		{"4460917 avatargimmick 8ya_Undersea Piano 2", "4460917 avatargimmick 8ya_Undersea Piano 2", ""},
		{"Savarum_251217_BaseSet", "Savarum_251217_BaseSet", ""},
		{"Gothic Doll", "Gothic Doll", ""},
		{"BlendShare-0.0.11", "BlendShare", "0.0.11"},
	}
	for _, c := range cases {
		base, version := splitVersion(c.in)
		if base != c.base || version != c.version {
			t.Errorf("splitVersion(%q) = (%q, %q), want (%q, %q)", c.in, base, version, c.base, c.version)
		}
	}
}

func TestMatchKeyPairsFoldersWithArchives(t *testing.T) {
	pairs := [][2]string{
		{"Gothic Doll", "Gothic_Doll.zip"},
		{"オフショルワンピ＆ゆるみつあみv1.1.1", "オフショルワンピ_ゆるみつあみv1.1.1.zip"},
		{"[ Space Cat ]", "[ Space Cat ].zip"},
		{"Kipfel_1.2.0", "Kipfel v1.1.1.zip"},
		{"YM_Kipfel_Rabbit Ear&Tail.unitypackage", "YM_Kipfel_Rabbit_Ear_Tail.zip"},
		{"青さは止んだ → Nanatsukaze", "青さは止んだ → Nanatsukaze.zip"},
		{"Re)Lovely Ribbon Lolita", "Re)Lovely Ribbon Lolita.zip"},
		{"BlendShare-0.0.11", "BlendShare-0.0.11.zip"},
	}
	for _, p := range pairs {
		if a, b := matchKey(p[0]), matchKey(p[1]); a != b || a == "" {
			t.Errorf("matchKey(%q)=%q should equal matchKey(%q)=%q", p[0], a, p[1], b)
		}
	}
	if matchKey("kip-taiyaki.zip") == matchKey("kip-taiyaki-eat.zip") {
		t.Errorf("different assets must not share a key")
	}
	if matchKey("Kipfel_Kitsune_1.0.0") == matchKey("Kipfel_1.2.0") {
		t.Errorf("Kipfel Kitsune must not group with the Kipfel avatar")
	}
}

func TestBoothIDs(t *testing.T) {
	if got := boothIDFromName("4460917 avatargimmick 8ya_Undersea Piano 2"); got != "4460917" {
		t.Errorf("expected id from folder name, got %q", got)
	}
	for _, name := range []string{"Savarum_251217_BaseSet", "20230712015053vn3license_en.pdf", "Kipfel_1.2.0"} {
		if got := boothIDFromName(name); got != "" {
			t.Errorf("%q should not yield a BOOTH id, got %q", name, got)
		}
	}
	text := "URL=https://hamanosis.booth.pm/items/6834468\n see https://booth.pm/ja/items/5901276?_gl=1*abc and https://hamanosis.booth.pm/"
	ids := boothIDsInText(text)
	if len(ids) != 2 || ids[0] != "6834468" || ids[1] != "5901276" {
		t.Errorf("unexpected ids %v", ids)
	}
	if got := boothIDFromURL("https://mukumi.booth.pm/items/5813187"); got != "5813187" {
		t.Errorf("unexpected id from URL %q", got)
	}
}

func TestDisplayName(t *testing.T) {
	if got := displayName("hamanosis_Small_Lady_Kipfel"); got != "hamanosis Small Lady Kipfel" {
		t.Errorf("got %q", got)
	}
	if got := displayName("LuminousWandv1.4.zip"); got != "LuminousWand" {
		t.Errorf("got %q", got)
	}
}
