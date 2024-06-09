package proto

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadLength(t *testing.T) {
	for i, d := range []struct {
		length   int64
		rawBytes []byte
	}{
		{0x00000001, []byte{0x01}},
		{0x00000087, []byte{0x80, 0x87}},
		{0x00004321, []byte{0xC0, 0x43, 0x21}},
		{0x002acdef, []byte{0xE0, 0x2a, 0xcd, 0xef}},
		{0x10000080, []byte{0xF0, 0x10, 0x00, 0x00, 0x80}},
	} {
		t.Run(fmt.Sprintf("#%d length=%d", i, d.length), func(t *testing.T) {
			r := NewReader(bytes.NewBuffer(d.rawBytes)).(*reader)
			l, err := r.readLength()
			require.NoError(t, err, "read length error")
			require.Equal(t, d.length, l, "expected length is wrong")
		})
	}
}

func TestReadRandom(t *testing.T) {
	randomBytes := make([]byte, 4)
	_, err := rand.Read(randomBytes)
	require.NoError(t, err, "read random bytes error")

	r := NewReader(bytes.NewBuffer(randomBytes)).(*reader)
	_, err = r.readLength()
	require.NoError(t, err, "read length error")

}
