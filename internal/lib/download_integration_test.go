package lib

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestAllTracksPipeline_ManyDubs(t *testing.T) {
	info := EpisodeInfo{
		Title: "Test Episode",
		EpisodeMetadata: EpisodeMetadata{
			SeriesTitle:   "Test Series",
			SeasonNumber:  1,
			EpisodeNumber: 1,
			AudioLocale:   "ja-JP",
			Versions: []*DubVersion{
				{AudioLocale: "en-US", GUID: "G-EN"},
				{AudioLocale: "de-DE", GUID: "G-DE"},
				{AudioLocale: "fr-FR", GUID: "G-FR"},
				{AudioLocale: "pt-BR", GUID: "G-PT"},
			},
		},
	}

	variants := resolveAudioVariants("G-JA", info, "all")
	if len(variants) != 5 {
		t.Fatalf("expected 5 audio variants, got %d: %v", len(variants), variants)
	}
	if variants[0].ContentID != "G-JA" || variants[0].AudioLocale != "ja-JP" {
		t.Errorf("first = %+v, want ja-JP/G-JA", variants[0])
	}
	if !sortCheck(variantsToLocales(variants)[1:]) {
		t.Errorf("not sorted: %v", variantsToLocales(variants)[1:])
	}

	subs := makeSubMap("en-US", "ja-JP", "pt-BR", "de-DE", "fr-FR", "ar-SA", "es-419", "it-IT", "ru-RU")
	subLanguages := resolveSubtitleLanguages(subs, "all")
	if len(subLanguages) < 5 {
		t.Errorf("expected >=5 sub langs, got %d: %v", len(subLanguages), subLanguages)
	}
	if !sortCheck(subLanguages) {
		t.Errorf("subs not sorted: %v", subLanguages)
	}

	audioTracks := tracksFromVariants(variants, "/tmp/ep_a_")
	subtitleTracks := tracksFromLocales(subLanguages, "/tmp/ep_s_")
	args := buildMuxArgs("/tmp/v.mp4", audioTracks, subtitleTracks, nil, "/tmp/o.mkv", info)

	for i := 0; i < len(variants); i++ {
		requireArg(t, args, fmt.Sprintf("%d:a:0", i+1))
	}
	subStart := len(variants) + 1
	for i := 0; i < len(subLanguages); i++ {
		requireArg(t, args, fmt.Sprintf("%d:0", subStart+i))
	}
}

func TestAllTracksPipeline_FewDubs(t *testing.T) {
	info := EpisodeInfo{
		Title: "Test Episode",
		EpisodeMetadata: EpisodeMetadata{
			SeriesTitle:   "Test Series",
			SeasonNumber:  1,
			EpisodeNumber: 1,
			AudioLocale:   "ja-JP",
			Versions: []*DubVersion{
				{AudioLocale: "en-US", GUID: "G-EN"},
				{AudioLocale: "es-419", GUID: "G-ES"},
			},
		},
	}

	variants := resolveAudioVariants("G-JA", info, "all")
	if len(variants) != 3 {
		t.Fatalf("expected 3 audio variants, got %d", len(variants))
	}
	if !sortCheck(variantsToLocales(variants)[1:]) {
		t.Errorf("not sorted: %v", variantsToLocales(variants)[1:])
	}

	subs := makeSubMap("en-US", "es-419")
	subLanguages := resolveSubtitleLanguages(subs, "all")
	if len(subLanguages) != 2 {
		t.Errorf("expected 2 sub langs, got %d", len(subLanguages))
	}

	audioTracks := tracksFromVariants(variants, "/tmp/ep_b_")
	subtitleTracks := tracksFromLocales(subLanguages, "/tmp/ep_b_")
	args := buildMuxArgs("/tmp/v.mp4", audioTracks, subtitleTracks, nil, "/tmp/o.mkv", info)

	for i := 0; i < len(variants); i++ {
		requireArg(t, args, fmt.Sprintf("%d:a:0", i+1))
	}
	subStart := len(variants) + 1
	for i := 0; i < len(subLanguages); i++ {
		requireArg(t, args, fmt.Sprintf("%d:0", subStart+i))
	}
}

