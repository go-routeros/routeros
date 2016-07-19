package routeros

import (
	"fmt"
	"strings"
)

// Pair is a Key-Value pair for RouterOS Attribute, Query, and Reply words
// use slices of pairs instead of map because we care about order
type Pair struct {
	Key   string
	Value string
	// Op is used for Query words to signify logical operations
	// valid operators are -, =, <, >
	// see http://wiki.mikrotik.com/wiki/Manual:API#Queries for details.
	Op string
}

type Query struct {
	Pairs    []Pair
	Op       string
	Proplist []string
}

func (c *Client) Query(command string, q Query) (*Reply, error) {
	sentence := []string{command}
	if len(q.Proplist) > 0 {
		word := fmt.Sprintf("=.proplist=%s", strings.Join(q.Proplist, ","))
		sentence = append(sentence, word)
	}
	if len(q.Pairs) > 0 {
		for _, v := range q.Pairs {
			word := fmt.Sprintf("?%s%s=%s", v.Op, v.Key, v.Value)
			sentence = append(sentence, word)
		}
		if q.Op != "" {
			word := fmt.Sprintf("?#%s", q.Op)
			sentence = append(sentence, word)
		}
	}
	return c.RunArgs(sentence)
}

func (c *Client) Call(command string, params []Pair) (*Reply, error) {
	sentence := []string{command}
	if len(params) > 0 {
		for _, v := range params {
			word := fmt.Sprintf("=%s=%s", v.Key, v.Value)
			sentence = append(sentence, word)
		}
	}
	return c.RunArgs(sentence)
}
