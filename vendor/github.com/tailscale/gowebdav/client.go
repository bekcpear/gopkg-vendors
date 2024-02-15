package gowebdav

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	pathpkg "path"
	"strings"
	"time"
)

const XInhibitRedirect = "X-Gowebdav-Inhibit-Redirect"

// Client defines our structure
type Client struct {
	root        string
	headers     http.Header
	interceptor func(method string, rq *http.Request)
	c           *http.Client
	auth        Authorizer
}

type Opts struct {
	URI       string
	Auth      Authorizer
	Transport http.RoundTripper
}

// NewClient creates a new instance of client
func NewClient(uri, user, pw string) *Client {
	return NewAuthClient(uri, NewAutoAuth(user, pw))
}

// NewAuthClient creates a new client instance with a custom Authorizer
func NewAuthClient(uri string, auth Authorizer) *Client {
	return New(&Opts{URI: uri, Auth: auth})
}

func New(opts *Opts) *Client {
	if opts.Auth == nil {
		opts.Auth = NewAutoAuth("", "")
	}

	c := &http.Client{
		CheckRedirect: func(rq *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return ErrTooManyRedirects
			}
			if via[0].Header.Get(XInhibitRedirect) != "" {
				return http.ErrUseLastResponse
			}
			return nil
		},
		Transport: opts.Transport,
	}
	return &Client{root: FixSlash(opts.URI), headers: make(http.Header), interceptor: nil, c: c, auth: opts.Auth}
}

// SetHeader lets us set arbitrary headers for a given client
func (c *Client) SetHeader(key, value string) {
	c.headers.Add(key, value)
}

// SetInterceptor lets us set an arbitrary interceptor for a given client
func (c *Client) SetInterceptor(interceptor func(method string, rq *http.Request)) {
	c.interceptor = interceptor
}

// SetTimeout exposes the ability to set a time limit for requests
func (c *Client) SetTimeout(timeout time.Duration) {
	c.c.Timeout = timeout
}

// SetTransport exposes the ability to define custom transports
func (c *Client) SetTransport(transport http.RoundTripper) {
	c.c.Transport = transport
}

// SetJar exposes the ability to set a cookie jar to the client.
func (c *Client) SetJar(jar http.CookieJar) {
	c.c.Jar = jar
}

// Connect connects to our dav server
func (c *Client) Connect(ctx context.Context) error {
	rs, err := c.options(ctx, "/")
	if err != nil {
		return err
	}

	err = rs.Body.Close()
	if err != nil {
		return err
	}

	if rs.StatusCode != 200 {
		return NewPathError("Connect", c.root, rs.StatusCode)
	}

	return nil
}

type props struct {
	Status      string   `xml:"DAV: status"`
	Name        string   `xml:"DAV: prop>displayname,omitempty"`
	Type        xml.Name `xml:"DAV: prop>resourcetype>collection,omitempty"`
	Size        string   `xml:"DAV: prop>getcontentlength,omitempty"`
	ContentType string   `xml:"DAV: prop>getcontenttype,omitempty"`
	ETag        string   `xml:"DAV: prop>getetag,omitempty"`
	Created     string   `xml:"DAV: prop>creationdate,omitempty"`
	Modified    string   `xml:"DAV: prop>getlastmodified,omitempty"`
}

type response struct {
	Href  string  `xml:"DAV: href"`
	Props []props `xml:"DAV: propstat"`
}

func getProps(r *response, status string) *props {
	for _, prop := range r.Props {
		if strings.Contains(prop.Status, status) {
			return &prop
		}
	}
	return nil
}

