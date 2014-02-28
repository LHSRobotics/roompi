package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"

	"code.google.com/p/go.net/websocket"
	"github.com/howeyc/fsnotify"
	"github.com/saljam/mjpeg"
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

	r      roomba.Roomba
	stream *mjpeg.Stream
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

func raspiwatcher() {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	err = w.Watch(path.Dir(*picFile))
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 0)

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
		if len(buf) < int(info.Size()) {
			buf = make([]byte, int(info.Size()))
		}

		io.ReadFull(file, buf)
		file.Close()

		stream.UpdateJPEG(buf)
	}
}

func main() {
	flag.Parse()
	f, err := os.OpenFile(*tty, os.O_RDWR, 0)
	if err != nil {
		log.Fatal("can't open serial file", err)
	}

	stream = mjpeg.NewStream()
	r = roomba.Roomba{f}

	err = r.Start()
	if err != nil {
		log.Println("can't send start command to roomba", err)
	}

	if *raspi {
		if _, err := os.Stat(path.Dir(*picFile)); os.IsNotExist(err) {
			os.MkdirAll(path.Dir(*picFile), 0775)
		}
		go raspiwatcher()
		go raspistill()
	} else {
		// TODO start v4l capture here
	}

	http.Handle("/cmd", websocket.Handler(sockHandler))
	http.Handle("/cam", stream)
	http.Handle("/", http.FileServer(http.Dir(*dataRoot)))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
