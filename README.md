# Dev HTML Proxy

`dev-html-proxy` is a Go-based development tool designed to facilitate the testing and development of web applications. It acts as a proxy server that can automatically trigger web page reloads upon receiving a specific request. This tool is especially useful for developers working on web interfaces, allowing for a more efficient iteration process.

## Features

- **Proxy Server**: Forwards all web requests to a specified target host, enabling the use of local development tools with live sites.
- **WebSocket Support**: Utilizes WebSockets to communicate with client browsers for real-time interaction.
- **Automatic Page Refresh**: Provides an endpoint to trigger a refresh command to all connected clients, simplifying the process of testing changes.

## Prerequisites

Before installing `dev-html-proxy`, ensure you have Go 1.22 or newer installed on your system. You can check your Go version by running:

```bash
go version
```

## Installation

With Go 1.22 or newer, installation is straightforward using the `go install` command. This command will compile and install the `dev-html-proxy` binary to your `$GOPATH/bin` directory.

```bash
go install github.com/adyatlov/dev-html-proxy@latest
```

Ensure that your `$GOPATH/bin` is in your system's `PATH` so that you can run `dev-html-proxy` from any terminal.

## Usage

After installation, start `dev-html-proxy` by running:

```bash
dev-html-proxy -target="http://example.com" -port="8481" -trigger-port="8482"
```

- `-target` specifies the target host to proxy to.
- `-port` is the HTTP port for serving browser requests.
- `-trigger-port` is the HTTP port for triggering page refreshes.

To trigger a refresh on all connected clients, use the following command:

```bash
curl http://localhost:8482/
```

## Contributing

Contributions are welcome! Feel free to report issues, submit fixes, or propose new features. Please follow the standard GitHub pull request process to submit your contributions.
