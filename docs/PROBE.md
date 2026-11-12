# Go discovery diagnostic

This repository includes a portable Go diagnostic executable. It is **not** the desktop UI and is not a release artifact. It uses the exact management identity endpoints verified from Zero1Local source.

```bash
go test -race ./...
go vet ./...
go run ./cmd/z1l-discover -probe 192.168.0.100
go run ./cmd/z1l-discover -scan
```

The commands make read-only HTTP GET requests only. No server configuration or shares are changed. A successful result includes the candidate endpoint, name, source-reported version, and setup flag, encoded as JSON. A fake local HTTP service can imitate these endpoints; **never** treat a probe response as cryptographic trust or mount authorization.

LAN scanning visits the current /24 containing each eligible private-network interface, probes TCP port 80 then legacy port 8088, uses up to 24 workers and has a 750ms per-request timeout in the diagnostic command. It does not reach other VLANs unless those interfaces are directly connected; manual endpoints support already routed private networks. The future GUI must also support explicitly authorized CIDR scans.

Cross-platform native GUI and share mount work remain gated as documented in ROADMAP.md.
