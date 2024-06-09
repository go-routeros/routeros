package proto

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadWrite(t *testing.T) {
	for i, test := range []struct {
		in  []string
		out string
		tag string
	}{
		{[]string{"!done"}, `[]`, ""},
		{[]string{"!done", ".tag=abc123"}, `[]`, "abc123"},
		{strings.Split("!re =tx-byte=123456789 =only-key", " "), "[{`tx-byte` `123456789`} {`only-key` ``}]", ""},
	} {
		t.Run(fmt.Sprintf("#%d out=%s tag=%s", i, test.out, test.tag), func(t *testing.T) {
			buf := &bytes.Buffer{}
			// Write sentence into buf.
			w := NewWriter(buf)
			w.BeginSentence()
			for _, word := range test.in {
				w.WriteWord(word)
			}
			err := w.EndSentence()
			require.NoErrorf(t, err, "#%d input(%#q)", i, test.in)

			// Read sentence from buf.
			r := NewReader(buf)
			sen, err := r.ReadSentence()
			require.NoErrorf(t, err, "#%d input(%#q)", i, test.in)

			x := fmt.Sprintf("%#q", sen.List)
			require.Equal(t, test.out, x, "#%d input(%#q)", i, test.in)
			require.Equal(t, test.tag, sen.Tag, "#%d input(%#q)", i, test.in)
		})
	}
}
