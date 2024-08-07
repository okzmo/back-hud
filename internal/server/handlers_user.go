package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"goback/internal/utils"
	"goback/proto/protoMess"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/h2non/bimg"
	"github.com/labstack/echo/v4"
	"github.com/lxzan/gws"
	"google.golang.org/protobuf/proto"
)

type ChangeInformations struct {
	UserId        string  `json:"user_id"`
	Username      *string `json:"username,omitempty"`
	UsernameColor *string `json:"username_color,omitempty"`
	Email         *string `json:"email,omitempty"`
	DisplayName   *string `json:"display_name,omitempty"`
	Status        *string `json:"status,omitempty"`
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

func (s *Server) HandlerChangeNameColor(c echo.Context) error {
	resp := make(map[string]any)

	body := new(ChangeInformations)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["message"] = "An error occured when parsing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.ChangeNameColor(body.UserId, *body.UsernameColor)
	if err != nil {
		log.Println(err)
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
		cmd := exec.Command("gifsicle",
			"--crop", fmt.Sprintf("%d,%d+%dx%d", cropX, cropY, cropWidth, cropHeight),
			"--lossy=90",
			"--output", "-",
			"--", "-",
		)

		cmd.Stdin = bytes.NewReader(imageBuffer)
		var outputBuf bytes.Buffer
		cmd.Stdout = &outputBuf

		err = cmd.Run()
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to crop GIF with gifsicle")
		}

		imageToUpload = outputBuf.Bytes()
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

	go func() {
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
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err = s.s3.PutObject(&s3.PutObjectInput{
			Bucket: aws.String("Hudori"),
			Key:    aws.String(bannerKey),
			Body:   bytes.NewReader(imageToUpload),
		})
		if err != nil {
			log.Println(err)
		}
	}()
	wg.Wait()

	banner, err := s.db.UpdateBanner(userId, bannerKey)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Failed to update link")
	}

	resp["banner"] = banner

	resp["message"] = "success"
	return c.JSON(http.StatusOK, resp)
}

type ChangeAvatar struct {
	UserId string `json:"user_id"`
	Avatar string `json:"avatar"`
}

func (s *Server) HandlerChangeAvatar(c echo.Context) error {
	resp := make(map[string]any)

	body := new(ChangeInformations)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["message"] = "An error occured when changing your avatar."
		return c.JSON(http.StatusBadRequest, resp)
	}

	userId := strings.Split(c.Request().Header.Get("X-User-ID"), ":")[1]

	cropX, _ := strconv.Atoi(c.FormValue("cropX"))
	cropY, _ := strconv.Atoi(c.FormValue("cropY"))
	cropWidth, _ := strconv.Atoi(c.FormValue("cropWidth"))
	cropHeight, _ := strconv.Atoi(c.FormValue("cropHeight"))
	oldAvatarName := c.FormValue("old_avatar")
	serverId := c.FormValue("server_id")
	friendsStr := c.FormValue("friends")

	var friends []string
	err := json.Unmarshal([]byte(friendsStr), &friends)
	if err != nil {
		log.Println(err)
	}
	log.Println(friends)
	log.Println(serverId)

	file, err := c.FormFile("avatar")
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
	var avatarKey string
	randId, _ := utils.GenerateRandomId(6)
	if mimeType == "image/gif" {
		cmd := exec.Command("gifsicle",
			"--crop", fmt.Sprintf("%d,%d+%dx%d", cropX, cropY, cropWidth, cropHeight),
			"--lossy=90",
			"--output", "-",
			"--", "-",
		)

		cmd.Stdin = bytes.NewReader(imageBuffer)
		var outputBuf bytes.Buffer
		cmd.Stdout = &outputBuf

		err = cmd.Run()
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to crop GIF with gifsicle")
		}

		imageToUpload = outputBuf.Bytes()
		avatarKey = userId + "-avatar-" + randId + ".gif"
	} else {
		croppedImage, err := bimg.NewImage(imageBuffer).Extract(cropY, cropX, cropWidth, cropHeight)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to crop image")
		}

		imageToUpload, err = bimg.NewImage(croppedImage).Convert(bimg.JPEG)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to convert image to jpg")
		}
		avatarKey = userId + "-avatar-" + randId + ".jpg"
	}

	go func() {
		res, err := s.s3.GetObject(&s3.GetObjectInput{
			Bucket: aws.String("Hudori"),
			Key:    aws.String(oldAvatarName),
		})
		if err != nil {
			log.Println(err)
		}

		_, err = s.s3.DeleteObject(&s3.DeleteObjectInput{
			Bucket:    aws.String("Hudori"),
			Key:       aws.String(oldAvatarName),
			VersionId: res.VersionId,
		})
		if err != nil {
			log.Println(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err = s.s3.PutObject(&s3.PutObjectInput{
			Bucket: aws.String("Hudori"),
			Key:    aws.String(avatarKey),
			Body:   bytes.NewReader(imageToUpload),
		})
		if err != nil {
			log.Println(err)
		}
	}()
	wg.Wait()

	avatar, err := s.db.UpdateAvatar(userId, avatarKey)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Failed to update link")
	}

	resp["avatar"] = avatar

	wsMess := &protoMess.WSMessage{
		Type: "new_avatar",
		Content: &protoMess.WSMessage_ChangeAvatar{
			ChangeAvatar: &protoMess.ChangeAvatar{
				UserId: userId,
				Avatar: avatar,
			},
		},
	}
	data, err := proto.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}

	compMess := utils.CompressMess(data)
	if serverId != "" {
		Pub(globalEmitter, serverId, gws.OpcodeBinary, compMess)
	}

	if len(friends) > 0 {
		for _, friend := range friends {
			if connFriend, ok := s.ws.sessions.Load(strings.Split(friend, ":")[1]); ok {
				connFriend.WriteMessage(gws.OpcodeBinary, compMess)
			}
		}
	}

	resp["message"] = "success"
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerChangeStatus(c echo.Context) error {
	body := new(ChangeInformations)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		return c.String(400, "")
	}

	err := s.db.UpdateUserStatus(strings.Split(body.UserId, ":")[1], *body.Status)
	if err != nil {
		log.Println(err)
		return c.String(400, "")
	}

	servers, err := s.db.GetUserServers(body.UserId)
	if err != nil {
		log.Println(err)
	}

	friends, err := s.db.GetFriends(body.UserId)
	if err != nil {
		log.Println(err)
	}

	wsMess := &protoMess.WSMessage{
		Type: "change_status",
		Content: &protoMess.WSMessage_ChangeStatus{
			ChangeStatus: &protoMess.ChangeStatus{
				UserId: body.UserId,
				Status: *body.Status,
			},
		},
	}

	data, err := proto.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}

	compMess := utils.CompressMess(data)

	for _, s := range servers {
		Pub(globalEmitter, s.ID, gws.OpcodeBinary, compMess)
	}

	for _, f := range friends {
		if connFriend, ok := s.ws.sessions.Load(strings.Split(f.ID, ":")[1]); ok {
			connFriend.WriteMessage(gws.OpcodeBinary, compMess)
		}
	}

	return c.String(200, "")
}
