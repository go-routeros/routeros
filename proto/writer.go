package proto

import (
	"fmt"
	"io"
	"sync"
)

// Writer writes words to a RouterOS device.
type Writer struct {
	io.Writer
	err error
	sync.Mutex
}

// NewWriter returns a new Writer to write to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{Writer: w}
}

// Err returns the last error that occurred on this Writer.
func (w *Writer) Err() error {
	return w.err
}

// BeginSentence simply calls Lock().
func (w *Writer) BeginSentence() {
	w.Lock()
}

// EndSentence writes an empty word and calls Unlock(). It returns Err().
func (w *Writer) EndSentence() error {
	w.WriteWord("")
	err := w.err
	w.Unlock()
	return err
}

// Printf formats the string and sends it using WriteWord.
func (w *Writer) Printf(format string, a ...interface{}) {
	word := fmt.Sprintf(format, a...)
	w.WriteWord(word)
}

// WriteWord writes one RouterOS word.
func (w *Writer) WriteWord(word string) {
	w.writeBytes([]byte(word))
}

func (w *Writer) writeBytes(word []byte) {
	if w.err != nil {
		return
	}
	err := w.writeLength(len(word))
	if err != nil {
		w.err = err
		return
	}
	_, err = w.Write(word)
	if err != nil {
		w.err = err
		return
	}
}

func (w *Writer) writeLength(l int) error {
	_, err := w.Write(encodeLength(l))
	return err
}

func encodeLength(l int) []byte {
	switch {
	case l < 0x80:
		return []byte{byte(l)}
	case l < 0x4000:
		return []byte{byte(l>>8) | 0x80, byte(l)}
	case l < 0x200000:
		return []byte{byte(l>>16) | 0xC0, byte(l >> 8), byte(l)}
	case l < 0x10000000:
		return []byte{byte(l>>24) | 0xE0, byte(l >> 16), byte(l >> 8), byte(l)}
	default:
		return []byte{0xF0, byte(l >> 24), byte(l >> 16), byte(l >> 8), byte(l)}
	}
}
