package fleet

import (
	"context"
	"regexp"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/serverstore"
)

type ServerStats struct {
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemUsageBytes uint64  `json:"mem_usage_bytes"`
	MemLimitBytes uint64  `json:"mem_limit_bytes"`
	Players       string  `json:"players"`
}

var playerCountRE = regexp.MustCompile(`There are (\d+) of a max of (\d+) players online`)

// Top takes one resource-usage snapshot across the whole fleet, combining
// Docker-level CPU/memory stats with an in-game player count (via RCON)
// for whichever servers are currently running.
func (f *Fleet) Top(ctx context.Context) ([]ServerStats, error) {
	names, err := serverstore.List(f.Root)
	if err != nil {
		return nil, err
	}

	var views []ServerStats
	for _, name := range names {
		meta, err := serverstore.Load(f.Root, name)
		if err != nil {
			return nil, err
		}
		view, err := f.statsFor(ctx, name, meta)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

// SingleStats takes the same resource-usage snapshot as Top, for one
// server, so a live per-server view (e.g. the web dashboard's stats
// stream) doesn't have to pay for inspecting the whole fleet on every
// tick.
func (f *Fleet) SingleStats(ctx context.Context, name string) (ServerStats, error) {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return ServerStats{}, err
	}
	return f.statsFor(ctx, name, meta)
}

func (f *Fleet) statsFor(ctx context.Context, name string, meta *serverstore.Metadata) (ServerStats, error) {
	view := ServerStats{Name: name, Status: "not created", Players: "-"}

	info, err := f.Docker.Inspect(ctx, meta.ContainerName)
	if err != nil {
		return ServerStats{}, err
	}
	if info == nil || info.State == nil {
		return view, nil
	}
	view.Status = string(info.State.Status)

	if info.State.Running {
		if stats, err := f.Docker.Stats(ctx, meta.ContainerName); err == nil {
			view.CPUPercent = stats.CPUPercent
			view.MemUsageBytes = stats.MemUsageBytes
			view.MemLimitBytes = stats.MemLimitBytes
		}
		if out, err := f.Exec(ctx, name, "list"); err == nil {
			view.Players = parsePlayerCount(out)
		}
	}

	return view, nil
}

func parsePlayerCount(rconOutput string) string {
	m := playerCountRE.FindStringSubmatch(rconOutput)
	if m == nil {
		return "-"
	}
	return m[1] + "/" + m[2]
}
