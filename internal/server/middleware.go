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

		sess, err := s.db.GetSession(sessionCookie.Value)
		if err != nil {
			return echo.NewHTTPError(401, "Invalid session")
		}

		if sess.UserAgent != c.Request().Header.Get("X-User-Agent") {
			return echo.NewHTTPError(401, "Invalid session")
		}

		userId := c.Request().Header.Get("X-User-ID")

		if userId != "" && userId != sess.UserId {
			return echo.NewHTTPError(401, "Invalid session")
		}

		return next(c)
	}
}
