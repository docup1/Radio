package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	hd := map[string][]string{"Authorization": {"Bearer " + os.Getenv("TOKEN")}}
	d, _, err := websocket.DefaultDialer.Dial("ws://localhost:18000/api/streams/01a07dbe-6741-75c0-a74b-333eb2a87773/ws", hd)
	if err != nil {
		fmt.Println("dial:", err)
		os.Exit(1)
	}
	defer d.Close()
	fmt.Println("connected; initial:")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		d.SetReadDeadline(deadline)
		mt, b, err := d.ReadMessage()
		if err != nil {
			break
		}
		fmt.Println("  ", mt, string(b)[:min(len(b), 200)])
	}
	fmt.Println("sending start...")
	d.WriteJSON(map[string]any{"type": "start", "data": map[string]any{"loop": false}})
	end := time.Now().Add(14 * time.Second)
	for time.Now().Before(end) {
		d.SetReadDeadline(end)
		mt, b, err := d.ReadMessage()
		if err != nil {
			fmt.Println("read end:", err)
			break
		}
		if mt == websocket.BinaryMessage {
			fmt.Printf("  AUDIO %d bytes (%s)\n", len(b), time.Now().Format("15:04:05.000"))
			continue
		}
		var f map[string]any
		json.Unmarshal(b, &f)
		fmt.Printf("  %s %s (%s)\n", f["type"], b, time.Now().Format("15:04:05.000"))
	}
	fmt.Println("done")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
