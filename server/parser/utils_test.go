package parser

import (
	"encoding/hex"
	"gotest.tools/v3/assert"
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
			assert.NilError(t, err)
			imei, err := DecodeIMEI(imeiBytes)
			if test.errWant != nil {
				assert.ErrorIs(t, err, test.errWant)
			} else {
				assert.NilError(t, err)
				assert.Equal(t, test.imeiResult, imei)
			}
		})
	}
}

func TestEncodeIMEI(t *testing.T) {
	tests := map[string]struct {
		errWant error
		imeiHex string
		imei    string
	}{
		"success": {
			errWant: nil,
			imeiHex: "000F333536333037303432343431303133",
			imei:    "356307042441013",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			imeiHex, err := EncodeIMEIToHex(test.imei)
			if test.errWant != nil {
				assert.ErrorIs(t, err, test.errWant)
			} else {
				assert.NilError(t, err)
				assert.Equal(t, imeiHex, test.imeiHex)
			}
		})
	}
}
