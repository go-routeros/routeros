package routeros

import (
	"crypto/rand"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-routeros/routeros/v3/proto"
)

func TestRandomData(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)

		randomBytes := make([]byte, 1024)
		_, err := rand.Read(randomBytes)
		require.NoError(t, err, "read random bytes error")

		s.readSentence(t, "/login @ [{`name` `userTest`} {`password` `passTest`}]")
		s.writeSentence(t, "!done", string(randomBytes))
	}()

	err := c.Login("userTest", "passTest")
	require.Error(t, err)

}

func TestLoginPre643(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/login @ [{`name` `userTest`} {`password` `passTest`}]")
		s.writeSentence(t, "!done", "=ret=abc123")
		s.readSentence(t, "/login @ [{`name` `userTest`} {`response` `0021277bff9ac7caf06aa608e46616d47f`}]")
		s.writeSentence(t, "!done")
	}()

	err := c.Login("userTest", "passTest")
	require.NoError(t, err)
}

func TestLoginPost643(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/login @ [{`name` `userTest`} {`password` `passTest`}]")
		s.writeSentence(t, "!done")
	}()

	err := c.Login("userTest", "passTest")
	require.NoError(t, err)
}

func TestLoginIncorrectPre643(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/login @ [{`name` `userTest`} {`password` `passTest`}]")
		s.writeSentence(t, "!done", "=ret=abc123")
		s.readSentence(t, "/login @ [{`name` `userTest`} {`response` `0021277bff9ac7caf06aa608e46616d47f`}]")
		s.writeSentence(t, "!trap", "=message=incorrect login")
		s.writeSentence(t, "!done")
	}()

	err := c.Login("userTest", "passTest")
	require.Error(t, err, "Login succeeded; want error")

	var top *DeviceError
	require.Truef(t, errors.As(err, &top), "want=DeviceError, have=%#v", err)
	require.Contains(t, []string{"incorrect login"}, top.fetchMessage())
}

func TestLoginIncorrectPost643(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/login @ [{`name` `userTest`} {`password` `passTest`}]")
		s.writeSentence(t, "!trap", "=message=invalid user name or password (6)")
		s.writeSentence(t, "!done")
	}()

	err := c.Login("userTest", "passTest")
	require.Error(t, err, "Login succeeded; want error")

	var top *DeviceError
	require.Truef(t, errors.As(err, &top), "want=DeviceError, have=%#v", err)
	require.Contains(t, []string{"invalid user name or password (6)"}, top.fetchMessage())
}

func TestLoginNoChallenge(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/login @ [{`name` `userTest`} {`password` `passTest`}]")
		s.writeSentence(t, "!done")
	}()

	require.NoError(t, c.Login("userTest", "passTest"))
}

func TestLoginInvalidChallenge(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/login @ [{`name` `userTest`} {`password` `passTest`}]")
		s.writeSentence(t, "!done", "=ret=Invalid Hex String")
	}()

	err := c.Login("userTest", "passTest")
	require.Error(t, err, "Login succeeded; want error")
	require.Truef(t, errors.Is(err, ErrInvalidChallengeReceived),
		"want=ErrInvalidChallengeReceived, have=%#v", err)
}

func TestLoginEOF(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)
	require.NoError(t, s.Close())

	err := c.Login("userTest", "passTest")
	require.Error(t, err, "Login succeeded; want error")
	require.EqualError(t, err, io.ErrClosedPipe.Error())
}

func TestCloseTwice(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, s)
	require.NoError(t, c.Close())
	require.NoError(t, c.Close())
}

func TestAsyncTwice(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)
	defer deferCloser(t, s)

	c.Async()

	errC := c.Async()
	err := <-errC
	require.EqualError(t, err, errAlreadyAsync.Error())
	require.NoError(t, <-errC, errAsyncLoopEnded.Error())
}

func TestProtoRun(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/ip/address @ []")
		s.writeSentence(t, "!re", "=address=1.2.3.4/32")
		s.writeSentence(t, "!done")
	}()

	sen, err := c.Run("/ip/address")
	require.NoError(t, err)

	want := "!re @ [{`address` `1.2.3.4/32`}]\n!done @ []"
	require.Equal(t, want, sen.String(), "for /ip/address")
}

func TestRunWithListen(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/ip/address @l1 []")
		s.writeSentence(t, "!re", ".tag=l1", "=address=1.2.3.4/32")
		s.writeSentence(t, "!done", ".tag=l1")
	}()

	listen, err := c.Listen("/ip/address")
	require.NoError(t, err)

	sen := <-listen.Chan()
	want := "!re @l1 [{`address` `1.2.3.4/32`}]"
	require.Equal(t, want, sen.String(), "for /ip/address")

	sen = <-listen.Chan()
	require.Nil(t, sen, "Listener should have been closed after EOF")
	require.NoError(t, listen.Err())
}

func TestProtoRunAsync(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)
	c.Async()

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/ip/address @r1 []")
		s.writeSentence(t, "!re", ".tag=r1", "=address=1.2.3.4/32")
		s.writeSentence(t, "!done", ".tag=r1")
	}()

	sen, err := c.Run("/ip/address")
	require.NoError(t, err)

	want := "!re @r1 [{`address` `1.2.3.4/32`}]\n!done @r1 []"
	require.Equal(t, want, sen.String(), "for /ip/address")
}