func TestSingleAudioTrack(t *testing.T) {
	info := EpisodeInfo{
		EpisodeMetadata: EpisodeMetadata{
			AudioLocale: "ja-JP",
			Versions: []*DubVersion{
				{AudioLocale: "en-US", GUID: "G-EN"},
				{AudioLocale: "de-DE", GUID: "G-DE"},
			},
		},
	}

	for _, loc := range []string{"ja-JP", "en-US", "de-DE"} {
		variants := resolveAudioVariants("G-JA", info, loc)
		if len(variants) != 1 {
			t.Errorf("%s: expected 1 variant, got %d", loc, len(variants))
			continue
		}
		if variants[0].AudioLocale != loc {
			t.Errorf("%s: wrong locale %s", loc, variants[0].AudioLocale)
		}
	}
}

func TestSingleSubtitleTrack(t *testing.T) {
	subs := makeSubMap("en-US", "ja-JP", "de-DE", "fr-FR")

	for _, loc := range []string{"ja-JP", "en-US"} {
		got := resolveSubtitleLanguages(subs, loc)
		if !reflect.DeepEqual(got, []string{loc}) {
			t.Errorf("%s: got %v, want [%s]", loc, got, loc)
		}
	}
	if got := resolveSubtitleLanguages(subs, "xx-XX"); got != nil {
		t.Errorf("missing locale: got %v, want nil", got)
	}
}

func TestDownloadArgsOrder(t *testing.T) {
	info := EpisodeInfo{
		Title: "Test",
		EpisodeMetadata: EpisodeMetadata{
			SeriesTitle:   "S",
			SeasonNumber:  1,
			EpisodeNumber: 1,
			AudioLocale:   "ja-JP",
			Versions:      []*DubVersion{{AudioLocale: "en-US", GUID: "G-EN"}},
		},
	}

	variants := resolveAudioVariants("G-JA", info, "all")
	audioTracks := tracksFromVariants(variants, "/tmp/a_")
	subLanguages := resolveSubtitleLanguages(makeSubMap("en-US", "pt-BR", "fr-FR"), "all")
	subtitleTracks := tracksFromLocales(subLanguages, "/tmp/s_")
	args := buildMuxArgs("/tmp/v.mp4", audioTracks, subtitleTracks, nil, "/tmp/o.mkv", info)

	if n := len(findArg(args, "-i")); n != 1+len(audioTracks)+len(subtitleTracks) {
		t.Errorf("expected %d -i args, got %d", 1+len(audioTracks)+len(subtitleTracks), n)
	}
	if n := len(findArg(args, "-map")); n != 1+len(audioTracks)+len(subtitleTracks) {
		t.Errorf("expected %d -map args, got %d", 1+len(audioTracks)+len(subtitleTracks), n)
	}
	if len(findArg(args, "-c:v")) == 0 {
		t.Error("-c:v missing")
	}
	if len(findArg(args, "-metadata:g")) < 4 {
		t.Errorf("expected >=4 global metadata args, got %d", len(findArg(args, "-metadata:g")))
	}
}

func TestMuxArgs_Metadata(t *testing.T) {
	infos := []EpisodeInfo{
		{Title: "Ep A", EpisodeMetadata: EpisodeMetadata{SeasonNumber: 1, EpisodeNumber: 2, SeriesTitle: "Show"}},
		{Title: "Ep B", EpisodeMetadata: EpisodeMetadata{SeasonNumber: 2, EpisodeNumber: 5, SeriesTitle: "Show"}},
	}
	audio := []mediaTrack{{Path: "/tmp/a.m4a", Language: "ja-JP"}}

	for _, info := range infos {
		args := buildMuxArgs("/tmp/v.mp4", audio, nil, nil, "/tmp/o.mkv", info)
		wants := []string{
			fmt.Sprintf("title=S%02dE%02d - %s", info.EpisodeMetadata.SeasonNumber, info.EpisodeMetadata.EpisodeNumber, info.Title),
			"show=" + info.EpisodeMetadata.SeriesTitle,
			fmt.Sprintf("track=%d", info.EpisodeMetadata.EpisodeNumber),
			fmt.Sprintf("season_number=%d", info.EpisodeMetadata.SeasonNumber),
		}
		for _, w := range wants {
			if !contains(args, w) {
				t.Errorf("missing metadata %q", w)
			}
		}
	}
}

