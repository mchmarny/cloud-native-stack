# NVIDIA Cloud Native Stack

NVIDIA Cloud Native Stack (CNS) combines production-ready documentation with Go-based tooling (CLI, API server, recipe logic, and supporting libraries) to help deploy, validate, and operate GPU-accelerated Kubernetes clusters.

## Repository Overview

- `cmd/` – Go entry points for the CLI (`eidos`) and API server.
- `pkg/` – Shared libraries covering recipe logic, collector, logging, measurement handling, and more.
- `deployments/` – Kubernetes manifests for CNS services.
- `docs/` – Installation guides, Ansible playbooks, optimizations, troubleshooting, and the full platform README.
- `tools/` – Utility scripts for release automation and remote builds.

## Installer 

Quickest (and easiest) way to install `eidos` CLI is to use the automated installer. Copy and run the following command:

```shell
curl -sfL https://raw.githubusercontent.com/mchmarny/cloud-native-stack/refs/heads/main/install | bash -s --
```

## Documentation

All deployment, operations, and troubleshooting material now lives in [docs/README.md](docs/README.md). That document includes hardware prerequisites, component matrices, installation guides, and references to every playbook and optimization guide.

## Users: Choosing How to Interact

### Server API

1. Build the binary:
	```bash
	go build -o bin/eidos-server ./cmd/server
	```
2. Export configuration (defaults live in `pkg/server/config.go`):
	```bash
	export PORT=8080
	export LOG_LEVEL=info
	```
3. Run the server:
	```bash
	./bin/eidos-server
	```
4. Request recipe:
	```bash
	curl "http://localhost:8080/v1/recipe?os=ubuntu&service=eks"
	```

### CLI (`eidos`)

1. Build the CLI:
	```bash
	go build -o bin/eidos ./cmd/eidos
	```
2. Get help:
	```bash
	./bin/eidos --help
	```
3. Snapshot the local system (example):
	```bash
	./bin/eidos snapshot --output snapshot.json
	```

## Contributing

Issues and pull requests are welcome. Please review [CONTRIBUTING.md](CONTRIBUTING.md) for coding standards, DCO requirements, and the preferred development workflow before opening a change.

## Need Help?

- Track releases and changelog details via [GitHub Releases](https://github.com/NVIDIA/cloud-native-stack/releases).
- File questions or feature requests in [GitHub Issues](https://github.com/NVIDIA/cloud-native-stack/issues).
- For the historical CNS documentation, jump straight to [docs/README.md](docs/README.md).
