package automationipc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	config   Config
	listener net.Listener
	handler  Handler
	done     chan struct{}
	once     sync.Once
	wg       sync.WaitGroup
}

func Listen(directory string, binding Binding, handler Handler) (*Server, error) {
	nonce, e := secret()
	if e != nil {
		return nil, e
	}
	listener, e := listen(directory)
	if e != nil {
		return nil, e
	}
	s := &Server{config: Config{Endpoint: listener.Addr().String(), Nonce: nonce, Binding: binding}, listener: listener, handler: handler, done: make(chan struct{})}
	s.wg.Add(1)
	go s.serve()
	return s, nil
}
func (s *Server) Config() Config { return s.config }
func (s *Server) Close() error {
	var e error
	s.once.Do(func() { close(s.done); e = s.listener.Close() })
	s.wg.Wait()
	return e
}
func (s *Server) serve() {
	defer s.wg.Done()
	slots := make(chan struct{}, 4)
	for {
		c, e := s.listener.Accept()
		if e != nil {
			return
		}
		select {
		case slots <- struct{}{}:
			s.wg.Add(1)
			go func() { defer func() { <-slots; s.wg.Done() }(); s.exchange(c) }()
		default:
			_ = c.Close()
		}
	}
}
func (s *Server) exchange(c net.Conn) {
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	var r Request
	if readFrame(c, &r) != nil || !valid(s.config, r) {
		_ = writeResponse(c, Response{Status: "rejected", Error: "invalid check binding"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	go func() {
		select {
		case <-s.done:
			cancel()
		case <-ctx.Done():
		}
	}()
	out, e := s.handler(ctx, r)
	if e != nil {
		out = Response{Status: "rejected", Error: "local check request refused"}
	}
	_ = writeResponse(c, out)
}

var errLocalTransport = errors.New("private local check transport unavailable")

func writeResponse(w io.Writer, out Response) error {
	raw, err := json.Marshal(out)
	if err != nil || len(raw) > MaxResponse {
		return writeFrame(w, Response{Status: "rejected", Error: "bounded local check response unavailable"})
	}
	return writeFrame(w, out)
}
