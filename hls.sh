#!/bin/bash

cd ui

raspivid -n -w 640 -h 480 -fps 25 -vf -t 0 -b 300000 -o - \
| ffmpeg -y \
	-i - \
	-c:v copy \
	-map 0:0 \
	-f ssegment \
	-segment_time 4 \
	-segment_format mpegts \
	-segment_list stream.m3u8 \
	-segment_list_size 720 \
	-segment_list_flags live \
	-segment_list_type m3u8 \
	"%08d.ts" 

trap "rm stream.m3u8 *.ts" EXIT
