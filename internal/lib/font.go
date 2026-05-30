package lib

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var fontCDNDomains = []string{
	"fonts.gstatic.com",
	"fonts.googleapis.com",
}

var fontExtensions = []string{".ttf", ".otf", ".woff", ".woff2"}

func dlFontsForSubs(subtitleFiles []mediaTrack) []string {
	if DlFonts == nil || !*DlFonts {
		return nil
	}

	allFonts := make(map[string]bool)
	for _, sub := range subtitleFiles {
		data, err := os.ReadFile(sub.Path)
		if err != nil {
			fmt.Printf("Warning: could not read %s for font extraction: %v\n", sub.Path, err)
			continue
		}
		for _, name := range ExtractFontNames(data) {
			allFonts[name] = true
		}
	}

	if len(allFonts) == 0 {
		return nil
	}

	fmt.Println("Downloading subtitle fonts...")
	dir, err := os.MkdirTemp("", "crdl-fonts-*")
	if err != nil {
		fmt.Printf("Warning: could not create font temp dir: %v\n", err)
		return nil
	}

	var paths []string
	for name := range allFonts {
		path, err := downloadGoogleFont(name, dir)
		if err != nil {
			fmt.Printf("  Skipped %q: %v\n", name, err)
			continue
		}
		paths = append(paths, path)
	}

	if len(paths) > 0 {
		fmt.Println("Downloaded subtitle fonts!")
	}
	return paths
}

func downloadGoogleFont(name, dir string) (string, error) {
	cssURL := fmt.Sprintf("https://fonts.googleapis.com/css2?family=%s", strings.ReplaceAll(name, " ", "+"))
	resp, err := http.Get(cssURL)
	if err != nil {
		return "", fmt.Errorf("HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 400 {
		return "", fmt.Errorf("not available on Google Fonts")
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Google Fonts returned status %d", resp.StatusCode)
	}

	css, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read error: %w", err)
	}

	if !bytes.Contains(css, []byte("@font-face")) {
		return "", fmt.Errorf("response is not a font stylesheet")
	}

	fontURL := extractFontURL(string(css))
	if fontURL == "" {
		return "", fmt.Errorf("no font file URL found in CSS response")
	}

	resp, err = http.Get(fontURL)
	if err != nil {
		return "", fmt.Errorf("font file HTTP error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("font file returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("font file read error: %w", err)
	}

	if !validFontFile(data) {
		return "", fmt.Errorf("downloaded file is not a valid font (wrong format or too small)")
	}

	ext := ".ttf"
	if bytes.HasPrefix(data, []byte("OTTO")) {
		ext = ".otf"
	} else if bytes.HasPrefix(data, []byte("wOFF")) {
		ext = ".woff"
	} else if bytes.HasPrefix(data, []byte("wOF2")) {
		ext = ".woff2"
	}
	safeName := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' {
			return '_'
		}
		return r
	}, name)
	outPath := filepath.Join(dir, safeName+ext)
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return "", fmt.Errorf("write error: %w", err)
	}
	fmt.Printf("  Downloaded font: %s (%s)\n", name, ext)
	return outPath, nil
}

func validFontFile(data []byte) bool {
	if len(data) < 16 {
		return false
	}
	switch {
	case bytes.HasPrefix(data, []byte{0x00, 0x01, 0x00, 0x00}):
		return true // TrueType
	case bytes.HasPrefix(data, []byte("OTTO")):
		return true // OpenType
	case bytes.HasPrefix(data, []byte("wOFF")):
		return true // WOFF
	case bytes.HasPrefix(data, []byte("wOF2")):
		return true // WOFF2
	case bytes.HasPrefix(data, []byte("ttcf")):
		return true // TrueType Collection
	default:
		return false
	}
}

func extractFontURL(css string) string {
	for _, line := range strings.Split(css, "\n") {
		idx := strings.Index(line, "url(")
		if idx < 0 {
			continue
		}
		start := idx + 4
		end := strings.Index(line[start:], ")")
		if end < 0 {
			continue
		}
		raw := strings.TrimSpace(line[start : start+end])
		raw = strings.Trim(raw, "'\"")
		if !strings.HasPrefix(raw, "http") {
			raw = "https:" + raw
		}
		if !isFontURL(raw) {
			continue
		}
		return raw
	}
	return ""
}

func isFontURL(raw string) bool {
	for _, domain := range fontCDNDomains {
		if !strings.Contains(raw, domain) {
			continue
		}
		for _, ext := range fontExtensions {
			if strings.HasSuffix(raw, ext) {
				return true
			}
		}
	}
	return false
}
