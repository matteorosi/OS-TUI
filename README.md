# ostui

A terminal user interface (TUI) for managing OpenStack clouds, built with Go, Bubble Tea, and Gophercloud. Inspired by k9s.

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

## Installation

```bash
# Clone the repository
git clone https://github.com/your-org/ostui.git
cd ostui

# Build the binary
go build -o ostui ./cmd/ostui

# (Optional) Install globally
go install ./cmd/ostui
```

**Requirements:** Go 1.22+

---

## Quick Start

```bash
# Run against a cloud defined in your clouds.yaml
ostui --cloud mycloud

# Specify a project and enable debug output
ostui --cloud mycloud --project myproject --debug
```

Authentication uses your existing `clouds.yaml`:

```bash
# Default location
~/.config/openstack/clouds.yaml

# Custom location via environment variable
OS_CLIENT_CONFIG_FILE=/path/to/clouds.yaml ostui --cloud mycloud
```

---

## Usage

### Flags

| Flag | Description |
|---|---|
| `--cloud <name>` | Cloud configuration name from `clouds.yaml` (required) |
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
cmd/ostui/          ← entry point
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
