package routeros

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type liveTest struct {
	*testing.T
	c *Client
}

type testConfig struct {
	Address  string
	Username string
	Password string
}

func fetchConfig(t *testing.T) *testConfig {
	cfg := &testConfig{
		Address:  os.Getenv("ROUTEROS_TEST_ADDRESS"),
		Username: os.Getenv("ROUTEROS_TEST_USERNAME"),
		Password: os.Getenv("ROUTEROS_TEST_PASSWORD"),
	}

	if cfg.Address == "" || cfg.Username == "" || cfg.Password == "" {
		t.Skip("skipping integration tests because address or username or password is missing")
	}

	return cfg
}

func newLiveTest(t *testing.T) *liveTest {
	tt := &liveTest{T: t}
	tt.connect()
	return tt
}

func (t *liveTest) connect() {
	cfg := fetchConfig(t.T)

	var err error
	t.c, err = DialContext(context.Background(), cfg.Address, cfg.Username, cfg.Password)
	require.NoError(t, err)
}

func (t *liveTest) runContext(ctx context.Context, sentence ...string) *Reply {
	t.Logf("Run: %#q", sentence)

	r, err := t.c.RunArgsContext(ctx, sentence)
	require.NoError(t, err)
	require.NotNil(t, r)
	require.NotNil(t, r.Done, "done not received")

	t.Logf("Reply: %s", r)
	return r
}

func (t *liveTest) getUptime() {
	// allow test to fail after 5 seconds if we didn't receive answer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r := t.runContext(ctx, "/system/resource/print")
	require.Len(t, r.Re, 1, "expected 1 response")

	_, ok := r.Re[0].Map["uptime"]
	require.True(t, ok, "missing uptime")
}

func deferCloser(t *testing.T, c io.Closer) {
	require.NoError(t, c.Close())
}

func TestRunSync(tt *testing.T) {
	t := newLiveTest(tt)
	defer deferCloser(tt, t.c)
	t.getUptime()
}

func TestRunAsync(tt *testing.T) {
	// allow test to fail after 5 seconds if we didn't receive answer
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	t := newLiveTest(tt)
	defer deferCloser(tt, t.c)

	t.c.AsyncContext(ctx)
	require.True(tt, t.c.async, "client should be in async mode")
	t.getUptime()
}

func TestRunError(tt *testing.T) {
	t := newLiveTest(tt)
	defer deferCloser(tt, t.c)
	for i, sentence := range [][]string{
		{"/xxx"},
		{"/ip/address/add", "=address=127.0.0.2/32", "=interface=xxx"},
	} {
		t.Logf("#%d: Run: %#q", i, sentence)
		_, err := t.c.RunArgs(sentence)
		if err == nil {
			t.Error("Success; want error from RouterOS device trying to run an invalid command")
		}
	}
}

func TestDialInvalidPort(t *testing.T) {
	con, err := Dial("127.0.0.1:xxx", "x", "x")
	if con != nil {
		assert.NoError(t, con.Close())
	}

	require.Error(t, err)
	require.IsType(t, &net.OpError{}, errors.Unwrap(err))

	var e *net.DNSError
	require.True(t, errors.As(err, &e))
	require.Contains(t, e.Err, "unknown port")
}

func TestDialTimeout(t *testing.T) {
	con, err := DialTimeout("255.255.255.0:8729", "x", "x", time.Millisecond)
	if con != nil {
		assert.NoError(t, con.Close())
	}

	require.Error(t, err)

	var e net.Error
	require.Truef(t, errors.As(err, &e), "want=net.Error have=%q", err)
	require.Truef(t, e.Timeout(), `expected="i/o timeout", have=%q`, e)
}

func TestDialTLSTimeout(t *testing.T) {
	con, err := DialTLSTimeout("255.255.255.0:8729", "x", "x", nil, time.Millisecond)
	if con != nil {
		assert.NoError(t, con.Close())
	}

	require.Error(t, err)

	var e net.Error
	require.Truef(t, errors.As(err, &e), "want=net.Error have=%q", err)
	require.Truef(t, e.Timeout(), `expected="i/o timeout", have=%q`, e)
}

func TestDialTLSInvalidPort(t *testing.T) {
	con, err := DialTLS("127.0.0.1:xxx", "x", "x", nil)
	if con != nil {
		assert.NoError(t, con.Close())
	}

	require.Error(t, err)
	require.IsType(t, &net.OpError{}, errors.Unwrap(err))

	var e *net.DNSError
	require.True(t, errors.As(err, &e))
	require.Contains(t, e.Err, "unknown port")
}

func TestInvalidLogin(t *testing.T) {
	cfg := fetchConfig(t)

	c, err := Dial(cfg.Address, "xxx", "APasswordThatWillNeverExistir")
	if c != nil {
		assert.NoError(t, c.Close())
	}

	require.Error(t, err, "dial succeeded; want error")

	var devErr *DeviceError
	require.Truef(t, errors.As(err, &devErr), "wait for device error: %v", err)
	require.Contains(t, []string{"cannot log in", "invalid user name or password (6)"}, devErr.fetchMessage())
}

func TestTrapHandling(tt *testing.T) {
	t := newLiveTest(tt)
	defer deferCloser(tt, t.c)

	cmd := []string{"/ip/dns/static/add", "=type=A", "=name=example.com", "=ttl=30", "=address=1.0.0.0"}

	_, _ = t.c.RunArgs(cmd)
	_, err := t.c.RunArgs(cmd)
	require.Error(tt, err, "should've returned an error due to a duplicate")

	var devErr *DeviceError
	require.True(t, errors.As(err, &devErr), "should've returned a DeviceError")
	require.Contains(tt, devErr.Sentence.Map["message"], "entry already exists")
}
