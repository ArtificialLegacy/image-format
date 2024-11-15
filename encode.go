package imageformat

import (
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
)

type ImageOptions struct {
	HueShift            uint8
	AlphaThreshold      uint8
	UseAlphaMask        bool
	CompressAlphaMask   bool
	UniformHue          bool
	ExcludeMaskedPixels bool
}

func Encode(w io.Writer, img image.Image, opts ImageOptions) error {
	zw := zlib.NewWriter(w)
	defer zw.Close()

	imgwidth := img.Bounds().Dx()
	imgheight := img.Bounds().Dy()
	if imgwidth > 65535 || imgheight > 65535 {
		return fmt.Errorf("image too large: %d x %d", imgwidth, imgheight)
	}

	header := make([]byte, header_size)

	binary.BigEndian.PutUint32(header[0:4], format_tag)

	binary.LittleEndian.PutUint16(header[4:6], uint16(imgwidth))
	binary.LittleEndian.PutUint16(header[6:8], uint16(imgheight))

	header[16] = opts.HueShift
	header[17] = buildFlags(opts)

	imgf := formatImage(img, opts)
	pixd, alphad := processImageFormat(imgf, opts)

	if !imgf.Opaque {
		binary.LittleEndian.PutUint32(header[8:12], uint32(len(alphad)))
	}

	if !opts.UniformHue {
		binary.LittleEndian.PutUint32(header[12:16], uint32(len(pixd)))
	}

	n, err := w.Write(header)
	if err != nil {
		return fmt.Errorf("error writing header: %v", err)
	}
	if n != header_size {
		return fmt.Errorf("header length written is incorrect length: %d, expected %d", n, header_size)
	}

	if opts.UseAlphaMask {
		n, err = zw.Write(alphad)
		if err != nil {
			return fmt.Errorf("error writing alpha mask: %v", err)
		}
		if n != len(alphad) {
			return fmt.Errorf("alpha mask length written is incorrect length: %d, expected %d", n, len(alphad))
		}
	}

	if !opts.UniformHue {
		n, err = zw.Write(pixd)
		if err != nil {
			return fmt.Errorf("error writing pixel data: %v", err)
		}
		if n != len(pixd) {
			return fmt.Errorf("pixel data length written is incorrect length: %d, expected %d", n, len(pixd))
		}
	}

	return nil
}

func buildFlags(opts ImageOptions) (flags byte) {
	if opts.CompressAlphaMask {
		flags |= flag_compress_alpha
	}
	if opts.ExcludeMaskedPixels {
		flags |= flag_exclude_masked_pixels
	}

	return
}

// Image data before processing is applied
type imageformat struct {
	AlphaMask []bool
	PixelData []byte

	Opaque bool
}

func formatImage(img image.Image, opts ImageOptions) *imageformat {
	imgwidth := img.Bounds().Dx()
	imgheight := img.Bounds().Dy()
	imgmin := img.Bounds().Min

	size := imgwidth * imgheight

	f := imageformat{
		AlphaMask: make([]bool, size),
		PixelData: make([]byte, size),
	}

	index := 0
	opaque := true

	for y := imgmin.Y; y < imgmin.Y+imgheight; y++ {
		for x := imgmin.X; x < imgmin.X+imgwidth; x++ {
			y, a := colorToGray(img.At(x, y), opts.AlphaThreshold)

			f.AlphaMask[index] = a
			f.PixelData[index] = y

			index++

			if opaque && !a {
				opaque = false
			}
		}
	}

	f.Opaque = opaque

	return &f
}

const (
	rweight = 299
	gweight = 587
	bweight = 114
)

func colorToGray(c color.Color, threshold uint8) (uint8, bool) {
	rgba, ok := color.RGBAModel.Convert(c).(color.RGBA)
	if !ok {
		return 0, true
	}

	r := int(rgba.R) * rweight
	g := int(rgba.G) * gweight
	b := int(rgba.B) * bweight

	y := uint8(math.Round(float64(r+g+b) / (rweight + gweight + bweight)))

	return y, rgba.A > threshold
}

func processImageFormat(f *imageformat, opts ImageOptions) ([]byte, []byte) {
	if f.Opaque {
		return f.PixelData, []byte{}
	}

	compress := opts.CompressAlphaMask
	uniform := opts.UniformHue
	exclude := opts.ExcludeMaskedPixels

	alpha := make([]byte, (len(f.AlphaMask)+7)/8)
	pix := make([]byte, len(f.PixelData))

	var alphaIndex int
	var alphaMod int
	var alphaByte byte

	var currentAlpha bool
	var alphaLength int

	var pixelIndex int

	for i, p := range f.PixelData {
		av := f.AlphaMask[i]

		if !uniform && (!exclude || av) {
			pix[pixelIndex] = p
			pixelIndex++
		}

		if !compress {
			// per bit alpha flags
			if av {
				alphaByte |= 0b1000_0000 >> alphaMod
			}

			alphaMod++
			if alphaMod == 8 {
				alphaMod = 0
				alpha[alphaIndex] = alphaByte
				alphaByte = 0
				alphaIndex++
			}
		} else {
			// rle alpha flags
			if alphaLength == 0 {
				currentAlpha = av
				alphaLength++
			} else {
				if currentAlpha != av {
					ab := byte(alphaLength)
					if currentAlpha {
						ab |= 1 << 7
					}

					alpha[alphaIndex] = ab
					alphaIndex++

					alphaLength = 1
					currentAlpha = av
				} else {
					alphaLength++
				}

				if alphaLength == 127 {
					ab := byte(alphaLength)
					if currentAlpha {
						ab |= 1 << 7
					}

					alpha[alphaIndex] = ab
					alphaIndex++

					alphaLength = 0
				}
			}
		}
	}

	// add any buffered alpha values
	if alphaMod > 0 {
		alpha[alphaIndex] = alphaByte
		alphaIndex++
	}

	if alphaLength > 0 {
		ab := byte(alphaLength)
		if currentAlpha {
			ab |= 1 << 7
		}

		alpha[alphaIndex] = ab
		alphaIndex++
	}

	return pix[0:pixelIndex], alpha[0:alphaIndex]
}
