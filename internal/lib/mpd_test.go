package lib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/unki2aut/go-mpd"
)

func discoverTestData(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir("../../testdata")
	if err != nil {
		t.Skipf("no testdata/ directory — run setup script first")
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join("../../testdata", e.Name(), "manifest.xml")
		if _, err := os.Stat(p); err == nil {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 0 {
		t.Skipf("no testdata subdirs with manifest.xml — run setup script first")
	}
	sort.Strings(dirs)
	return dirs
}

func loadMPD(t *testing.T, path string) *mpd.MPD {
	t.Helper()
	m := loadManifestFromFile(path)
	if m == nil {
		t.Fatalf("failed to load manifest: %s", path)
	}
	return m
}

func TestGetVideoSet(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			set := getVideoSet(loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml")))
			if set == nil {
				t.Fatal("getVideoSet returned nil")
			}
			if set.MimeType != "video/mp4" {
				t.Errorf("MimeType = %q, want video/mp4", set.MimeType)
			}
			if len(set.Representations) == 0 {
				t.Error("no representations")
			}
		})
	}
}

func TestGetVideoSet_NilOnEmpty(t *testing.T) {
	if got := getVideoSet(&mpd.MPD{}); got != nil {
		t.Error("getVideoSet on empty MPD should return nil")
	}
}

func TestGetAudioSet(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			set := getAudioSet(loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml")), "ja-JP")
			if set == nil {
				t.Fatal("getAudioSet returned nil")
			}
			if set.MimeType != "audio/mp4" {
				t.Errorf("MimeType = %q, want audio/mp4", set.MimeType)
			}
			if set.Lang == nil || *set.Lang != "ja-JP" {
				t.Errorf("Lang = %v, want ja-JP", set.Lang)
			}
		})
	}
}

func TestGetAudioSet_WrongLocaleFallsBack(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			set := getAudioSet(loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml")), "fr-FR")
			if set == nil {
				t.Fatal("getAudioSet should return fallback when locale not found")
			}
			if set.MimeType != "audio/mp4" {
				t.Errorf("fallback MimeType = %q", set.MimeType)
			}
		})
	}
}

func TestGetAudioSet_NilOnEmpty(t *testing.T) {
	if got := getAudioSet(&mpd.MPD{}, "ja-JP"); got != nil {
		t.Error("getAudioSet on empty MPD should return nil")
	}
}

func TestGetBaseUrl_Video(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			set := getVideoSet(loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml")))
			base, repID := getBaseUrl(set, true, "1080p")
			if base == nil {
				t.Fatal("getBaseUrl for 1080p returned nil")
			}
			if *repID == "" {
				t.Error("empty representation ID")
			}
			if b, _ := getBaseUrl(set, true, "9999p"); b != nil {
				t.Error("invalid quality should return nil")
			}
		})
	}
}

func TestGetBaseUrl_Audio(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			set := getAudioSet(loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml")), "ja-JP")
			for _, q := range []string{"192k", "128k", "96k"} {
				base, repID := getBaseUrl(set, false, q)
				if base == nil {
					t.Errorf("audio %q returned nil base", q)
				}
				if repID == nil || *repID == "" {
					t.Errorf("audio %q returned empty rep ID", q)
				}
			}
			if b, _ := getBaseUrl(set, false, "999k"); b != nil {
				t.Error("invalid audio quality should return nil")
			}
		})
	}
}

func TestExpandTimeline(t *testing.T) {
	r := int64(2)
	zeroR := int64(0)
	tests := []struct {
		name     string
		timeline []*mpd.SegmentTimelineS
		start    int64
		want     []int64
	}{
		{"empty", nil, 1, []int64{}},
		{"single no repeat", []*mpd.SegmentTimelineS{{D: 100}}, 1, []int64{1}},
		{"repeat=2", []*mpd.SegmentTimelineS{{D: 100, R: &r}}, 1, []int64{1, 2, 3}},
		{"repeat=0", []*mpd.SegmentTimelineS{{D: 100, R: &zeroR}}, 5, []int64{5}},
		{"multiple", []*mpd.SegmentTimelineS{{D: 100, R: &r}, {D: 200}}, 1, []int64{1, 2, 3, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandTimeline(tt.timeline, tt.start)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d; got=%v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLoadManifestFromFile(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			m := loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml"))
			if len(m.Period) == 0 {
				t.Fatal("no Periods")
			}
			if len(m.Period[0].AdaptationSets) < 2 {
				t.Fatal("fewer than 2 AdaptationSets")
			}
		})
	}
}

func TestGetPssh(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			pssh := getPssh(loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml")))
			if pssh == nil {
				t.Fatal("getPssh returned nil")
			}
			if *pssh == "" {
				t.Fatal("getPssh returned empty")
			}
		})
	}
}

