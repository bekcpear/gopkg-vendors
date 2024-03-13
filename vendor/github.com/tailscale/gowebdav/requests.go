package gowebdav

import (
	"context"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
)

func (c *Client) req(ctx context.Context, method, path string, body io.Reader, intercept func(*http.Request)) (rs *http.Response, err error) {
	var r *http.Request
	var uri = PathEscape(Join(c.root, path))

	if r, err = http.NewRequestWithContext(ctx, method, uri, body); err != nil {
		return
	}

	for k, vals := range c.headers {
		for _, v := range vals {
			r.Header.Add(k, v)
		}
	}

	if intercept != nil {
		intercept(r)
	}

	if c.interceptor != nil {
		c.interceptor(method, r)
	}

	return c.c.Do(r)
}

func (c *Client) mkcol(ctx context.Context, path string) (status int, err error) {
	rs, err := c.req(ctx, "MKCOL", path, nil, nil)
	if err != nil {
		return
	}
	defer rs.Body.Close()

	status = rs.StatusCode
	if status == 405 {
		status = 201
	}

	return
}

func (c *Client) options(ctx context.Context, path string) (*http.Response, error) {
	return c.req(ctx, "OPTIONS", path, nil, func(rq *http.Request) {
		rq.Header.Add("Depth", "0")
	})
}

func (c *Client) propfind(ctx context.Context, path string, self bool, body string, resp interface{}, parse func(resp interface{}) error) error {
	rs, err := c.req(ctx, "PROPFIND", path, strings.NewReader(body), func(rq *http.Request) {
		if self {
			rq.Header.Add("Depth", "0")
		} else {
			rq.Header.Add("Depth", "1")
		}
		rq.Header.Add("Content-Type", "application/xml;charset=UTF-8")
		rq.Header.Add("Accept", "application/xml,text/xml")
		rq.Header.Add("Accept-Charset", "utf-8")
		// TODO add support for 'gzip,deflate;q=0.8,q=0.7'
		rq.Header.Add("Accept-Encoding", "")
	})
	if err != nil {
		return err
	}
	defer rs.Body.Close()

	if rs.StatusCode != 207 {
		return NewPathError("PROPFIND", path, rs.StatusCode)
	}

	return parseXML(rs.Body, resp, parse)
}

func (c *Client) doCopyMove(
	ctx context.Context,
	method string,
	oldpath string,
	newpath string,
	overwrite bool,
) (
	status int,
	r io.ReadCloser,
	err error,
) {
	rs, err := c.req(ctx, method, oldpath, nil, func(rq *http.Request) {
		rq.Header.Add("Destination", PathEscape(Join(c.root, newpath)))
		if overwrite {
			rq.Header.Add("Overwrite", "T")
		} else {
			rq.Header.Add("Overwrite", "F")
		}
	})
	if err != nil {
		return
	}
	status = rs.StatusCode
	r = rs.Body
	return
}

func (c *Client) copymove(ctx context.Context, method string, oldpath string, newpath string, overwrite bool) (err error) {
	s, data, err := c.doCopyMove(ctx, method, oldpath, newpath, overwrite)
	if err != nil {
		return
	}
	if data != nil {
		defer data.Close()
	}

	switch s {
	case 201, 204:
		return nil

	case 207:
		// TODO handle multistat errors, worst case ...
		log.Printf("TODO handle %s - %s multistatus result %s\n", method, oldpath, String(data))

	case 409:
		err := c.createParentCollection(ctx, newpath)
		if err != nil {
			return err
		}

		return c.copymove(ctx, method, oldpath, newpath, overwrite)
	}

	return NewPathError(method, oldpath, s)
}

func (c *Client) put(ctx context.Context, path string, stream io.Reader) (status int, err error) {
	rs, err := c.req(ctx, "PUT", path, stream, nil)
	if err != nil {
		return
	}
	defer rs.Body.Close()

	status = rs.StatusCode
	return
}

func (c *Client) createParentCollection(ctx context.Context, itemPath string) (err error) {
	parentPath := path.Dir(itemPath)
	if parentPath == "." || parentPath == "/" {
		return nil
	}

	return c.MkdirAll(ctx, parentPath, 0755)
}