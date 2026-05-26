package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/unki2aut/go-mpd"
)

const ep1XML = "Solo Leveling/Solo Leveling S01E01 [1080p].xml"
const ep2XML = "Solo Leveling/Solo Leveling S01E02 [1080p].xml"

func epManifest(t *testing.T, path string) *mpd.MPD {
	t.Helper()
	m := loadManifestFromFile(path)
	if m == nil {
		t.Fatalf("failed to load manifest: %s", path)
	}
	return m
}

func TestManifestStructure_Ep1(t *testing.T) {
	m := epManifest(t, ep1XML)

	if len(m.Period) == 0 {
		t.Fatal("no periods in manifest")
	}

	videoSet := getVideoSet(m)
	audioSet := getAudioSet(m, "ja-JP")

	if videoSet == nil {
		t.Fatal("no video adaptation set")
	}
	if videoSet.MimeType != "video/mp4" {
		t.Errorf("video MimeType = %q, want video/mp4", videoSet.MimeType)
	}

	if audioSet == nil {
		t.Fatal("no audio adaptation set")
	}
	if audioSet.MimeType != "audio/mp4" {
		t.Errorf("audio MimeType = %q, want audio/mp4", audioSet.MimeType)
	}
	if audioSet.Lang == nil || *audioSet.Lang != "ja-JP" {
		t.Errorf("audio lang = %v, want ja-JP", audioSet.Lang)
	}

	pssh := getPssh(m)
	if pssh == nil || *pssh == "" {
		t.Fatal("pssh missing")
	}

	defaultKID := getDefaultKID(m)
	if defaultKID == nil || *defaultKID == "" {
		t.Fatal("default_KID missing")
	}
}

func TestManifestStructure_Ep2(t *testing.T) {
	m := epManifest(t, ep2XML)

	if len(m.Period) == 0 {
		t.Fatal("no periods in manifest")
	}

	videoSet := getVideoSet(m)
	audioSet := getAudioSet(m, "ja-JP")

	if videoSet == nil {
		t.Fatal("no video adaptation set")
	}
	if videoSet.MimeType != "video/mp4" {
		t.Errorf("video MimeType = %q, want video/mp4", videoSet.MimeType)
	}

	if audioSet == nil {
		t.Fatal("no audio adaptation set")
	}
	if audioSet.MimeType != "audio/mp4" {
		t.Errorf("audio MimeType = %q, want audio/mp4", audioSet.MimeType)
	}
	if audioSet.Lang == nil || *audioSet.Lang != "ja-JP" {
		t.Errorf("audio lang = %v, want ja-JP", audioSet.Lang)
	}

	pssh := getPssh(m)
	if pssh == nil || *pssh == "" {
		t.Fatal("pssh missing")
	}

	defaultKID := getDefaultKID(m)
	if defaultKID == nil || *defaultKID == "" {
		t.Fatal("default_KID missing")
	}
}

func TestManifest_KIDsDifferBetweenEpisodes(t *testing.T) {
	kid1 := getDefaultKID(epManifest(t, ep1XML))
	kid2 := getDefaultKID(epManifest(t, ep2XML))
	if *kid1 == *kid2 {
		t.Error("ep1 and ep2 default_KID should differ")
	}

	pssh1 := getPssh(epManifest(t, ep1XML))
	pssh2 := getPssh(epManifest(t, ep2XML))
	if *pssh1 == *pssh2 {
		t.Error("ep1 and ep2 PSSH should differ")
	}
}

func TestVideoQualities_Ep1(t *testing.T) {
	set := getVideoSet(epManifest(t, ep1XML))
	qualities := []string{"1080p", "720p", "480p", "360p", "240p"}

	for _, q := range qualities {
		base, repID := getBaseUrl(set, true, q)
		if base == nil {
			t.Errorf("getBaseUrl(ep1, video, %q) returned nil base", q)
		}
		if repID == nil || *repID == "" {
			t.Errorf("getBaseUrl(ep1, video, %q) returned empty rep ID", q)
		}
	}

	base, _ := getBaseUrl(set, true, "999p")
	if base != nil {
		t.Error("getBaseUrl(ep1, video, invalid) should return nil")
	}
}

