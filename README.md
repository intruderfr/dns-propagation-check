# dns-propagation-check

A fast Go CLI that checks DNS record propagation across **12+ global public resolvers** in parallel and reports any disagreements. Useful during DNS migrations, zero-downtime cutovers, and debugging geographic replication.

Unlike `dig` (single resolver) or browser-based propagation tools (slow, ad-riddled), this runs locally, fits in CI/CD pipelines, and exits non-zero when resolvers disagree — so you can gate deployments on "the change has fully propagated."

## Features

- Queries 14 public DNS resolvers in parallel (Google, Cloudflare, Quad9, OpenDNS, AdGuard, Yandex, CleanBrowsing, Comodo, Level3, Verisign, AWS, DNS.SB, Mullvad, Control D).
- Supports A, AAAA, CNAME, MX, TXT, NS, CAA, SOA, SRV record types.
- Parallel queries with configurable timeout.
- Human-readable table output or machine-readable JSON.
- Exit code reflects consistency — `0` if all resolvers agree, `2` if there's disagreement, `1` on internal error. Makes it CI-friendly.
- Optional watch mode (`--watch 30s`) polls until all resolvers converge, handy for "wait for DNS to propagate" steps in release scripts.
- Zero external Go dependencies — standard library only.

## Installation

### Go install (recommended)

```bash
go install github.com/intruderfr/dns-propagation-check@latest
```

The binary lands in `$(go env GOPATH)/bin` as `dns-propagation-check`.

### Build from source

```bash
git clone https://github.com/intruderfr/dns-propagation-check.git
cd dns-propagation-check
go build -o dnsprop .
./dnsprop example.com
```

## Usage

### Basic query

```bash
dns-propagation-check example.com
```

Queries the A record for `example.com` against every resolver and prints a table.

### Pick a record type

```bash
dns-propagation-check --type MX gmail.com
dns-propagation-check --type TXT _dmarc.example.com
dns-propagation-check -t AAAA cloudflare.com
```

### JSON output (for pipelines)

```bash
dns-propagation-check --json example.com | jq '.consistent'
```

### Custom timeout

```bash
dns-propagation-check --timeout 3s slow-dns.example.org
```

### Watch mode — wait for propagation

```bash
dns-propagation-check --watch 15s --max-wait 10m example.com
```

Re-polls every 15s up to 10 minutes and exits `0` as soon as all resolvers return identical answers. Drop this in your post-deploy script to gate health checks on true propagation, not just "the authoritative server accepted the change."

### Use custom resolvers

```bash
dns-propagation-check --resolvers 8.8.8.8,1.1.1.1,9.9.9.9 example.com
```

## Example output

```
$ dns-propagation-check example.com
Query: example.com (A)

RESOLVER           PROVIDER           ANSWER                         LATENCY   STATUS
1.1.1.1            Cloudflare         93.184.216.34                  14ms      OK
1.0.0.1            Cloudflare (2)     93.184.216.34                  15ms      OK
8.8.8.8            Google             93.184.216.34                  21ms      OK
8.8.4.4            Google (2)         93.184.216.34                  23ms      OK
9.9.9.9            Quad9              93.184.216.34                  31ms      OK
208.67.222.222     OpenDNS            93.184.216.34                  42ms      OK
94.140.14.14       AdGuard            93.184.216.34                  55ms      OK
77.88.8.8          Yandex             93.184.216.34                  112ms     OK
185.228.168.9      CleanBrowsing      93.184.216.34                  67ms      OK
8.26.56.26         Comodo             93.184.216.34                  89ms      OK
4.2.2.2            Level3             93.184.216.34                  44ms      OK
64.6.64.6          Verisign           93.184.216.34                  38ms      OK
185.222.222.222    DNS.SB             93.184.216.34                  73ms      OK
194.242.2.2        Mullvad            93.184.216.34                  118ms     OK

Result: CONSISTENT — all 14 resolvers returned the same answer.
```

When resolvers disagree:

```
Result: INCONSISTENT — 2 distinct answer sets observed.

  Group A (12 resolvers): 93.184.216.34
  Group B (2 resolvers — Yandex, Mullvad): 93.184.216.33

Hint: this usually means propagation is incomplete. Re-run with --watch to
wait for convergence.
```

## Why this exists

DNS changes don't propagate instantly. Authoritative updates happen in seconds, but cached answers at downstream resolvers can linger for the full TTL — sometimes longer when resolvers misrepresent TTL. During a cutover, "it works from my machine" is meaningless; what matters is whether global resolvers agree.

This tool answers that question with one command. Drop it in your CI/CD pipeline as a gate between "apply DNS change" and "swing production traffic," and you stop deploying into half-propagated DNS.

## Exit codes

| Code | Meaning |
|------|---------|
| `0`  | All resolvers returned identical answers (or watch converged) |
| `2`  | Resolvers disagree (or watch timed out before convergence) |
| `1`  | Internal error — invalid input, network failure, etc. |

## License

MIT — see [LICENSE](LICENSE).

## Author

**Aslam Ahamed** — Head of IT @ Prestige One Developments, Dubai
[LinkedIn](https://www.linkedin.com/in/aslam-ahamed/)
