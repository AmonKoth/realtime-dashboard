# Real-Time Analytics Dashboard

A Go server that collects live events from several sources and pushes the results to a browser dashboard over WebSockets.

Data comes from two real sources:

- **Wikimedia EventStreams**: every edit across Wikipedia and the other Wikimedia projects, received as a live Server-Sent Events stream.
- **The Go runtime**: the program's own goroutine count and heap size, internal statistics.

Any other program can also send events to `POST /api/events`.

## Running

Open <http://localhost:9090>. Stop the server with Ctrl+C; it shuts down cleanly within a few seconds.

## Flags

| Flag | Default | Description |
|---|---|---|
| `-port` | `:9090` | Address to listen on. Use `127.0.0.1:9090` to accept only local connections. |
| `-tick` | `1s` | How often a snapshot is sent to browsers. |
| `-window` | `1m` | How far back the statistics look. Also the oldest timestamp accepted by `POST /api/events`. |
| `-runtime` | `1s` | How often goroutine count and heap size are measured. |
| `-web` | `web` | Folder containing `index.html` and `app.js`. |