# OS-Tui

A terminal user interface (TUI) for managing OpenStack clouds, built with Go, Bubble Tea, and Gophercloud.

![Go Version](https://img.shields.io/badge/go-1.22+-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)

---

## Screenshots

**Sidebar — resource navigation**
<img width="878" alt="Sidebar" src="https://github.com/user-attachments/assets/7e68bf7e-67c8-4012-9f0e-ad7f17e4bf55" />

**Limits — quota usage with colored bars**
<img width="890" alt="Limits" src="https://github.com/user-attachments/assets/8a850714-868e-439b-9746-1968bff48548" />

---


## Features

- **Full resource browsing** — navigate all major OpenStack services from a single interface. Every resource is drill-down navigable with `Enter`.
- **Relationship graph** — press `g` to visualize connected objects (volumes, ports, networks, floating IPs, load balancers) as an ASCII graph.
- **Command mode** — press `:` for instant navigation with Tab autocomplete and inline suggestions.
- **Shell passthrough** — use `:!` to run any `openstack` CLI command. Output appears in a scrollable viewport inside the TUI.
- **Log streaming** — press `l` on any server for live console logs with pause/resume and adjustable refresh interval.
- **Token caching** — authentication tokens are cached to disk and reused across sessions. No re-auth on every launch.
- **Parallel client creation** — all service clients are initialized concurrently for fast startup.
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

## Installation

**Requirements:** Go 1.22+

### Run directly

```bash
git clone https://github.com/your-org/ostui.git
cd ostui
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

---

## Quick Start

### 1. Find your cloud name

ostui reads from your existing `clouds.yaml`. To see which clouds are configured:

```bash
# Default location
cat ~/.config/openstack/clouds.yaml

# The cloud names are the top-level keys, e.g.:
# clouds:
#   mycloud:        ← this is your cloud name
#     auth:
#       auth_url: https://...
#   production:     ← or this one
#     auth:
#       ...
```

You can also check which cloud is currently active:

```bash
echo $OS_CLOUD
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
| `--cloud <n>` | Cloud name from `clouds.yaml` (required) |
| `--project <n>` | OpenStack project to work with (optional) |
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
| `/` | Filter list |
| `:` | Command mode |
| `?` | Context-sensitive help |
| `c` | Switch cloud |
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
| `quit` | | Exit |
| `!<cmd>` | | Run `openstack <cmd>` inline |

---

## Project structure

```
cmd/ostui/
  main.go           ← entry point
internal/
  client/           ← OpenStack client interfaces (compute, network, storage, dns, lb…)
  ui/
    app.go          ← root model, state machine
    compute/        ← servers, flavors, keypairs, hypervisors, limits, logs, graph
    network/        ← networks, subnets, routers, ports, floating IPs, security groups
    storage/        ← volumes, snapshots
    image/          ← images
    identity/       ← projects, users, token
    dns/            ← zones, record sets
    loadbalancer/   ← load balancers, listeners, pools
    graph/          ← generic relationship graph
    shell/          ← openstack CLI passthrough
```

---

## Stack

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** — TUI framework
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** — terminal styling
- **[gophercloud](https://github.com/gophercloud/gophercloud)** — OpenStack SDK
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
