# Contributing

## Getting started

```sh
git clone https://github.com/CuteTenshii/crunchyroll-downloader.git
git checkout dev      # development branch
```

All work should target the `dev` branch. `main` is reserved for stable releases.

## Running tests

```sh
go test ./...
```

This runs two categories of tests:

### Always-offline (no setup needed)

These test pure logic and never touch the network or filesystem
beyond test fixtures. They run immediately after cloning the repo.

| File | What it tests |
|---|---|
| `utils_test.go` | `languageLabel`, `sortedLanguageKeys` |
| `download_test.go` | `resolveAudioVariants`, `resolveSubtitleLanguages` |
| `output_test.go` | `buildMuxArgs` (ffmpeg argument construction) |
| `mpd_test.go` (partial) | `TestGetVideoSet_NilOnEmpty`, `TestGetAudioSet_NilOnEmpty`, `TestExpandTimeline` |
| `download_integration_test.go` (partial) | `TestSingleAudioTrack`, `TestSingleSubtitleTrack`, `TestAllTracksPipeline_*`, `TestMuxArgs_*`, `TestDownloadArgsOrder` |

### MPD & integration tests (requires setup)

These parse real Crunchyroll MPD manifests and validate the full
download pipeline against actual output. They **discover** all
subdirectories under `testdata/` that contain a `manifest.xml` and
run against every one found. No hardcoded names, call your test
data whatever you want.

If no `testdata/` directory exists, these tests **skip** with a clear
message.

## Generating test data

The test data is copyrighted (contains Crunchyroll media) and never
committed to the repo. Each developer generates their own using:

```sh
./scripts/setup-testdata.sh \
  -u "https://www.crunchyroll.com/watch/<episode-id>" \
  -e "<your-etp-rt-cookie>" \
  -n "<any-name>"
```

The `-n` flag is freeform, use any name (e.g. `ep1`, `my-show`, `dub-test`).
The tests don't care what you call it.

The script:
1. Builds the downloader binary if needed
2. Runs it with `-setup-dir testdata/<name>` which downloads **all**
   audio dubs, all subtitle languages, the video track, and metadata
   JSONs into a predictable directory tree
3. Does **not** mux, leaves every track as a standalone file

### Output structure

```
testdata/<name>/
├── episode_info.json       # GetEpisodeInfo API response (dub versions)
├── playback.json           # GetEpisode API response (subtitle URLs, token)
├── manifest.xml            # Primary variant's MPD manifest
├── video.mp4               # Decrypted video track
├── video.mp4.enc           # Encrypted video (for decrypt-only tests)
├── audio/                  # One .m4a per dub
│   ├── ja-JP.m4a
│   ├── en-US.m4a
│   └── ...
├── subtitles/              # One .ass per language
│   ├── en-US.ass
│   └── ...
└── manifests/              # Additional dub manifests
    └── en-US.xml
```

### Adding more test data

Run the script again with a different name:

```sh
./scripts/setup-testdata.sh \
  -u "https://www.crunchyroll.com/watch/<another-ep>" \
  -e "<your-etp-rt-cookie>" \
  -n "<another-name>"
```

Tests that compare across episodes (KID diffs, PSSH diffs, etc.)
automatically run against every unique pair of discovered directories.
The more testdata you add, the more thorough the cross-comparison.

## Prerequisites for setup

- A Crunchyroll Premium account (free trial works)
- CDM files: either a `.wvd` file, or `client_id.bin` + `private_key.pem` in the project root
- [FFmpeg](https://www.ffmpeg.org) (only needed for normal downloading, not for setup mode)
