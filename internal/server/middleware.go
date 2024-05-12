package server

import (
	"github.com/labstack/echo/v4"
)

func (s *Server) SessionAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sessionCookie, err := c.Cookie("session")
		if err != nil {
			return echo.NewHTTPError(401, "No session cookie available")
		}

		_, err = s.db.GetSession(sessionCookie.Value)
		if err != nil {
			return echo.NewHTTPError(401, "Invalid session")
		}

		return next(c)
	}
}
