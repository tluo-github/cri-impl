package container

import (
	"encoding/hex"
	"errors"
	"github.com/satori/go.uuid"
	"strings"
)

type ID string

var badIdFormatErr = errors.New("Bad container ID format")

func RandID() ID {
	return ID(strings.ReplaceAll(uuid.NewV4().String(), "-", ""))
}

func ParseId(id string) (ID, error) {
	if len(id) != 32 {
		return ID(""), badIdFormatErr
	}
	if _, err := hex.DecodeString(id); err != nil {
		return ID(""), badIdFormatErr
	}
	return ID(id), nil
}
