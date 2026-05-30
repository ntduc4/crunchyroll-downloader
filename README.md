# Crunchyroll Downloader

Downloads anime from Crunchyroll and outputs them in a MKV file.

You won't be banned or anything, I downloaded all Kaguya-Sama seasons to test during 30 mins and everything went fine

## Features

- Supports downloading one or all audio and subtitle languages
- Supports choosing the audio and video quality
- Decrypts Widevine DRM (requires: a `.wvd` file or `client_id.bin` and `private_key.pem` files)
- Adds metadata (like episode name) to the MKV container
- Parallel segment downloads (10 workers) for faster downloads
- Retry with backoff on connection errors
- Batch download from a list of URLs
- ASS subtitle processing: fixes ScaledBorderAndShadow, SRT conversion artifacts, OriginalScript; optional PlayRes and timestamp fixes
- Font download and embedding into MKV (`--dl-fonts`)

## Requirements

- [FFmpeg](https://www.ffmpeg.org/download.html#get-packages)
- To download Premium-only content, a Crunchyroll Premium account. No, this can't be bypassed and a free trial should be enough
- Either a `.wvd` file, or a `client_id.bin` and `private_key.pem`

## Download

Check the [latest release](https://github.com/CuteTenshii/crunchyroll-downloader/releases/latest) and download the file that corresponds to your OS.

## Usage

- Open a Terminal/Command prompt, and go to the folder where you downloaded the binary/cloned the repo
- Run the program with the options you want:
```shell
Usage of ./crunchyroll-downloader:
  -audio-lang string
        Audio language, comma-separated list, or 'all' (default "all")
  -audio-quality string
        Audio quality (default "192k")
  -dl-fonts
        Download subtitle fonts from Google Fonts and embed them in the MKV
  -etp-rt string
        The "etp_rt" cookie value of your account
  -layout-res-fix
        Override PlayRes to match video resolution (default false)
  -no-ass-fix
        Disable all ASS subtitle processing
  -original-script-fix
        Comment out OriginalScript line that can confuse renderers (default true)
  -scaled-border-and-shadow-fix
        Fix ScaledBorderAndShadow for correct border/shadow scaling (default true)
  -season int
        Season number. Not used if an episode link is entered
  -srt-ass-fix
        Clean up SRT-to-ASS conversion artifacts (default true)
  -subs-lang string
        Subtitles language, comma-separated list, or 'all' (default "all")
  -subtitle-timestamp-fix
        Auto-detect and fix subtitle timing offset (default false)
  -url string
        URL of the episode/season to download
  -urls string
        Path to a text file with one URL per line
  -video-quality string
        Video quality (default "1080p")
```

Language codes use the `ja-JP`, `en-US`, `de-DE` format. Use `all` to download every available track.
Use a comma-separated list to select multiple specific tracks: `--audio-lang ja-JP,en-US --subs-lang en-US,pt-BR`.

### ASS subtitle processing flags

Crunchyroll's ASS subtitles are authored for their own player and can render incorrectly in standard video players. These flags fix common issues:

| Flag | Default | Behaviour |
|---|---|---|
| `--scaled-border-and-shadow-fix` | `true` | Sets `ScaledBorderAndShadow: yes` so border widths and shadow distances scale correctly with video resolution |
| `--srt-ass-fix` | `true` | Removes zero-duration events and normalizes newlines from SRT-to-ASS conversion |
| `--original-script-fix` | `true` | Comments out the `OriginalScript` metadata that can confuse some renderers |
| `--layout-res-fix` | `false` | Overrides `PlayResX`/`PlayResY` in the ASS header to match the actual video dimensions |
| `--subtitle-timestamp-fix` | `false` | Auto-detects a constant offset (>2s) and shifts all subtitle timestamps to start at 0 |
| `--no-ass-fix` | `false` | Disables all of the above — passes through the original ASS file as-served |

When `--dl-fonts` is enabled, font names referenced in the ASS styles and `\fn` override tags are downloaded from Google Fonts and attached to the MKV as embedded attachments.

Ex: to download the first season of *Hell's Paradise*:
```shell
./crunchyroll-downloader --url https://www.crunchyroll.com/series/GJ0H7Q5ZJ/hells-paradise --season 1 --etp-rt replace_this
```

To download a specific episode:
```shell
./crunchyroll-downloader --url https://www.crunchyroll.com/watch/GE00198973JAJP/dawn-and-confusion --etp-rt replace_this
```

To batch download from a file (one URL per line):
```shell
./crunchyroll-downloader --urls list.txt --etp-rt replace_this --subs-lang pt-BR
```

To download every available audio dub and subtitle language into a single MKV:
```shell
./crunchyroll-downloader --url https://www.crunchyroll.com/watch/GE00198973JAJP/dawn-and-confusion --etp-rt replace_this --audio-lang all --subs-lang all
```

## Building

### Requirements

- [Go](https://go.dev/dl/)

### Guide

- Clone this repository
- Open a Terminal/Command prompt, and go to the folder where you cloned the repo
- Run `go build ./cmd/crunchyroll-downloader`

## Help

### How do I get my `etp_rt` cookie?

- Go to https://crunchyroll.com
- Open Developer Tools
- Firefox: Go to *Storage* then *Cookies*<br />Chrome: Go to *Application* then *Cookies*
- Select the Crunchyroll domain, then copy the `etp_rt` cookie value

![](.github/screenshots/etp-rt-cookie.png)

### What is a `.wvd` file and do I really need one?

Yes, Crunchyroll uses DRM-only content. This file is used to get a Widevine license, which gives the keys to decrypt the media.

If you don't have a rooted Android device or are just lazy, search "ready to use cdms" and you'll find plenty of websites providing those files.
