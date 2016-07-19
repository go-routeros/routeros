package routeros

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"testing"

	"github.com/andre-luiz-dos-santos/routeros-go/proto"
)

type TestVars struct {
	Username string
	Password string
	Address  string
}

// Make sure we have the env vars to run, handle bailing if we don't
func PrepVars(t *testing.T) TestVars {
	var tv TestVars

	addr := os.Getenv("ROS_TEST_TARGET")
	if addr == "" {
		t.Skip("Can't run test because ROS_TEST_TARGET undefined")
	} else {
		tv.Address = addr
	}

	username := os.Getenv("ROS_TEST_USER")
	if username == "" {
		tv.Username = "admin"
		t.Logf("ROS_TEST_USER not defined. Assuming %s\n", tv.Username)
	} else {
		tv.Username = username
	}

	password := os.Getenv("ROS_TEST_PASSWORD")
	if password == "" {
		tv.Password = "admin"
		t.Logf("ROS_TEST_PASSWORD not defined. Assuming %s\n", tv.Password)
	} else {
		tv.Password = password
	}

	return tv
}

// Test logging in and out
func TestLogin(t *testing.T) {
	tv := PrepVars(t)
	c := &Client{
		Address:  tv.Address,
		Username: tv.Username,
		Password: tv.Password,
	}
	err := c.Connect()
	if err != nil {
		t.Fatal(err)
	}
}

// Test running a command (uptime)
func TestCommand(t *testing.T) {
	tv := PrepVars(t)
	c := &Client{
		Address:  tv.Address,
		Username: tv.Username,
		Password: tv.Password,
	}
	err := c.Connect()
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.Call("/system/resource/getall", nil)
	if err != nil {
		t.Fatal(err)
	}
	uptime := res.Re[0].Map["uptime"]
	t.Logf("Uptime: %s\n", uptime)
}

func TestCommandAsyncA(t *testing.T) {
	tv := PrepVars(t)
	c := &Client{
		Address:  tv.Address,
		Username: tv.Username,
		Password: tv.Password,
	}
	err := c.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Async()
	res, err := c.Call("/system/resource/getall", nil)
	if err != nil {
		t.Fatal(err)
	}
	uptime := res.Re[0].Map["uptime"]
	t.Logf("Uptime: %s\n", uptime)
}

// func TestCommandAsyncB(t *testing.T) {
// 	tv := PrepVars(t)
// 	c := &Client{
// 		Address:  tv.Address,
// 		Username: tv.Username,
// 		Password: tv.Password,
// 	}
// 	err := c.Connect()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	go func() {
// 		defer c.Close()
// 		res, err := c.Call("/system/resource/getall", nil)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		uptime := res.Re[0].Map["uptime"]
// 		t.Logf("Uptime: %s\n", uptime)
// 	}()
// 	c.Loop()
// }

// Test querying data (getting IP addresses on ether1)
func TestQuery(t *testing.T) {
	tv := PrepVars(t)
	c := &Client{
		Address:  tv.Address,
		Username: tv.Username,
		Password: tv.Password,
	}
	err := c.Connect()
	if err != nil {
		t.Fatal(err)
	}

	res, err := c.Query("/ip/address/print", Query{
		Pairs:    []Pair{Pair{"interface", "ether1", "="}},
		Proplist: []string{"address"},
	})
	if err != nil {
		t.Error(err)
	}

	t.Log("IP addresses on ether1:")
	for _, v := range res.Re {
		for _, sv := range v.List {
			t.Log(sv)
		}
	}
}

// Test adding some bridges (test of Call)
func TestCallAddBridges(t *testing.T) {
	tv := PrepVars(t)
	c := &Client{
		Address:  tv.Address,
		Username: tv.Username,
		Password: tv.Password,
	}
	err := c.Connect()
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 10; i++ {
		var pairs []Pair
		bName := "test-bridge" + strconv.Itoa(i)
		pairs = append(pairs, Pair{Key: "name", Value: bName})
		pairs = append(pairs, Pair{Key: "comment", Value: "test bridge number " + strconv.Itoa(i)})
		pairs = append(pairs, Pair{Key: "arp", Value: "disabled"})
		res, err := c.Call("/interface/bridge/add", pairs)
		if err != nil {
			t.Errorf("Error adding bridge: %s\n", err)
		}
		t.Logf("reply from adding bridge: %+v\n", res)
	}
}

