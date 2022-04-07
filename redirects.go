// Package redirects provides Netlify style _redirects file format parsing.
package redirects

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Params is a map of key/value pairs.
type Params map[string]interface{}

// Has returns true if the param is present.
func (p *Params) Has(key string) bool {
	if p == nil {
		return false
	}

	_, ok := (*p)[key]
	return ok
}

// Get returns the key value.
func (p *Params) Get(key string) interface{} {
	if p == nil {
		return nil
	}

	return (*p)[key]
}

// A Rule represents a single redirection or rewrite rule.
type Rule struct {
	// From is the path which is matched to perform the rule.
	From string

	// To is the destination which may be relative, or absolute
	// in order to proxy the request to another URL.
	To string

	// Status is one of the following:
	//
	// - 3xx a redirect
	// - 200 a rewrite
	// - defaults to 301 redirect
	//
	// When proxying this field is ignored.
	//
	Status int

	// Force is used to force a rewrite or redirect even
	// when a response (or static file) is present.
	Force bool

	// Params is an optional arbitrary map of key/value pairs.
	Params Params
}

// IsRewrite returns true if the rule represents a rewrite (status 200).
func (r *Rule) IsRewrite() bool {
	return r.Status == 200
}

// IsProxy returns true if it's a proxy rule (aka contains a hostname).
func (r *Rule) IsProxy() bool {
	u, err := url.Parse(r.To)
	if err != nil {
		return false
	}

	return u.Host != ""
}

// Must parse utility.
func Must(v []Rule, err error) []Rule {
	if err != nil {
		panic(err)
	}

	return v
}

// Parse the given reader.
func Parse(r io.Reader) (rules []Rule, err error) {
	s := bufio.NewScanner(r)

	for s.Scan() {
		line := strings.TrimSpace(s.Text())

		// empty
		if line == "" {
			continue
		}

		// comment
		if strings.HasPrefix(line, "#") {
			continue
		}

		// fields
		fields := strings.Fields(line)

		// missing dst
		if len(fields) <= 1 {
			return nil, fmt.Errorf("missing destination path: %q", line)
		}

		// src and dst
		rule := Rule{
			From:   fields[0],
			To:     fields[1],
			Status: 301,
		}

		// status
		if len(fields) > 2 {
			code, force, err := parseStatus(fields[2])
			if err != nil {
				return nil, errors.Wrapf(err, "parsing status %q", fields[2])
			}

			rule.Status = code
			rule.Force = force
		}

		// params
		if len(fields) > 3 {
			rule.Params = parseParams(fields[3:])
		}

		rules = append(rules, rule)
	}

	err = s.Err()
	return
}

// ParseString parses the given string.
func ParseString(s string) ([]Rule, error) {
	return Parse(strings.NewReader(s))
}

// parseParams returns parsed param key/value pairs.
func parseParams(pairs []string) Params {
	m := make(Params)

	for _, p := range pairs {
		parts := strings.Split(p, "=")
		if len(parts) > 1 {
			m[parts[0]] = parts[1]
		} else {
			m[parts[0]] = true
		}
	}

	return m
}

// parseStatus returns the status code and force when "!" suffix is present.
func parseStatus(s string) (code int, force bool, err error) {
	if strings.HasSuffix(s, "!") {
		force = true
		s = strings.Replace(s, "!", "", -1)
	}

	code, err = strconv.Atoi(s)
	return
}
