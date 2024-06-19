package server

import (
	"bytes"
	"goback/internal/utils"
	"image"
	"image/draw"
	"image/gif"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/h2non/bimg"
	"github.com/labstack/echo/v4"
)

type ChangeInformations struct {
	UserId      string  `json:"user_id"`
	Username    *string `json:"username,omitempty"`
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
}

func (s *Server) HandlerChangeEmail(c echo.Context) error {
	resp := make(map[string]any)

	body := new(ChangeInformations)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "email"
		resp["message"] = "An error occured when changing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.ChangeEmail(body.UserId, *body.Email)
	if err != nil {
		log.Println(err)
		resp["name"] = "email"
		resp["message"] = "An error occured when changing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["message"] = "success"

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerGetUser(c echo.Context) error {
	resp := make(map[string]any)

	userId := c.Param("userId")

	user, err := s.db.GetUser(userId, "", "")
	if err != nil {
		log.Println(err)
		resp["message"] = "An error occured when fetching the user."
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["user"] = user

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerChangeUsername(c echo.Context) error {
	resp := make(map[string]any)

	body := new(ChangeInformations)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "username"
		resp["message"] = "An error occured when changing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	_, err := s.db.GetUser("", *body.Username, "")
	if err == nil {
		resp["name"] = "username"
		resp["message"] = "This username is already in use."
		return c.JSON(http.StatusBadRequest, resp)
	}

	err = s.db.ChangeUsername(body.UserId, *body.Username)
	if err != nil {
		log.Println(err)
		resp["name"] = "username"
		resp["message"] = "An error occured when changing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["message"] = "success"

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerChangeDisplayName(c echo.Context) error {
	resp := make(map[string]any)

	body := new(ChangeInformations)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "display_name"
		resp["message"] = "An error occured when changing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.ChangeDisplayName(body.UserId, *body.DisplayName)
	if err != nil {
		log.Println(err)
		resp["name"] = "display_name"
		resp["message"] = "An error occured when changing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["message"] = "success"

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerChangeBanner(c echo.Context) error {
	resp := make(map[string]any)

	body := new(ChangeInformations)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["message"] = "An error occured when changing your banner."
		return c.JSON(http.StatusBadRequest, resp)
	}

	userId := strings.Split(c.Request().Header.Get("X-User-ID"), ":")[1]

	cropX, _ := strconv.Atoi(c.FormValue("cropX"))
	cropY, _ := strconv.Atoi(c.FormValue("cropY"))
	cropWidth, _ := strconv.Atoi(c.FormValue("cropWidth"))
	cropHeight, _ := strconv.Atoi(c.FormValue("cropHeight"))
	oldBannerName := c.FormValue("old_banner")

	file, err := c.FormFile("banner")
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Failed to get file")
	}

	if file.Size > 8*1024*1024 {
		resp["message"] = "File size exceeds 8MB limit"
		return c.JSON(http.StatusBadRequest, resp)
	}

	src, err := file.Open()
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Failed to open file")
	}
	defer src.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, src)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to read image")
	}

	imageBuffer := buf.Bytes()
	mimeType := http.DetectContentType(imageBuffer)

	var imageToUpload []byte
	var bannerKey string
	randId, _ := utils.GenerateRandomId(6)
	if mimeType == "image/gif" {

		gifImage, err := gif.DecodeAll(bytes.NewReader(imageBuffer))
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to decode GIF")
		}

		croppedGif := &gif.GIF{
			Image:     make([]*image.Paletted, len(gifImage.Image)),
			Delay:     gifImage.Delay,
			Disposal:  gifImage.Disposal,
			LoopCount: gifImage.LoopCount,
		}

		var wg sync.WaitGroup
		var mu sync.Mutex

		for i, frame := range gifImage.Image {
			wg.Add(1)
			go func(i int, frame *image.Paletted) {
				defer wg.Done()

				var frameBuf bytes.Buffer
				err := gif.Encode(&frameBuf, frame, nil)
				if err != nil {
					log.Println("Failed to encode frame:", err)
					return
				}

				croppedFrame, err := bimg.NewImage(frameBuf.Bytes()).Extract(cropY, cropX, cropWidth, cropHeight)
				if err != nil {
					log.Println("Failed to crop frame", err)
					return
				}

				croppedFrameImg, _, err := image.Decode(bytes.NewReader(croppedFrame))
				if err != nil {
					log.Println("Failed to decode cropped frame", err)
					return
				}

				palettedFrame := image.NewPaletted(croppedFrameImg.Bounds(), frame.Palette)
				draw.Draw(palettedFrame, palettedFrame.Rect, croppedFrameImg, image.Point{}, draw.Over)

				mu.Lock()
				croppedGif.Image[i] = palettedFrame
				mu.Unlock()
			}(i, frame)
		}
		wg.Wait()

		var croppedBuf bytes.Buffer
		err = gif.EncodeAll(&croppedBuf, croppedGif)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to encode cropped GIF")
		}

		imageToUpload = croppedBuf.Bytes()
		bannerKey = userId + "-banner-" + randId + ".gif"
	} else {
		croppedImage, err := bimg.NewImage(imageBuffer).Extract(cropY, cropX, cropWidth, cropHeight)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to crop image")
		}

		imageToUpload, err = bimg.NewImage(croppedImage).Convert(bimg.JPEG)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to convert image to jpg")
		}
		bannerKey = userId + "-banner-" + randId + ".jpg"
	}

	res, err := s.s3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String("Hudori"),
		Key:    aws.String(oldBannerName),
	})
	if err != nil {
		log.Println(err)
	}

	_, err = s.s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket:    aws.String("Hudori"),
		Key:       aws.String(oldBannerName),
		VersionId: res.VersionId,
	})
	if err != nil {
		log.Println(err)
	}

	_, err = s.s3.PutObject(&s3.PutObjectInput{
		Bucket: aws.String("Hudori"),
		Key:    aws.String(bannerKey),
		Body:   bytes.NewReader(imageToUpload),
	})
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Failed to upload file")
	}

	banner, err := s.db.UpdateBanner(userId, bannerKey)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Failed to update link")
	}

	resp["banner"] = banner

	return c.JSON(http.StatusOK, resp)
}