// Test getting list of interfaces (test Query)
func TestQueryMultiple(t *testing.T) {
	tv := PrepVars(t)
	c := &Client{
		Address:  tv.Address,
		Username: tv.Username,
		Password: tv.Password,
	}
	err := c.Connect()
	if err != nil {
		t.Fatal(err)
	}

	var q Query
	q.Pairs = append(q.Pairs, Pair{Key: "type", Value: "bridge", Op: "="})

	res, err := c.Query("/interface/print", q)
	if err != nil {
		t.Error(err)
	}
	if len(res.Re) <= 1 {
		t.Error("Did not get multiple SubPairs from bridge interface query")
	}
}

// Test query with proplist
func TestQueryWithProplist(t *testing.T) {
	tv := PrepVars(t)
	c := &Client{
		Address:  tv.Address,
		Username: tv.Username,
		Password: tv.Password,
	}
	err := c.Connect()
	if err != nil {
		t.Fatal(err)
	}

	var q Query
	q.Proplist = append(q.Proplist, "name")
	q.Proplist = append(q.Proplist, "comment")
	q.Proplist = append(q.Proplist, ".id")
	q.Pairs = append(q.Pairs, Pair{Key: "type", Value: "bridge", Op: "="})
	res, err := c.Query("/interface/print", q)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range res.Re {
		b := s.Map
		t.Logf("Found bridge %s (%s)\n", b["name"], b["comment"])

	}
}

// Test query with proplist
func TestCallRemoveBridges(t *testing.T) {
	tv := PrepVars(t)
	c := &Client{
		Address:  tv.Address,
		Username: tv.Username,
		Password: tv.Password,
	}
	err := c.Connect()
	if err != nil {
		t.Fatal(err)
	}

	var q Query
	q.Proplist = append(q.Proplist, ".id")
	q.Pairs = append(q.Pairs, Pair{Key: "type", Value: "bridge", Op: "="})
	res, err := c.Query("/interface/print", q)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range res.Re {
		v := s.Map
		var pairs []Pair
		pairs = append(pairs, Pair{Key: ".id", Value: v[".id"]})
		_, err = c.Call("/interface/bridge/remove", pairs)
		if err != nil {
			t.Errorf("error removing bridge: %s\n", err)
		}
	}
}

// Test call that should trigger error response from router
func TestCallCausesError(t *testing.T) {
	tv := PrepVars(t)
	c := &Client{
		Address:  tv.Address,
		Username: tv.Username,
		Password: tv.Password,
	}
	err := c.Connect()
	if err != nil {
		t.Fatal(err)
	}

	var pairs []Pair
	pairs = append(pairs, Pair{Key: "address", Value: "192.168.99.1/32"})
	pairs = append(pairs, Pair{Key: "comment", Value: "this address should never be added"})
	pairs = append(pairs, Pair{Key: "interface", Value: "badbridge99"})
	_, err = c.Call("/ip/address/add", pairs)
	if err != nil {
		t.Logf("Error adding address to nonexistent bridge: %s\n", err)
	} else {
		t.Error("did not get error when adding address to nonexistent bridge")
	}
}

// Test query that should trigger error response from router
func TestQueryCausesError(t *testing.T) {
	tv := PrepVars(t)
	c := &Client{
		Address:  tv.Address,
		Username: tv.Username,
		Password: tv.Password,
	}
	err := c.Connect()
	if err != nil {
		t.Fatal(err)
	}

	var q Query
	q.Proplist = append(q.Proplist, ".id")
	_, err = c.Query("/ip/address/sneeze", q)
	if err != nil {
		t.Logf("Error querying with nonexistent command: %s\n", err)
	} else {
		t.Error("did not get error when querying nonexistent command")
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
		{r(s("!trap")), `RouterOS: unknown error: !trap []`},
		{r(s("!trap", "=message=abc123")), `RouterOS: abc123`},
		{r(s("!fatal")), `RouterOS: unknown error: !fatal []`},
		{r(s("!fatal", "=message=abc123")), `RouterOS: abc123`},
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
