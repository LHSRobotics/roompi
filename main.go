package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

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

const picFile = "/tmp/pic.jpg"

//raspistill -n -w 640 -h 480 -q 5 -o poomba/ui/pic.jpg -tl 100 -t 999999999 -th 0:0:0
// TODO read stdout
func raspistill() {
	args := []string{"raspistill", "-n",
		"-w", "854", "-h", "480",
		"-rot", "180",
		"-q", "5", "-tl", "100",
		"-o", picFile,
	}
	for {
		start := time.Now()
		c := exec.Command(args[0], args[1:]...)
		output, err := c.CombinedOutput()
		if err != nil {
			log.Printf("error running raspistill:", err, output)
		} else if time.Since(start) < time.Second {
			log.Printf("raspistill exited too quickly:", output)
		}
	}
}

func camStreamer(w http.ResponseWriter, r *http.Request) {
	const boundary = "MAGIC"
	const headerf = "\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: image/jpeg\r\n" + 
		"Content-Length: %d\r\n" +
		"X-Timestamp: 0.000000\r\n"+
		"\r\n"
		
	w.Header().Add("Content-Type", "multipart/x-mixed-replace;boundary="+boundary)

	for {
		f, err := ioutil.ReadFile(picFile)
		if err != nil {
			log.Println(err)
		}
		w.Write([]byte(fmt.Sprintf(headerf, len(f))))
		w.Write(f)
		log.Println("wrote some", len(f))
		time.Sleep(100* time.Millisecond)
	}
}

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

	go raspistill() // change this to launch streamer

	http.Handle("/cmd", websocket.Handler(sockHandler))
	http.HandleFunc("/cam", camStreamer)
	http.Handle("/", http.FileServer(http.Dir(*dataRoot)))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
