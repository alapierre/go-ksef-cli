package bmp

import (
	"bytes"

	gobmp "golang.org/x/image/bmp"

	"github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
)

var logger = logrus.WithField("component", "bmp")

func Qr(content string) ([]byte, error) {

	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		logger.Fatal(err)
	}

	img := qr.Image(300)

	var buf bytes.Buffer

	if err := gobmp.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