func TestVideoQualities_Ep2(t *testing.T) {
	set := getVideoSet(epManifest(t, ep2XML))
	qualities := []string{"1080p", "720p", "480p", "360p", "240p"}

	for _, q := range qualities {
		base, repID := getBaseUrl(set, true, q)
		if base == nil {
			t.Errorf("getBaseUrl(ep2, video, %q) returned nil base", q)
		}
		if repID == nil || *repID == "" {
			t.Errorf("getBaseUrl(ep2, video, %q) returned empty rep ID", q)
		}
	}

	base, _ := getBaseUrl(set, true, "999p")
	if base != nil {
		t.Error("getBaseUrl(ep2, video, invalid) should return nil")
	}
}

func TestAudioQualities_Ep1(t *testing.T) {
	set := getAudioSet(epManifest(t, ep1XML), "ja-JP")
	qualities := []string{"192k", "128k", "96k"}

	for _, q := range qualities {
		base, repID := getBaseUrl(set, false, q)
		if base == nil {
			t.Errorf("getBaseUrl(ep1, audio, %q) returned nil base", q)
		}
		if repID == nil || *repID == "" {
			t.Errorf("getBaseUrl(ep1, audio, %q) returned empty rep ID", q)
		}
	}

	base, _ := getBaseUrl(set, false, "999k")
	if base != nil {
		t.Error("getBaseUrl(ep1, audio, invalid) should return nil")
	}
}

func TestAudioQualities_Ep2(t *testing.T) {
	set := getAudioSet(epManifest(t, ep2XML), "ja-JP")
	qualities := []string{"192k", "128k", "96k"}

	for _, q := range qualities {
		base, repID := getBaseUrl(set, false, q)
		if base == nil {
			t.Errorf("getBaseUrl(ep2, audio, %q) returned nil base", q)
		}
		if repID == nil || *repID == "" {
			t.Errorf("getBaseUrl(ep2, audio, %q) returned empty rep ID", q)
		}
	}

	base, _ := getBaseUrl(set, false, "999k")
	if base != nil {
		t.Error("getBaseUrl(ep2, audio, invalid) should return nil")
	}
}

func episodeInfoWithDubs(audioLocale string, versions []*DubVersion) EpisodeInfo {
	return EpisodeInfo{
		Title: "Test Episode",
		EpisodeMetadata: EpisodeMetadata{
			SeriesTitle:   "Test Series",
			SeasonNumber:  1,
			EpisodeNumber: 1,
			AudioLocale:   audioLocale,
			Versions:      versions,
		},
	}
}

func subtitlesWith(t *testing.T, locales []string) map[string]*Subtitle {
	t.Helper()
	m := make(map[string]*Subtitle, len(locales))
	for _, loc := range locales {
		m[loc] = &Subtitle{Language: loc, URL: fmt.Sprintf("http://example.com/%s.ass", loc)}
	}
	return m
}

