package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFontURL_Basic(t *testing.T) {
	css := `@font-face {
  font-family: 'Arial';
  src: url(https://fonts.gstatic.com/s/arial/v17/hash.woff2) format('woff2');
}`
	url := extractFontURL(css)
	if url != "https://fonts.gstatic.com/s/arial/v17/hash.woff2" {
		t.Errorf("got %q, want https://fonts.gstatic.com/s/arial/v17/hash.woff2", url)
	}
}

func TestExtractFontURL_WithQuotes(t *testing.T) {
	css := `src: url("https://fonts.gstatic.com/s/noto/v1/font.ttf") format('truetype')`
	url := extractFontURL(css)
	if url != "https://fonts.gstatic.com/s/noto/v1/font.ttf" {
		t.Errorf("got %q, want https://fonts.gstatic.com/s/noto/v1/font.ttf", url)
	}
}

func TestExtractFontURL_WithSingleQuotes(t *testing.T) {
	css := `src: url('https://fonts.gstatic.com/s/roboto/v18/font.otf') format('opentype')`
	url := extractFontURL(css)
	if url != "https://fonts.gstatic.com/s/roboto/v18/font.otf" {
		t.Errorf("got %q, want https://fonts.gstatic.com/s/roboto/v18/font.otf", url)
	}
}

func TestExtractFontURL_NoURL(t *testing.T) {
	css := `@font-face { font-family: 'Test'; }`
	url := extractFontURL(css)
	if url != "" {
		t.Errorf("expected empty, got %q", url)
	}
}

func TestExtractFontURL_EmptyCSS(t *testing.T) {
	url := extractFontURL("")
	if url != "" {
		t.Errorf("expected empty, got %q", url)
	}
}

func TestExtractFontURL_ProtocolRelative(t *testing.T) {
	css := `src: url(//fonts.gstatic.com/s/a/v1/font.woff2) format('woff2')`
	url := extractFontURL(css)
	if url != "https://fonts.gstatic.com/s/a/v1/font.woff2" {
		t.Errorf("got %q, want https://fonts.gstatic.com/s/a/v1/font.woff2", url)
	}
}

func TestDlFontsForSubs_FlagDisabled(t *testing.T) {
	DlFonts = boolPtr(false)
	paths := dlFontsForSubs(nil)
	if paths != nil {
		t.Errorf("expected nil, got %v", paths)
	}
}

func TestDlFontsForSubs_FlagNil(t *testing.T) {
	DlFonts = nil
	paths := dlFontsForSubs(nil)
	if paths != nil {
		t.Errorf("expected nil, got %v", paths)
	}
}

func TestDlFontsForSubs_NoFonts(t *testing.T) {
	DlFonts = boolPtr(true)
	dir := t.TempDir()
	subPath := filepath.Join(dir, "sub.ass")
	os.WriteFile(subPath, []byte("[Events]\nDialogue: 0,0:00:01.00,0:00:05.00,Default,,0,0,0,,hello\n"), 0644)

	paths := dlFontsForSubs([]mediaTrack{{Path: subPath, Language: "en-US"}})
	if len(paths) != 0 {
		t.Errorf("expected no font paths, got %v", paths)
	}
}

func TestBuildMuxArgs_WithFonts(t *testing.T) {
	info := EpisodeInfo{
		Title: "Ep",
		EpisodeMetadata: EpisodeMetadata{
			SeasonNumber:  1,
			EpisodeNumber: 1,
			SeriesTitle:   "S",
		},
	}
	fonts := []string{"/tmp/font1.ttf", "/tmp/font2.otf"}
	args := buildMuxArgs("/tmp/v.mp4", nil, nil, fonts, "/tmp/o.mkv", info)

	if !containsStr(args, "-attach") {
		t.Error("expected -attach in args")
	}
	if !containsStr(args, "/tmp/font1.ttf") {
		t.Error("expected font1 path in args")
	}
	if !containsStr(args, "/tmp/font2.otf") {
		t.Error("expected font2 path in args")
	}
	if !containsStr(args, "mimetype=application/x-truetype-font") {
		t.Error("expected ttf mimetype")
	}
	if !containsStr(args, "mimetype=application/vnd.ms-opentype") {
		t.Error("expected otf mimetype")
	}
}

