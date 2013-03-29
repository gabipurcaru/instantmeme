package main

import (
	"code.google.com/p/freetype-go/freetype"
	"code.google.com/p/freetype-go/freetype/raster"
	"crypto/md5"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
)

var (
	fontSrc      = "impact.ttf"
	fontBytes, _ = ioutil.ReadFile(fontSrc)
	font, _      = freetype.ParseFont(fontBytes)
	dpi          = float64(72)
	max_width    = 1200
	max_height   = 1200
)

func Fix32ToInt(n raster.Fix32) int {
	return int(n) / 256
}

// writes some text centered horizontally at a given height
// on an image
func drawMiddle(img draw.Image, text string, h int, color image.Image) {
	tmp := image.NewRGBA(img.Bounds())
	pt := freetype.Pt(0, h)
	context := freetype.NewContext()
	context.SetClip(img.Bounds())
	context.SetDst(tmp)
	context.SetDPI(72)
	context.SetFont(font)
	context.SetSrc(color)
	context.SetFontSize(float64(img.Bounds().Dx()) / 10 * 0.6)
	res, err := context.DrawString(text, pt)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}
	x := Fix32ToInt(res.X)
	width := int(img.Bounds().Dx())
	if x > width {
		// out of bounds
		fmt.Println("Out of bounds")
	}
	context.SetDst(img)
	pt = freetype.Pt((width-x)/2, h)
	_, err = context.DrawString(text, pt)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}
}

func Process(url string, top string, bottom string, color image.Image) (image.Image, error) {
	data, err := http.Get(url)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return nil, fmt.Errorf("Invalid URL")
	}
	source, _, err := image.Decode(data.Body)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return nil, fmt.Errorf("Invalid or unsupported image")
	}
	if source.Bounds().Dx() > max_width || source.Bounds().Dy() > max_height {
		return nil, fmt.Errorf("Image is too large")
	}
	img := image.NewRGBA(source.Bounds())
	h := img.Bounds().Dy()
	h1 := int(float64(h) * 0.14)
	h2 := h - h1

	draw.Draw(img, img.Bounds(), source, image.Point{0, 0}, draw.Over)
	drawMiddle(img, top, h1, color)
	drawMiddle(img, bottom, h2, color)

	return img, nil
}

func serveImage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	url := r.Form.Get("source")
	top := r.Form.Get("top")
	bottom := r.Form.Get("bottom")
	white := r.Form.Get("white")

	h := md5.New()
	fmt.Fprintf(h, "%s|%s|%s|%s", url, top, bottom, white)
	hash := fmt.Sprintf("%x", h.Sum(nil))

	file, err := ioutil.ReadFile("cache/" + hash)
	if err == nil {
		w.Header().Add("Content-Type", "image/png")
		w.Write(file)
		fmt.Println("Served from cache")
		return
	}

	var color image.Image = nil
	if white != "" {
		color = image.White
	} else {
		color = image.Black
	}
	img, err := Process(url, top, bottom, color)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintln(w, err.Error())
		return
	}
	cache_file, _ := os.Create("cache/" + hash)
	w.Header().Add("Content-Type", "image/png")
	png.Encode(w, img)
	png.Encode(cache_file, img)
}

func main() {
	http.HandleFunc("/", serveImage)
	http.ListenAndServe(":8080", nil)
}