func TestAllTracksPipeline_Ep1_5Dubs(t *testing.T) {
	info := episodeInfoWithDubs("ja-JP", []*DubVersion{
		{AudioLocale: "en-US", GUID: "EP1-EN-GUID"},
		{AudioLocale: "de-DE", GUID: "EP1-DE-GUID"},
		{AudioLocale: "fr-FR", GUID: "EP1-FR-GUID"},
		{AudioLocale: "pt-BR", GUID: "EP1-PT-GUID"},
	})

	variants := resolveAudioVariants("EP1-JA-GUID", info, "all")
	if len(variants) != 5 {
		t.Fatalf("expected 5 audio variants, got %d: %v", len(variants), variants)
	}

	if variants[0].AudioLocale != "ja-JP" || variants[0].ContentID != "EP1-JA-GUID" {
		t.Errorf("first variant = %+v, want ja-JP/EP1-JA-GUID", variants[0])
	}

	locales := make([]string, len(variants))
	for i, v := range variants {
		locales[i] = v.AudioLocale
	}
	if !sortCheck(locales[1:]) {
		t.Errorf("additional variants not sorted: %v", locales[1:])
	}

	subs := subtitlesWith(t, []string{"en-US", "ja-JP", "pt-BR", "de-DE", "fr-FR", "ar-SA", "es-419", "it-IT", "ru-RU"})
	subLanguages := resolveSubtitleLanguages(subs, "all")
	if len(subLanguages) < 5 {
		t.Errorf("expected at least 5 subtitle languages, got %d: %v", len(subLanguages), subLanguages)
	}
	if !sortCheck(subLanguages) {
		t.Errorf("subtitle languages not sorted: %v", subLanguages)
	}

	audioTracks := make([]mediaTrack, len(variants))
	for i, v := range variants {
		audioTracks[i] = mediaTrack{Path: fmt.Sprintf("/tmp/ep1_audio_%s.m4a", v.AudioLocale), Language: v.AudioLocale}
	}
	subtitleTracks := make([]mediaTrack, len(subLanguages))
	for i, loc := range subLanguages {
		subtitleTracks[i] = mediaTrack{Path: fmt.Sprintf("/tmp/ep1_subs_%s.ass", loc), Language: loc}
	}

	args := buildMuxArgs("/tmp/ep1_video.mp4", audioTracks, subtitleTracks, "/tmp/ep1_output.mkv", info)

	expectedSubCount := len(subLanguages)
	for i := 0; i < len(variants); i++ {
		mapArg := fmt.Sprintf("%d:a:0", i+1)
		found := false
		for _, a := range args {
			if a == mapArg {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing audio map %q", mapArg)
		}
	}

	subStartIdx := len(variants) + 1
	for i := 0; i < expectedSubCount; i++ {
		mapArg := fmt.Sprintf("%d:0", subStartIdx+i)
		found := false
		for _, a := range args {
			if a == mapArg {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing subtitle map %q", mapArg)
		}
	}
}

func TestAllTracksPipeline_Ep2_3Dubs(t *testing.T) {
	info := episodeInfoWithDubs("ja-JP", []*DubVersion{
		{AudioLocale: "en-US", GUID: "EP2-EN-GUID"},
		{AudioLocale: "es-419", GUID: "EP2-ES-GUID"},
	})

	variants := resolveAudioVariants("EP2-JA-GUID", info, "all")
	if len(variants) != 3 {
		t.Fatalf("expected 3 audio variants, got %d: %v", len(variants), variants)
	}

	if variants[0].AudioLocale != "ja-JP" || variants[0].ContentID != "EP2-JA-GUID" {
		t.Errorf("first variant = %+v, want ja-JP/EP2-JA-GUID", variants[0])
	}

	locales := make([]string, len(variants))
	for i, v := range variants {
		locales[i] = v.AudioLocale
	}
	if !sortCheck(locales[1:]) {
		t.Errorf("additional variants not sorted: %v", locales[1:])
	}

	subs := subtitlesWith(t, []string{"en-US", "es-419"})
	subLanguages := resolveSubtitleLanguages(subs, "all")
	if len(subLanguages) != 2 {
		t.Errorf("expected 2 subtitle languages, got %d: %v", len(subLanguages), subLanguages)
	}

	audioTracks := make([]mediaTrack, len(variants))
	for i, v := range variants {
		audioTracks[i] = mediaTrack{Path: fmt.Sprintf("/tmp/ep2_audio_%s.m4a", v.AudioLocale), Language: v.AudioLocale}
	}
	subtitleTracks := make([]mediaTrack, len(subLanguages))
	for i, loc := range subLanguages {
		subtitleTracks[i] = mediaTrack{Path: fmt.Sprintf("/tmp/ep2_subs_%s.ass", loc), Language: loc}
	}

	args := buildMuxArgs("/tmp/ep2_video.mp4", audioTracks, subtitleTracks, "/tmp/ep2_output.mkv", info)

	for i := 0; i < len(variants); i++ {
		mapArg := fmt.Sprintf("%d:a:0", i+1)
		found := false
		for _, a := range args {
			if a == mapArg {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing audio map %q", mapArg)
		}
	}

	subStartIdx := len(variants) + 1
	for i := 0; i < len(subLanguages); i++ {
		mapArg := fmt.Sprintf("%d:0", subStartIdx+i)
		found := false
		for _, a := range args {
			if a == mapArg {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing subtitle map %q", mapArg)
		}
	}
}

func TestSingleAudioTrack_Ep1(t *testing.T) {
	info := episodeInfoWithDubs("ja-JP", []*DubVersion{
		{AudioLocale: "en-US", GUID: "EP1-EN-GUID"},
		{AudioLocale: "de-DE", GUID: "EP1-DE-GUID"},
	})

	variants := resolveAudioVariants("EP1-JA-GUID", info, "ja-JP")
	if len(variants) != 1 {
		t.Fatalf("expected 1 audio variant, got %d", len(variants))
	}
	if variants[0].ContentID != "EP1-JA-GUID" || variants[0].AudioLocale != "ja-JP" {
		t.Errorf("wrong variant: %+v", variants[0])
	}

	variants = resolveAudioVariants("EP1-JA-GUID", info, "en-US")
	if len(variants) != 1 {
		t.Fatalf("expected 1 audio variant, got %d", len(variants))
	}
	if variants[0].ContentID != "EP1-EN-GUID" || variants[0].AudioLocale != "en-US" {
		t.Errorf("wrong variant: %+v", variants[0])
	}
}

func TestSingleSubtitleTrack_Ep1(t *testing.T) {
	subs := subtitlesWith(t, []string{"en-US", "ja-JP", "de-DE", "fr-FR"})

	got := resolveSubtitleLanguages(subs, "ja-JP")
	if !reflect.DeepEqual(got, []string{"ja-JP"}) {
		t.Errorf("expected [ja-JP], got %v", got)
	}

	got = resolveSubtitleLanguages(subs, "en-US")
	if !reflect.DeepEqual(got, []string{"en-US"}) {
		t.Errorf("expected [en-US], got %v", got)
	}

	got = resolveSubtitleLanguages(subs, "xx-XX")
	if got != nil {
		t.Errorf("expected nil for missing locale, got %v", got)
	}
}

func TestGetAudioTrackOutputPath(t *testing.T) {
	tests := []struct {
		file, locale string
		include      bool
		want         string
	}{
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
	tests := []struct {
		file, locale string
		include      bool
		want         string
	}{
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

func sortCheck(ss []string) bool {
	for i := 1; i < len(ss); i++ {
		if ss[i] < ss[i-1] {
			return false
		}
	}
	return true
}

func TestAudioSet_FallbackInBothEpisodes(t *testing.T) {
	for _, path := range []string{ep1XML, ep2XML} {
		m := epManifest(t, path)

		set := getAudioSet(m, "xx-XX")
		if set == nil {
			t.Errorf("%s: getAudioSet with unknown locale returned nil", path)
			continue
		}
		if set.MimeType != "audio/mp4" {
			t.Errorf("%s: fallback MimeType = %q", path, set.MimeType)
		}

		knownSet := getAudioSet(m, "ja-JP")
		if knownSet == nil {
			t.Errorf("%s: getAudioSet with known locale returned nil", path)
		}
	}
}

func TestExpandTimeline_Ep1VsEp2(t *testing.T) {
	for _, path := range []string{ep1XML, ep2XML} {
		m := epManifest(t, path)
		videoSet := getVideoSet(m)
		if videoSet == nil || videoSet.SegmentTemplate == nil || videoSet.SegmentTemplate.SegmentTimeline == nil {
			t.Fatalf("%s: video set has no segment timeline", path)
		}

		timeline := expandTimeline(videoSet.SegmentTemplate.SegmentTimeline.S, 1)
		if len(timeline) == 0 {
			t.Errorf("%s: expandTimeline returned empty", path)
		}

		if timeline[0] != 1 {
			t.Errorf("%s: first segment should be 1, got %d", path, timeline[0])
		}

		for i := 1; i < len(timeline); i++ {
			if timeline[i] <= timeline[i-1] {
				t.Errorf("%s: segments not strictly increasing at position %d: %d -> %d",
					path, i, timeline[i-1], timeline[i])
			}
		}
	}
}

func TestMuxArgs_HasCorrectMetadataForBothEpisodes(t *testing.T) {
	ep1Info := EpisodeInfo{
		Title: "If I Had One More Chance",
		EpisodeMetadata: EpisodeMetadata{
			SeasonNumber:  1,
			EpisodeNumber: 2,
			SeriesTitle:   "Solo Leveling",
		},
	}
	ep2Info := EpisodeInfo{
		Title: "The Tenth S-Rank Hunter",
		EpisodeMetadata: EpisodeMetadata{
			SeasonNumber:  1,
			EpisodeNumber: 1,
			SeriesTitle:   "Solo Leveling",
		},
	}

	audio := []mediaTrack{{Path: "/tmp/a.m4a", Language: "ja-JP"}}

	for _, tc := range []struct {
		name string
		info EpisodeInfo
	}{
		{"ep1-like", ep1Info},
		{"ep2-like", ep2Info},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := buildMuxArgs("/tmp/v.mp4", audio, nil, "/tmp/o.mkv", tc.info)

			wantTitle := fmt.Sprintf("title=S%02dE%02d - %s",
				tc.info.EpisodeMetadata.SeasonNumber,
				tc.info.EpisodeMetadata.EpisodeNumber,
				tc.info.Title)
			wantShow := "show=" + tc.info.EpisodeMetadata.SeriesTitle
			wantTrack := fmt.Sprintf("track=%d", tc.info.EpisodeMetadata.EpisodeNumber)
			wantSeason := fmt.Sprintf("season_number=%d", tc.info.EpisodeMetadata.SeasonNumber)

			for _, want := range []string{wantTitle, wantShow, wantTrack, wantSeason} {
				found := false
				for _, a := range args {
					if a == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("missing metadata arg %q", want)
				}
			}
		})
	}
}

func TestDownloadArgsOrder(t *testing.T) {
	info := episodeInfoWithDubs("ja-JP", []*DubVersion{
		{AudioLocale: "en-US", GUID: "G-EN"},
	})

	subs := subtitlesWith(t, []string{"en-US", "pt-BR", "fr-FR"})

	variants := resolveAudioVariants("G-JA", info, "all")
	audioTracks := make([]mediaTrack, len(variants))
	for i, v := range variants {
		audioTracks[i] = mediaTrack{Path: "/tmp/a_" + v.AudioLocale + ".m4a", Language: v.AudioLocale}
	}

	subLanguages := resolveSubtitleLanguages(subs, "all")
	subtitleTracks := make([]mediaTrack, len(subLanguages))
	for i, loc := range subLanguages {
		subtitleTracks[i] = mediaTrack{Path: "/tmp/s_" + loc + ".ass", Language: loc}
	}

	args := buildMuxArgs("/tmp/v.mp4", audioTracks, subtitleTracks, "/tmp/o.mkv", info)

	iIdx := findArg(args, "-i")
	if len(iIdx) != 1+len(audioTracks)+len(subtitleTracks) {
		t.Errorf("expected %d -i args, got %d", 1+len(audioTracks)+len(subtitleTracks), len(iIdx))
	}

	mapIdx := findArg(args, "-map")
	expectedMaps := 1 + len(audioTracks) + len(subtitleTracks)
	if len(mapIdx) != expectedMaps {
		t.Errorf("expected %d -map args, got %d", expectedMaps, len(mapIdx))
	}

	copyIdx := findArg(args, "-c:v")
	if len(copyIdx) == 0 {
		t.Error("-c:v missing")
	}

	metaIdx := findArg(args, "-metadata:g")
	if len(metaIdx) < 4 {
		t.Errorf("expected at least 4 global metadata args, got %d", len(metaIdx))
	}
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