// ReadDir reads the contents of a remote directory
func (c *Client) ReadDir(ctx context.Context, path string) ([]os.FileInfo, error) {
	path = FixSlashes(path)
	files := make([]os.FileInfo, 0)
	skipSelf := true
	parse := func(resp interface{}) error {
		r := resp.(*response)

		if skipSelf {
			skipSelf = false
			if p := getProps(r, "200"); p != nil && p.Type.Local == "collection" {
				r.Props = nil
				return nil
			}
			return NewPathError("ReadDir", path, 405)
		}

		if p := getProps(r, "200"); p != nil {
			f := new(File)
			if ps, err := url.PathUnescape(r.Href); err == nil {
				f.name = pathpkg.Base(ps)
			} else {
				f.name = p.Name
			}
			f.path = path + f.name
			f.created = parseTime(&p.Created)
			f.modified = parseTime(&p.Modified)
			f.etag = p.ETag
			f.contentType = p.ContentType

			if p.Type.Local == "collection" {
				f.path += "/"
				f.size = 0
				f.isdir = true
			} else {
				f.size = parseInt64(&p.Size)
				f.isdir = false
			}

			files = append(files, *f)
		}

		r.Props = nil
		return nil
	}

	err := c.propfind(ctx, path, false,
		`<d:propfind xmlns:d='DAV:'>
			<d:prop>
				<d:displayname/>
				<d:resourcetype/>
				<d:getcontentlength/>
				<d:getcontenttype/>
				<d:getetag/>
				<d:creationdate/>
				<d:getlastmodified/>
			</d:prop>
		</d:propfind>`,
		&response{},
		parse)

	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			err = NewPathErrorErr("ReadDir", path, err)
		}
	}
	return files, err
}

// Stat returns the file stats for a specified path
func (c *Client) Stat(ctx context.Context, path string) (os.FileInfo, error) {
	var f *File
	parse := func(resp interface{}) error {
		r := resp.(*response)
		if p := getProps(r, "200"); p != nil && f == nil {
			f = new(File)
			f.name = p.Name
			f.path = path
			f.etag = p.ETag
			f.contentType = p.ContentType

			if p.Type.Local == "collection" {
				if !strings.HasSuffix(f.path, "/") {
					f.path += "/"
				}
				f.size = 0
				f.created = parseTime(&p.Created)
				f.modified = parseTime(&p.Modified)
				f.isdir = true
			} else {
				f.size = parseInt64(&p.Size)
				f.created = parseTime(&p.Created)
				f.modified = parseTime(&p.Modified)
				f.isdir = false
			}
		}

		r.Props = nil
		return nil
	}

	err := c.propfind(ctx, path, true,
		`<d:propfind xmlns:d='DAV:'>
			<d:prop>
				<d:displayname/>
				<d:resourcetype/>
				<d:getcontentlength/>
				<d:getcontenttype/>
				<d:getetag/>
				<d:creationdate/>
				<d:getlastmodified/>
			</d:prop>
		</d:propfind>`,
		&response{},
		parse)

	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			err = NewPathErrorErr("ReadDir", path, err)
		}
	}
	return f, err
}

// Remove removes a remote file
func (c *Client) Remove(ctx context.Context, path string) error {
	return c.RemoveAll(ctx, path)
}

// RemoveAll removes remote files
func (c *Client) RemoveAll(ctx context.Context, path string) error {
	rs, err := c.req(ctx, "DELETE", path, nil, nil)
	if err != nil {
		return NewPathError("Remove", path, 400)
	}
	err = rs.Body.Close()
	if err != nil {
		return err
	}

	if rs.StatusCode == 200 || rs.StatusCode == 204 || rs.StatusCode == 404 {
		return nil
	}

	return NewPathError("Remove", path, rs.StatusCode)
}

// Mkdir makes a directory
func (c *Client) Mkdir(ctx context.Context, path string, _ os.FileMode) (err error) {
	path = FixSlashes(path)
	status, err := c.mkcol(ctx, path)
	if err != nil {
		return
	}
	if status == 201 {
		return nil
	}

	return NewPathError("Mkdir", path, status)
}

// MkdirAll like mkdir -p, but for webdav
func (c *Client) MkdirAll(ctx context.Context, path string, _ os.FileMode) (err error) {
	path = FixSlashes(path)
	status, err := c.mkcol(ctx, path)
	if err != nil {
		return
	}
	if status == 201 {
		return nil
	}
	if status == 409 {
		paths := strings.Split(path, "/")
		sub := "/"
		for _, e := range paths {
			if e == "" {
				continue
			}
			sub += e + "/"
			status, err = c.mkcol(ctx, sub)
			if err != nil {
				return
			}
			if status != 201 {
				return NewPathError("MkdirAll", sub, status)
			}
		}
		return nil
	}

	return NewPathError("MkdirAll", path, status)
}

