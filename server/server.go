package server

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
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
	description string
	closer      io.Closer
	listener    net.Listener
	passphrase  string
	port        int
	secret      barcode.Barcode
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

func (s *Server) Close() error {

	if s.closer == nil {
		return fmt.Errorf("server not yet started")
	} else {
		return s.closer.Close()
	}

}

func (s *Server) Serve() error {

	mux := http.NewServeMux()

	mux.HandleFunc("/favicon.ico", s.ignore)
	mux.HandleFunc("/", s.show)

	srv := &http.Server{
		Handler: mux,
	}

	s.closer = srv

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
