package lib

import (
	"os"
	"strings"
	"testing"
)

func TestProcessASS_ScaledBorderAndShadow(t *testing.T) {
	input := []byte("[Script Info]\nTitle: test\n[Events]\nDialogue: 0,0:00:01.00,0:00:05.00,Default,,0,0,0,,hello\n")
	output := ProcessASS(input, 0, true, false, false, false, false)
	lines := strings.Split(string(output), "\n")
	found := false
	for _, l := range lines {
		if strings.TrimSpace(l) == "ScaledBorderAndShadow: yes" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ScaledBorderAndShadow: yes not found in output")
	}
}

func TestProcessASS_ScaledBorderAndShadow_ReplacesExisting(t *testing.T) {
	input := []byte("[Script Info]\nScaledBorderAndShadow: no\n[Events]\n")
	output := ProcessASS(input, 0, true, false, false, false, false)
	if !strings.Contains(string(output), "ScaledBorderAndShadow: yes") {
		t.Error("should replace existing value")
	}
	if strings.Contains(string(output), "ScaledBorderAndShadow: no") {
		t.Error("should not keep old value")
	}
}

func TestProcessASS_RemoveOriginalScript(t *testing.T) {
	input := []byte("[Script Info]\nOriginalScript: 1920x1080\n[Events]\n")
	output := ProcessASS(input, 0, false, false, true, false, false)
	if !strings.Contains(string(output), "; OriginalScript:") {
		t.Errorf("expected OriginalScript to be commented out, got: %s", output)
	}
}

func TestProcessASS_SrtAssFix_RemovesZeroDuration(t *testing.T) {
	input := []byte("[Events]\nDialogue: 0,0:00:01.00,0:00:05.00,Default,,0,0,0,,normal\nDialogue: 0,0:00:06.00,0:00:06.00,Default,,0,0,0,,zero-duration\n")
	output := ProcessASS(input, 0, false, true, false, false, false)
	if strings.Contains(string(output), "zero-duration") {
		t.Error("zero-duration event should be removed")
	}
	if !strings.Contains(string(output), "normal") {
		t.Error("normal event should remain")
	}
}

func TestProcessASS_TimestampFix(t *testing.T) {
	input := []byte("[Events]\nDialogue: 0,0:00:05.00,0:00:10.00,Default,,0,0,0,,hello\n")
	output := ProcessASS(input, 0, false, false, false, false, true)
	if !strings.Contains(string(output), "0:00:00.00") {
		t.Errorf("expected first event to start at 0, got: %s", output)
	}
	if !strings.Contains(string(output), "0:00:05.00") {
		t.Errorf("expected end to be 5s after shift, got: %s", output)
	}
}

func TestProcessASS_TimestampFix_NearZero(t *testing.T) {
	input := []byte("[Events]\nDialogue: 0,0:00:00.50,0:00:05.00,Default,,0,0,0,,hello\n")
	output := ProcessASS(input, 0, false, false, false, false, true)
	if !strings.Contains(string(output), "0:00:00.50") {
		t.Error("should not shift timestamps when first event starts near 0")
	}
}

func TestProcessASS_LayoutResFix(t *testing.T) {
	input := []byte("[Script Info]\nPlayResX: 1920\nPlayResY: 1080\n[Events]\n")
	output := ProcessASS(input, 720, false, false, false, true, false)
	if !strings.Contains(string(output), "PlayResX: 1280") {
		t.Errorf("expected PlayResX: 1280 for 720p, got: %s", output)
	}
	if !strings.Contains(string(output), "PlayResY: 720") {
		t.Errorf("expected PlayResY: 720, got: %s", output)
	}
}

func TestProcessASS_NoFixFlag(t *testing.T) {
	input := []byte("[Script Info]\nScaledBorderAndShadow: no\n")
	output := ProcessASS(input, 0, false, false, false, false, false)
	if !strings.Contains(string(output), "ScaledBorderAndShadow: no") {
		t.Error("should not modify when fix is false")
	}
}

func TestProcessSubtitleFile(t *testing.T) {
	content := []byte("[Script Info]\nScaledBorderAndShadow: no\n[Events]\nDialogue: 0,0:00:01.00,0:00:05.00,Default,,0,0,0,,hello\n")
	tmpFile := t.TempDir() + "/test.ass"
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	ScaledBorderAndShadowFix = boolPtr(true)
	SrtAssFix = boolPtr(false)
	OriginalScriptFix = boolPtr(false)
	SubtitleTimestampFix = boolPtr(false)
	LayoutResFix = boolPtr(false)
	NoASSFix = boolPtr(false)

	if err := processSubtitleFile(tmpFile, 0); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(tmpFile)
	if !strings.Contains(string(data), "ScaledBorderAndShadow: yes") {
		t.Error("file content should be fixed")
	}
}

func TestExtractFontNames_FromStyle(t *testing.T) {
	data := []byte("[V4+ Styles]\nStyle: Default,Arial,16,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,2,0,2,10,10,10,1\nStyle: Alt,Noto Sans,14,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,2,0,2,10,10,10,1\n[Events]\n")
	fonts := ExtractFontNames(data)
	if !containsStr(fonts, "Arial") {
		t.Errorf("expected Arial, got %v", fonts)
	}
	if !containsStr(fonts, "Noto Sans") {
		t.Errorf("expected Noto Sans, got %v", fonts)
	}
}

func TestExtractFontNames_FromOverride(t *testing.T) {
	data := []byte("[Events]\nDialogue: 0,0:00:01.00,0:00:05.00,Default,,0,0,0,,{\\fnComic Sans}hello\n")
	fonts := ExtractFontNames(data)
	if !containsStr(fonts, "Comic Sans") {
		t.Errorf("expected Comic Sans, got %v", fonts)
	}
}

func TestExtractFontNames_Deduplicates(t *testing.T) {
	data := []byte("[V4+ Styles]\nStyle: Default,Arial,16,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,2,0,2,10,10,10,1\n[Events]\nDialogue: 0,0:00:01.00,0:00:05.00,Default,,0,0,0,,{\\fnArial}hello\n")
	fonts := ExtractFontNames(data)
	count := 0
	for _, f := range fonts {
		if f == "Arial" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected Arial once, got %d occurrences", count)
	}
}

func TestExtractFontNames_Empty(t *testing.T) {
	data := []byte("[Events]\nDialogue: 0,0:00:01.00,0:00:05.00,Default,,0,0,0,,hello\n")
	fonts := ExtractFontNames(data)
	if len(fonts) != 0 {
		t.Errorf("expected no fonts, got %v", fonts)
	}
}

func TestExtractFontNames_OnlyStyleHeaderNoFonts(t *testing.T) {
	data := []byte("[V4+ Styles]\n[Events]\n")
	fonts := ExtractFontNames(data)
	if len(fonts) != 0 {
		t.Errorf("expected no fonts, got %v", fonts)
	}
}

func boolPtr(b bool) *bool { return &b }

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
