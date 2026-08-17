# Breez
> Instant, lightweight local tunneling for developers.

Breez is an open-source, developer-focused tunneling tool built in Go. It securely exposes your local HTTP development servers to the public internet through a high-performance WebSocket proxy without requiring complex networking setups, port forwarding, or third-party router configurations.

---

## Core Features

- 🌐 **Local DNS (`*.breez.local`)**: Zero-latency local domain resolution directly to your local ports without requiring internet.
- ⚡ **Instant Public URLs**: Expose local ports (`localhost:3000`) to the public internet via secure WebSocket tunnels.
- 🔀 **Smart Local Reverse Proxy**: Map custom names (`myapp.breez.local`) directly to any local service (`localhost:3000`).
- 🎨 **Elevated Terminal UI/UX**: Real-time request monitoring with color-coded HTTP method badges, status codes, and latency tracking.
- 🚀 **Single Lightweight Binary**: High-performance Go binary with minimal resource overhead.
- 💻 **Cross-Platform**: Support for Linux, macOS, and Windows.

---

## Installation

### Linux & macOS

Run the following command in your terminal to automatically download and install the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/devlopersabbir/breez/main/scripts/install.sh | bash
```

To uninstall:

```bash
curl -fsSL https://raw.githubusercontent.com/devlopersabbir/breez/main/scripts/uninstall.sh | bash
```

### Windows (PowerShell)

Run the following script in PowerShell:

```powershell
iwr -useb https://raw.githubusercontent.com/devlopersabbir/breez/main/scripts/install.ps1 | iex
```

To uninstall:

```powershell
iwr -useb https://raw.githubusercontent.com/devlopersabbir/breez/main/scripts/uninstall.ps1 | iex
```

---

## Usage & Examples

### 1. Local Development (`*.breez.local`)

Assign a clean, zero-latency local domain to your development server:

```bash
breez start 3000 --name myapp
```

**Output:**
```text
  ┌────────────────────────────────────────────────────────┐
  │  ☁  BREEZ  v0.2.0                    ● Online (Local)  │
  ├────────────────────────────────────────────────────────┤
  │  Local Domain:  http://myapp.breez.local               │
  │  Target Port:   http://localhost:3000                  │
  │  DNS Resolver:  127.0.0.1:53 (*.breez.local)           │
  ├────────────────────────────────────────────────────────┤
  │  [o] Open in Browser   [c] Copy URL   [q] Quit         │
  └────────────────────────────────────────────────────────┘

  Live Request Logs: (monitoring local traffic...)

  14:52:10  GET     /api/v1/health   [200 OK]        1.2ms
  14:52:12  POST    /api/v1/users    [201 Created]  14.8ms
  14:52:15  GET     /assets/logo.png [304 Cache]     0.8ms
```

### 2. Dual Tunnel (Local DNS + Public Internet)

Expose your service publicly while keeping local domain access:

```bash
breez serve 3000 --subdomain myapp
```

### 3. List Active Routes

```bash
breez list
```

### 4. Local DNS Resolver Setup (macOS)

One-command setup to route `*.breez.local` queries locally via `/etc/resolver`:

```bash
breez dns setup
breez dns status
```

### 5. Check Installed Version

```bash
breez version
```

---

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the [issues page](https://github.com/devlopersabbir/breez/issues) if you want to contribute, submit a pull request, or suggest new features.

---

## Final Note

Thank you for checking out **Breez**! Happy coding, and may your webhooks and local testing always flow smoothly. 🚀
