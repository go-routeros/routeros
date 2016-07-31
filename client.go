/*
Package routeros is a pure Go client library for accessing Mikrotik devices using the RouterOS API.
*/
package routeros

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"gopkg.in/routeros.v1/proto"
)

// Client is a RouterOS API client.
type Client struct {
	Address  string
	Username string
	Password string
	Queue    int
	conn     net.Conn
	r        proto.Reader
	w        *proto.Writer
	closing  bool
	async    bool
	nextTag  int64
	tags     map[string]sentenceProcessor
	sync.Mutex
}

// Connect connects and login to the RouterOS device.
func (c *Client) Connect() error {
	if c.conn != nil {
		return errAlreadyConnected
	}
	conn, err := net.Dial("tcp", c.Address)
	if err != nil {
		return err
	}
	return c.connect(conn)
}

// ConnectTLS connects and login to the RouterOS device using TLS.
func (c *Client) ConnectTLS(tlsConfig *tls.Config) error {
	if c.conn != nil {
		return errAlreadyConnected
	}
	conn, err := tls.Dial("tcp", c.Address, tlsConfig)
	if err != nil {
		return err
	}
	return c.connect(conn)
}

// Close closes the connection to the RouterOS device.
func (c *Client) Close() {
	if c.conn == nil || c.closing {
		return
	}
	c.closing = true
	c.conn.Close()
}

func (c *Client) connect(conn net.Conn) error {
	c.conn = conn
	c.r = proto.NewReader(conn)
	c.w = proto.NewWriter(conn)

	err := c.login()
	if err != nil {
		c.Close()
		return err
	}

	return nil
}

func (c *Client) login() error {
	r, err := c.Run("/login")
	if err != nil {
		return err
	}
	ret, ok := r.Done.Map["ret"]
	if !ok {
		return errors.New("RouterOS: /login: no ret (challenge) received")
	}
	b, err := hex.DecodeString(ret)
	if err != nil {
		return fmt.Errorf("RouterOS: /login: invalid ret (challenge) hex string received: %s", err)
	}

	r, err = c.Run("/login", "=name="+c.Username, "=response="+c.challengeResponse(b))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) challengeResponse(cha []byte) string {
	h := md5.New()
	h.Write([]byte{0})
	io.WriteString(h, c.Password)
	h.Write(cha)
	return fmt.Sprintf("00%x", h.Sum(nil))
}
