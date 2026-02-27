# OSTUI – OpenStack TUI Management Tool

A terminal user interface (TUI) for managing OpenStack clouds, built with Go, Bubble Tea, and Gophercloud.

## Features
- Manage compute, network, storage, identity, image, limits, DNS, and load balancer services.
- Parallel client creation for fast startup.
- Token caching with automatic refresh.
- Debug mode with verbose output.
- Interactive TUI with keyboard shortcuts and context‑sensitive help.
- Configurable cloud selection via `--cloud` flag or `OS_CLOUD` environment variable.

## Installation

```bash
# Clone the repository
git clone https://github.com/your-org/ostui.git
cd ostui

# Build the binary
go build ./...

# (Optional) Install globally
go install ./...
```

## Quick Start

```bash
# Run the TUI against a cloud defined in your clouds.yaml
ostui --cloud mycloud
```

You can also specify a project name or enable debug output:

```bash
ostui --cloud mycloud --project myproject --debug
```

## Usage

- `--cloud <name>` (required): Name of the cloud configuration in `clouds.yaml`.
- `--project <name>` (optional): Identifier for the OpenStack project you want to work with.
- `--debug`: Enable verbose debug logging.
- Environment variable `OS_CLIENT_CONFIG_FILE` can be used to point to a custom `clouds.yaml`.

The TUI presents a list of resources (servers, networks, volumes, etc.) that can be inspected, filtered, and acted upon using keyboard shortcuts. Press `?` for the help overlay.

## Contributing

Contributions are welcome! Please follow the project's coding standards:

- Keep functions small and pure where possible.
- Write tests for new functionality.
- Run `go test ./...` and `go build ./...` before submitting a PR.

Open an issue to discuss major changes before submitting a pull request.

## License

The license for this project has not been specified yet. Add an appropriate license file (e.g., MIT, Apache‑2.0) and update this section accordingly.
