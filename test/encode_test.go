package test

import (
	"image/png"
	"os"
	"testing"

	imageformat "github.com/ArtificialLegacy/image-format"
)

const hueShift = 135

func TestEncode(t *testing.T) {
	fs, err := os.Open("./test.png")
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()

	img, err := png.Decode(fs)
	if err != nil {
		t.Fatal(err)
	}

	opts := imageformat.ImageOptions{
		HueShift:            hueShift,
		UseAlphaMask:        true,
		ExcludeMaskedPixels: true,
		CompressAlphaMask:   true,
		UniformHue:          true,
	}

	w, err := os.OpenFile("./output.blub", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = imageformat.Encode(w, img, opts)
	if err != nil {
		t.Fatal(err)
	}
}
