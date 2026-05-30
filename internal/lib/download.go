package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	widevine "github.com/iyear/gowidevine"
	"github.com/unki2aut/go-mpd"
)

const maxWorkers = 10

var (
	Token         string
	AudioLang     *string
	SubtitlesLang *string
	VideoQuality  *string
	AudioQuality  *string
	EtpRt         *string
	DebugDump     *bool
	DecryptOnly   *bool
	SeasonNumber  *int
	SetupDir      *string

	NoASSFix              *bool
	ScaledBorderAndShadowFix *bool
	SrtAssFix             *bool
	OriginalScriptFix     *bool
	SubtitleTimestampFix  *bool
	LayoutResFix          *bool
	DlFonts               *bool
)

func parseVideoQuality(q string) int {
	q = strings.ReplaceAll(q, "p", "")
	n, _ := strconv.Atoi(q)
	return n
}

func buildUrl(base, representationId, file string, partNum *int64) string {
	if partNum != nil {
		file = strings.ReplaceAll(file, "$Number$", fmt.Sprintf("%05d", *partNum))
		file = strings.ReplaceAll(file, "$Number%05d$", fmt.Sprintf("%05d", *partNum))
	}
	return base + strings.ReplaceAll(file, "$RepresentationID$", representationId)
}

func downloadPart(url string) ([]byte, error) {
	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Origin", "https://static.crunchyroll.com")
		req.Header.Set("Referer", "https://static.crunchyroll.com/")
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if attempt < maxRetries-1 {
				continue
			}
			return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, err)
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			if attempt < maxRetries-1 {
				continue
			}
			return nil, fmt.Errorf("failed after %d retries, status: %d", maxRetries, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if attempt < maxRetries-1 {
				continue
			}
			return nil, fmt.Errorf("failed reading body after %d retries: %w", maxRetries, err)
		}
		if looksLikeTextResponse(body) {
			return nil, fmt.Errorf("unexpected text response for %s", url)
		}
		return body, nil
	}
	return nil, fmt.Errorf("failed after %d retries", maxRetries)
}

func looksLikeTextResponse(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return true
	}
	if len(trimmed) >= 5 {
		prefix := strings.ToLower(string(trimmed[:5]))
		if prefix == "<html" || prefix == "<?xml" {
			return true
		}
	}
	textMarkers := [][]byte{
		[]byte("AccessDenied"),
		[]byte("Forbidden"),
		[]byte("Request has expired"),
	}
	for _, marker := range textMarkers {
		if bytes.Contains(trimmed, marker) {
			return true
		}
	}
	return false
}

func getFilename(set *mpd.AdaptationSet) string {
	if set == nil {
		f, _ := os.CreateTemp("", "crdl-subs-*.ass")
		return f.Name()
	}
	for _, representation := range set.Representations {
		if representation.Height != nil {
			f, _ := os.CreateTemp("", "crdl-video-*.mp4")
			return f.Name()
		} else if representation.Bandwidth != nil {
			f, _ := os.CreateTemp("", "crdl-audio-*.mp3")
			return f.Name()
		}
	}
	return ""
}

type segmentJob struct {
	index int
	url   string
}

