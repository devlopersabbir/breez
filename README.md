# Breez
> Instant, lightweight local tunneling for developers.

Breez is an open-source, developer-focused tunneling tool built in Go. It securely exposes your local HTTP development servers to the public internet through a high-performance WebSocket proxy without requiring complex networking setups, port forwarding, or third-party router configurations.

---

## Core Features

- ⚡ **Instant Public URLs**: Expose local ports (`localhost:3000`) to the public internet in seconds.
- 🚀 **Lightweight & Fast**: Single binary written in Go with minimal resource overhead.
- 🔄 **Multiplexed WebSockets**: Efficient HTTP request & response framing over WebSocket connections.
- 🌐 **Random Subdomains**: Auto-allocated clean subdomains for every active tunnel session.
- 💻 **Cross-Platform**: Support for Linux, macOS, and Windows.
- 🔒 **Secure Protocol**: Built for secure HTTP/HTTPS transport.

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

### Expose a Local Development Server

To expose a local web application running on port `3000`:

```bash
breez serve 3000
```

**Output:**

```text
✔ Tunnel Created Successfully!

Local:  http://localhost:3000
Public: http://k72mfa.breez.run

Status: Connected (Press Ctrl+C to stop)
---------------------------------------------
➜ GET / 200 (12ms)
➜ POST /api/webhooks 200 (35ms)
```

### Request a Specific Subdomain

```bash
breez serve 3000 --subdomain myapp
```

### Check Installed Version

```bash
breez version
```

---

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the [issues page](https://github.com/devlopersabbir/breez/issues) if you want to contribute, submit a pull request, or suggest new features.

---

## Final Note

Thank you for checking out **Breez**! Happy coding, and may your webhooks and local testing always flow smoothly. 🚀