func TestBuildMuxArgs_FontsInMuxOrder(t *testing.T) {
	info := EpisodeInfo{
		EpisodeMetadata: EpisodeMetadata{
			SeasonNumber: 1, EpisodeNumber: 1, SeriesTitle: "S",
		},
	}

	args := buildMuxArgs("/tmp/v.mp4", nil, nil, []string{"/tmp/a.ttf", "/tmp/b.woff2"}, "/tmp/o.mkv", info)

	foundA := false
	foundB := false
	for _, a := range args {
		if a == "/tmp/a.ttf" {
			foundA = true
		}
		if a == "/tmp/b.woff2" {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Error("expected both font paths in args")
	}

	if !containsStr(args, "mimetype=font/woff2") {
		t.Error("expected woff2 mimetype")
	}
}

func TestBuildMuxArgs_NoFontsStillWorks(t *testing.T) {
	info := EpisodeInfo{
		EpisodeMetadata: EpisodeMetadata{SeasonNumber: 1, EpisodeNumber: 1, SeriesTitle: "S"},
	}
	args := buildMuxArgs("/tmp/v.mp4", nil, nil, nil, "/tmp/o.mkv", info)
	if !containsStr(args, "-map") {
		t.Error("expected -map in args even without fonts")
	}
	if len(findArg(args, "-attach")) > 0 {
		t.Error("expected no -attach without fonts")
	}
}

func TestValidFontFile_Empty(t *testing.T) {
	if validFontFile(nil) {
		t.Error("nil should be invalid")
	}
	if validFontFile([]byte{}) {
		t.Error("empty should be invalid")
	}
}

func TestValidFontFile_TooSmall(t *testing.T) {
	if validFontFile([]byte{0x00, 0x01, 0x00}) {
		t.Error("3 bytes should be invalid")
	}
}

func TestValidFontFile_TrueType(t *testing.T) {
	if !validFontFile([]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) {
		t.Error("TrueType magic should be valid")
	}
}

func TestValidFontFile_OpenType(t *testing.T) {
	if !validFontFile([]byte("OTTO0123456789AB")) {
		t.Error("OpenType magic should be valid")
	}
}

func TestValidFontFile_WOFF(t *testing.T) {
	if !validFontFile([]byte("wOFF0123456789AB")) {
		t.Error("WOFF magic should be valid")
	}
}

func TestValidFontFile_WOFF2(t *testing.T) {
	if !validFontFile([]byte("wOF20123456789AB")) {
		t.Error("WOFF2 magic should be valid")
	}
}

func TestValidFontFile_TrueTypeCollection(t *testing.T) {
	if !validFontFile([]byte("ttcf0123456789AB")) {
		t.Error("ttcf magic should be valid")
	}
}

func TestValidFontFile_RandomBytes(t *testing.T) {
	if validFontFile([]byte("GIF89a0123456789")) {
		t.Error("GIF header should not be valid")
	}
	if validFontFile([]byte("<html>0123456789")) {
		t.Error("HTML should not be valid")
	}
}

func TestExtractFontURL_RejectsNonFontURL(t *testing.T) {
	css := `@font-face {
  src: url(https://example.com/image.png) format('png');
}`
	url := extractFontURL(css)
	if url != "" {
		t.Errorf("expected empty for non-font URL, got %q", url)
	}
}

func TestExtractFontURL_RejectsNonFontDomain(t *testing.T) {
	css := `@font-face {
  src: url(https://cdn.example.com/font.ttf) format('truetype');
}`
	url := extractFontURL(css)
	if url != "" {
		t.Errorf("expected empty for non-font-CDN domain, got %q", url)
	}
}

func TestExtractFontURL_AcceptsGstatic(t *testing.T) {
	css := `@font-face {
  src: url(https://fonts.gstatic.com/s/roboto/v18/KFOlCnqEu92Fr1MmEU9fBBc9.ttf) format('truetype');
}`
	url := extractFontURL(css)
	if url == "" {
		t.Error("expected gstatic ttf URL to be accepted")
	}
}

func TestIsFontURL_Valid(t *testing.T) {
	cases := []string{
		"https://fonts.gstatic.com/s/a/v1/font.ttf",
		"https://fonts.gstatic.com/s/a/v1/font.otf",
		"https://fonts.gstatic.com/s/a/v1/font.woff",
		"https://fonts.gstatic.com/s/a/v1/font.woff2",
		"//fonts.gstatic.com/s/a/v1/font.ttf",
	}
	for _, c := range cases {
		if !isFontURL(c) {
			t.Errorf("expected valid: %s", c)
		}
	}
}

func TestIsFontURL_Invalid(t *testing.T) {
	cases := []string{
		"https://example.com/font.ttf",
		"https://fonts.gstatic.com/s/a/v1/font.png",
		"https://fonts.gstatic.com/s/a/v1/font.svg",
		"",
		"not-a-url",
	}
	for _, c := range cases {
		if isFontURL(c) {
			t.Errorf("expected invalid: %s", c)
		}
	}
}
