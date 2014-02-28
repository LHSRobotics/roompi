package main

// Using the Video4Linux docs and examples at http://linuxtv.org/downloads/v4l-dvb-apis/.

import (
	"log"
	"syscall"
	"unsafe"

	"launchpad.net/gommap"
)

// #include <stdlib.h>
// #include <errno.h>
// #include <string.h>
// #include <sys/select.h>
// #include <linux/videodev2.h>
//
// int selectfd(int fd)
// {
// 	struct timeval tout = { 2, 0 };
// 	fd_set fds;
// 
// 	FD_ZERO(&fds);
// 	FD_SET(fd, &fds);
// 
// 	return select(fd + 1, &fds, NULL, NULL, &tout);
// }
import "C"

func ioctl(fd int, request, argp uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), request, argp)
	if errno != 0 {
		return errno
	}
	return nil
}

// vidCapture captures frames using the Video4Linux API.
// We use select()+mmap() because many drivers can't do read().
func vidCapture() {
	fd, err := syscall.Open(*videoSource, syscall.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		log.Println("disabling video: can't open video device file:", err)
		return
	}
	defer syscall.Close(fd)

	// Check camera capabilities.
	var cap C.struct_v4l2_capability
	err = ioctl(fd, C.VIDIOC_QUERYCAP, uintptr(unsafe.Pointer(&cap)))
	if err != nil {
		log.Fatalf("%s is not a V4L2 device: %v", *videoSource, err)
	}
	if (cap.capabilities & C.V4L2_CAP_VIDEO_CAPTURE) == 0 {
		log.Fatalf("%s does not support video capture", *videoSource)
	}
	if (cap.capabilities & C.V4L2_CAP_STREAMING) == 0 {
		log.Fatalf("%s does not support streaming io", *videoSource)
	}

	// Ask for MJPEG format.
	// TODO encode JPEGs ourselves on cameras which don't do MJPEG
	fmt := C.struct_v4l2_format{
		_type: C.V4L2_BUF_TYPE_VIDEO_CAPTURE,
	}
	pix := (*C.struct_v4l2_pix_format)(unsafe.Pointer(&fmt.fmt))
	pix.width = 640
	pix.height = 480
	pix.pixelformat = C.V4L2_PIX_FMT_MJPEG
	pix.field = C.V4L2_FIELD_INTERLACED
	err = ioctl(fd, C.VIDIOC_S_FMT, uintptr(unsafe.Pointer(&fmt)))
	if err != nil {
		log.Fatalf("couldn't set capture format: %v", err)
	}

	// Sort out buffers.
	req := C.struct_v4l2_requestbuffers{
		count:  4,
		_type:  C.V4L2_BUF_TYPE_VIDEO_CAPTURE,
		memory: C.V4L2_MEMORY_MMAP,
	}

	err = ioctl(fd, C.VIDIOC_REQBUFS, uintptr(unsafe.Pointer(&req)))
	if err != nil {
		log.Fatalf("couldn't get capture buffers: %v", err)
	}
	if req.count < 2 {
		log.Fatalf("not enough buffers: %v", err)
	}

	buffers := make([]gommap.MMap, req.count)

	for i := 0; i < len(buffers); i++ {
		buf := C.struct_v4l2_buffer{
			_type:  C.V4L2_BUF_TYPE_VIDEO_CAPTURE,
			memory: C.V4L2_MEMORY_MMAP,
			index:  C.__u32(i),
		}
		offset := (*C.__u32)(unsafe.Pointer(&buf.m))

		err = ioctl(fd, C.VIDIOC_QUERYBUF, uintptr(unsafe.Pointer(&buf)))
		if err != nil {
			log.Fatalf("couldn't query buffer %d: %v", i, err)
		}
		buffers[i], err = gommap.MapRegion(uintptr(fd),
			int64(*offset), int64(buf.length),
			gommap.PROT_READ|gommap.PROT_WRITE,
			gommap.MAP_SHARED)
		if err != nil {
			log.Fatalf("failed to map buffer %d: %v", i, err)
		}
	}

	// Start capture
	for i := 0; i < len(buffers); i++ {
		buf := C.struct_v4l2_buffer{
			_type:  C.V4L2_BUF_TYPE_VIDEO_CAPTURE,
			memory: C.V4L2_MEMORY_MMAP,
			index:  C.__u32(i),
		}
		err = ioctl(fd, C.VIDIOC_QBUF, uintptr(unsafe.Pointer(&buf)))
		if err != nil {
			log.Fatalf("couldn't queue buffer %d: %v", i, err)
		}
	}

	typ := C.V4L2_BUF_TYPE_VIDEO_CAPTURE
	err = ioctl(fd, C.VIDIOC_STREAMON, uintptr(unsafe.Pointer(&typ)))
	if err != nil {
		log.Fatalf("couldn't start stream: %v", err)
	}

	// Loop over frames!
	for {
		errno := C.selectfd(C.int(fd))
		if errno == C.EINTR || errno == C.EAGAIN {
			continue
		}
		if errno < 0 {
			log.Fatalf("select err: %v", C.GoString(C.strerror(errno)))
		}
		
		buf := C.struct_v4l2_buffer{
			_type:  C.V4L2_BUF_TYPE_VIDEO_CAPTURE,
			memory: C.V4L2_MEMORY_MMAP,
		}
		
		err = ioctl(fd, C.VIDIOC_DQBUF, uintptr(unsafe.Pointer(&buf)))
		if err == syscall.EAGAIN {
			continue
		}
		if err != nil {
			log.Fatalf("couldn't dequeue buffer: %v", err)
		}
		
		stream.UpdateJPEG(buffers[buf.index][:buf.bytesused])
		
		err = ioctl(fd, C.VIDIOC_QBUF, uintptr(unsafe.Pointer(&buf)))
		if err != nil {
			log.Fatalf("couldn't queue buffer %d: %v", buf.index, err)
		}
	}
}
