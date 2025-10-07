package main_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E(t *testing.T) {

	assert := assert.New(t)
	require := require.New(t)

	os.MkdirAll(".test", os.ModePerm)

	// encrypt with offkey

	secret := fmt.Sprintf("this is my very secret message from %d", time.Now().Unix())

	log.Printf("encrypting secret %q", secret)

	cmd := exec.Command(
		"sh",
		"-c",
		fmt.Sprintf("echo %s | go run offkey.go -o=false", secret),
	)

	url := make(chan string)

	stdout, err := cmd.StdoutPipe()
	require.NoError(err)

	go func(r io.ReadCloser, url chan string) {

		var (
			buf     = make([]byte, 1024)
			matcher = regexp.MustCompile(`(?m)Open "(.+)"`)
		)

		n, err := r.Read(buf)
		require.NoError((err))

		matches := matcher.FindStringSubmatch(string(buf[:n]))
		assert.Len(matches, 2)

		url <- matches[1]

	}(stdout, url)

	go func() {
		require.NoError(cmd.Run())
	}()

	// navigate to browser

	target := <-url

	pw, err := playwright.Run()
	require.NoError(err)

	defer pw.Stop()

	browser, err := pw.Chromium.Launch()
	require.NoError(err)

	defer browser.Close()

	page, err := browser.NewPage()
	require.NoError(err)

	_, err = page.Goto(target)
	require.NoError(err)

	// extract passphrase and search for the hint

	var passphrase string

	pp, err := page.QuerySelector("#passphrase")
	assert.NoError(err)
	assert.True(pp.IsVisible())

	text, err := pp.InnerText()
	assert.NoError(err)
	assert.Regexp("(?:[a-z]+-){9}[a-z]+", text)

	passphrase = text

	log.Printf("using passphrase %q", passphrase)

	pp, err = page.QuerySelector("#passphrase-hint")
	assert.NoError(err)
	assert.True(pp.IsHidden())

	code, err := page.Screenshot()
	require.NoError(err)

	err = ioutil.WriteFile(".test/secret-with-passphrase.png", code, os.ModePerm)
	require.NoError(err)

	// print the document, search for the passphrase and extract the hint

	page.EmulateMedia(playwright.PageEmulateMediaOptions{
		Media: playwright.MediaPrint,
	})

	pp, err = page.QuerySelector("#passphrase")
	assert.NoError(err)
	assert.True(pp.IsHidden())

	pp, err = page.QuerySelector("#passphrase-hint")
	assert.NoError(err)
	assert.True(pp.IsVisible())

	// extract qr code

	code, err = page.Screenshot()
	require.NoError(err)

	err = ioutil.WriteFile(".test/secret-for-printing.png", code, os.ModePerm)
	require.NoError(err)

	err = exec.Command(
		"sh",
		"-c",
		"zbarimg --raw -q .test/secret-for-printing.png > .test/secret.age",
	).Run()

	require.NoError(err)

	// decrypt with age

	out, err := exec.Command(
		"sh",
		"-c",
		fmt.Sprintf(`expect -c 'spawn age --decrypt .test/secret.age; expect -- "Enter passphrase:*"; send -- "%s\n"; expect eof'`, passphrase),
	).CombinedOutput()
	assert.NoError(err)

	var (
		buf  = bytes.NewBuffer(out)
		last string
	)

	for {

		line, err := buf.ReadString('\n')

		if err != nil {

			if err == io.EOF {
				break
			} else {
				require.NoError(err)
			}

		}

		last = line
		last = strings.TrimSuffix(last, "\n")
		last = strings.TrimSuffix(last, "\r")

	}

	// Strip ANSI escape sequences (age emits cursor control codes to clean up prompts)
	ansiEscape := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	last = ansiEscape.ReplaceAllString(last, "")

	assert.Equal(secret, last)

	log.Printf("decrypted secret %q", last)

}