func TestGetAudioTrackOutputPath(t *testing.T) {
	tests := []struct{ file, locale string; include bool; want string }{
		{"Show/Show S01E01 [1080p].mkv", "ja-JP", false, "Show/Show S01E01 [1080p].audio.m4a"},
		{"Show/Show S01E01 [1080p].mkv", "en-US", true, "Show/Show S01E01 [1080p].audio.en-US.m4a"},
		{"Show/Show S01E01 [1080p].mkv", "de-DE", true, "Show/Show S01E01 [1080p].audio.de-DE.m4a"},
	}
	for _, tt := range tests {
		got := getAudioTrackOutputPath(tt.file, tt.locale, tt.include)
		if got != tt.want {
			t.Errorf("getAudioTrackOutputPath(%q, %q, %v) = %q, want %q",
				tt.file, tt.locale, tt.include, got, tt.want)
		}
	}
}

func TestGetAudioManifestPath(t *testing.T) {
	tests := []struct{ file, locale string; include bool; want string }{
		{"Show/Show S01E01 [1080p].mkv", "ja-JP", false, "Show/Show S01E01 [1080p].xml"},
		{"Show/Show S01E01 [1080p].mkv", "en-US", true, "Show/Show S01E01 [1080p].audio.en-US.xml"},
	}
	for _, tt := range tests {
		got := getAudioManifestPath(tt.file, tt.locale, tt.include)
		if got != tt.want {
			t.Errorf("getAudioManifestPath(%q, %q, %v) = %q, want %q",
				tt.file, tt.locale, tt.include, got, tt.want)
		}
	}
}

func makeSubMap(locales ...string) map[string]*Subtitle {
	m := make(map[string]*Subtitle, len(locales))
	for _, loc := range locales {
		m[loc] = &Subtitle{Language: loc, URL: fmt.Sprintf("http://x.com/%s.ass", loc)}
	}
	return m
}

func tracksFromVariants(variants []episodeVariant, prefix string) []mediaTrack {
	tracks := make([]mediaTrack, len(variants))
	for i, v := range variants {
		tracks[i] = mediaTrack{Path: prefix + v.AudioLocale + ".m4a", Language: v.AudioLocale}
	}
	return tracks
}

func tracksFromLocales(locales []string, prefix string) []mediaTrack {
	tracks := make([]mediaTrack, len(locales))
	for i, loc := range locales {
		tracks[i] = mediaTrack{Path: prefix + loc + ".ass", Language: loc}
	}
	return tracks
}

func requireArg(t *testing.T, args []string, want string) {
	t.Helper()
	for _, a := range args {
		if a == want {
			return
		}
	}
	t.Errorf("missing arg %q", want)
}

func contains(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func sortCheck(ss []string) bool {
	for i := 1; i < len(ss); i++ {
		if ss[i] < ss[i-1] {
			return false
		}
	}
	return true
}

func variantsToLocales(variants []episodeVariant) []string {
	locales := make([]string, len(variants))
	for i, v := range variants {
		locales[i] = v.AudioLocale
	}
	return locales
}

func findArg(args []string, prefix string) []int {
	var indices []int
	for i, a := range args {
		if strings.HasPrefix(a, prefix) || a == prefix {
			indices = append(indices, i)
		}
	}
	return indices
}
