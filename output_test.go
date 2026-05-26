package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildMuxArgs_SingleAudio_SingleSub(t *testing.T) {
	info := EpisodeInfo{
		Title: "Episode Title",
		EpisodeMetadata: EpisodeMetadata{
			SeasonNumber:  1,
			EpisodeNumber: 3,
			SeriesTitle:   "Test Show",
		},
	}

	audio := []mediaTrack{{Path: "/tmp/audio.m4a", Language: "ja-JP"}}
	subs := []mediaTrack{{Path: "/tmp/subs.ass", Language: "en-US"}}

	args := buildMuxArgs("/tmp/video.mp4", audio, subs, "/tmp/output.mkv", info)

	assertArg := func(name, want string) {
		found := false
		for _, a := range args {
			if a == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: missing arg %q in %v", name, want, args)
		}
	}

	assertArg("video input", "-i")
	assertArg("video file", "/tmp/video.mp4")
	assertArg("audio file", "/tmp/audio.m4a")
	assertArg("sub file", "/tmp/subs.ass")
	assertArg("map video", "0:v:0")
	assertArg("map audio", "1:a:0")
	assertArg("map subs", "2:0")
	assertArg("output", "/tmp/output.mkv")
	assertArg("c:v copy", "-c:v")
	assertArg("c:a copy", "-c:a")
	assertArg("c:s copy", "-c:s")
	assertArg("disposition default", "-disposition:a:0")

	if args[len(args)-1] != "/tmp/output.mkv" {
		t.Errorf("last arg should be output, got %q", args[len(args)-1])
	}
}

func TestBuildMuxArgs_MultiAudio_MultiSub(t *testing.T) {
	info := EpisodeInfo{
		Title: "Episode Title",
		EpisodeMetadata: EpisodeMetadata{
			SeasonNumber:  2,
			EpisodeNumber: 5,
			SeriesTitle:   "Test Show",
		},
	}

	audio := []mediaTrack{
		{Path: "/tmp/audio_ja.m4a", Language: "ja-JP"},
		{Path: "/tmp/audio_en.m4a", Language: "en-US"},
		{Path: "/tmp/audio_de.m4a", Language: "de-DE"},
	}
	subs := []mediaTrack{
		{Path: "/tmp/subs_en.ass", Language: "en-US"},
		{Path: "/tmp/subs_pt.ass", Language: "pt-BR"},
	}

	args := buildMuxArgs("/tmp/video.mp4", audio, subs, "/tmp/output.mkv", info)

	assertContains := func(name, substr string) {
		found := false
		for _, a := range args {
			if strings.Contains(a, substr) || a == substr {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: missing %q in %v", name, substr, args)
		}
	}

	assertContains("video input", "/tmp/video.mp4")
	assertContains("audio 1 input", "/tmp/audio_ja.m4a")
	assertContains("audio 2 input", "/tmp/audio_en.m4a")
	assertContains("audio 3 input", "/tmp/audio_de.m4a")
	assertContains("sub 1 input", "/tmp/subs_en.ass")
	assertContains("sub 2 input", "/tmp/subs_pt.ass")
	assertContains("video map", "0:v:0")
	assertContains("audio 1 map", "1:a:0")
	assertContains("audio 2 map", "2:a:0")
	assertContains("audio 3 map", "3:a:0")
	assertContains("sub 1 map", "4:0")
	assertContains("sub 2 map", "5:0")
	assertContains("disposition default", "-disposition:a:0")

	assertContains("audio 1 title", "title=ja-JP")
	assertContains("audio 2 title", "title=English")
	assertContains("audio 3 title", "title=Deutsch")
	assertContains("sub 1 title", "title=English")
	assertContains("sub 2 title", "title=Português (Brasil)")
}

func TestBuildMuxArgs_NoAudioStillProducesValidArgs(t *testing.T) {
	info := EpisodeInfo{
		Title: "Ep",
		EpisodeMetadata: EpisodeMetadata{
			SeasonNumber:  1,
			EpisodeNumber: 1,
			SeriesTitle:   "S",
		},
	}

	args := buildMuxArgs("/tmp/v.mp4", nil, nil, "/tmp/o.mkv", info)
	if args == nil {
		t.Fatal("args should not be nil")
	}

	assertContains := func(substr string) {
		for _, a := range args {
			if strings.Contains(a, substr) || a == substr {
				return
			}
		}
		t.Errorf("missing %q in %v", substr, args)
	}

	assertContains("0:v:0")
	assertContains("/tmp/o.mkv")
}

func TestBuildMuxArgs_MetadataFields(t *testing.T) {
	info := EpisodeInfo{
		Title: "The Episode",
		EpisodeMetadata: EpisodeMetadata{
			SeasonNumber:  3,
			EpisodeNumber: 7,
			SeriesTitle:   "Great Show",
		},
	}

	audio := []mediaTrack{{Path: "/tmp/a.m4a", Language: "ja-JP"}}
	subs := []mediaTrack{{Path: "/tmp/s.ass", Language: "en-US"}}

	args := buildMuxArgs("/tmp/v.mp4", audio, subs, "/tmp/o.mkv", info)

	assertContains := func(name, want string) {
		found := false
		for _, a := range args {
			if a == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: missing %q", name, want)
		}
	}

	assertContains("title metadata", "title=S03E07 - The Episode")
	assertContains("show metadata", "show=Great Show")
	assertContains("track metadata", "track=7")
	assertContains("season metadata", "season_number=3")
}

func TestResolveAudioVariants_DedupContent(t *testing.T) {
	// When primary AudioLocale also appears in Versions, the primary
	// contentID should win, not overwritten by the version GUID.
	info := makeInfo("ja-JP", []*DubVersion{
		{AudioLocale: "ja-JP", GUID: "DUPLICATE-JA"}, // same locale as primary
		{AudioLocale: "en-US", GUID: "GUID-EN"},
	})

	got := resolveAudioVariants("ORIG-ID", info, "all")

	// Primary comes first with original contentID
	if got[0].ContentID != "ORIG-ID" || got[0].AudioLocale != "ja-JP" {
		t.Errorf("first variant = %v, want {ORIG-ID ja-JP}", got[0])
	}

	// en-US should still be present
	foundEN := false
	for _, v := range got {
		if v.AudioLocale == "en-US" && v.ContentID == "GUID-EN" {
			foundEN = true
		}
	}
	if !foundEN {
		t.Error("en-US variant missing or wrong contentID")
	}

	// No duplicate ja-JP from versions
	jaCount := 0
	for _, v := range got {
		if v.AudioLocale == "ja-JP" {
			jaCount++
		}
	}
	if jaCount != 1 {
		t.Errorf("expected 1 ja-JP variant, got %d", jaCount)
	}
}

func TestSortedLanguageKeys_Deterministic(t *testing.T) {
	m := map[string]int{
		"z": 1, "a": 2, "m": 3, "b": 4, "c": 5,
	}
	got1 := sortedLanguageKeys(m)
	got2 := sortedLanguageKeys(m)
	if !reflect.DeepEqual(got1, got2) {
		t.Errorf("sortedLanguageKeys not deterministic: %v vs %v", got1, got2)
	}

	for i := 1; i < len(got1); i++ {
		if got1[i] <= got1[i-1] {
			t.Errorf("sortedLanguageKeys not sorted: %v", got1)
			break
		}
	}
}
