package proto

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeLength(t *testing.T) {
	for i, d := range []struct {
		length   int
		rawBytes []byte
	}{
		{0x00000001, []byte{0x01}},
		{0x00000087, []byte{0x80, 0x87}},
		{0x00004321, []byte{0xC0, 0x43, 0x21}},
		{0x002acdef, []byte{0xE0, 0x2a, 0xcd, 0xef}},
		{0x10000080, []byte{0xF0, 0x10, 0x00, 0x00, 0x80}},
	} {
		t.Run(fmt.Sprintf("#%d length=%d", i, d.length), func(t *testing.T) {
			require.Equal(t, d.rawBytes, encodeLength(d.length), "expected bytes is wrong")
		})
	}
}