func TestGetDefaultKID(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			kid := getDefaultKID(loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml")))
			if kid == nil {
				t.Fatal("getDefaultKID returned nil")
			}
			if *kid == "" {
				t.Fatal("getDefaultKID returned empty")
			}
		})
	}
}

func TestManifestStructure(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			m := loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml"))
			if len(m.Period) == 0 {
				t.Fatal("no periods")
			}
			vs := getVideoSet(m)
			if vs == nil || vs.MimeType != "video/mp4" {
				t.Fatal("missing video adaptation set")
			}
			as := getAudioSet(m, "ja-JP")
			if as == nil || as.MimeType != "audio/mp4" {
				t.Fatal("missing audio adaptation set")
			}
			if as.Lang == nil || *as.Lang != "ja-JP" {
				t.Errorf("audio lang = %v, want ja-JP", as.Lang)
			}
			if p := getPssh(m); p == nil || *p == "" {
				t.Fatal("pssh missing")
			}
			if k := getDefaultKID(m); k == nil || *k == "" {
				t.Fatal("default_KID missing")
			}
		})
	}
}

func TestKIDsDifferAcrossEpisodes(t *testing.T) {
	dirs := discoverTestData(t)
	if len(dirs) < 2 {
		t.Skip("need at least 2 testdata dirs to compare")
	}
	for i := 0; i < len(dirs); i++ {
		for j := i + 1; j < len(dirs); j++ {
			t.Run(dirs[i]+"-vs-"+dirs[j], func(t *testing.T) {
				a := filepath.Join("../../testdata", dirs[i], "manifest.xml")
				b := filepath.Join("../../testdata", dirs[j], "manifest.xml")
				if *getDefaultKID(loadMPD(t, a)) == *getDefaultKID(loadMPD(t, b)) {
					t.Error("KIDs are identical")
				}
				if *getPssh(loadMPD(t, a)) == *getPssh(loadMPD(t, b)) {
					t.Error("PSSH values are identical")
				}
			})
		}
	}
}

func TestVideoQualities(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			set := getVideoSet(loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml")))
			for _, q := range []string{"1080p", "720p", "480p", "360p", "240p"} {
				base, repID := getBaseUrl(set, true, q)
				if base == nil {
					t.Errorf("video %q returned nil base", q)
				}
				if repID == nil || *repID == "" {
					t.Errorf("video %q returned empty rep ID", q)
				}
			}
			if b, _ := getBaseUrl(set, true, "999p"); b != nil {
				t.Error("invalid quality should return nil")
			}
		})
	}
}

func TestAudioQualities(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			set := getAudioSet(loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml")), "ja-JP")
			for _, q := range []string{"192k", "128k", "96k"} {
				base, repID := getBaseUrl(set, false, q)
				if base == nil {
					t.Errorf("audio %q returned nil base", q)
				}
				if repID == nil || *repID == "" {
					t.Errorf("audio %q returned empty rep ID", q)
				}
			}
			if b, _ := getBaseUrl(set, false, "999k"); b != nil {
				t.Error("invalid audio quality should return nil")
			}
		})
	}
}

func TestAudioSet_Fallback(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			m := loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml"))
			if s := getAudioSet(m, "xx-XX"); s == nil {
				t.Error("fallback returned nil")
			} else if s.MimeType != "audio/mp4" {
				t.Errorf("fallback MimeType = %q", s.MimeType)
			}
			if getAudioSet(m, "ja-JP") == nil {
				t.Error("known locale returned nil")
			}
		})
	}
}

