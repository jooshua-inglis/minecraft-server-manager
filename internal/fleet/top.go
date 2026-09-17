package fleet

import (
	"context"
	"regexp"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/serverstore"
)

type ServerStats struct {
	Name          string
	Status        string
	CPUPercent    float64
	MemUsageBytes uint64
	MemLimitBytes uint64
	Players       string
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

		view := ServerStats{Name: name, Status: "not created", Players: "-"}

		info, err := f.Docker.Inspect(ctx, meta.ContainerName)
		if err != nil {
			return nil, err
		}
		if info == nil || info.State == nil {
			views = append(views, view)
			continue
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

		views = append(views, view)
	}
	return views, nil
}

func parsePlayerCount(rconOutput string) string {
	m := playerCountRE.FindStringSubmatch(rconOutput)
	if m == nil {
		return "-"
	}
	return m[1] + "/" + m[2]
}
