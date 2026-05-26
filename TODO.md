# TODO

Things that are planned to do if I ever get to it

Features present in [multi-downloader-nx](https://github.com/anidl/multi-downloader-nx) but missing here, roughly ordered by impact.

Created on 1.3.0, some feature refer to that version may no longer exist

## Auth

- [ ] Username/password login (currently only etp-rt cookie)
- [ ] Token-based auth (`--token`)

## Discovery

- [ ] Built-in search (`--search`) with pagination and type filter (series, movie, episode)
- [ ] `--movie-listing` support for movie downloads
- [ ] `--extid` / `--externalid` for legacy Crunchyroll IDs

## Audio & subtitle selection

- [ ] Hardsub download (`--hslang`), download video with burned-in subtitles
- [ ] ISO 639-2 language codes as aliases (`eng`, `jpn`, `spa`) alongside BCP-47

## Subtitle processing

- [ ] Font download and embedding (`--dlFonts`)
- [ ] ASS subtitle fixes:
  - [ ] `--layoutResFix`
  - [ ] `--scaledBorderAndShadowFix`
  - [ ] `--srtAssFix` (Closed Caption converter fix)
  - [ ] `--originalScriptFix`
  - [ ] `--subtitleTimestampFix`
- [ ] VTT→ASS conversion (`--noASSConv` toggle)
- [ ] Configurable font size (`--fontSize`)

## Quality & streams

- [ ] Per-device endpoint selection for video (`--vstream`) and audio (`--astream`), e.g. androidtv gives CBR video, android gives 192k audio
- [ ] Numeric quality levels (`-q 0` = max)

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

## Infrastructure

- [ ] GUI mode
- [ ] HiDive and ADN support (if ever multi-service)
- [ ] Configurable timeout and wait time
- [ ] `--tsd` (Total Session Death), kill all active streaming sessions

## Sonarr integration

- [ ] Custom script support for Sonarr, acceptable as a post-download script that Sonarr calls after an episode is grabbed
- [ ] Webhook or API endpoint for Sonarr to trigger downloads remotely
- [ ] Matching Sonarr naming conventions for automatic import (`{Series Title} - S{season:00}E{episode:00}`)
- [ ] Episode metadata format compatible with Sonarr's expected structure
