package utils

import (
	"bytes"
	"crypto/rand"
	"image"
	"image/draw"
	"image/gif"
	"math/big"
	"net/mail"

	"github.com/h2non/bimg"
)

func EmailValid(email string) bool {
	emailAddress, err := mail.ParseAddress(email)
	return err == nil && emailAddress.Address == email
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRandomId(length ...int) (string, error) {
	idLength := 8
	if len(length) > 0 {
		idLength = length[0]
	}

	id := make([]byte, idLength)
	for i := range id {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}

		id[i] = charset[num.Int64()]
	}

	return string(id), nil
}

func CropImageGIF(i int, frame *image.Paletted, croppedGif *gif.GIF, cropX, cropY, cropWidth, cropHeight int) error {
	var frameBuf bytes.Buffer
	err := gif.Encode(&frameBuf, frame, nil)
	if err != nil {
		return err
	}

	croppedFrame, err := bimg.NewImage(frameBuf.Bytes()).Extract(cropY, cropX, cropWidth, cropHeight)
	if err != nil {
		return err
	}

	croppedFrameImg, _, err := image.Decode(bytes.NewReader(croppedFrame))
	if err != nil {
		return err
	}

	palettedFrame := image.NewPaletted(croppedFrameImg.Bounds(), frame.Palette)
	draw.Draw(palettedFrame, palettedFrame.Rect, croppedFrameImg, image.Point{}, draw.Over)

	croppedGif.Image[i] = palettedFrame

	return nil
}
