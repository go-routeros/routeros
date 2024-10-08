package routeros

import (
	"errors"
	"fmt"

	"github.com/go-routeros/routeros/v3/proto"
)

var (
	errAlreadyAsync   = errors.New("method Async() has already been called")
	errAsyncLoopEnded = errors.New("method Async(): loop has ended - probably read error")
)

// UnknownReplyError records the sentence whose Word is unknown.
type UnknownReplyError struct {
	Sentence *proto.Sentence
}

func (err *UnknownReplyError) Error() string {
	return "unknown RouterOS reply word: " + err.Sentence.Word
}

// DeviceError records the sentence containing the error received from the device.
// The sentence may have Word !trap or !fatal.
type DeviceError struct {
	Sentence *proto.Sentence
}

type decodedDeviceError string

// Well-known errors. We could just dynamically create these out of the various
// strings we get back from RouterOS, but by listing them here we capture knowledge
// and experience recognizing the different messages. This makes it easier for clients
// to write good error-handling code.
var (
	ErrInvalidUserNameOrPassword = decodedDeviceError("invalid user name or password (6)")
	ErrIncorrectLogin            = decodedDeviceError("incorrect login")
	ErrAlreadyExists             = decodedDeviceError("failure:entry already exists")
)

func (e decodedDeviceError) Error() string { return string(e) }
func (err *DeviceError) fetchMessage() string {
	if m := err.Sentence.Map["message"]; m != "" {
		return m
	}

	return "unknown error: " + err.Sentence.String()
}

func (err *DeviceError) Error() string {
	return fmt.Sprintf("from RouterOS device: %s", err.fetchMessage())
}
