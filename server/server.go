package server

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/aaronarduino/goqrsvg"
	svg "github.com/ajstarks/svgo"
	"github.com/boombuler/barcode"
	"github.com/pkg/errors"
	"github.com/yawn/offkey/crypto"
	"github.com/yawn/offkey/log"
)

var Log *log.Log

//go:embed index.html
var indexHtml string

type Server struct {
	closer     io.Closer
	listener   net.Listener
	passphrase string
	port       int
	secret     barcode.Barcode
	shown      bool
	token      string
	tpl        *template.Template
}

func New(secret []byte) (*Server, error) {

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
		listener:   listener,
		passphrase: passphrase,
		port:       port,
		secret:     sec,
		token:      token,
		tpl:        tpl,
	}, nil

}

func (s *Server) Close() error {

	Log.Msg("[3/3] Done, long-term password removed for security reasons")

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

	Log.Msg("[1/3] Navigate to http://127.0.0.1:%d/?token=%s", s.port, s.token)

	if err := srv.Serve(s.listener); err != http.ErrServerClosed {
		return err
	}

	return nil

}

func (s *Server) ignore(res http.ResponseWriter, req *http.Request) {
	http.NotFound(res, req)
}

func (s *Server) show(res http.ResponseWriter, req *http.Request) {

	if s.shown {
		return
	}

	s.shown = true

	token := req.URL.Query().Get("token")

	if token == "" {
		Log.Msg("[ERR] No token supplied in request - some other process is tying to access your offkey?")
		os.Exit(1)
	} else if token != s.token {
		Log.Msg("[ERR] Invalid token supplied - some other process is trying to access your offkey?")
		os.Exit(2)
	}

	Log.Password("[2/3] Print the document, write down following passphrase with a permanent pen and send ctrl-c to continue: %s", s.passphrase)

	buf := bytes.NewBuffer(nil)
	enc := svg.New(buf)

	qs := goqrsvg.NewQrSVG(s.secret, 5)
	qs.StartQrSVG(enc)
	qs.WriteQrSVG(enc)

	s.tpl.Execute(res, struct {
		Code template.HTML
	}{
		Code: template.HTML(buf.String()),
	})

}
