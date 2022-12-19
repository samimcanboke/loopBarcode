package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/vladimirvivien/go4vl/device"
	"github.com/vladimirvivien/go4vl/v4l2"
	"image"
	_ "image/jpeg"
	"log"
	"readBarcode/gozbar"
	"strings"
)

var frames <-chan []byte

func main() {
	devName := "/dev/v4l/by-id/usb-CR_USB_Camera_20220907-video-index0"
	format := "yuyv"
	width := 640
	height := 480
	frameRate := 30
	buffSize := 4
	camera, err := device.Open(devName,
		device.WithIOType(v4l2.IOTypeMMAP),
		device.WithPixFormat(v4l2.PixFormat{PixelFormat: getFormatType(format), Width: uint32(width), Height: uint32(height), Field: v4l2.FieldAny}),
		device.WithFPS(uint32(frameRate)),
		device.WithBufferSize(uint32(buffSize)),
	)
	brightnessSetErr := camera.SetControlBrightness(0)
	if brightnessSetErr != nil {
		fmt.Println("brightnessSetErr", brightnessSetErr)
	}

	if err != nil {
		log.Fatalf("failed to open device: %s", err)
	}
	defer camera.Close()

	caps := camera.Capability()
	log.Printf("device [%s] opened\n", devName)
	log.Printf("device info: %s", caps.String())

	// set device format
	currFmt, err := camera.GetPixFormat()
	if err != nil {
		log.Fatalf("unable to get format: %s", err)
	}
	log.Printf("Current format: %s", currFmt)
	pixfmt := currFmt.PixelFormat
	streamInfo := fmt.Sprintf("%s - %s [%dx%d] %d fps",
		caps.Card,
		v4l2.PixelFormats[currFmt.PixelFormat],
		currFmt.Width, currFmt.Height, frameRate,
	)

	// start capture
	ctx, cancel := context.WithCancel(context.TODO())
	if err := camera.Start(ctx); err != nil {
		log.Fatalf("stream capture: %s", err)
	}
	defer func() {
		cancel()
		camera.Close()
	}()

	// video stream

	frames = camera.GetOutput()

	var frame []byte
	for frame = range frames {
		fmt.Println(frame)
		if len(frame) == 0 {
			log.Print("skipping empty frame")
			continue
		}
		img, _, err := image.Decode(bytes.NewReader(frame))
		if err != nil {
			fmt.Println(err)
		}

		bounds := img.Bounds()
		fmt.Println(bounds)

		scanner := gozbar.NewScanner().
			SetEnabledAll(true)

		src := gozbar.NewImage(img)
		symbols, _ := scanner.ScanImage(src)

		for _, s := range symbols {
			data := s.Data
			points := s.Boundary

			fmt.Println(data, points)
		}

	}

	log.Printf("device capture started (buffer size set %d)", camera.BufferCount())
	fmt.Println(frames)
	fmt.Println("pixfmt", pixfmt)
	fmt.Println("streamInfo", streamInfo)

}

func getFormatType(fmtStr string) v4l2.FourCCType {
	switch strings.ToLower(fmtStr) {
	case "jpeg":
		return v4l2.PixelFmtJPEG
	case "mpeg":
		return v4l2.PixelFmtMPEG
	case "mjpeg":
		return v4l2.PixelFmtMJPEG
	case "h264", "h.264":
		return v4l2.PixelFmtH264
	case "yuyv":
		return v4l2.PixelFmtYUYV
	case "rgb":
		return v4l2.PixelFmtRGB24
	}
	return v4l2.PixelFmtMPEG
}
