// Package mojang resolves Minecraft usernames to UUIDs, which is needed
// whenever mcm edits whitelist.json/ops.json/banned-players.json directly
// on disk (RCON can resolve names itself; the offline file-editing path
// can't).
package mojang

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

var ErrNotFound = errors.New("mojang: no such Minecraft account")

// LookupUUID resolves a username to its Mojang account UUID (dashed
// form) and canonical (correctly-cased) username. It requires network
// access to api.mojang.com.
func LookupUUID(ctx context.Context, username string) (uuid, canonicalName string, err error) {
	endpoint := "https://api.mojang.com/users/profiles/minecraft/" + url.PathEscape(username)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("mojang: looking up %q: %w", username, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", "", ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("mojang: looking up %q: unexpected status %s", username, resp.Status)
	}

	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", "", fmt.Errorf("mojang: decoding response for %q: %w", username, err)
	}

	return dashUUID(body.ID), body.Name, nil
}

// OfflineUUID computes the deterministic UUID a server running with
// online-mode=false would assign to username, using the same algorithm
// Minecraft itself uses (a name-based v3-style UUID over
// "OfflinePlayer:<username>").
func OfflineUUID(username string) string {
	sum := md5.Sum([]byte("OfflinePlayer:" + username))
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	return dashUUID(hex.EncodeToString(sum[:]))
}

func dashUUID(compact string) string {
	if len(compact) != 32 {
		return compact
	}
	return compact[0:8] + "-" + compact[8:12] + "-" + compact[12:16] + "-" + compact[16:20] + "-" + compact[20:32]
}
