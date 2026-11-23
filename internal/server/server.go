package server

import (
	"encoding/json"
	"log"
	"net/http"
	"realtime-dashboard/internal/hub"
	"realtime-dashboard/internal/metrics"
	"time"

	"github.com/gorilla/websocket"
)

const (
	readBufferSize    = 1024
	writeBufferSize   = 1024
	readerCap         = 1024 * 1024
	maxClockAllowance = time.Second * 5
	defaultMaxAge     = time.Minute
)

type SnapshotProvider interface {
	Latest() metrics.Snapshot
}

type Server struct {
	hub      *hub.Hub
	events   chan<- metrics.Event
	provider SnapshotProvider
	webDir   string
	maxAge   time.Duration
	upgrader websocket.Upgrader
}

func NewServer(h *hub.Hub, ev chan<- metrics.Event, pro SnapshotProvider, mAge time.Duration, dir string) *Server {
	if mAge <= 0 {
		mAge = defaultMaxAge
	}
	return &Server{
		hub:      h,
		events:   ev,
		provider: pro,
		webDir:   dir,
		maxAge:   mAge,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  readBufferSize,
			WriteBufferSize: writeBufferSize,
		},
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.Dir(s.webDir)))
	mux.HandleFunc("GET /ws", s.wsHandle)
	mux.HandleFunc("POST /api/events", s.ingestHandle)
	mux.HandleFunc("GET /api/snapshot", s.snapshotHandle)
	mux.HandleFunc("GET /api/health", s.healthHandle)
	return mux
}

func (s *Server) wsHandle(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("connection not upgraded: %v", err)
		return
	}
	hub.NewClient(s.hub, conn).Start()
}

func (s *Server) ingestHandle(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, readerCap)
	ev := metrics.Event{}
	err := json.NewDecoder(r.Body).Decode(&ev)
	if err != nil {
		http.Error(w, "could not decode the data: "+err.Error(), http.StatusBadRequest)
		return
	}

	ev.Source = "http"
	now := time.Now()
	if ev.Timestamp.IsZero() {
		ev.Timestamp = now
	}
	if ev.Timestamp.After(now.Add(maxClockAllowance)) {
		http.Error(w, "timestamp is in the future", http.StatusBadRequest)
		return
	}
	if ev.Timestamp.Before(now.Add(-s.maxAge)) {
		http.Error(w, "timestamp is too old", http.StatusBadRequest)
		return
	}
	err = ev.Validate()
	if err != nil {
		http.Error(w, "data is not valid: "+err.Error(), http.StatusBadRequest)
		return
	}

	select {
	case s.events <- ev:
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("received"))
	default:
		http.Error(w, "server busy try again", http.StatusServiceUnavailable)
	}
}

func (s *Server) snapshotHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(s.provider.Latest())
	if err != nil {
		log.Printf("encoding error occurred during snapshot: %v", err)
	}
}

func (s *Server) healthHandle(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("ok"))
	if err != nil {
		log.Printf("healthcheck error %v", err)
	}
}
