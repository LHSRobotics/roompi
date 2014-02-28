package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/howeyc/fsnotify"
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

var (
	addr     = flag.String("addr", ":8001", "http address to listen on")
	dataRoot = flag.String("data", "./ui", "data dir")
	tty      = flag.String("tty", "/dev/ttyAMA0", "path to serial interface")
	videoDev = flag.String("video", "/dev/video0", "video input device")
	raspi    = flag.Bool("raspistill", false, "use raspistill for video")
	picFile  = flag.String("pic", "/tmp/roompi/pic.jpg", "temp file to store camera image")

	r roomba.Roomba
)

//raspistill -n -w 640 -h 480 -q 5 -o poomba/ui/pic.jpg -tl 100 -t 999999999 -th 0:0:0
// TODO replace this stuff with cgo calls to mmal (or wait for v4l to happen...)
func raspistill() {
	args := []string{"raspistill", "-n",
		"-w", "854", "-h", "480",
		"-rot", "180",
		"-q", "5", "-tl", "500",
		"-o", *picFile,
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

var streams struct {
	m map[chan []byte]bool
	sync.Mutex
}

const boundaryWord = "MJPEGBOUNDARY"

func camStreamer() {
	const headerf = "\r\n" +
		"--" + boundaryWord + "\r\n" +
		"Content-Type: image/jpeg\r\n" +
		"Content-Length: %d\r\n" +
		"X-Timestamp: 0.000000\r\n" +
		"\r\n"

	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	err = w.Watch(path.Dir(*picFile))
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, len(headerf))

	for {
		select {
		case <-w.Event:
		case err := <-w.Error:
			log.Println(err)
		}

		file, err := os.Open(*picFile)
		if err != nil {
			log.Println(err)
			continue
		}
		info, err := file.Stat()
		if err != nil {
			log.Println(err)
			continue
		}
		if len(buf) < int(info.Size())+len(headerf)+20 {
			buf = make([]byte, int(info.Size())*2)
		}

		header := fmt.Sprintf(headerf, info.Size())
		copy(buf, header)

		io.ReadFull(file, buf[len(header):])
		file.Close()

		streams.Lock()
		for s := range streams.m {
			s <- buf
		}
		streams.Unlock()
	}
}

func camHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "multipart/x-mixed-replace;boundary="+boundaryWord)

	c := make(chan []byte)
	streams.Lock()
	streams.m[c] = true
	streams.Unlock()

	for {
		b := <-c
		_, err := w.Write(b)
		if err != nil {
			break
		}
	}

	streams.Lock()
	delete(streams.m, c)
	streams.Unlock()
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
		log.Println("can't send start command to roomba", err)
	}

	if *raspi {
		if _, err := os.Stat(path.Dir(*picFile)); os.IsNotExist(err) {
			os.MkdirAll(path.Dir(*picFile), 0775)
		}
		go raspistill() // change this to launch streamer
	} else {
		// TODO start v4l capture here
	}
	streams.m = make(map[chan []byte]bool)
	go camStreamer()

	http.Handle("/cmd", websocket.Handler(sockHandler))
	http.HandleFunc("/cam", camHandler)
	http.Handle("/", http.FileServer(http.Dir(*dataRoot)))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
