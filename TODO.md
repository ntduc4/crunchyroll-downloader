# TODO

Things that are planned to do if I ever get to it

Features present in [multi-downloader-nx](https://github.com/anidl/multi-downloader-nx) but missing here, roughly ordered by impact.

Created on 1.3.0, some feature refer to that version may no longer exist

## Auth

- [ ] Username/password login (currently only etp-rt cookie)
- [ ] Token-based auth (`--token`)

## Audio & subtitle selection

<!-- - [ ] Hardsub download (`--hslang`), download video with burned-in subtitles -->
- [ ] ISO 639-2 language codes as aliases (`eng`, `jpn`, `spa`) alongside BCP-47
- [ ] `--subsOnly` flag, download only subtitles (no video or audio)
- [ ] `--ignore-dubs` flag, skip dubbed content

## Subtitle processing

- [x] Font download and embedding (`--dlFonts`)
- [x] ASS subtitle fixes:
  - [x] `--layoutResFix`
  - [x] `--scaledBorderAndShadowFix`
  - [x] `--srtAssFix` (Closed Caption converter fix)
  - [x] `--originalScriptFix`
  - [x] `--subtitleTimestampFix`
- [ ] VTT→ASS conversion (`--noASSConv` toggle)
- [ ] Configurable font size (`--fontSize`)

## Muxing

- [ ] MKVToolNix support (`--forceMuxer mkvmerge`)
- [ ] MP4 output option (`--mp4`)
- [ ] User-selectable default audio/subtitle track (`--defaultAudio`, `--defaultSub`)
- [ ] `--skipmux` / `--skipSubMux` / `--nocleanup` fine-grained control
- [ ] Multi-dub A/V sync (`--syncTiming`)

## Output

- [ ] Filename template system with variables (`${showTitle}`, `${season}`, `${episode}`, `${height}`, `${service}`, etc.)
- [ ] Configurable zero-padding for episode numbers (`--numbers`)
- [ ] Custom template variable overrides (`--override`)

## Content

- [ ] Chapter fetching and embedding (`--chapters`)

## Episode selection

- [ ] Episode ranges (`-e 1-4`) and lists (`-e 1,2,3,4`)
- [ ] Special episode selection (`-e S1-S4`)
- [ ] `--but` flag (download everything except the selected episodes)
- [ ] `--absolute`, use absolute episode numbers instead of season-indexed

## Download

- [ ] `--novids` / `--noaudio` / `--nosubs` toggles
- [ ] Configurable part size (`--partsize`)
- [ ] Proxy support (`--proxy`, `--proxyAll`)
- [ ] `--overwrite` flag, force overwrite existing files instead of skipping
- [ ] `--tmpDir` flag, custom directory for temporary download files

## Infrastructure

- [ ] Configurable timeout and wait time
- [ ] `--tsd` (Total Session Death), kill all active streaming sessions

## Sonarr integration

### Near-term (custom script)

- [ ] Output naming compatible with Sonarr's import parser (`{Series Title} - S{season:00}E{episode:00}`)
- [ ] Episode metadata format compatible with Sonarr's expected structure
- [ ] Post-download script Sonarr calls via Connect on grab/import (one-shot, no daemon)

### Far future (download client)

- [ ] Daemon mode (`--daemon` / `--serve`) for a persistent process
- [ ] HTTP API with Sonarr-compatible endpoints: queue status, history, health check
- [ ] API key auth matching Sonarr's `APIKey` header
- [ ] Episode lookup by series name, season, and episode number (search without a URL)
- [ ] Download request queuing with progress reporting
- [ ] Appear in Sonarr's Download Client UI alongside qBittorrent / SABnzbd
