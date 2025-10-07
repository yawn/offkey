package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"time"

	"github.com/pkg/browser"
	"github.com/yawn/offkey/server"
)

var Version string

var (
	fDescription string
	fOpen        bool
	fVersion     bool
)

func main() {

	flag.BoolVar(&fOpen, "o", true, "try to open URL in browser automatically")
	flag.BoolVar(&fVersion, "v", false, "show version")
	flag.StringVar(&fDescription, "d", "", "a description of your secret")

	flag.Parse()

	if fVersion {

		buildInfo, ok := debug.ReadBuildInfo()

		if !ok {
			panic("missing build information")
		}

		if Version != "" {
			buildInfo.Main.Version = Version
		}

		var rev string

		for _, setting := range buildInfo.Settings {

			if setting.Key == "vcs.revision" {
				rev = setting.Value
				break
			}

		}

		if rev == "" {
			panic("missing build vcs information")
		}

		fmt.Printf("%s (%s)\n", buildInfo.Main.Version, rev[:7])

		return

	}

	secret, err := io.ReadAll(os.Stdin)

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

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)

	s.Serve(ctx)

}
