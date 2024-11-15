package test

import (
	"image/png"
	"os"
	"testing"

	imageformat "github.com/ArtificialLegacy/image-format"
)

const test_image = "./output.blub"

const (
	test_width  = 1500
	test_height = 1500
)

func openImage(t *testing.T) *os.File {
	fs, err := os.Open(test_image)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	return fs
}

func TestDecode(t *testing.T) {
	fs := openImage(t)
	defer fs.Close()

	img, err := imageformat.Decode(fs)
	if err != nil {
		t.Fatal(err)
	}

	r, err := os.OpenFile("./output.png", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	err = png.Encode(r, img)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDecodeConfig(t *testing.T) {
	fs := openImage(t)
	defer fs.Close()

	conf, err := imageformat.DecodeConfig(fs)
	if err != nil {
		t.Fatal(err)
	}

	if conf.Width != test_width {
		t.Fatalf("Width is incorrect: %d, expected %d", conf.Width, test_width)
	}
	if conf.Height != test_height {
		t.Fatalf("Height is incorrect: %d, expected %d", conf.Height, test_height)
	}

	if !conf.Options.UseAlphaMask {
		t.Fatalf("UseAlphaMask is incorrect: %t, expected %t", conf.Options.UseAlphaMask, true)
	}
	if !conf.Options.ExcludeMaskedPixels {
		t.Fatalf("ExcludeMaskedPixels is incorrect: %t, expected %t", conf.Options.ExcludeMaskedPixels, true)
	}
	if !conf.Options.CompressAlphaMask {
		t.Fatalf("CompressAlphaMask is incorrect: %t, expected %t", conf.Options.CompressAlphaMask, true)
	}
}
