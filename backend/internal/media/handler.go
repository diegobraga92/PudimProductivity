package media

import (
	"errors"
	"image"
	_ "image/gif"  // register decoders
	_ "image/jpeg" // register decoders
	_ "image/png"  // register decoders
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// scanResponse is the payload returned by POST /api/v1/media/scan-isbn.
type scanResponse struct {
	ISBN string `json:"isbn"`
}

// ScanISBNHandler decodes a barcode from an uploaded image (multipart field
// "image") and returns the ISBN/EAN payload. The client then feeds the ISBN
// into the existing book-lookup endpoints.
func ScanISBNHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB cap
		shared.WriteError(w, http.StatusBadRequest, "expected multipart form")
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "missing 'image' file field")
		return
	}
	defer func() { _ = file.Close() }()

	img, _, err := image.Decode(file)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "could not decode image (jpg/png/gif)")
		return
	}

	isbn, err := DecodeISBN(r.Context(), img)
	if err != nil {
		if errors.Is(err, ErrNoBarcode) {
			shared.WriteError(w, http.StatusUnprocessableEntity, "no barcode found in image")
			return
		}
		log.Error().Err(err).Msg("barcode decode failed")
		shared.WriteError(w, http.StatusInternalServerError, "barcode decode failed")
		return
	}

	shared.WriteJSON(w, http.StatusOK, scanResponse{ISBN: isbn})
}
