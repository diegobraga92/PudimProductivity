package media

import (
	"context"
	"image"
	"image/color"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
)

// encodeEAN13 renders an EAN-13 barcode as a white/black image using the same
// library's writer — a round-trip test of our decoder.
func encodeEAN13(t *testing.T, payload string) image.Image {
	t.Helper()
	writer := oned.NewEAN13Writer()
	bitMatrix, err := writer.Encode(payload, gozxing.BarcodeFormat_EAN_13, 300, 100, nil)
	if err != nil {
		t.Fatalf("encode EAN-13: %v", err)
	}

	img := image.NewGray(image.Rect(0, 0, bitMatrix.GetWidth(), bitMatrix.GetHeight()))
	for y := 0; y < bitMatrix.GetHeight(); y++ {
		for x := 0; x < bitMatrix.GetWidth(); x++ {
			if bitMatrix.Get(x, y) {
				img.SetGray(x, y, color.Gray{Y: 0})
			} else {
				img.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return img
}

// TestDecodeISBN_RoundTrip encodes and decodes a real ISBN-13 payload.
func TestDecodeISBN_RoundTrip(t *testing.T) {
	const isbn = "9780553386790" // The Demon-Haunted World (EAN-13)
	img := encodeEAN13(t, isbn)

	got, err := DecodeISBN(context.Background(), img)
	if err != nil {
		t.Fatalf("DecodeISBN: %v", err)
	}
	if got != isbn {
		t.Errorf("DecodeISBN: got %q, want %q", got, isbn)
	}
}

// TestDecodeISBN_NoBarcode returns ErrNoBarcode for a plain white image.
func TestDecodeISBN_NoBarcode(t *testing.T) {
	blank := image.NewGray(image.Rect(0, 0, 200, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 200; x++ {
			blank.SetGray(x, y, color.Gray{Y: 255})
		}
	}

	_, err := DecodeISBN(context.Background(), blank)
	if err == nil {
		t.Fatal("expected an error for a blank image")
	}
}
