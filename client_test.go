package routeros

import (
	"bytes"
	"flag"
	"io"
	"testing"

	"github.com/go-routeros/routeros/proto"
)

var (
	routerosAddress  = flag.String("routeros.address", "", "RouterOS address:port")
	routerosUsername = flag.String("routeros.username", "admin", "RouterOS user name")
	routerosPassword = flag.String("routeros.password", "admin", "RouterOS password")
)

type liveTest struct {
	*testing.T
	c *Client
}

func newLiveTest(t *testing.T) *liveTest {
	tt := &liveTest{T: t}
	tt.connect()
	return tt
}

func (t *liveTest) connect() {
	if *routerosAddress == "" {
		t.Skip("Flag -routeros.address not set")
	}
	var err error
	t.c, err = Dial(*routerosAddress, *routerosUsername, *routerosPassword)
	if err != nil {
		t.Fatal(err)
	}
}

func (t *liveTest) run(sentence ...string) *Reply {
	t.Logf("Run: %#q", sentence)
	r, err := t.c.RunArgs(sentence)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Reply: %s", r)
	return r
}

func (t *liveTest) getUptime() {
	r := t.run("/system/resource/print")
	if len(r.Re) != 1 {
		t.Fatalf("len(!re)=%d; want 1", len(r.Re))
	}
	_, ok := r.Re[0].Map["uptime"]
	if !ok {
		t.Fatal("Missing uptime")
	}
}

func TestRunSync(tt *testing.T) {
	t := newLiveTest(tt)
	defer t.c.Close()
	t.getUptime()
}

func TestRunAsync(tt *testing.T) {
	t := newLiveTest(tt)
	defer t.c.Close()
	t.c.Async()
	t.getUptime()
}

func TestRunError(tt *testing.T) {
	t := newLiveTest(tt)
	defer t.c.Close()
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

type sentenceTester struct {
	sentences []*proto.Sentence
}

func newSentenceTester(sentences []*proto.Sentence) *Client {
	return &Client{r: &sentenceTester{sentences}}
}

func (p *sentenceTester) ReadSentence() (*proto.Sentence, error) {
	if len(p.sentences) == 0 {
		return nil, io.EOF
	}
	s := p.sentences[0]
	p.sentences = p.sentences[1:]
	return s, nil
}

func TestReceive(t *testing.T) {
	// Return a list of sentences.
	r := func(sentences ...*proto.Sentence) []*proto.Sentence {
		return sentences
	}
	// Return one sentence.
	s := func(words ...string) *proto.Sentence {
		b := &bytes.Buffer{}
		w := proto.NewWriter(b)
		for _, word := range words {
			w.WriteWord(word)
		}
		w.WriteWord("")
		w.Flush()
		r := proto.NewReader(b)
		sen, err := r.ReadSentence()
		if err != nil {
			t.Fatalf("ReadSentence(%#q)=%#v", words, err)
		}
		return sen
	}
	// Valid replies.
	for i, test := range []struct {
		in  []*proto.Sentence
		out string
	}{
		{r(s("!done")), `!done []`},
		{r(s(), s("!done")), `!done []`},
		{r(s("!done", "=name")), "!done [{`name` ``}]"},
		{r(s("!done", "=ret=abc123")), "!done [{`ret` `abc123`}]"},
		{r(s("!re", "=name=value"), s("!done")), "!re [{`name` `value`}]\n!done []"},
	} {
		c := newSentenceTester(test.in)
		reply, err := c.readReply()
		if err != nil {
			t.Errorf("#%d: Input(%#q)=%#v", i, test.in, err)
			continue
		}
		x := reply.String()
		if x != test.out {
			t.Errorf("#%d: Input(%#q)=%#q; want %#q", i, test.in, x, test.out)
		}
	}
	// Must return EOF.
	for i, test := range []struct {
		in []*proto.Sentence
	}{
		{r()},
		{r(s())},
		{r(s("!re", "=name=value"))},
	} {
		c := newSentenceTester(test.in)
		_, err := c.readReply()
		if err != io.EOF {
			t.Errorf("#%d: Input(%#q)=%s; want EOF", i, test.in, err)
		}
	}
	// Must return ErrUnknownReply.
	for i, test := range []struct {
		in  []*proto.Sentence
		out string
	}{
		{r(s("=name")), `unknown RouterOS reply word: =name`},
		{r(s("=ret=abc123")), `unknown RouterOS reply word: =ret=abc123`},
	} {
		c := newSentenceTester(test.in)
		_, err := c.readReply()
		_, ok := err.(*UnknownReplyError)
		if !ok {
			t.Errorf("#%d: Input(%#q)=%T; want *UnknownReplyError", i, test.in, err)
			continue
		}
		x := err.Error()
		if x != test.out {
			t.Errorf("#%d: Input(%#q)=%#q; want %#q", i, test.in, x, test.out)
		}
	}
	// Must return ErrFromDevice.
	for i, test := range []struct {
		in  []*proto.Sentence
		out string
	}{
		{r(s("!trap")), `from RouterOS device: unknown error: !trap []`},
		{r(s("!trap", "=message=abc123")), `from RouterOS device: abc123`},
		{r(s("!fatal")), `from RouterOS device: unknown error: !fatal []`},
		{r(s("!fatal", "=message=abc123")), `from RouterOS device: abc123`},
	} {
		c := newSentenceTester(test.in)
		_, err := c.readReply()
		_, ok := err.(*DeviceError)
		if !ok {
			t.Errorf("#%d: Input(%#q)=%T; want *DeviceError", i, test.in, err)
			continue
		}
		x := err.Error()
		if x != test.out {
			t.Errorf("#%d: Input(%#q)=%#q; want %#q", i, test.in, x, test.out)
		}
	}
}
