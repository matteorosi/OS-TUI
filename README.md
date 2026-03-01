# OS-TUI

A terminal user interface (TUI) for managing OpenStack clouds, built with Go, Bubble Tea, and Gophercloud.

![Go Version](https://img.shields.io/badge/go-1.22+-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)
![Release](https://img.shields.io/github/v/release/matteorosi/OS-TUI)

---

## Screenshots

**Sidebar — resource navigation with quick reference panel**
<img width="878" alt="Sidebar" src="https://github.com/user-attachments/assets/7e68bf7e-67c8-4012-9f0e-ad7f17e4bf55" />

**Limits — quota usage with colored bars**
<img width="890" alt="Limits" src="https://github.com/user-attachments/assets/8a850714-868e-439b-9746-1968bff48548" />

---

## Features

- **Full resource browsing** — navigate all major OpenStack services from a single interface. Every resource is drill-down navigable with `Enter`.
- **Global search** — press `/` from the sidebar to search across all services simultaneously. Queries run in parallel against Compute, Network, Storage, and more.
- **Relationship graph** — press `g` to visualize connected objects (volumes, ports, networks, floating IPs, load balancers) as an ASCII graph.
- **Topology view** — press `T` for a flat tree of all resources grouped by network.
- **Command mode** — press `:` for instant navigation with Tab autocomplete and inline suggestions.
- **Shell passthrough** — use `:!` to run any `openstack` CLI command. Output appears in a scrollable viewport inside the TUI.
- **Log streaming** — press `l` on any server for live console logs with pause/resume and adjustable refresh interval.
- **Token caching** — authentication tokens are cached to disk and reused across sessions. No re-auth on every launch.
- **Parallel client creation** — all service clients are initialized concurrently for fast startup.
- **Dynamic layout** — sidebar and tables adapt to terminal dimensions automatically.
- **Debug mode** — verbose output with `--debug` flag.
- **Context-sensitive help** — press `?` for keybindings relevant to the current view.

---

## Services

| Service | Resources |
|---|---|
| **Compute** | Servers, Images, Flavors, Keypairs, Hypervisors, Availability Zones, Limits |
| **Network** | Networks, Subnets, Routers, Ports, Floating IPs, Security Groups, Load Balancers |
| **Storage** | Volumes, Snapshots |
| **Identity** | Projects, Users, Token |
| **DNS** | Zones, Record Sets |

---

## Relationship Graph

Press `g` on any resource to see its connected objects as an ASCII graph.

```
╭──────────────────╮  ╭──────────────────╮  ╭──────────────────╮
│ Vol: /dev/vda    │  │ Vol: /dev/vdb    │  │ Vol: /dev/vdc    │
╰────────┬─────────╯  ╰────────┬─────────╯  ╰────────┬─────────╯
         │                     │                      │
╭────────┴─────────────────────┴──────────────────────┴──╮
│ Server: web-01                  Status: ACTIVE          │
╰────────────────────────────┬────────────────────────────╯
                             │
                  ╭──────────┴──────────╮
                  │ Port                │ ── ╭──────────────────╮
                  │ IP: 192.168.1.10    │    │ Net: prod-net    │
                  ╰─────────────────────╯    ╰──────────────────╯
                             │
                  ╭──────────┴──────────╮
                  │ FIP: 1.2.3.4        │
                  ╰─────────────────────╯

[g] close  [j/k] scroll
```

Available for: Servers, Networks, Volumes, Floating IPs, Load Balancers.

---

## Global Search

Press `/` from the sidebar to open the global search overlay. Type to search — queries run live in parallel across all OpenStack services.

Results are grouped by category (Servers, Networks, Volumes, etc.) and can be opened directly with `Enter`.

---

## Installation

**Requirements:** Go 1.22+

### Run directly

```bash
git clone https://github.com/matteorosi/OS-TUI.git
cd OS-TUI
go run ./cmd/ostui/main.go --cloud mycloud
```

### Build a binary

```bash
go build -o ostui ./cmd/ostui/main.go
./ostui --cloud mycloud
```

### Install globally

```bash
go install ./cmd/ostui
ostui --cloud mycloud
```

### Download pre-built binary

Download the latest release for your platform from [GitHub Releases](https://github.com/matteorosi/OS-TUI/releases).

---

## Quick Start

### 1. Find your cloud name

ostui reads from your existing `clouds.yaml`. To see which clouds are configured:

```bash
cat ~/.config/openstack/clouds.yaml

# clouds:
#   mycloud:        ← this is your cloud name
#     auth:
#       auth_url: https://...
```

### 2. Launch

```bash
go run ./cmd/ostui/main.go --cloud mycloud

# With debug output
go run ./cmd/ostui/main.go --cloud mycloud --debug

# Custom clouds.yaml location
OS_CLIENT_CONFIG_FILE=/path/to/clouds.yaml go run ./cmd/ostui/main.go --cloud mycloud
```

---

## Usage

### Flags

| Flag | Description |
|---|---|
| `--cloud <name>` | Cloud name from `clouds.yaml` (required) |
| `--project <name>` | OpenStack project to work with (optional) |
| `--debug` | Enable verbose debug logging |

### Keyboard shortcuts

| Key | Action |
|---|---|
| `j` / `k` | Move down / up |
| `Enter` | Open detail / drill-down |
| `Esc` | Go back |
| `g` | Open relationship graph |
| `l` | View server logs |
| `i` | Inspect (raw fields) |
| `y` | JSON view |
| `v` | Console URL |
| `/` | Global search (from sidebar) or filter list (in resource views) |
| `:` | Command mode |
| `?` | Context-sensitive help |
| `c` | Switch cloud |
| `T` | Topology view |
| `q` | Quit |

### Command mode

| Command | Alias | Target |
|---|---|---|
| `servers` | `srv` | Servers |
| `networks` | `net` | Networks |
| `volumes` | `vol` | Volumes |
| `images` | `img` | Images |
| `limits` | `quota` | Quota |
| `dns` | `zones` | DNS Zones |
| `loadbalancers` | `lb` | Load Balancers |
| `routers` | | Routers |
| `floatingips` | `fip` | Floating IPs |
| `secgroups` | `sg` | Security Groups |
| `topology` | `topo` | Topology view |
| `search` | | Global search |
| `quit` | | Exit |
| `!<cmd>` | | Run `openstack <cmd>` inline |

---

## Project structure

```
cmd/ostui/
  main.go               ← entry point
internal/
  cache/                ← in-memory TTL cache
  client/               ← OpenStack client interfaces (compute, network, storage, dns, lb…)
  config/               ← clouds.yaml loader
  ui/
    app.go              ← root model, state machine
    uiconst/            ← shared UI constants (column widths, table heights)
    common/             ← reusable components (table, confirm dialog, action menu)
    compute/            ← servers, flavors, keypairs, hypervisors, limits, logs, graph
    network/            ← networks, subnets, routers, ports, floating IPs, security groups
    storage/            ← volumes, snapshots
    image/              ← images
    identity/           ← projects, users, token
    dns/                ← zones, record sets
    loadbalancer/       ← load balancers, listeners, pools
    graph/              ← generic relationship graph
    search/             ← global search across all services
    shell/              ← openstack CLI passthrough
    topology/           ← topology view
```

---

## Stack

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** — TUI framework
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** — terminal styling
- **[Bubbles](https://github.com/charmbracelet/bubbles)** — UI components (table, textinput, spinner, list)
- **[gophercloud](https://github.com/gophercloud/gophercloud)** — OpenStack SDK
- **[errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup)** — parallel API queries
- **Go 1.22**

---

## Contributing

Contributions are welcome! Please follow the project's coding standards:

- Keep functions small and pure where possible.
- Write tests for new functionality.
- Run `go test ./...` and `go build ./...` before submitting a PR.

Open an issue to discuss major changes before submitting a pull request.

---

## License

Apache 2.0
