# Crunchyroll API Reference, Daemon Auto-Discovery

> Discovered 2026-05-31 by probing the live API with an SG-based account.

## Authentication

All CMS/discover endpoints require a Bearer access token obtained from the auth endpoint:

```
POST https://www.crunchyroll.com/auth/v1/token
Authorization: Basic bm9haWhkZXZtXzZpeWcwYThsMHE6
Content-Type: application/x-www-form-urlencoded

device_id=<uuid>&device_type=Firefox+on+Linux&grant_type=etp_rt_cookie
```

Cookies: `device_id=<uuid>` & `etp_rt=<cookie-value>`

Response:
```json
{
  "access_token": "<jwt>",
  "refresh_token": "<etp_rt_value>",
  "expires_in": 300,
  "country": "SG",
  "token_type": "Bearer"
}
```

The access token expires in 5 minutes. Auto-refresh on 401 uses `GetAccessToken(*EtpRt)` (already implemented in `internal/lib/token.go` + `http_request.go`).

---

## 1. Series Search `GET /content/v2/discover/search`

**Primary endpoint for auto-discovery.** Searches across series, episodes, and movies. Returns results grouped by type.

### Request

```
GET https://www.crunchyroll.com/content/v2/discover/search
  ?q=Solo+Leveling
  &n=10
  &locale=en-US
```

Headers:
- `Authorization: Bearer <token>`
- `User-Agent: Mozilla/5.0 ...`

### Response

```json
{
  "total": 4,
  "data": [
    {
      "type": "top_results",
      "count": 16,
      "items": [ ... ]
    },
    {
      "type": "series",
      "count": 16,
      "items": [ ... ]
    },
    {
      "type": "movie_listing",
      "count": 0,
      "items": []
    },
    {
      "type": "episode",
      "count": 733,
      "items": [ ... ]
    }
  ],
  "meta": {}
}
```

### Relevant Item Shape (within `"type": "series"` items)

```json
{
  "type": "series",
  "id": "GDKHZEJ0K",
  "title": "Solo Leveling",
  "slug_title": "solo-leveling",
  "external_id": "SRZ.283771"
}
```

Fields used:
| Field | Value | Notes |
|---|---|---|
| `id` | `"GDKHZEJ0K"` | Series content ID -> fed directly to `GetSeasons()` |
| `title` | `"Solo Leveling"` | Title for matching against Sonarr |
| `slug_title` | `"solo-leveling"` | URL slug, used for URL construction / logging |
| `type` | `"series"` | Filter on this |

### Notes

- `top_results` group contains the same series items (usually identical to the `series` group). Either group can be used.
- The query is tokenizing, searching `"Solo Leveling"` also matches `"Solo Camping for Two"`. Ranking is by `search_metadata.score` descending. The first result is almost always the exact match.
- `movie_listing` and `episode` groups can be ignored for series discovery.
- Response omits unavailable series entirely (region-locked content is invisible).
- No pagination observed, `n` param governs both groups and per-group item count.

---

## 2. Series Seasons `GET /content/v2/cms/series/{id}/seasons`

Already implemented in `internal/lib/season.go`. Verified with discovered ID `GDKHZEJ0K`.

```
GET https://www.crunchyroll.com/content/v2/cms/series/GDKHZEJ0K/seasons
  ?force_locale=
  &preferred_audio_language=ja-JP
  &locale=en-US
```

### Response (abbreviated)

```json
{
  "data": [
    {
      "id": "GR19CPDWM",
      "season_number": 1,
      "title": "Solo Leveling",
      "season_display_number": "1",
      "season_sequence_number": 1,
      "slug_title": "solo-leveling"
    },
    {
      "id": "GY09CX813",
      "season_number": 2,
      "title": "Solo Leveling Season 2 -Arise from the Shadow-",
      "season_display_number": "2",
      "season_sequence_number": 2
    }
  ]
}
```

Flow: `searchItem.id → GetSeasons(id)` returns seasons → match by `season_number`.

---

## 3. Season Episodes `GET /content/v2/cms/seasons/{seasonId}/episodes`

Already implemented in `internal/lib/season.go`.

```
GET https://www.crunchyroll.com/content/v2/cms/seasons/GR19CPDWM/episodes
  ?preferred_audio_language=ja-JP
  &locale=en-US
```

Each episode has `episode_number`, `id` (the episode content GUID for the preferred audio locale), and `versions` array for alternate audio locales.

---

## 4. Episode Playback `GET /playback/v3/{episodeId}/web/firefox/play`

Already implemented in `internal/lib/episode.go`.

Returns DASH manifest URL, subtitle map, and Widevine token. No changes needed.

---

## 5. Episode Info `GET /content/v2/cms/objects/{id}`

Already implemented in `internal/lib/episode.go`. Returns metadata including `series_title`, `season_number`, `episode_number`, `versions` (dub variants), and `audio_locale`.

---

## Summary: End-to-End Discovery Flow

```
Sonarr Title ("Solo Leveling")
    ↓
GET /content/v2/discover/search?q=Solo+Leveling&n=10
    ↓ filter data[].items where type=="series"
    ↓ pick best match (exact title match + highest score)
Series content ID: "GDKHZEJ0K"
    ↓ persist in mappings.json
    ↓ 
GET /content/v2/cms/series/GDKHZEJ0K/seasons
    ↓ match by season_number
Season ID: "GR19CPDWM" (season 1)
    ↓
GET /content/v2/cms/seasons/GR19CPDWM/episodes
    ↓ match by episode_number
Episode ID: "G4VUQVJ2X" (S01E01)
    ↓
GET /playback/v3/G4VUQVJ2X/web/firefox/play
    ↓ Widevine + DASH download (existing pipeline)
```

## Notes & Edge Cases

1. **Region locking:** Search only returns content available in the token's region. If Sonarr has a series not in SG, it won't appear in search results.
2. **Dead/altered IDs:** If Crunchyroll removes or changes a series ID, the next poll after the state file entry was created will fail at `GetSeasons()`. The daemon should detect this (empty seasons) and re-search.
3. **Exact match reliability:** For unambiguous titles (90%+ of cases), the first search result is correct. Ambiguous short names (e.g. "Mushishi" → "Mushishi" + "Mushishi Zoku Shou") need the Levenshtein fallback.
4. **Search `n` param:** Controls items per group. 10 is sufficient for matching; higher values increase response size.