func TestRunEmptySentence(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/ip/address @ []")
		s.writeSentence(t)
		s.writeSentence(t, "!re", "=address=1.2.3.4/32")
		s.writeSentence(t, "!done")
	}()

	sen, err := c.Run("/ip/address")
	require.NoError(t, err)

	want := "!re @ [{`address` `1.2.3.4/32`}]\n!done @ []"
	require.Equal(t, want, sen.String(), "for /ip/address")
}

func TestRunEOF(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/ip/address @ []")
	}()

	_, err := c.Run("/ip/address")
	require.Error(t, err, "Run succeeded; want error")
	require.Truef(t, errors.Is(err, io.EOF), "want=io.EOF, have=%#v", err)
}

func TestRunEOFAsync(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)
	c.Async()

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/ip/address @r1 []")
		s.writeSentence(t, "!re", "=address=1.2.3.4/32")
	}()

	_, err := c.Run("/ip/address")
	require.Error(t, err, "Run succeeded; want error")
	require.Truef(t, errors.Is(err, io.EOF), "want=io.EOF, have=%#v", err)
}

func TestRunInvalidSentence(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/ip/address @ []")
		s.writeSentence(t, "!xxx")
	}()

	_, err := c.Run("/ip/address")
	require.Error(t, err, "Run succeeded; want error")

	var unkErr *UnknownReplyError
	require.Truef(t, errors.As(err, &unkErr), "want=UnknownReplyError, have=%#v", err)
	require.Equal(t, unkErr.Sentence.Word, "!xxx")
}

func TestRunTrap(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/ip/address @ []")
		s.writeSentence(t, "!trap", "=message=Some device error message")
		s.writeSentence(t, "!done")
	}()

	_, err := c.Run("/ip/address")
	require.Error(t, err, "Run succeeded; want error")

	var devErr *DeviceError
	require.Truef(t, errors.As(err, &devErr), "want=DeviceError, have=%#v", err)
	require.Equal(t, devErr.fetchMessage(), "Some device error message")
}

func TestRunTrapWithoutMessage(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/ip/address @ []")
		s.writeSentence(t, "!trap", "=some=unknown key")
		s.writeSentence(t, "!done")
	}()

	_, err := c.Run("/ip/address")
	require.Error(t, err, "Run succeeded; want error")

	var devErr *DeviceError
	require.Truef(t, errors.As(err, &devErr), "want=DeviceError, have=%#v", err)
	require.Equal(t, devErr.fetchMessage(), "unknown error: !trap @ [{`some` `unknown key`}]")
}

func TestRunFatal(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/ip/address @ []")
		s.writeSentence(t, fatalSentence, "=message=Some device error message")
	}()

	_, err := c.Run("/ip/address")
	require.Error(t, err, "Run succeeded; want error")

	var devErr *DeviceError
	require.Truef(t, errors.As(err, &devErr), "want=DeviceError, have=%#v", err)
	require.Equal(t, devErr.fetchMessage(), "Some device error message")
}

func TestRunAfterClose(t *testing.T) {
	c, s := newPair(t)
	require.NoError(t, c.Close())
	require.NoError(t, s.Close())

	_, err := c.Run("/ip/address")
	require.Error(t, err, "Run succeeded; want error")
	require.EqualError(t, err, io.EOF.Error())
}

func TestListen(t *testing.T) {
	c, s := newPair(t)
	defer deferCloser(t, c)

	go func() {
		defer deferCloser(t, s)
		s.readSentence(t, "/ip/address/listen @l1 []")
		s.writeSentence(t, "!re", ".tag=l1", "=address=1.2.3.4/32")
		s.readSentence(t, "/cancel @r2 [{`tag` `l1`}]")
		s.writeSentence(t, "!trap", "=category=2", ".tag=l1")
		s.writeSentence(t, "!done", "=tag=r2")
		s.writeSentence(t, "!done", "=tag=l1")
	}()

	c.Queue = 1
	listen, err := c.Listen("/ip/address/listen")
	require.NoError(t, err)

	reC := listen.Chan()

	_, err = listen.Cancel()
	require.Equal(t, err, io.EOF)

	sen := <-reC
	want := "!re @l1 [{`address` `1.2.3.4/32`}]"
	require.Equalf(t, want, sen.String(), "/ip/address/listen (%s); want (%s)", sen, want)

	sen = <-reC
	require.Nilf(t, sen, "Listen() channel should be closed after Close(); got %#q", sen)
	require.NoError(t, listen.Err())
}

type conn struct {
	*io.PipeReader
	*io.PipeWriter
}

func (c *conn) Close() error {
	if err := c.PipeReader.Close(); err != nil {
		return err
	}

	return c.PipeWriter.Close()
}

func newPair(t *testing.T) (*Client, *fakeServer) {
	ar, aw := io.Pipe()
	br, bw := io.Pipe()

	c, err := NewClient(&conn{ar, bw})
	require.NoError(t, err)

	return c, &fakeServer{
		proto.NewReader(br),
		proto.NewWriter(aw),
		&conn{br, aw},
	}
}

type fakeServer struct {
	r proto.Reader
	w proto.Writer
	io.Closer
}

func (f *fakeServer) readSentence(t *testing.T, want string) {
	sen, err := f.r.ReadSentence()
	require.NoError(t, err)
	require.Equal(t, want, sen.String(), "wrong sentence")
	t.Logf("< %s\n", sen)
}

func (f *fakeServer) writeSentence(t *testing.T, sentence ...string) {
	t.Logf("> %#q\n", sentence)
	f.w.BeginSentence()
	for _, word := range sentence {
		f.w.WriteWord(word)
	}

	require.NoError(t, f.w.EndSentence())
}