func downloadParts(baseUrl, representationId *string, set *mpd.AdaptationSet, outputPath, inputPath string) (string, error) {
	var parts []byte

	if inputPath != "" {
		data, err := os.ReadFile(inputPath)
		if err != nil {
			return "", err
		}
		parts = append(parts, data...)
		fmt.Println("Finished loading local dump!")
	} else if set.SegmentTemplate == nil || set.SegmentTemplate.Initialization == nil || set.SegmentTemplate.Media == nil || set.SegmentTemplate.SegmentTimeline == nil {
		data, err := downloadPart(*baseUrl)
		if err != nil {
			return "", err
		}
		parts = append(parts, data...)
		fmt.Println("Finished downloading!")
	} else {
		initUrl := buildUrl(*baseUrl, *representationId, *set.SegmentTemplate.Initialization, nil)
		initData, err := downloadPart(initUrl)
		if err != nil {
			return "", err
		}

		timeline := expandTimeline(set.SegmentTemplate.SegmentTimeline.S, 1)
		total := len(timeline)
		results := make([][]byte, total)
		var downloadErr error
		var errOnce sync.Once
		var done atomic.Int64

		jobs := make(chan segmentJob, total)
		var wg sync.WaitGroup

		for w := 0; w < maxWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for job := range jobs {
					data, err := downloadPart(job.url)
					if err != nil {
						errOnce.Do(func() { downloadErr = err })
						return
					}
					results[job.index] = data
					count := done.Add(1)
					fmt.Printf("\rDownloaded %v of %v segments (%v%%)", count, total, (100*count)/int64(total))
				}
			}()
		}

		for i, item := range timeline {
			url := buildUrl(*baseUrl, *representationId, *set.SegmentTemplate.Media, &item)
			jobs <- segmentJob{index: i, url: url}
		}
		close(jobs)
		wg.Wait()

		if downloadErr != nil {
			return "", downloadErr
		}

		fmt.Println("\nFinished downloading!")

		parts = append(parts, initData...)
		for _, data := range results {
			parts = append(parts, data...)
		}
	}

	filename := outputPath
	if filename == "" {
		filename = getFilename(set)
	}
	file, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if outputPath != "" && inputPath == "" {
		if err := os.WriteFile(filename+".enc", parts, 0644); err != nil {
			return "", err
		}
	}
	if err = widevine.DecryptMP4Auto(io.NopCloser(bytes.NewReader(parts)), keys, file); err != nil {
		return "", fmt.Errorf("widevine.DecryptMP4Auto: %w", err)
	}

	return filename, nil
}

