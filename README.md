
# Some Image Format?

After working with images for awhile when building ImgScal,
I've been wanting to make an image format for myself.

While this is not meant to be practical, I did have a few goals when making this:

1. Be a 1-2 day project.
2. Have an interesting result.
3. Learn something.

For #1 this meant keeping it simple,
and putting tight design constraints on the project.

For #2 I still wanted this format to be competent,
this required picking and optimizing for a niche.

For #3 I decided I wanted to find a way to incorporate run length encoding.
While I have understood how it works, this will be a first time implementation.

This format ultimately does well with monochromatic images,
or with boolean mask images.

## Design Constraints

* Monochromatic
  * Allows for storing pixel data as 1 byte grayscale,
    but adding the hue shift in the header allows for more flexibility.
* No Partial Transparency
  * Allows for 1 bit per pixel,
    and allows for more significant gains with run length encoding.
  * Having transparency to begin with allows for grayscale data
    for some pixels to be skipped.

## Format

After the 32 byte header,
the alpha mask and grayscale data is written using zlib compression.
The lengths stored in the header are the lengths before the zlib compression.

### Header (32 Bytes)

> All multi-byte sections are encoded in little endian.

#### Tag

| 0 | 1 | 2 | 3 |
| - | - | - | - |
|  'B' | 'L' | 'U' | 'B' |

#### Dimensions

| 4-5 | 6-7 |
| - | - |
| Width | Height |

#### Data Byte Lengths

| 8-11 | 12-15 |
| - | - |
| Alpha Data Length | Grayscale Data Length |

#### Hue Information

| 16 |
| - |
| Hue Shift |

#### Encoding Flags

| 17 |
| - |
| Flags |

| 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 |
| - | - | - | - | - | - | - | - |
| CompressAlphaMask | ExcludeMaskedPixels | - | - | - | - | - | - |

#### Reserved

| 18-31 |
| - |
| Reserved |

### Alpha Mask

The alpha mask can be disabled,
this is indicated by the length in the header being 0.

The `CompressAlphaMask` flag determines the mode used below:

#### Uncompressed

A 0 bit represents a transparent pixel, and a 1 bit represents an opaque pixel.

These are ordered starting from the left of each byte.
The least significant bits of the final byte that are unused are 0s.

#### Compressed

The alpha mask is compressed using run length encoding.
Here each byte represents a single boolean, and an amount.

The most significant bit represents the boolean value,
and the 7 least significant bits represents the amount.
It is arranged this way to make parsing them easier.

```go
boolean := (b & 128) != 0
amount := b & 127
```

### Grayscale Data

The grayscale data can be disabled,
this is indicated by the length in the header being 0.
If these is no grayscale data, all unmasked pixels are given a value of `0xFF`.

The flag `ExcludeMaskedPixels` determines if transparent pixels should be removed from the grayscale data.
In this case this data can only be read/written when the pixel is opaque.

## Resulting Image Sizes

### Alpha and Grayscale

| .png | .blub |
| - | - |
| 50.7kB | 18.8kB |

### Alpha Only

| .png | .blub |
| - | - |
| 16.6kB | 3.1kB |
