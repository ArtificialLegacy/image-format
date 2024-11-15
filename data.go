package imageformat

const header_size = 32

const format_tag = (uint32('B') << 24) | (uint32('L') << 16) | (uint32('U') << 8) | (uint32('B'))

const (
	flag_compress_alpha        = 0b1000_0000
	flag_exclude_masked_pixels = 0b0100_0000
)
