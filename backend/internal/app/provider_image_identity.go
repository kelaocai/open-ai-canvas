package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"image/png"
)

const protocolImageIdentityMaxBytes = 32 << 20

func protocolInlineImageIdentity(value string) [32]byte {
	if len(value) <= protocolImageIdentityMaxBytes*4/3+1024 {
		if _, data, err := decodeProviderDataURL(value); err == nil {
			return protocolImageIdentity(data)
		}
	}
	return sha256.Sum256([]byte(value))
}

// Normalize static PNG pixels before provenance is stamped or resources are created.
// Oversized, animated and unsupported images retain exact-byte identity instead.
func protocolImageIdentity(data []byte) [32]byte {
	fallback := sha256.Sum256(data)
	if len(data) > protocolImageIdentityMaxBytes || !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		return fallback
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	const maxPixels = 16 * 1024 * 1024
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Height > maxPixels || config.Width > maxPixels/config.Height {
		return fallback
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte("static-png-pixels:v1"))
	for offset := 8; offset+12 <= len(data); {
		length := uint64(binary.BigEndian.Uint32(data[offset : offset+4]))
		if length > uint64(len(data)-offset-12) {
			return fallback
		}
		end := offset + 12 + int(length)
		kind := string(data[offset+4 : offset+8])
		switch kind {
		case "acTL":
			return fallback
		case "iCCP", "sRGB", "gAMA", "cHRM", "cICP", "mDCv", "cLLI", "eXIf":
			// These chunks can change appearance even when decoded RGB values match.
			_, _ = hash.Write(data[offset:end])
		}
		offset = end
	}
	image, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return fallback
	}
	var dimensions [8]byte
	binary.BigEndian.PutUint32(dimensions[:4], uint32(config.Width))
	binary.BigEndian.PutUint32(dimensions[4:], uint32(config.Height))
	_, _ = hash.Write(dimensions[:])
	row := make([]byte, config.Width*8)
	for y := 0; y < config.Height; y++ {
		for x := 0; x < config.Width; x++ {
			r, g, b, a := image.At(x, y).RGBA()
			pixel := row[x*8 : x*8+8]
			binary.BigEndian.PutUint16(pixel[0:2], uint16(r))
			binary.BigEndian.PutUint16(pixel[2:4], uint16(g))
			binary.BigEndian.PutUint16(pixel[4:6], uint16(b))
			binary.BigEndian.PutUint16(pixel[6:8], uint16(a))
		}
		_, _ = hash.Write(row)
	}
	var identity [32]byte
	copy(identity[:], hash.Sum(nil))
	return identity
}
