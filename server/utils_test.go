package server

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDecodeIMEI(t *testing.T) {
	tests := map[string]struct {
		errWant    error
		imeiHex    string
		imeiResult string
	}{
		"success": {
			errWant:    nil,
			imeiHex:    "000F333536333037303432343431303133",
			imeiResult: "356307042441013",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			imeiBytes, err := hex.DecodeString(test.imeiHex)
			assertion := assert.New(t)
			assertion.Nil(err)
			imei, err := DecodeIMEI(imeiBytes)
			if test.errWant != nil {
				assertion.ErrorIs(err, test.errWant)
			} else {
				assertion.Nil(err)
				assertion.Equal(imei, test.imeiResult)
			}
		})
	}
}
