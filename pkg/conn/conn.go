// Package conn contains a RTSP connection implementation.
package conn

import (
	"bufio"
	"io"

	"github.com/aler9/gortsplib/v2/pkg/base"
)

const (
	readBufferSize = 4096
)

type RWLogger struct {
	rw      io.ReadWriter
	rlogger func([]byte, int, error)
	wlogger func([]byte, int, error)
}

func (cw *RWLogger) Read(p []byte) (int, error) {
	n, err := cw.rw.Read(p)
	if n > 0 && cw.rlogger != nil {
		cw.rlogger(p, n, err)
	}
	return n, err
}

func (cw *RWLogger) Write(p []byte) (int, error) {
	n, err := cw.rw.Write(p)
	if n > 0 && cw.wlogger != nil {
		cw.wlogger(p, n, err)
	}
	return n, err
}

// Conn is a RTSP connection.
type Conn struct {
	w   io.Writer
	br  *bufio.Reader
	req base.Request
	res base.Response
	fr  base.InterleavedFrame
}

// NewConn allocates a Conn.
func NewConn(rw io.ReadWriter) *Conn {
	return &Conn{
		w:  rw,
		br: bufio.NewReaderSize(rw, readBufferSize),
	}
}

func NewConnWithLogger(rw io.ReadWriter, rlogger func([]byte, int, error), wlogger func([]byte, int, error)) *Conn {
	rwl := &RWLogger{
		rw:      rw,
		rlogger: rlogger,
		wlogger: wlogger,
	}
	return &Conn{
		w:  rwl,
		br: bufio.NewReaderSize(rwl, readBufferSize),
	}
}

// ReadRequest reads a Request.
func (c *Conn) ReadRequest() (*base.Request, error) {
	err := c.req.Read(c.br)
	return &c.req, err
}

// ReadResponse reads a Response.
func (c *Conn) ReadResponse() (*base.Response, error) {
	err := c.res.Read(c.br)
	return &c.res, err
}

// ReadInterleavedFrame reads a InterleavedFrame.
func (c *Conn) ReadInterleavedFrame() (*base.InterleavedFrame, error) {
	err := c.fr.Read(c.br)
	return &c.fr, err
}

// ReadInterleavedFrameOrRequest reads an InterleavedFrame or a Request.
func (c *Conn) ReadInterleavedFrameOrRequest() (interface{}, error) {
	b, err := c.br.ReadByte()
	if err != nil {
		return nil, err
	}
	c.br.UnreadByte()

	if b == base.InterleavedFrameMagicByte {
		return c.ReadInterleavedFrame()
	}

	return c.ReadRequest()
}

// ReadInterleavedFrameOrResponse reads an InterleavedFrame or a Response.
func (c *Conn) ReadInterleavedFrameOrResponse() (interface{}, error) {
	b, err := c.br.ReadByte()
	if err != nil {
		return nil, err
	}
	c.br.UnreadByte()

	if b == base.InterleavedFrameMagicByte {
		return c.ReadInterleavedFrame()
	}

	return c.ReadResponse()
}

// ReadRequestIgnoreFrames reads a Request and ignores frames in between.
func (c *Conn) ReadRequestIgnoreFrames() (*base.Request, error) {
	for {
		recv, err := c.ReadInterleavedFrameOrRequest()
		if err != nil {
			return nil, err
		}

		if req, ok := recv.(*base.Request); ok {
			return req, nil
		}
	}
}

// ReadResponseIgnoreFrames reads a Response and ignores frames in between.
func (c *Conn) ReadResponseIgnoreFrames() (*base.Response, error) {
	for {
		recv, err := c.ReadInterleavedFrameOrResponse()
		if err != nil {
			return nil, err
		}

		if res, ok := recv.(*base.Response); ok {
			return res, nil
		}
	}
}

// WriteRequest writes a request.
func (c *Conn) WriteRequest(req *base.Request) error {
	buf, _ := req.Marshal()
	_, err := c.w.Write(buf)
	return err
}

// WriteResponse writes a response.
func (c *Conn) WriteResponse(res *base.Response) error {
	buf, _ := res.Marshal()
	_, err := c.w.Write(buf)
	return err
}

// WriteInterleavedFrame writes an interleaved frame.
func (c *Conn) WriteInterleavedFrame(fr *base.InterleavedFrame, buf []byte) error {
	n, _ := fr.MarshalTo(buf)
	_, err := c.w.Write(buf[:n])
	return err
}
