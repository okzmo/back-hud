package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type invitationBody struct {
	InvitationId string `json:"invitation_id"`
}

func (s *Server) HandlerCheckInvitationValidity(c echo.Context) error {
	resp := make(map[string]any)

	InvitationId := fmt.Sprintf("invitations:%s", c.Param("invitationId"))
	fmt.Println(InvitationId)

	invite, err := s.db.CheckInvitationValidity(InvitationId)
	if err != nil {
		resp["message"] = "This invitation is invalid or has expired."
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["invite"] = invite

	return c.JSON(http.StatusOK, resp)
}
