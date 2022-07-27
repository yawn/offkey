package server

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"net"
	"net/http"

	"github.com/aaronarduino/goqrsvg"
	svg "github.com/ajstarks/svgo"
	"github.com/boombuler/barcode"
	"github.com/pkg/errors"
	"github.com/yawn/offkey/crypto"
)

//go:embed index.html
var indexHtml string

// Server is a http server on the local loopback device that shows it's secret once
type Server struct {
	cancel      func()
	description string
	listener    net.Listener
	passphrase  string
	port        int
	secret      barcode.Barcode
	server      *http.Server
	token       string
	tpl         *template.Template
}

// New initializes the server over a given secret and optional description
func New(secret []byte, description string) (*Server, error) {

	var (
		passphrase = crypto.Passphrase()
		token      = crypto.Token()
	)

	sec, err := crypto.Encrypt(passphrase, secret)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to age encrypt secret")
	}

	tpl, err := template.New("index").Parse(indexHtml)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse index.html")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")

	if err != nil {
		return nil, errors.Wrapf(err, "failed to create listener")
	}

	port := listener.Addr().(*net.TCPAddr).Port

	return &Server{
		description: description,
		listener:    listener,
		passphrase:  passphrase,
		port:        port,
		secret:      sec,
		token:       token,
		tpl:         tpl,
	}, nil

}

// Close closes the server
func (s *Server) Close() {
	s.cancel()
}

// Serve starts serving the secret - it blocks until the context is cancelled
func (s *Server) Serve(ctx context.Context) error {

	ctx, cancel := context.WithCancel(ctx)

	mux := http.NewServeMux()

	mux.HandleFunc("/favicon.ico", s.ignore)
	mux.HandleFunc("/", s.show)

	srv := &http.Server{
		Handler: mux,
	}

	s.cancel = cancel

	go func() {
		<-ctx.Done()
		srv.Shutdown(ctx)
	}()

	if err := srv.Serve(s.listener); err != http.ErrServerClosed {
		return err
	}

	return nil

}

// URL return the url under which the secret will be available
func (s *Server) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d?token=%s", s.port, s.token)
}

// ignore is a generic 404 handler
func (s *Server) ignore(res http.ResponseWriter, req *http.Request) {
	http.NotFound(res, req)
}

// show is the handler that shows the secret once - provided the correct token is passed
func (s *Server) show(res http.ResponseWriter, req *http.Request) {

	// never store the secret in any caches
	res.Header().Set("Cache-Control", "no-store")

	token := req.URL.Query().Get("token")

	if token == "" || token != s.token {
		res.WriteHeader(http.StatusForbidden)
	} else {

		// close the server after this
		defer s.Close()

		buf := bytes.NewBuffer(nil)
		enc := svg.New(buf)

		qs := goqrsvg.NewQrSVG(s.secret, 5)
		qs.StartQrSVG(enc)
		qs.WriteQrSVG(enc)

		s.tpl.Execute(res, struct {
			Code        template.HTML
			Description string
			Passphrase  string
		}{
			Code:        template.HTML(buf.String()),
			Description: s.description,
			Passphrase:  s.passphrase,
		})

	}

}
