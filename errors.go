package routeros

import (
	"fmt"

	"github.com/go-routeros/routeros/proto"
)

// UnknownReplyError records the sentence whose Word is unknown.
type UnknownReplyError struct {
	Sentence *proto.Sentence
}

func (err *UnknownReplyError) Error() string {
	return fmt.Sprintf("unknown RouterOS reply word: %s", err.Sentence.Word)
}

// DeviceError records the sentence containing the error received from the device.
// The sentence may have Word !trap or !fatal.
type DeviceError struct {
	Sentence *proto.Sentence
}

func (err *DeviceError) Error() string {
	m := err.Sentence.Map["message"]
	if m == "" {
		m = fmt.Sprintf("unknown error: %s", err.Sentence)
	}
	return fmt.Sprintf("RouterOS: %s", m)
}
