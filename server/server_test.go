package server

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseData(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		imei := "546897541245687"
		data := `000000000000004A8E010000016B412CEE000100000000000000000000000000000000010005000100010100010011001D00010010015E2C880002000B000000003544C87A000E000000001DD7E06A00000100002994`
		dBytes, err := hex.DecodeString(data)
		assert.Nil(t, err)
		points, err := ParseData(dBytes, imei)
		assert.Nil(t, err)
		t.Log(points)
	})

}
