package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	widevine "github.com/iyear/gowidevine"
	"github.com/unki2aut/go-mpd"
)

const maxWorkers = 10

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

func downloadSubs(url string) string {
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

	filename := getFilename(nil)
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
		if subtitles[requestedLanguage] == nil {
			return nil
		}
		return []string{requestedLanguage}
	}

	locales := sortedLanguageKeys(subtitles)
	filtered := make([]string, 0, len(locales))
	for _, locale := range locales {
		if subtitles[locale] != nil {
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

func downloadEpisode(contentId string, videoQuality, audioQuality, subtitlesLang *string, info EpisodeInfo) {
	sanitize := func(s string) string {
		illegal := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
		res := s
		for _, char := range illegal {
			res = strings.ReplaceAll(res, char, "_")
		}
		return strings.TrimRight(res, " .")
	}

	cleanSeriesTitle := sanitize(info.EpisodeMetadata.SeriesTitle)
	audioVariants := resolveAudioVariants(contentId, info, *audioLang)
	primaryVariant := audioVariants[0]

	if _, err := os.Stat(cleanSeriesTitle); err != nil {
		_ = os.MkdirAll(cleanSeriesTitle, 0777)
	}

	outputFile := fmt.Sprintf("%s/%s S%02vE%02v [%s].mkv",
		cleanSeriesTitle,
		cleanSeriesTitle,
		info.EpisodeMetadata.SeasonNumber,
		info.EpisodeMetadata.EpisodeNumber,
		*videoQuality,
	)

	if _, err := os.Stat(outputFile); err == nil {
		fmt.Printf("Episode %v is already downloaded, skipping...\n", info.EpisodeMetadata.EpisodeNumber)
		return
	}

	episode := getEpisode(primaryVariant.ContentID)
	fmt.Printf("Downloading: %s (S%02vE%02v) from %s\n", info.Title, info.EpisodeMetadata.SeasonNumber, info.EpisodeMetadata.EpisodeNumber, info.EpisodeMetadata.SeriesTitle)

	manifestPath := strings.TrimSuffix(outputFile, ".mkv") + ".xml"
	var manifest *mpd.MPD
	if *decryptOnly {
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

	err := getLicense(*pssh, primaryVariant.ContentID, episode.Token, defaultKID)
	if err != nil {
		fmt.Printf("Error: %s", err)
		os.Exit(1)
	}

	subtitleLanguages := resolveSubtitleLanguages(episode.Subtitles, *subtitlesLang)
	subtitleFiles := make([]mediaTrack, 0, len(subtitleLanguages))
	for _, locale := range subtitleLanguages {
		subtitles := episode.Subtitles[locale]
		fmt.Printf("Downloading subtitles for %s...\n", languageLabel(locale))
		subtitleFiles = append(subtitleFiles, mediaTrack{Path: downloadSubs(subtitles.URL), Language: locale})
	}
	if len(subtitleFiles) > 0 {
		fmt.Println("Downloaded subtitles!")
	}

	baseUrl, representationId := getBaseUrl(videoSet, true, *videoQuality)
	if baseUrl == nil {
		print("Failed to get the video base URL, maybe the video quality you entered is wrong?\n")
		os.Exit(1)
	}
	videoOut := ""
	videoInput := ""
	if *debugDump {
		videoOut = strings.TrimSuffix(outputFile, ".mkv") + ".video.mp4"
	}
	if *decryptOnly {
		videoInput = strings.TrimSuffix(outputFile, ".mkv") + ".video.mp4.enc"
	}

	videoFile, err := downloadParts(baseUrl, representationId, videoSet, videoOut, videoInput)
	if err != nil {
		panic(err)
	}

	audioFiles := make([]mediaTrack, 0, len(audioVariants))
	includeLocaleInAudioDump := len(audioVariants) > 1
	baseAudioOut := ""
	if *debugDump {
		baseAudioOut = getAudioTrackOutputPath(outputFile, audioVariants[0].AudioLocale, includeLocaleInAudioDump)
	}
	baseAudioInput := ""
	if *decryptOnly {
		baseAudioInput = getAudioTrackOutputPath(outputFile, audioVariants[0].AudioLocale, includeLocaleInAudioDump) + ".enc"
	}
	audioBaseUrl, audioRepresentationId := getBaseUrl(audioSet, false, *audioQuality)
	if audioBaseUrl == nil {
		print("Failed to get the audio base URL, maybe the audio quality you entered is wrong?\n")
		os.Exit(1)
	}
	baseAudioFile, err := downloadParts(audioBaseUrl, audioRepresentationId, audioSet, baseAudioOut, baseAudioInput)
	if err != nil {
		panic(err)
	}
	audioFiles = append(audioFiles, mediaTrack{Path: baseAudioFile, Language: audioVariants[0].AudioLocale})

	if success := deleteStream(primaryVariant.ContentID, episode.Token); !success {
		print("Failed to remove the player stream, you will probably have issues downloading other episodes.\n")
	}

	for _, variant := range audioVariants[1:] {
		fmt.Printf("Downloading audio for %s...\n", languageLabel(variant.AudioLocale))
		variantEpisode := getEpisode(variant.ContentID)
		variantManifestPath := getAudioManifestPath(outputFile, variant.AudioLocale, true)
		var variantManifest *mpd.MPD
		if *decryptOnly {
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
		if err := getLicense(*variantPssh, variant.ContentID, variantEpisode.Token, variantDefaultKID); err != nil {
			fmt.Printf("Error: %s", err)
			os.Exit(1)
		}
		variantAudioBaseURL, variantAudioRepresentationID := getBaseUrl(variantAudioSet, false, *audioQuality)
		if variantAudioBaseURL == nil {
			print("Failed to get the audio base URL, maybe the audio quality you entered is wrong?\n")
			os.Exit(1)
		}
		variantAudioOut := ""
		if *debugDump {
			variantAudioOut = getAudioTrackOutputPath(outputFile, variant.AudioLocale, true)
		}
		variantAudioInput := ""
		if *decryptOnly {
			variantAudioInput = getAudioTrackOutputPath(outputFile, variant.AudioLocale, true) + ".enc"
		}
		variantAudioFile, err := downloadParts(variantAudioBaseURL, variantAudioRepresentationID, variantAudioSet, variantAudioOut, variantAudioInput)
		if err != nil {
			panic(err)
		}
		audioFiles = append(audioFiles, mediaTrack{Path: variantAudioFile, Language: variant.AudioLocale})
		if success := deleteStream(variant.ContentID, variantEpisode.Token); !success {
			print("Failed to remove the player stream, you will probably have issues downloading other episodes.\n")
		}
	}

	mergeEverything(videoFile, audioFiles, subtitleFiles, outputFile, info, *debugDump)
}

func downloadSeason(videoQuality, audioQuality, subtitlesLang *string, episodes []SeasonEpisode) {
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
		downloadEpisode(episode.ID, videoQuality, audioQuality, subtitlesLang, info)
	}
}
