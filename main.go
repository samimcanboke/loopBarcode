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
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"readBarcode/gozbar"
	"strings"
)

var frames <-chan []byte

func main() {
	port := ":9090"
	devName := "/dev/v4l/by-id/usb-CR_USB_Camera_20220907-video-index0"
	format := "MJPEG"
	width := 640
	height := 400
	frameRate := 120
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
	//pixfmt := currFmt.PixelFormat
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

	ctrls, err := v4l2.QueryAllExtControls(camera.Fd())
	if err != nil {
		log.Fatalf("failed to get ext controls: %s", err)
	}
	if len(ctrls) == 0 {
		log.Println("Device does not have extended controls")
		os.Exit(0)
	}
	for _, ctrl := range ctrls {
		printControl(ctrl)
	}
	//start := time.Now()
	for frame := range camera.GetOutput() {

		/*
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
		   		//log.Printf("Saved file: %s", fileName)
		   		if err := file.Close(); err != nil {
		   			log.Printf("failed to close file %s: %s", fileName, err)
		   		}

		   		fin, _ := os.Open(fileName)
		   		defer fin.Close()

		   		src, _ := jpeg.Decode(fin)


		*/
		imgNew, _, err1 := image.Decode(bytes.NewReader(frame))
		if err1 != nil {
			fmt.Println("imagedecodeerr", err1)
		}
		img := gozbar.NewImage(imgNew)
		scanner := gozbar.NewScanner().
			SetEnabledAll(true)

		symbols, scanImageErr := scanner.ScanImage(img)
		if scanImageErr != nil {
			fmt.Println("scanImageErr", scanImageErr)
		}
		for _, s := range symbols {
			fmt.Println(s.Type.Name(), s.Data, s.Quality, s.Boundary)
		}
	}
	//elapsed := time.Since(start)
	//log.Printf("Binomial took %s", elapsed)

	log.Printf("device capture started (buffer size set %d)", camera.BufferCount())
	http.HandleFunc("/stream", imageServ)
	log.Fatal(http.ListenAndServe(port, nil))

}

func imageServ(w http.ResponseWriter, req *http.Request) {
	mimeWriter := multipart.NewWriter(w)
	w.Header().Set("Content-Type", fmt.Sprintf("multipart/x-mixed-replace; boundary=%s", mimeWriter.Boundary()))
	partHeader := make(textproto.MIMEHeader)
	partHeader.Add("Content-Type", "image/jpeg")

	var frame []byte
	for frame = range frames {
		partWriter, err := mimeWriter.CreatePart(partHeader)
		if err != nil {
			log.Printf("failed to create multi-part writer: %s", err)
			return
		}

		if _, err := partWriter.Write(frame); err != nil {
			log.Printf("failed to write image: %s", err)
		}
	}
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

func printControl(ctrl v4l2.Control) {
	fmt.Printf("Control id (%d) name: %s\t[min: %d; max: %d; step: %d; default: %d current_val: %d]\n",
		ctrl.ID, ctrl.Name, ctrl.Minimum, ctrl.Maximum, ctrl.Step, ctrl.Default, ctrl.Value)

	if ctrl.IsMenu() {
		menus, err := ctrl.GetMenuItems()
		if err != nil {
			return
		}

		for _, m := range menus {
			fmt.Printf("\t(%d) Menu %s: [%d]\n", m.Index, m.Name, m.Value)
		}
	}

}
