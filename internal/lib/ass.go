package lib

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ProcessASS(data []byte, videoHeight int, fixSBS, fixSRT, fixOrig, fixLayout, fixTS bool) []byte {
	lines := strings.Split(string(data), "\n")

	if fixLayout && videoHeight > 0 {
		width := videoHeight * 16 / 9
		lines = setPlayRes(lines, width, videoHeight)
	}
	if fixSBS {
		lines = setScaledBorderAndShadow(lines)
	}
	if fixSRT {
		lines = cleanupSrtAss(lines)
	}
	if fixOrig {
		lines = removeOriginalScript(lines)
	}
	if fixTS {
		lines = shiftTimestamps(lines)
	}

	return []byte(strings.Join(lines, "\n"))
}

func ExtractFontNames(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	seen := make(map[string]bool)

	inStyles := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[V4+ Styles]" || trimmed == "[V4 Styles]" {
			inStyles = true
			continue
		}
		if inStyles && strings.HasPrefix(trimmed, "[") {
			inStyles = false
		}
		if inStyles && strings.HasPrefix(trimmed, "Style:") {
			fields := strings.Split(trimmed, ",")
			if len(fields) >= 2 {
				name := strings.TrimSpace(fields[1])
				if name != "" {
					seen[name] = true
				}
			}
		}
	}

	for _, line := range lines {
		rest := line
		for {
			idx := strings.Index(rest, "\\fn")
			if idx < 0 {
				break
			}
			rest = rest[idx+3:]
			end := strings.IndexAny(rest, "\\}")
			fontName := rest
			if end >= 0 {
				fontName = rest[:end]
			}
			fontName = strings.TrimSpace(fontName)
			if fontName != "" {
				seen[fontName] = true
			}
			if end < 0 {
				break
			}
			rest = rest[end:]
		}
	}

	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	return result
}

func processSubtitleFile(path string, videoHeight int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	fixSBS := ScaledBorderAndShadowFix != nil && *ScaledBorderAndShadowFix
	fixSRT := SrtAssFix != nil && *SrtAssFix
	fixOrig := OriginalScriptFix != nil && *OriginalScriptFix
	fixLayout := LayoutResFix != nil && *LayoutResFix
	fixTS := SubtitleTimestampFix != nil && *SubtitleTimestampFix

	processed := ProcessASS(data, videoHeight, fixSBS, fixSRT, fixOrig, fixLayout, fixTS)
	return os.WriteFile(path, processed, 0644)
}

func setPlayRes(lines []string, width, height int) []string {
	wStr, hStr := strconv.Itoa(width), strconv.Itoa(height)
	foundX, foundY := false, false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "PlayResX:") {
			lines[i] = "PlayResX: " + wStr
			foundX = true
		} else if strings.HasPrefix(trimmed, "PlayResY:") {
			lines[i] = "PlayResY: " + hStr
			foundY = true
		}
	}
	if foundX && foundY {
		return lines
	}

	insertAt := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "[Script Info]" {
			insertAt = i + 1
			break
		}
	}
	if insertAt < 0 {
		return lines
	}

	var newLines []string
	newLines = append(newLines, lines[:insertAt]...)
	if !foundX {
		newLines = append(newLines, "PlayResX: "+wStr)
	}
	if !foundY {
		newLines = append(newLines, "PlayResY: "+hStr)
	}
	newLines = append(newLines, lines[insertAt:]...)
	return newLines
}

func setScaledBorderAndShadow(lines []string) []string {
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "ScaledBorderAndShadow:") {
			lines[i] = "ScaledBorderAndShadow: yes"
			return lines
		}
	}
	for i, line := range lines {
		if strings.TrimSpace(line) == "[Script Info]" {
			newLines := make([]string, 0, len(lines)+1)
			newLines = append(newLines, lines[:i+1]...)
			newLines = append(newLines, "ScaledBorderAndShadow: yes")
			newLines = append(newLines, lines[i+1:]...)
			return newLines
		}
	}
	return lines
}

func cleanupSrtAss(lines []string) []string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "Dialogue:") {
			start, end := parseStartEnd(line)
			if start >= 0 && end >= 0 && end-start < 0.01 {
				continue
			}
			line = strings.ReplaceAll(line, "\\N", "\\n")
		}
		result = append(result, line)
	}
	return result
}

func removeOriginalScript(lines []string) []string {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "OriginalScript:") || strings.HasPrefix(trimmed, "Original Script:") {
			if !strings.HasPrefix(trimmed, ";") {
				lines[i] = "; " + line
			}
		}
	}
	return lines
}

func shiftTimestamps(lines []string) []string {
	var shift float64
	for _, line := range lines {
		if !strings.HasPrefix(line, "Dialogue:") {
			continue
		}
		start, _ := parseStartEnd(line)
		if start < 2.0 {
			return lines
		}
		shift = start
		break
	}
	if shift == 0 {
		return lines
	}

	result := make([]string, len(lines))
	for i, line := range lines {
		if !strings.HasPrefix(line, "Dialogue:") {
			result[i] = line
			continue
		}
		result[i] = shiftDialogueLine(line, shift)
	}
	return result
}

func parseStartEnd(line string) (float64, float64) {
	body := line[len("Dialogue:"):]
	c1 := strings.IndexByte(body, ',')
	if c1 < 0 {
		return -1, -1
	}
	c2 := strings.IndexByte(body[c1+1:], ',')
	if c2 < 0 {
		return -1, -1
	}
	c2 += c1 + 1
	c3 := strings.IndexByte(body[c2+1:], ',')
	if c3 < 0 {
		return -1, -1
	}
	c3 += c2 + 1

	start, err1 := parseASSTimestamp(strings.TrimSpace(body[c1+1 : c2]))
	end, err2 := parseASSTimestamp(strings.TrimSpace(body[c2+1 : c3]))
	if err1 != nil || err2 != nil {
		return -1, -1
	}
	return start, end
}

func shiftDialogueLine(line string, shift float64) string {
	body := line[len("Dialogue:"):]
	c1 := strings.IndexByte(body, ',')
	c2 := strings.IndexByte(body[c1+1:], ',') + c1 + 1
	c3 := strings.IndexByte(body[c2+1:], ',') + c2 + 1

	start, _ := parseASSTimestamp(strings.TrimSpace(body[c1+1 : c2]))
	end, _ := parseASSTimestamp(strings.TrimSpace(body[c2+1 : c3]))

	start -= shift
	end -= shift
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}

	return "Dialogue:" + body[:c1+1] + formatASSTimestamp(start) + "," + formatASSTimestamp(end) + body[c3:]
}

func parseASSTimestamp(s string) (float64, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid timestamp: %s", s)
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, err
	}
	m, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, err
	}
	sec, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err != nil {
		return 0, err
	}
	return float64(h)*3600 + float64(m)*60 + sec, nil
}

func formatASSTimestamp(t float64) string {
	h := int(t) / 3600
	t -= float64(h) * 3600
	m := int(t) / 60
	t -= float64(m) * 60
	return fmt.Sprintf("%d:%02d:%05.2f", h, m, t)
}