func downloadSubs(url, outputPath string) string {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Origin", "https://static.crunchyroll.com")
	req.Header.Set("Referer", "https://static.crunchyroll.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	filename := outputPath
	if filename == "" {
		filename = getFilename(nil)
	}
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	file.Write(body)
	file.Close()

	return filename
}

type episodeVariant struct {
	ContentID   string
	AudioLocale string
}

func resolveAudioVariants(contentID string, info EpisodeInfo, requestedLanguage string) []episodeVariant {
	variantsByLocale := map[string]string{
		info.EpisodeMetadata.AudioLocale: contentID,
	}
	for _, version := range info.EpisodeMetadata.Versions {
		if version == nil {
			continue
		}
		variantsByLocale[version.AudioLocale] = version.GUID
	}

	if requestedLanguage != "all" {
		variantID, ok := variantsByLocale[requestedLanguage]
		if !ok {
			print("! Invalid audio locale. Please put the locale in the \"ja-JP\", \"en-US\"... format, or use \"all\".\n")
			os.Exit(1)
		}
		return []episodeVariant{{ContentID: variantID, AudioLocale: requestedLanguage}}
	}

	variants := []episodeVariant{{ContentID: contentID, AudioLocale: info.EpisodeMetadata.AudioLocale}}
	locales := make([]string, 0, len(variantsByLocale)-1)
	for locale := range variantsByLocale {
		if locale == info.EpisodeMetadata.AudioLocale {
			continue
		}
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	for _, locale := range locales {
		variants = append(variants, episodeVariant{ContentID: variantsByLocale[locale], AudioLocale: locale})
	}

	return variants
}

func resolveSubtitleLanguages(subtitles map[string]*Subtitle, requestedLanguage string) []string {
	if requestedLanguage != "all" {
		sub := subtitles[requestedLanguage]
		if sub == nil || sub.URL == "" {
			return nil
		}
		return []string{requestedLanguage}
	}

	locales := sortedLanguageKeys(subtitles)
	filtered := make([]string, 0, len(locales))
	for _, locale := range locales {
		if locale == "" {
			continue
		}
		sub := subtitles[locale]
		if sub != nil && sub.URL != "" {
			filtered = append(filtered, locale)
		}
	}

	return filtered
}

func getAudioTrackOutputPath(outputFile, locale string, includeLocale bool) string {
	base := strings.TrimSuffix(outputFile, ".mkv") + ".audio"
	if includeLocale {
		base += "." + locale
	}
	return base + ".m4a"
}

func getAudioManifestPath(outputFile, locale string, includeLocale bool) string {
	base := strings.TrimSuffix(outputFile, ".mkv")
	if includeLocale {
		return base + ".audio." + locale + ".xml"
	}
	return base + ".xml"
}

func DownloadEpisode(contentId string, VideoQuality, AudioQuality, SubtitlesLang *string, info EpisodeInfo) {
	sanitize := func(s string) string {
		illegal := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
		res := s
		for _, char := range illegal {
			res = strings.ReplaceAll(res, char, "_")
		}
		return strings.TrimRight(res, " .")
	}

	cleanSeriesTitle := sanitize(info.EpisodeMetadata.SeriesTitle)
	audioVariants := resolveAudioVariants(contentId, info, *AudioLang)
	primaryVariant := audioVariants[0]

	setupMode := SetupDir != nil && *SetupDir != ""
	if setupMode {
		_ = os.MkdirAll(filepath.Join(*SetupDir, "audio"), 0777)
		_ = os.MkdirAll(filepath.Join(*SetupDir, "subtitles"), 0777)
		_ = os.MkdirAll(filepath.Join(*SetupDir, "manifests"), 0777)
	}

	var outputFile string
	if !setupMode {
		if _, err := os.Stat(cleanSeriesTitle); err != nil {
			_ = os.MkdirAll(cleanSeriesTitle, 0777)
		}

		outputFile = fmt.Sprintf("%s/%s S%02vE%02v [%s].mkv",
			cleanSeriesTitle,
			cleanSeriesTitle,
			info.EpisodeMetadata.SeasonNumber,
			info.EpisodeMetadata.EpisodeNumber,
			*VideoQuality,
		)

		if _, err := os.Stat(outputFile); err == nil {
			fmt.Printf("Episode %v is already downloaded, skipping...\n", info.EpisodeMetadata.EpisodeNumber)
			return
		}
	}

	episode := GetEpisode(primaryVariant.ContentID)
	fmt.Printf("Downloading: %s (S%02vE%02v) from %s\n", info.Title, info.EpisodeMetadata.SeasonNumber, info.EpisodeMetadata.EpisodeNumber, info.EpisodeMetadata.SeriesTitle)

	if setupMode {
		saveJSONFile(filepath.Join(*SetupDir, "episode_info.json"), info)
		saveJSONFile(filepath.Join(*SetupDir, "playback.json"), episode)
	}

	manifestPath := ""
	if *DebugDump || setupMode {
		manifestPath = strings.TrimSuffix(outputFile, ".mkv") + ".xml"
	}
	if setupMode {
		manifestPath = filepath.Join(*SetupDir, "manifest.xml")
	}
	if *DecryptOnly && manifestPath == "" {
		manifestPath = strings.TrimSuffix(outputFile, ".mkv") + ".xml"
	}
	var manifest *mpd.MPD
	if *DecryptOnly {
		manifest = loadManifestFromFile(manifestPath)
	} else {
		manifest = parseManifest(episode.ManifestURL, manifestPath)
	}
	pssh := getPssh(manifest)
	if pssh == nil {
		panic("PSSH not found")
	}
	defaultKID := getDefaultKID(manifest)
	videoSet := getVideoSet(manifest)
	audioSet := getAudioSet(manifest, primaryVariant.AudioLocale)
	if videoSet == nil || audioSet == nil {
		panic("missing video or audio adaptation set")
	}

	err := GetLicense(*pssh, primaryVariant.ContentID, episode.Token, defaultKID)
	if err != nil {
		fmt.Printf("Error: %s", err)
		os.Exit(1)
	}

	subtitleLanguages := resolveSubtitleLanguages(episode.Subtitles, *SubtitlesLang)
	subtitleFiles := make([]mediaTrack, 0, len(subtitleLanguages))
	videoHeight := parseVideoQuality(*VideoQuality)
	for _, locale := range subtitleLanguages {
		subtitles := episode.Subtitles[locale]
		fmt.Printf("Downloading subtitles for %s...\n", languageLabel(locale))
		var subPath string
		if setupMode {
			subPath = filepath.Join(*SetupDir, "subtitles", locale+".ass")
		}
		subFile := downloadSubs(subtitles.URL, subPath)
		if !*NoASSFix {
			if err := processSubtitleFile(subFile, videoHeight); err != nil {
				fmt.Printf("Warning: failed to process subtitles for %s: %v\n", locale, err)
			}
		}
		subtitleFiles = append(subtitleFiles, mediaTrack{Path: subFile, Language: locale})
	}
	if len(subtitleFiles) > 0 {
		fmt.Println("Downloaded subtitles!")
	}

	baseUrl, representationId := getBaseUrl(videoSet, true, *VideoQuality)
	if baseUrl == nil {
		print("Failed to get the video base URL, maybe the video quality you entered is wrong?\n")
		os.Exit(1)
	}
	videoOut := ""
	videoInput := ""
	if setupMode {
		videoOut = filepath.Join(*SetupDir, "video.mp4")
	} else if *DebugDump {
		videoOut = strings.TrimSuffix(outputFile, ".mkv") + ".video.mp4"
	}
	if *DecryptOnly && !setupMode {
		videoInput = strings.TrimSuffix(outputFile, ".mkv") + ".video.mp4.enc"
	}

	videoFile, err := downloadParts(baseUrl, representationId, videoSet, videoOut, videoInput)
	if err != nil {
		panic(err)
	}

	audioFiles := make([]mediaTrack, 0, len(audioVariants))
	baseAudioOut := ""
	if setupMode {
		baseAudioOut = filepath.Join(*SetupDir, "audio", audioVariants[0].AudioLocale+".m4a")
	} else if *DebugDump {
		includeLocaleInAudioDump := len(audioVariants) > 1
		baseAudioOut = getAudioTrackOutputPath(outputFile, audioVariants[0].AudioLocale, includeLocaleInAudioDump)
	}
	baseAudioInput := ""
	if *DecryptOnly && !setupMode {
		includeLocaleInAudioDump := len(audioVariants) > 1
		baseAudioInput = getAudioTrackOutputPath(outputFile, audioVariants[0].AudioLocale, includeLocaleInAudioDump) + ".enc"
	}
	audioBaseUrl, audioRepresentationId := getBaseUrl(audioSet, false, *AudioQuality)
	if audioBaseUrl == nil {
		print("Failed to get the audio base URL, maybe the audio quality you entered is wrong?\n")
		os.Exit(1)
	}
	baseAudioFile, err := downloadParts(audioBaseUrl, audioRepresentationId, audioSet, baseAudioOut, baseAudioInput)
	if err != nil {
		panic(err)
	}
	audioFiles = append(audioFiles, mediaTrack{Path: baseAudioFile, Language: audioVariants[0].AudioLocale})

	if success := DeleteStream(primaryVariant.ContentID, episode.Token); !success {
		print("Failed to remove the player stream, you will probably have issues downloading other episodes.\n")
	}

	for _, variant := range audioVariants[1:] {
		fmt.Printf("Downloading audio for %s...\n", languageLabel(variant.AudioLocale))
		variantEpisode := GetEpisode(variant.ContentID)
		variantManifestPath := ""
		if *DebugDump || setupMode {
			variantManifestPath = getAudioManifestPath(outputFile, variant.AudioLocale, true)
		}
		if setupMode {
			variantManifestPath = filepath.Join(*SetupDir, "manifests", variant.AudioLocale+".xml")
		}
		if *DecryptOnly && variantManifestPath == "" {
			variantManifestPath = getAudioManifestPath(outputFile, variant.AudioLocale, true)
		}
		var variantManifest *mpd.MPD
		if *DecryptOnly {
			variantManifest = loadManifestFromFile(variantManifestPath)
		} else {
			variantManifest = parseManifest(variantEpisode.ManifestURL, variantManifestPath)
		}
		variantPssh := getPssh(variantManifest)
		if variantPssh == nil {
			panic("PSSH not found")
		}
		variantDefaultKID := getDefaultKID(variantManifest)
		variantAudioSet := getAudioSet(variantManifest, variant.AudioLocale)
		if variantAudioSet == nil {
			panic("missing audio adaptation set")
		}
		if err := GetLicense(*variantPssh, variant.ContentID, variantEpisode.Token, variantDefaultKID); err != nil {
			fmt.Printf("Error: %s", err)
			os.Exit(1)
		}
		variantAudioBaseURL, variantAudioRepresentationID := getBaseUrl(variantAudioSet, false, *AudioQuality)
		if variantAudioBaseURL == nil {
			print("Failed to get the audio base URL, maybe the audio quality you entered is wrong?\n")
			os.Exit(1)
		}
		variantAudioOut := ""
		if setupMode {
			variantAudioOut = filepath.Join(*SetupDir, "audio", variant.AudioLocale+".m4a")
		} else if *DebugDump {
			variantAudioOut = getAudioTrackOutputPath(outputFile, variant.AudioLocale, true)
		}
		variantAudioInput := ""
		if *DecryptOnly && !setupMode {
			variantAudioInput = getAudioTrackOutputPath(outputFile, variant.AudioLocale, true) + ".enc"
		}
		variantAudioFile, err := downloadParts(variantAudioBaseURL, variantAudioRepresentationID, variantAudioSet, variantAudioOut, variantAudioInput)
		if err != nil {
			panic(err)
		}
		audioFiles = append(audioFiles, mediaTrack{Path: variantAudioFile, Language: variant.AudioLocale})
		if success := DeleteStream(variant.ContentID, variantEpisode.Token); !success {
			print("Failed to remove the player stream, you will probably have issues downloading other episodes.\n")
		}
	}

	fontFiles := dlFontsForSubs(subtitleFiles)

	if setupMode {
		fmt.Println("\nSetup finished! All tracks saved to", *SetupDir)
		return
	}

	mergeEverything(videoFile, audioFiles, subtitleFiles, fontFiles, outputFile, info, *DebugDump)
}

func saveJSONFile(path string, v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}

func DownloadSeason(VideoQuality, AudioQuality, SubtitlesLang *string, episodes []SeasonEpisode) {
	fmt.Printf("Downloading season %v of %s (%v episodes)\n\n", episodes[0].SeasonNumber, episodes[0].SeriesTitle, len(episodes))

	for _, episode := range episodes {
		info := EpisodeInfo{
			EpisodeMetadata: EpisodeMetadata{
				SeriesTitle:        episode.SeriesTitle,
				SeasonNumber:       episode.SeasonNumber,
				EpisodeNumber:      episode.EpisodeNumber,
				AudioLocale:        episode.AudioLocale,
				Versions:           episode.Versions,
				AvailabilityStarts: episode.AvailabilityStarts,
			},
			Title: episode.Title,
		}
		DownloadEpisode(episode.ID, VideoQuality, AudioQuality, SubtitlesLang, info)
	}
}

func ProcessURL(url string) {
	contentType := strings.Split(url, "/")[3]
	contentId := strings.Split(url, "/")[4]
	if contentType != "watch" && contentType != "series" {
		fmt.Printf("Invalid URL (must be /watch/ or /series/): %s\n", url)
		return
	}

	if contentType == "watch" {
		info := GetEpisodeInfo(contentId)
		DownloadEpisode(contentId, VideoQuality, AudioQuality, SubtitlesLang, info)
	} else {
		seasons := GetSeasons(contentId)

		if SeasonNumber != nil && *SeasonNumber != 0 {
			var seasonId string
			for _, season := range seasons {
				if season.SeasonNumber == *SeasonNumber {
					seasonId = season.ID
					break
				}
			}
			if seasonId == "" {
				fmt.Printf("This anime has no season %v!\n", *SeasonNumber)
				return
			}

			episodes := GetSeasonEpisodes(seasonId)
			DownloadSeason(VideoQuality, AudioQuality, SubtitlesLang, episodes)
		} else {
			print("No season number specified, downloading all seasons...\n")

			for _, season := range seasons {
				episodes := GetSeasonEpisodes(season.ID)
				DownloadSeason(VideoQuality, AudioQuality, SubtitlesLang, episodes)
			}
		}
	}
}
