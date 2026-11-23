package hub

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"realtime-dashboard/internal/metrics"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

const (
	sleepDuration = time.Millisecond * 50
	readDeadLine  = time.Second
)

func TestBroadcastReachesAllClients(t *testing.T) {
	hub, url := startTestHub(t)

	conn1 := dialHelper(t, url)
	defer conn1.Close()

	conn2 := dialHelper(t, url)
	defer conn2.Close()

	snapshot := metrics.Snapshot{
		Time: time.Now(),
		Metrics: map[string]metrics.MetricStats{
			"test": {Count: 1},
		},
	}

	time.Sleep(sleepDuration)
	hub.Broadcast(snapshot)
	_, msg1, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("could not read the msg1: %v", err)
	}
	_, msg2, err := conn2.ReadMessage()
	if err != nil {
		t.Fatalf("could not read the msg2: %v", err)
	}
	text1 := string(msg1)
	text2 := string(msg2)
	if text1 != text2 {
		t.Fatal("messages are different")
	}
	if !strings.Contains(text1, "test") {
		t.Fatal("messages does not contain test")
	}
}

func TestSlowClientIsDropped(t *testing.T) {
	hub, url := startTestHub(t)

	normalClient := dialHelper(t, url)
	defer normalClient.Close()

	slow := &Client{hub: hub, send: make(chan []byte, sendBuffer)}
	hub.Register(slow)

	time.Sleep(sleepDuration)

	const total = sendBuffer * 2
	for i := 1; i <= total; i++ {
		hub.Broadcast(metrics.Snapshot{
			Time:    time.Now(),
			Metrics: map[string]metrics.MetricStats{"test": {Count: i}},
		})
		time.Sleep(time.Millisecond)
	}

	want := fmt.Sprintf(`"count":%d`, total)
	for {
		_, msg, err := normalClient.ReadMessage()
		if err != nil {
			t.Fatalf("healthy client stopped receiving: %v", err)
		}
		if strings.Contains(string(msg), want) {
			break
		}
	}

	timeout := time.After(time.Second)
	for {
		select {
		case _, ok := <-slow.send:
			if !ok {
				return
			}
		case <-timeout:
			t.Fatal("slow client was not dropped")
		}
	}
}

func dialHelper(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("could not dial the connection: %v", err)
	}
	err = conn.SetReadDeadline(time.Now().Add(readDeadLine))
	if err != nil {
		t.Fatalf("could not extend the deadline: %v", err)
	}
	return conn
}

func startTestHub(t *testing.T) (*Hub, string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	hub := NewHub()
	go hub.Run(ctx)

	upgrader := websocket.Upgrader{}
	handler := func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("could not upgrade the connection to websocket %v", err)
			return
		}
		NewClient(hub, conn).Start()
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	url := server.URL
	url = strings.Replace(url, "http", "ws", 1)

	return hub, url
}
