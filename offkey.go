package main

import (
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yawn/offkey/log"
	"github.com/yawn/offkey/server"
)

func main() {

	server.Log = log.New()

	secret, err := ioutil.ReadAll(os.Stdin)

	if err != nil {
		server.Log.Err("failed to read secret from stdin: %s", err.Error())
	}

	s, err := server.New(secret)

	if err != nil {
		server.Log.Err("failed to server secret: %s", err.Error())
	}

	time.AfterFunc(5*time.Minute, func() {
		s.Close()
	})

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func(sig chan os.Signal) {

		<-sig
		s.Close()

	}(sig)

	s.Serve()

}
