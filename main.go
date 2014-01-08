package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"code.google.com/p/go.net/websocket"

	"github.com/saljam/roomba"
)

type msg struct {
	Cmd  string
	Args []int
}

func sockHandler(ws *websocket.Conn) {
	log.Println(ws.Request().RemoteAddr, "connected")

	defer log.Println(ws.Request().RemoteAddr, "disconnected")

	dec := json.NewDecoder(ws)
	var m msg
	for {
		if err := dec.Decode(&m); err == io.EOF {
			log.Println("websocket EOF:", err, ws.LocalAddr())
			break
		} else if err != nil {
			log.Println("websocket error:", err, dec.Buffered(), ws.LocalAddr())
		}

		var err error
		switch m.Cmd {
		case "start":
			err = r.Start()
		case "clean":
			err = r.Clean()
		case "dock":
			err = r.Dock()
		case "power":
			err = r.Power()
		case "safe":
			err = r.Safe()
		case "drive":
			err = r.Drive(int16(m.Args[0]), int16(m.Args[1]))
			log.Println("driving", m.Args)
		}
		if err != nil {
			log.Println("can't send command to roomba", err)
		}
	}
}

var addr = flag.String("addr", ":8001", "http address to listen on")
var dataRoot = flag.String("data", "./ui", "data dir")
var tty = flag.String("tty", "/dev/ttyAMA0", "path to serial interface")

var r roomba.Roomba

func main() {
	flag.Parse()
	f, err := os.OpenFile(*tty, os.O_RDWR, 0)
	if err != nil {
		log.Fatal("can't open serial file", err)
	}

	r = roomba.Roomba{f}

	err = r.Start()
	if err != nil {
		log.Fatal("can't send command to roomba", err)
	}
	
	http.Handle("/cmd", websocket.Handler(sockHandler))
	http.Handle("/", http.FileServer(http.Dir(*dataRoot)))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
