package media

import (
	"context"
	"errors"
	"fmt"
	"image"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
)

// ErrNoBarcode is returned when an image contains no decodable barcode.
var ErrNoBarcode = errors.New("media: no barcode found in image")

// DecodeISBN extracts a product barcode (EAN-13 / UPC-A / EAN-8) from an image
// and returns the decoded payload — for book spines this is the ISBN-13.
// Pure-Go implementation (gozxing), so no external service or API key is
// required (unlike a Google Vision pipeline).
func DecodeISBN(ctx context.Context, img image.Image) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", fmt.Errorf("media: build bitmap: %w", err)
	}

	hints := map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_TRY_HARDER: true,
	}

	// MultiFormatUPCEANReader covers EAN-13 (ISBN), UPC-A and EAN-8 — the
	// barcode families printed on books.
	reader := oned.NewMultiFormatUPCEANReader(hints)
	result, err := reader.Decode(bitmap, hints)
	if err != nil {
		return "", ErrNoBarcode
	}

	text := result.GetText()
	if text == "" {
		return "", ErrNoBarcode
	}

	// Keep only digits (ISBN/EAN payloads are numeric; strip formatting like
	// dashes or "ISBN" prefixes).
	digits := make([]byte, 0, len(text))
	for i := 0; i < len(text); i++ {
		if text[i] >= '0' && text[i] <= '9' {
			digits = append(digits, text[i])
		}
	}
	if len(digits) == 0 {
		return "", ErrNoBarcode
	}

	// EAN-13 is the book standard; UPC-A (12) and legacy 10-digit ISBNs are
	// also accepted and normalized by the caller.
	if len(digits) != 13 && len(digits) != 12 && len(digits) != 10 {
		return "", fmt.Errorf("media: unsupported barcode length %d (want 10/12/13)", len(digits))
	}
	return string(digits), nil
}