func TestExpandTimeline_All(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			m := loadMPD(t, filepath.Join("../../testdata", d, "manifest.xml"))
			vs := getVideoSet(m)
			if vs == nil || vs.SegmentTemplate == nil || vs.SegmentTemplate.SegmentTimeline == nil {
				t.Fatal("video set has no segment timeline")
			}
			tl := expandTimeline(vs.SegmentTemplate.SegmentTimeline.S, 1)
			if len(tl) == 0 {
				t.Error("expandTimeline returned empty")
			}
			if tl[0] != 1 {
				t.Errorf("first segment = %d, want 1", tl[0])
			}
			for i := 1; i < len(tl); i++ {
				if tl[i] <= tl[i-1] {
					t.Errorf("not increasing at %d: %d -> %d", i, tl[i-1], tl[i])
				}
			}
		})
	}
}

func TestIntegration_All(t *testing.T) {
	for _, d := range discoverTestData(t) {
		t.Run(d, func(t *testing.T) {
			dir := filepath.Join("../../testdata", d)
			info := loadEpisodeInfo(t, dir)
			ep := loadPlaybackJSON(t, dir)
			manifest := loadMPD(t, filepath.Join(dir, "manifest.xml"))

			if getVideoSet(manifest) == nil {
				t.Fatal("manifest has no video adaptation set")
			}
			if getDefaultKID(manifest) == nil || *getDefaultKID(manifest) == "" {
				t.Fatal("default_KID missing")
			}
			if getPssh(manifest) == nil || *getPssh(manifest) == "" {
				t.Fatal("pssh missing")
			}

			variants := resolveAudioVariants("", info, "all")
			if len(variants) == 0 {
				t.Fatal("no audio variants resolved")
			}

			for _, v := range variants {
				p := filepath.Join(dir, "audio", v.AudioLocale+".m4a")
				if _, err := os.Stat(p); os.IsNotExist(err) {
					t.Errorf("missing audio: %s", p)
				}
			}
			for _, v := range variants[1:] {
				p := filepath.Join(dir, "manifests", v.AudioLocale+".xml")
				if _, err := os.Stat(p); os.IsNotExist(err) {
					t.Errorf("missing variant manifest: %s", p)
				}
			}

			subLanguages := resolveSubtitleLanguages(ep.Subtitles, "all")
			if len(subLanguages) == 0 {
				t.Error("no subtitle languages resolved")
			}
			gotSubs := map[string]bool{}
			if entries, _ := os.ReadDir(filepath.Join(dir, "subtitles")); entries != nil {
				for _, e := range entries {
					if !e.IsDir() {
						gotSubs[e.Name()] = true
					}
				}
			}
			for _, loc := range subLanguages {
				if !gotSubs[loc+".ass"] {
					t.Errorf("missing subtitle: %s.ass", loc)
				}
			}

			if _, err := os.Stat(filepath.Join(dir, "video.mp4")); os.IsNotExist(err) {
				t.Error("video.mp4 missing")
			}
		})
	}
}

func loadEpisodeInfo(t *testing.T, dir string) EpisodeInfo {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "episode_info.json"))
	if err != nil {
		t.Fatalf("failed to read episode_info.json: %v", err)
	}
	var info EpisodeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("failed to parse episode_info.json: %v", err)
	}
	if info.Title == "" {
		t.Fatal("episode_info.json has empty title")
	}
	if info.EpisodeMetadata.SeriesTitle == "" {
		t.Fatal("episode_info.json has empty series title")
	}
	if info.EpisodeMetadata.AudioLocale == "" {
		t.Fatal("episode_info.json has empty audio locale")
	}
	return info
}

func loadPlaybackJSON(t *testing.T, dir string) Episode {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "playback.json"))
	if err != nil {
		t.Fatalf("failed to read playback.json: %v", err)
	}
	var ep Episode
	if err := json.Unmarshal(data, &ep); err != nil {
		t.Fatalf("failed to parse playback.json: %v", err)
	}
	if ep.ManifestURL == "" {
		t.Fatal("playback.json has empty manifest URL")
	}
	if ep.Token == "" {
		t.Fatal("playback.json has empty token")
	}
	return ep
}
