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

func (s *Server) Close() {
	s.cancel()
}

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

func (s *Server) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d?token=%s", s.port, s.token)
}

func (s *Server) ignore(res http.ResponseWriter, req *http.Request) {
	http.NotFound(res, req)
}

func (s *Server) show(res http.ResponseWriter, req *http.Request) {

	res.Header().Set("Cache-Control", "no-store")

	token := req.URL.Query().Get("token")

	if token == "" || token != s.token {
		res.WriteHeader(http.StatusForbidden)
	} else {

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
