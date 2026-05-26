# Changelog

## 1.3.1

- Fix xml file perssiting when no debug flag was set

## 1.3.0

- Multi-language audio support: download all available dubs into a single MKV
- Multi-language subtitle support: download all available subtitle languages
- `-audio-lang` and `-subs-lang` now default to `all`; specific locales still accepted
- `-setup-dir <path>` flag: saves all tracks + metadata JSONs to a directory, skips muxing
- `scripts/setup-testdata.sh` to generate offline test fixtures from any episode
- Integration test suite, discovers all testdata directories, no hardcoded episode names
- Fixed panic on subtitle entries with empty URLs
- Removed content ID length gate, supports all Crunchyroll ID formats (9, 10, 14 chars)
- Project restructured to Go conventions: `cmd/`, `internal/lib/`, `testdata/`
- Added `CONTRIBUTING.md` with developer setup guide

## 1.2.0

- Parallel segment downloads (10 workers) for much faster downloads
- Retry with backoff on connection errors instead of crashing
- Added `--urls` flag to batch download from a text file with one URL per line
- Invalid URLs in batch mode are skipped instead of stopping the whole process

## 1.1.1

- Optimized code, tried to handle errors
- Some random fixes
- Added a way to automatically refetch an access token if the current one expires

## 1.1.0

- Added support for downloading entire seasons
- Fixed MPD parsing
- Temporary downloaded files (video, audio segments and subtitles) are now stored in the OS temporary files then deleted
- Fixed FFmpeg merge command
- Docs improvements
- Support for `device_id.bin` and `private_key.pem` files

## 1.0.0

Initial release
