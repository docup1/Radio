package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

const sid = "01a07dbe-6741-75c0-a74b-333eb2a87773"

type frame struct {
	Type   string         `json:"type,omitempty"`
	SongId string         `json:"songId,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
}

func readWindow(d *websocket.Conn, dur time.Duration, label string, wantSong bool) []string {
	deadline := time.Now().Add(dur)
	songs := []string{}
	end := deadline.Add(2 * time.Second)
	for time.Now().Before(end) {
		d.SetReadDeadline(deadline)
		mt, data, err := d.ReadMessage()
		if err != nil {
			fmt.Printf("[%s] end (%v)\n", label, err)
			return songs
		}
		if mt == websocket.BinaryMessage {
			fmt.Printf("[%s] AUDIO %d bytes\n", label, len(data))
			continue
		}
		var f frame
		if json.Unmarshal(data, &f) != nil {
			fmt.Printf("[%s] raw: %s\n", label, string(data))
			continue
		}
		switch f.Type {
		case "song":
			songs = append(songs, f.SongId)
			fmt.Printf("[%s] song %s\n", label, f.SongId[:8])
		case "song_ended":
			fmt.Printf("[%s] song_ended %s\n", label, f.SongId[:8])
		case "state":
			da := f.Data
			fmt.Printf("[%s] state active=%v song=%v item=%v pos=%v qlen=%v\n", label,
				da["is_active"], short(da["current_song_id"]), short(da["current_item_id"]), da["position"], da["queue_length"])
		case "stream_stopped":
			fmt.Printf("[%s] stream_stopped\n", label)
		case "stream_ended":
			fmt.Printf("[%s] stream_ended %v\n", label, f.Data["message"])
			songs = append(songs, "END")
		case "error":
			fmt.Printf("[%s] error %v\n", label, f.Data["message"])
		default:
			fmt.Printf("[%s] type=%s\n", label, f.Type)
		}
	}
	return songs
}

func short(v any) string {
	s, ok := v.(string)
	if !ok || s == "" {
		return "-"
	}
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

func dial(mode string) *websocket.Conn {
	hd := map[string][]string{}
	if mode == "owner" {
		hd["Authorization"] = []string{"Bearer " + os.Getenv("TOKEN")}
	}
	u := "ws://localhost:18000/api/streams/" + sid + "/ws"
	d, resp, err := websocket.DefaultDialer.Dial(u, hd)
	if err != nil {
		fmt.Println("dial err:", err, "resp:", resp)
		os.Exit(1)
	}
	fmt.Println("connected as", mode)
	return d
}

func main() {
	mode := "owner"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	d := dial(mode)
	defer d.Close()

	fmt.Println("-- initial --")
	readWindow(d, 1500*time.Millisecond, "init", false)

	if mode == "anon" {
		d.WriteJSON(frame{Type: "start", Data: map[string]any{"loop": false}})
		fmt.Println("-- anon start attempt --")
		readWindow(d, 2*time.Second, "anon", false)
		return
	}

	fmt.Println("-- start --")
	d.WriteJSON(frame{Type: "start", Data: map[string]any{"loop": false}})
	songs1 := readWindow(d, 6*time.Second, "start", true)
	fmt.Println("songs@start:", len(songs1))

	time.Sleep(300 * time.Millisecond)
	fmt.Println("-- skip --")
	d.WriteJSON(frame{Type: "skip"})
	songs2 := readWindow(d, 5*time.Second, "skip", true)
	fmt.Println("songs@skip:", len(songs2))

	time.Sleep(300 * time.Millisecond)
	fmt.Println("-- stop --")
	d.WriteJSON(frame{Type: "stop"})
	readWindow(d, 3*time.Second, "stop", false)

	fmt.Println("-- post-stop silence --")
	readWindow(d, 2*time.Second, "silence", false)

	fmt.Println("DONE", mode)
}