// Rename moves a file from A to B
func (c *Client) Rename(ctx context.Context, oldpath, newpath string, overwrite bool) error {
	return c.copymove(ctx, "MOVE", oldpath, newpath, overwrite)
}

// Copy copies a file from A to B
func (c *Client) Copy(ctx context.Context, oldpath, newpath string, overwrite bool) error {
	return c.copymove(ctx, "COPY", oldpath, newpath, overwrite)
}

// Read reads the contents of a remote file
func (c *Client) Read(ctx context.Context, path string) ([]byte, error) {
	var stream io.ReadCloser
	var err error

	if stream, err = c.ReadStream(ctx, path); err != nil {
		return nil, err
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(stream)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ReadStream reads the stream for a given path
func (c *Client) ReadStream(ctx context.Context, path string) (io.ReadCloser, error) {
	return c.ReadStreamOffset(ctx, path, 0)
}

// ReadStreamOffset reads the stream for a given path, starting offset bytes
// into the stream.
func (c *Client) ReadStreamOffset(ctx context.Context, path string, offset int) (io.ReadCloser, error) {
	rs, err := c.req(ctx, "GET", path, nil, func(req *http.Request) {
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}
	})
	if err != nil {
		return nil, NewPathErrorErr("ReadStream", path, err)
	}

	if rs.StatusCode == 200 {
		return rs.Body, nil
	}

	rs.Body.Close()
	return nil, NewPathError("ReadStream", path, rs.StatusCode)
}

// ReadStreamRange reads the stream representing a subset of bytes for a given path,
// utilizing HTTP Range Requests if the server supports it.
// The range is expressed as offset from the start of the file and length, for example
// offset=10, length=10 will return bytes 10 through 19.
//
// If the server does not support partial content requests and returns full content instead,
// this function will emulate the behavior by skipping `offset` bytes and limiting the result
// to `length`.
func (c *Client) ReadStreamRange(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error) {
	rs, err := c.req(ctx, "GET", path, nil, func(r *http.Request) {
		if length > 0 {
			r.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
		} else {
			r.Header.Add("Range", fmt.Sprintf("bytes=%d-", offset))
		}
	})
	if err != nil {
		return nil, NewPathErrorErr("ReadStreamRange", path, err)
	}

	if rs.StatusCode == http.StatusPartialContent {
		// server supported partial content, return as-is.
		return rs.Body, nil
	}

	// server returned success, but did not support partial content, so we have the whole
	// stream in rs.Body
	if rs.StatusCode == 200 {
		// discard first 'offset' bytes.
		if _, err := io.Copy(io.Discard, io.LimitReader(rs.Body, offset)); err != nil {
			return nil, NewPathErrorErr("ReadStreamRange", path, err)
		}

		// return a io.ReadCloser that is limited to `length` bytes.
		return &limitedReadCloser{rc: rs.Body, remaining: int(length)}, nil
	}

	rs.Body.Close()
	return nil, NewPathError("ReadStream", path, rs.StatusCode)
}

// Write writes data to a given path
func (c *Client) Write(ctx context.Context, path string, data []byte, _ os.FileMode) (err error) {
	s, err := c.put(ctx, path, bytes.NewReader(data))
	if err != nil {
		return
	}

	switch s {

	case 200, 201, 204:
		return nil

	case 404, 409:
		err = c.createParentCollection(ctx, path)
		if err != nil {
			return
		}

		s, err = c.put(ctx, path, bytes.NewReader(data))
		if err != nil {
			return
		}
		if s == 200 || s == 201 || s == 204 {
			return
		}
	}

	return NewPathError("Write", path, s)
}

// WriteStream writes a stream
func (c *Client) WriteStream(ctx context.Context, path string, stream io.Reader, _ os.FileMode) (err error) {

	err = c.createParentCollection(ctx, path)
	if err != nil {
		return err
	}

	s, err := c.put(ctx, path, stream)
	if err != nil {
		return err
	}

	switch s {
	case 200, 201, 204:
		return nil

	default:
		return NewPathError("WriteStream", path, s)
	}
}
