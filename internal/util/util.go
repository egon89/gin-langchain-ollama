package util

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/jackc/pgx/v5/pgtype"
)

func NewUUID(value string) (pgtype.UUID, error) {
	var id pgtype.UUID

	err := id.Scan(value) // parses string into UUID

	return id, err
}

func ComputeSHA256Checksum(data []byte) string {
	h := sha256.New()
	// Write the data to the hasher
	h.Write(data)
	// Compute the final sum
	checksum := h.Sum(nil)

	// Return the checksum as a hexadecimal string to avoid issues with binary data in the database
	return hex.EncodeToString(checksum)
}
