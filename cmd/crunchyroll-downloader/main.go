package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	lib "crunchyroll-downloader/internal/lib"
)

func main() {
	url := flag.String("url", "", "URL of the episode/season to download")
	urlsFile := flag.String("urls", "", "Path to a text file with one URL per line")

	lib.AudioLang = flag.String("audio-lang", "all", "Audio language or 'all'")
	lib.SubtitlesLang = flag.String("subs-lang", "all", "Subtitles language or 'all'")
	lib.VideoQuality = flag.String("video-quality", "1080p", "Video quality")
	lib.AudioQuality = flag.String("audio-quality", "192k", "Audio quality")
	lib.SeasonNumber = flag.Int("season", 0, "Season number. Not used if an episode link is entered")
	lib.EtpRt = flag.String("etp-rt", "", "The \"etp_rt\" cookie value of your account")
	lib.DebugDump = flag.Bool("debug-dump", false, "Keep decrypted tracks and raw encrypted inputs")
	lib.DecryptOnly = flag.Bool("decrypt-only", false, "Decrypt from local .xml and .enc dumps instead of downloading media")
	lib.SetupDir = flag.String("setup-dir", "", "Save all tracks and metadata to directory for testing (skips muxing)")

	lib.NoASSFix = flag.Bool("no-ass-fix", false, "Disable all ASS subtitle processing")
	lib.ScaledBorderAndShadowFix = flag.Bool("scaled-border-and-shadow-fix", true, "Fix ScaledBorderAndShadow for correct border/shadow scaling")
	lib.SrtAssFix = flag.Bool("srt-ass-fix", true, "Clean up SRT-to-ASS conversion artifacts")
	lib.OriginalScriptFix = flag.Bool("original-script-fix", true, "Comment out OriginalScript line that can confuse renderers")
	lib.SubtitleTimestampFix = flag.Bool("subtitle-timestamp-fix", false, "Auto-detect and fix subtitle timing offset")
	lib.LayoutResFix = flag.Bool("layout-res-fix", false, "Override PlayRes to match video resolution")
	lib.DlFonts = flag.Bool("dl-fonts", false, "Download subtitle fonts from Google Fonts and embed them in the MKV")

	flag.Parse()

	if *url == "" && *urlsFile == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *lib.EtpRt == "" {
		fmt.Println("You must specify the \"-etp-rt\" option!\n- Open Crunchyroll on your browser and log in.\n- Open developer tools (Ctrl+Shift+I), go to \"Application\", and then \"Cookies\".\n- The value of the \"ept_rt\" cookie is what you need to input into this option.")
		os.Exit(1)
	}

	lib.Token = lib.GetAccessToken(*lib.EtpRt)

	if *urlsFile != "" {
		file, err := os.Open(*urlsFile)
		if err != nil {
			fmt.Printf("Failed to open URLs file: %s\n", err)
			os.Exit(1)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var urls []string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && strings.HasPrefix(line, "http") {
				urls = append(urls, line)
			}
		}

		fmt.Printf("Found %d URLs to download\n\n", len(urls))
		for i, u := range urls {
			fmt.Printf("=== [%d/%d] %s ===\n", i+1, len(urls), u)
			lib.ProcessURL(u)
			fmt.Println()
		}
	} else {
		lib.ProcessURL(*url)
	}
}
