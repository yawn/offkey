package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/browser"
	"github.com/yawn/offkey/server"
)

var (
	fDescription string
	fOpen        bool
)

func main() {

	flag.BoolVar(&fOpen, "o", true, "try to open URL in browser automatically")
	flag.StringVar(&fDescription, "d", "", "a description of your secret")

	flag.Parse()

	secret, err := ioutil.ReadAll(os.Stdin)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s, err := server.New(secret, fDescription)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if fOpen {

		if err := browser.OpenURL(s.URL()); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	} else {
		fmt.Printf("Open %q in your browser\n", s.URL())
	}

	time.AfterFunc(5*time.Minute, func() {
		s.Close()
	})

	s.Serve()

}
