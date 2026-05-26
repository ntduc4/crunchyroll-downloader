package lib

import (
	"testing"

	"github.com/unki2aut/go-mpd"
)

func TestGetVideoSet(t *testing.T) {
	manifest := loadManifestFromFile("testdata/solo-leveling/ep1.xml")

	set := getVideoSet(manifest)
	if set == nil {
		t.Fatal("getVideoSet returned nil")
	}
	if set.MimeType != "video/mp4" {
		t.Errorf("getVideoSet MimeType = %q, want video/mp4", set.MimeType)
	}
	if len(set.Representations) == 0 {
		t.Error("getVideoSet has no representations")
	}
}

func TestGetVideoSet_NilOnEmpty(t *testing.T) {
	empty := &mpd.MPD{}
	if got := getVideoSet(empty); got != nil {
		t.Error("getVideoSet on empty MPD should return nil")
	}
}

func TestGetAudioSet(t *testing.T) {
	manifest := loadManifestFromFile("testdata/solo-leveling/ep1.xml")

	set := getAudioSet(manifest, "ja-JP")
	if set == nil {
		t.Fatal("getAudioSet returned nil for ja-JP")
	}
	if set.MimeType != "audio/mp4" {
		t.Errorf("getAudioSet MimeType = %q, want audio/mp4", set.MimeType)
	}
	if set.Lang == nil || *set.Lang != "ja-JP" {
		t.Errorf("getAudioSet Lang = %v, want ja-JP", set.Lang)
	}
}

func TestGetAudioSet_WrongLocaleFallsBack(t *testing.T) {
	manifest := loadManifestFromFile("testdata/solo-leveling/ep1.xml")

	set := getAudioSet(manifest, "fr-FR")
	if set == nil {
		t.Fatal("getAudioSet should return fallback when locale not found")
	}
	if set.MimeType != "audio/mp4" {
		t.Errorf("getAudioSet fallback MimeType = %q, want audio/mp4", set.MimeType)
	}
}

func TestGetAudioSet_NilOnEmpty(t *testing.T) {
	empty := &mpd.MPD{}
	if got := getAudioSet(empty, "ja-JP"); got != nil {
		t.Error("getAudioSet on empty MPD should return nil")
	}
}

func TestGetBaseUrl_Video(t *testing.T) {
	manifest := loadManifestFromFile("testdata/solo-leveling/ep1.xml")
	set := getVideoSet(manifest)

	base, repID := getBaseUrl(set, true, "1080p")
	if base == nil {
		t.Fatal("getBaseUrl for 1080p returned nil base")
	}
	if *repID == "" {
		t.Error("getBaseUrl returned empty representation ID")
	}

	base2, _ := getBaseUrl(set, true, "9999p")
	if base2 != nil {
		t.Error("getBaseUrl for invalid quality should return nil")
	}
}

func TestGetBaseUrl_Audio(t *testing.T) {
	manifest := loadManifestFromFile("testdata/solo-leveling/ep1.xml")
	set := getAudioSet(manifest, "ja-JP")

	for _, quality := range []string{"192k", "128k", "96k"} {
		base, repID := getBaseUrl(set, false, quality)
		if base == nil {
			t.Errorf("getBaseUrl for audio %q returned nil base", quality)
		}
		if repID == nil || *repID == "" {
			t.Errorf("getBaseUrl for audio %q returned empty rep ID", quality)
		}
	}

	base, _ := getBaseUrl(set, false, "999k")
	if base != nil {
		t.Error("getBaseUrl for invalid audio quality should return nil")
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
		{
			name:     "empty",
			timeline: nil,
			start:    1,
			want:     []int64{},
		},
		{
			name: "single no repeat",
			timeline: []*mpd.SegmentTimelineS{
				{D: 100},
			},
			start: 1,
			want:  []int64{1},
		},
		{
			name: "single with repeat=2",
			timeline: []*mpd.SegmentTimelineS{
				{D: 100, R: &r},
			},
			start: 1,
			want:  []int64{1, 2, 3},
		},
		{
			name: "repeat=0 treated as 1 segment",
			timeline: []*mpd.SegmentTimelineS{
				{D: 100, R: &zeroR},
			},
			start: 5,
			want:  []int64{5},
		},
		{
			name: "multiple entries",
			timeline: []*mpd.SegmentTimelineS{
				{D: 100, R: &r},
				{D: 200},
			},
			start: 1,
			want:  []int64{1, 2, 3, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandTimeline(tt.timeline, tt.start)
			if len(got) != len(tt.want) {
				t.Fatalf("expandTimeline length = %d, want %d; got=%v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("expandTimeline[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLoadManifestFromFile(t *testing.T) {
	m := loadManifestFromFile("testdata/solo-leveling/ep1.xml")
	if m == nil {
		t.Fatal("loadManifestFromFile returned nil")
	}
	if len(m.Period) == 0 {
		t.Fatal("manifest has no Periods")
	}
	if len(m.Period[0].AdaptationSets) < 2 {
		t.Fatal("manifest has fewer than 2 AdaptationSets")
	}
}

func TestGetPssh(t *testing.T) {
	manifest := loadManifestFromFile("testdata/solo-leveling/ep1.xml")
	pssh := getPssh(manifest)
	if pssh == nil {
		t.Fatal("getPssh returned nil")
	}
	if *pssh == "" {
		t.Fatal("getPssh returned empty string")
	}
}

func TestGetDefaultKID(t *testing.T) {
	manifest := loadManifestFromFile("testdata/solo-leveling/ep1.xml")
	kid := getDefaultKID(manifest)
	if kid == nil {
		t.Fatal("getDefaultKID returned nil")
	}
	if *kid == "" {
		t.Fatal("getDefaultKID returned empty string")
	}
}
