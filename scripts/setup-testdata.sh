#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BIN="$PROJECT_DIR/crunchyroll-downloader"

usage() {
  echo "Usage: $0 -u <url> -e <etp-rt> -n <name> [-q <video-quality>] [-a <audio-quality>]"
  echo ""
  echo "  -u   Crunchyroll episode URL"
  echo "  -e   etp_rt cookie value"
  echo "  -n   Test case name (e.g. ep1)"
  echo "  -q   Video quality (default: 1080p)"
  echo "  -a   Audio quality (default: 192k)"
  echo ""
  echo "Downloads all audio dubs, subtitles, video, and metadata into:"
  echo "  testdata/<name>/"
  echo ""
  echo "Example:"
  echo "  $0 -u 'https://www.crunchyroll.com/watch/...' -e 'your-cookie' -n ep1"
  exit 1
}

URL=""
ETPRT=""
NAME=""
VIDEO_Q="1080p"
AUDIO_Q="192k"

while getopts "u:e:n:q:a:h" opt; do
  case $opt in
    u) URL="$OPTARG" ;;
    e) ETPRT="$OPTARG" ;;
    n) NAME="$OPTARG" ;;
    q) VIDEO_Q="$OPTARG" ;;
    a) AUDIO_Q="$OPTARG" ;;
    h) usage ;;
    *) usage ;;
  esac
done

if [ -z "$URL" ] || [ -z "$ETPRT" ] || [ -z "$NAME" ]; then
  usage
fi

if [ ! -f "$BIN" ]; then
  echo "Building downloader binary..."
  (cd "$PROJECT_DIR" && go build -o crunchyroll-downloader ./cmd/crunchyroll-downloader)
fi

SETUP_DIR="$PROJECT_DIR/testdata/$NAME"

echo "=== Setup: $NAME ==="
echo "URL:        $URL"
echo "Video:      $VIDEO_Q"
echo "Audio:      $AUDIO_Q"
echo "Output:     $SETUP_DIR"
echo ""

rm -rf "$SETUP_DIR"

"$BIN" \
  -url "$URL" \
  -etp-rt "$ETPRT" \
  -setup-dir "$SETUP_DIR" \
  -video-quality "$VIDEO_Q" \
  -audio-quality "$AUDIO_Q" \
  -audio-lang all \
  -subs-lang all

echo ""
echo "=== Setup complete ==="
echo "Files saved to: $SETUP_DIR"
echo ""
echo "Directory contents:"
find "$SETUP_DIR" -type f | sort
echo ""
echo "Run integration test:"
echo "  go test ./internal/lib/ -run TestIntegration -v"
