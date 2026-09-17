// Package modpackapi searches Modrinth and CurseForge for modpacks, so
// operators can find a ref to pass to `mcm modpack install` without
// already knowing its exact slug.
package modpackapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Result is one modpack search hit. Ref is exactly what should be passed
// to `mcm modpack install --source <Source> <Ref>`.
type Result struct {
	Source      string
	Ref         string
	Name        string
	Description string
	Downloads   int64
}

// SearchModrinth queries Modrinth's public search API for modpacks
// matching query. No API key is required.
func SearchModrinth(ctx context.Context, query string, limit int) ([]Result, error) {
	if limit <= 0 {
		limit = 10
	}

	endpoint := "https://api.modrinth.com/v2/search?" + url.Values{
		"query":  {query},
		"facets": {`[["project_type:modpack"]]`},
		"limit":  {fmt.Sprintf("%d", limit)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("modrinth: searching %q: %w", query, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("modrinth: searching %q: unexpected status %s", query, resp.Status)
	}

	var body struct {
		Hits []struct {
			Slug        string `json:"slug"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Downloads   int64  `json:"downloads"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("modrinth: decoding search response: %w", err)
	}

	results := make([]Result, 0, len(body.Hits))
	for _, h := range body.Hits {
		results = append(results, Result{
			Source:      "modrinth",
			Ref:         h.Slug,
			Name:        h.Title,
			Description: h.Description,
			Downloads:   h.Downloads,
		})
	}
	return results, nil
}

// Minecraft's gameId and the "Modpacks" classId, as assigned by
// CurseForge's catalog (stable, documented constants of their API).
const (
	curseForgeGameID         = 432
	curseForgeModpackClass   = 4471
	curseForgeSortPopularity = "2"
)

// SearchCurseForge queries CurseForge's search API for modpacks matching
// query. Requires an API key from https://console.curseforge.com.
func SearchCurseForge(ctx context.Context, apiKey, query string, limit int) ([]Result, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("searching CurseForge requires cf_api_key to be set in mcm's config")
	}
	if limit <= 0 {
		limit = 10
	}

	endpoint := "https://api.curseforge.com/v1/mods/search?" + url.Values{
		"gameId":       {fmt.Sprintf("%d", curseForgeGameID)},
		"classId":      {fmt.Sprintf("%d", curseForgeModpackClass)},
		"searchFilter": {query},
		"pageSize":     {fmt.Sprintf("%d", limit)},
		"sortField":    {curseForgeSortPopularity},
		"sortOrder":    {"desc"},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("curseforge: searching %q: %w", query, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("curseforge: searching %q: unexpected status %s", query, resp.Status)
	}

	var body struct {
		Data []struct {
			Slug          string  `json:"slug"`
			Name          string  `json:"name"`
			Summary       string  `json:"summary"`
			DownloadCount float64 `json:"downloadCount"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("curseforge: decoding search response: %w", err)
	}

	results := make([]Result, 0, len(body.Data))
	for _, m := range body.Data {
		results = append(results, Result{
			Source:      "curseforge",
			Ref:         m.Slug,
			Name:        m.Name,
			Description: m.Summary,
			Downloads:   int64(m.DownloadCount),
		})
	}
	return results, nil
}
