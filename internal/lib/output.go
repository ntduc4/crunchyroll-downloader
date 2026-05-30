package lib

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type mediaTrack struct {
	Path     string
	Language string
}

// mergeEverything merges audio, video and subtitles in a single MKV container.
func mergeEverything(videoFile string, audioFiles, subtitleFiles []mediaTrack, fontFiles []string, outputFile string, info EpisodeInfo, keepSources bool) {
	args := buildMuxArgs(videoFile, audioFiles, subtitleFiles, fontFiles, outputFile, info)

	cmd := exec.Command("ffmpeg", args...)
	if err := cmd.Run(); err != nil {
		panic(err)
	}

	if !keepSources {
		_ = os.Remove(videoFile)
		for _, audioFile := range audioFiles {
			_ = os.Remove(audioFile.Path)
		}
		for _, subtitleFile := range subtitleFiles {
			_ = os.Remove(subtitleFile.Path)
		}
		for _, fontFile := range fontFiles {
			_ = os.Remove(fontFile)
		}
	}

	fmt.Printf("\nDownload finished! Output file: %s\n\n", outputFile)
}

func buildMuxArgs(videoFile string, audioFiles, subtitleFiles []mediaTrack, fontFiles []string, outputFile string, info EpisodeInfo) []string {
	args := []string{
		"-i", videoFile,
	}

	for _, audioFile := range audioFiles {
		args = append(args, "-i", audioFile.Path)
	}
	for _, subtitleFile := range subtitleFiles {
		args = append(args, "-i", subtitleFile.Path)
	}

	args = append(args, "-map", "0:v:0")
	for i := range audioFiles {
		args = append(args, "-map", fmt.Sprintf("%d:a:0", i+1))
	}
	for i := range subtitleFiles {
		args = append(args, "-map", fmt.Sprintf("%d:0", i+1+len(audioFiles)))
	}

	args = append(args,
		"-c:v", "copy", "-c:a", "copy",
		"-c:s", "copy",
		"-metadata:g", "title="+fmt.Sprintf("S%02vE%02v - %s", info.EpisodeMetadata.SeasonNumber, info.EpisodeMetadata.EpisodeNumber, info.Title),
		"-metadata:g", "show="+info.EpisodeMetadata.SeriesTitle,
		"-metadata:g", "track="+fmt.Sprintf("%v", info.EpisodeMetadata.EpisodeNumber),
		"-metadata:g", "season_number="+fmt.Sprintf("%v", info.EpisodeMetadata.SeasonNumber),
	)

	for i, audioFile := range audioFiles {
		args = append(args, "-metadata:s:a:"+fmt.Sprintf("%d", i), fmt.Sprintf("title=%s", languageLabel(audioFile.Language)))
	}
	if len(audioFiles) > 0 {
		args = append(args, "-disposition:a:0", "default")
	}
	for i, subtitleFile := range subtitleFiles {
		args = append(args, "-metadata:s:s:"+fmt.Sprintf("%d", i), fmt.Sprintf("title=%s", languageLabel(subtitleFile.Language)))
	}

	for i, fontFile := range fontFiles {
		args = append(args, "-attach", fontFile)
		mime := "application/x-truetype-font"
		if strings.HasSuffix(fontFile, ".otf") {
			mime = "application/vnd.ms-opentype"
		} else if strings.HasSuffix(fontFile, ".woff2") {
			mime = "font/woff2"
		}
		args = append(args, "-metadata:s:t:"+fmt.Sprintf("%d", i), "mimetype="+mime)
	}

	args = append(args, outputFile)

	return args
}
