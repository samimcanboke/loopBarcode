package main

import (
	"context"
	"fmt"
	"github.com/vladimirvivien/go4vl/device"
	"github.com/vladimirvivien/go4vl/v4l2"
	_ "image/jpeg"
	"log"
	"os"
	"strings"
)

var frames <-chan []byte

func main() {
	devName := "/dev/v4l/by-id/usb-CR_USB_Camera_20220907-video-index0"
	format := "MJPEG"
	width := 640
	height := 400
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
	fmt.Println("streamInfo", streamInfo)

	// start capture
	ctx, cancel := context.WithCancel(context.TODO())
	if err := camera.Start(ctx); err != nil {
		log.Fatalf("stream capture: %s", err)
	}
	defer func() {
		cancel()
		camera.Close()
	}()
	totalFrames := 1
	count := 0
	for frame := range camera.GetOutput() {
		fileName := fmt.Sprintf("capture_%d.jpg", count)
		file, err := os.Create(fileName)
		if err != nil {
			log.Printf("failed to create file %s: %s", fileName, err)
			continue
		}
		if _, err := file.Write(frame); err != nil {
			log.Printf("failed to write file %s: %s", fileName, err)
			continue
		}
		log.Printf("Saved file: %s", fileName)
		if err := file.Close(); err != nil {
			log.Printf("failed to close file %s: %s", fileName, err)
		}
		count++
		if count >= totalFrames {
			break
		}
	}

	/*for frame := range camera.GetOutput() {
		fileName := "barcode.jpg"
		file, err := os.Create(fileName)
		if err != nil {
			log.Printf("failed to create file %s: %s", fileName, err)
			continue
		}
		if _, err := file.Write(frame); err != nil {
			log.Printf("failed to write file %s: %s", fileName, err)
			continue
		}
		log.Printf("Saved file: %s", fileName)
		if err := file.Close(); err != nil {
			log.Printf("failed to close file %s: %s", fileName, err)
		}

		fin, _ := os.Open(fileName)
		defer fin.Close()
		src, _ := jpeg.Decode(fin)
		fmt.Println(src)

		img := gozbar.NewImage(src)
		scanner := gozbar.NewScanner().
			SetEnabledAll(true)

		symbols, scanImageErr := scanner.ScanImage(img)
		if scanImageErr != nil {
			fmt.Println("scanImageErr", scanImageErr)
		}
		for _, s := range symbols {
			fmt.Println(s.Type.Name(), s.Data, s.Quality, s.Boundary)
		}

		count++
		if count >= totalFrames {
			break
		}
	}*/

	// video stream

	/*
		frames =

		var frame []byte
		for frame = range frames {
			if len(frame) == 0 {
				log.Print("skipping empty frame")
				continue
			}
			//fmt.Println(frame)
			break
			img, _, imErr := image.Decode(bytes.NewReader(frame))
			if imErr != nil {

				fmt.Println("imErr", imErr)
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

		}*/

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
