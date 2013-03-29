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
	fontSrc       = "impact.ttf"
	fontBytes, _  = ioutil.ReadFile(fontSrc)
	font, _       = freetype.ParseFont(fontBytes)
	dpi           = float64(72)
	max_width     = 1200 // maximum allowed width
	max_height    = 1200 // and height
	font_scaling  = 0.06 // font scaling, as a factor of image width
	height_factor = 0.14 // where to draw the top/bottom text, as a factor
						 // of image height
)

// turns a raster.Fix32 into number of pixels
// this is needed because freetype-go doesn't provide such a utility
func Fix32ToInt(n raster.Fix32) int {
	return int(n) / 256
}

// writes some text centered horizontally at a given height
// on an image
func drawMiddle(img draw.Image, text string, h int, color image.Image) {
	// in order to figure out where to draw the text to make it centered,
	// we use a temporary image, write on that, see how much width the written
	// text occupies, compute where to write the text on the real image,
	// and then write it
	tmp := image.NewNRGBA(image.Rectangle{image.Point{0, 0}, image.Point{1, 1}})
	pt := freetype.Pt(0, h)
	context := freetype.NewContext()
	context.SetClip(img.Bounds())
	context.SetDst(tmp)
	context.SetDPI(dpi)
	context.SetFont(font)
	context.SetSrc(color)
	context.SetFontSize(float64(img.Bounds().Dx()) * font_scaling)
	res, err := context.DrawString(text, pt)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}
	x := Fix32ToInt(res.X)
	width := int(img.Bounds().Dx())
	if x > width {
		fmt.Println("Out of bounds")
	}
	context.SetDst(img)
	pt = freetype.Pt((width-x)/2, h)
	_, err = context.DrawString(text, pt)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}
}

// process a set of parameters, returning an image
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

	// h1 is the height of the top text, and h2 is the height of the bottom one
	h1 := int(float64(h) * height_factor)
	h2 := h - h1

	draw.Draw(img, img.Bounds(), source, image.Point{0, 0}, draw.Over)
	drawMiddle(img, top, h1, color)
	drawMiddle(img, bottom, h2, color)

	return img, nil
}


// this is the main handler; it decides whether to call Process to create
// the image, or to serve it from cache
func serveImage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	url := r.Form.Get("source")
	top := r.Form.Get("top")
	bottom := r.Form.Get("bottom")
	white := r.Form.Get("white")

	// compute a hash value for the set of parameters
	h := md5.New()
	fmt.Fprintf(h, "%s|%s|%s|%s", url, top, bottom, white)
	hash := fmt.Sprintf("%x", h.Sum(nil))

	file, err := ioutil.ReadFile("cache/" + hash)
	if err == nil {
		// if image available in cache, don't re-create it
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

	// return the image to the user, and also write it in cache (at this
	// point, we are certain that the image is not already cached)
	png.Encode(w, img)
	png.Encode(cache_file, img)
}

func main() {
	http.HandleFunc("/", serveImage)
	http.ListenAndServe(":8080", nil)
}
