package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	targetHost  string
	httpPort    string
	triggerPort string
	showHelp    bool
)

func init() {
	flag.StringVar(&targetHost, "target", "", "The target host to proxy to (e.g. http://example.com).")
	flag.StringVar(&httpPort, "port", "8481", "HTTP port for serving browser requests.")
	flag.StringVar(&triggerPort, "trigger-port", "8482", "HTTP port for triggering page refresh.")
	flag.BoolVar(&showHelp, "help", false, "Show help")
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	clients   = make([]*websocket.Conn, 0)
	clientsMu sync.Mutex
)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error upgrading to WebSocket: %v", err)
		return
	}
	defer ws.Close()

	clientsMu.Lock()
	clients = append(clients, ws)
	clientsMu.Unlock()

	for {
		if _, _, err := ws.NextReader(); err != nil {
			ws.Close()
			removeClient(ws)
			break
		}
	}
}

func removeClient(ws *websocket.Conn) {
	clientsMu.Lock()
	for i, client := range clients {
		if client == ws {
			clients = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	clientsMu.Unlock()
}

func broadcastMessage(message string) {
	slog.Debug("Broadcasting", "message", message)
	clientsMu.Lock()
	for _, client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
			slog.Error("WebSocket send error: %v", err)
			client.Close()
			removeClient(client)
		}
	}
	clientsMu.Unlock()
}

const jsCode = `
<script>
    function connectWebSocket() {
        // Create a WebSocket connection
        var ws = new WebSocket('ws://' + window.location.host + '/dev-html-proxy-ws');

        // Message received on the WebSocket
        ws.onmessage = function(event) {
            if (event.data === "refresh") {
                window.location.reload();
            }
        };

        // WebSocket closed unexpectedly
        ws.onclose = function() {
            // Try to reconnect in 5 seconds
            console.log("WebSocket connection closed. Attempting to reconnect...");
            setTimeout(function() {
                connectWebSocket();
            }, 5000);
        };

        // Optional: Handle WebSocket errors
        ws.onerror = function(err) {
            console.error("WebSocket encountered an error:", err);
            ws.close();
        };
    }

    // Initial connection attempt
    connectWebSocket();
</script>
`

func startProxyServer(targetHost, port string) {
	targetURL, err := url.Parse(targetHost)
	if err != nil {
		slog.Error("Could not parse target URL: %v", err)
		os.Exit(1)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ModifyResponse = func(res *http.Response) error {
		slog.Debug("Modifying response")
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		bodyStr := string(body)
		modifiedBody := strings.Replace(bodyStr, "</body>", jsCode+"</body>", 1)
		res.Body = io.NopCloser(bytes.NewBufferString(modifiedBody))
		res.ContentLength = int64(len(modifiedBody))
		res.Header.Set("Content-Length", fmt.Sprint(len(modifiedBody)))
		return nil
	}
	proxy.Transport = &retryRoundTripper{http.DefaultTransport}

	proxyMux := http.NewServeMux()
	proxyMux.HandleFunc("/dev-html-proxy-ws", handleWebSocket)
	proxyMux.HandleFunc("/", proxy.ServeHTTP)

	log.Fatal(http.ListenAndServe(":"+port, proxyMux))
}

func startTriggerServer(port string) {
	triggerMux := http.NewServeMux()
	triggerMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		broadcastMessage("refresh")
		fmt.Fprintf(w, "Broadcasted refresh message\n")
	})

	log.Fatal(http.ListenAndServe(":"+port, triggerMux))
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	flag.Parse()
	// Print usage if -help flag is provided
	if showHelp {
		flag.Usage()
		os.Exit(0)
	}
	// Print usage if no target host is provided
	if targetHost == "" {
		fmt.Println("Error: -target flag is required")
		flag.Usage()
		os.Exit(1)
	}
	// if target doesn't start from http:// or https://, add http://
	if !strings.HasPrefix(targetHost, "http://") && !strings.HasPrefix(targetHost, "https://") {
		targetHost = "http://" + targetHost
	}

	if targetHost == "http://localhost:"+httpPort {
		fmt.Println("Error: target host cannot be the same as the proxy server")
		os.Exit(1)
	}

	go startProxyServer(targetHost, httpPort)
	go startTriggerServer(triggerPort)

	fmt.Println("Dev HTTP Proxy")
	fmt.Printf("Listening on http://localhost:%s for proxy\n", httpPort)
	fmt.Printf("Forwarding requests to %s\n", targetHost)
	fmt.Printf("Listening on http://localhost:%s for refresh requests\n", triggerPort)
	fmt.Println("Use the following curl command to trigger a refresh on all connected pages:")
	fmt.Printf("curl http://localhost:%s/\n", triggerPort)

	select {} // Prevent the main goroutine from exiting
}
