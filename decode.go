package imageformat

import (
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/crazy3lf/colorconv"
)

type ImageConfig struct {
	Options ImageOptions

	Width  int
	Height int

	alphaLength uint32
	pixelLength uint32
}

func Decode(r io.Reader) (image.Image, error) {
	conf, err := DecodeConfig(r)
	if err != nil {
		return nil, err
	}

	_ = conf

	zr, err := zlib.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("unable to create a zlib reader: %s", err)
	}
	defer zr.Close()

	alphaMask := make([]byte, conf.alphaLength)
	pixelData := make([]byte, conf.pixelLength)

	if conf.Options.UseAlphaMask {
		_, err = io.ReadFull(zr, alphaMask)
		if err != nil {
			return nil, fmt.Errorf("unable to read alpha mask: %s", err)
		}
	}

	if !conf.Options.UniformHue {
		_, err = io.ReadFull(zr, pixelData)
		if err != nil {
			return nil, fmt.Errorf("unable to read pixel data: %s", err)
		}
	}

	img := decodeImageData(conf, pixelData, alphaMask)

	return img, nil
}

func DecodeConfig(r io.Reader) (*ImageConfig, error) {
	header := make([]byte, header_size)
	n, err := r.Read(header)
	if err != nil {
		return nil, fmt.Errorf("unable to read header: %s", err)
	}
	if n != header_size {
		return nil, fmt.Errorf("header size is not %d", header_size)
	}

	tag := binary.BigEndian.Uint32(header[0:4])
	if tag != format_tag {
		return nil, fmt.Errorf("invalid format tag: %d", tag)
	}

	width := binary.LittleEndian.Uint16(header[4:6])
	height := binary.LittleEndian.Uint16(header[6:8])

	alphaLength := binary.LittleEndian.Uint32(header[8:12])
	pixelLength := binary.LittleEndian.Uint32(header[12:16])

	hueShift := header[16]
	flags := header[17]

	return &ImageConfig{
		Options: ImageOptions{
			HueShift:            hueShift,
			AlphaThreshold:      0,
			UseAlphaMask:        alphaLength > 0,
			CompressAlphaMask:   flags&flag_compress_alpha != 0,
			UniformHue:          pixelLength == 0,
			ExcludeMaskedPixels: flags&flag_exclude_masked_pixels != 0,
		},

		Width:  int(width),
		Height: int(height),

		alphaLength: alphaLength,
		pixelLength: pixelLength,
	}, nil
}

func decodeImageData(conf *ImageConfig, pix, alpha []byte) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, conf.Width, conf.Height))

	alphaBuffered := false
	alphaBufferCount := uint8(0)

	alphaIndex := 0
	alphaMod := 0
	pixIndex := 0

	compress := conf.Options.CompressAlphaMask
	uniform := conf.Options.UniformHue
	exclude := conf.Options.ExcludeMaskedPixels
	hue := (float64(conf.Options.HueShift) / 256) * 360

	for y := 0; y < conf.Height; y++ {
		for x := 0; x < conf.Width; x++ {
			var av bool

			if !compress {
				av = (alpha[alphaIndex]>>(7-alphaMod))&1 == 1

				alphaMod++
				if alphaMod == 8 {
					alphaMod = 0
					alphaIndex++
				}
			} else {
				if alphaBufferCount == 0 {
					ar := alpha[alphaIndex]
					alphaIndex++

					alphaBuffered = (ar & 0b1000_0000) != 0
					alphaBufferCount = ar & 0b0111_1111
				}

				alphaBufferCount--
				av = alphaBuffered
			}

			gray := uint8(0xFF)

			if !uniform && (!exclude || av) {
				gray = pix[pixIndex]
				pixIndex++
			}

			a := uint8(0xFF)
			if !av {
				a = 0
			}

			red, green, blue := gray, gray, gray

			if hue != 0 {
				_, _, v := colorconv.RGBToHSV(red, green, blue)
				red, green, blue, _ = colorconv.HSVToRGB(hue, 1, v)
			}

			img.SetRGBA(x, y, color.RGBA{R: red, G: green, B: blue, A: a})
		}
	}

	return img
}
