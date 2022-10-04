package app

import (
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
)

func ParseHistoryQueryParams(c echo.Context, defaultLimit int64) (string, int64, int64, bool) {
	userID := c.QueryParam("userid")
	from, _ := strconv.ParseInt(c.QueryParam("from"), 10, 64)
	limit, _ := strconv.ParseInt(c.QueryParam("limit"), 10, 64)
	isBlocked, _ := strconv.ParseBool(c.QueryParam("isBlocked"))

	if limit == 0 {
		limit = defaultLimit
	}

	if from == 0 {
		from = time.Now().Unix()
	}

	return userID, from, limit, isBlocked
}

func ParseHistoriesQueryParams(c echo.Context, defaultLimit int64) ([]string, string, int64, int64) {
	userID := c.QueryParam("userid")
	topicsSuffix := strings.Split(c.QueryParam("topics"), ",")
	from, _ := strconv.ParseInt(c.QueryParam("from"), 10, 64)
	limit, _ := strconv.ParseInt(c.QueryParam("limit"), 10, 64)

	if limit == 0 {
		limit = defaultLimit
	}

	if from == 0 {
		from = time.Now().Unix()
	}

	return topicsSuffix, userID, from, limit
}

func ParseHistoryPSQueryParams(c echo.Context, defaultLimit int64) (string, string, string, int64, bool) {
	userID := c.QueryParam("userid")
	playerId := c.QueryParam("playerId")
	topic := c.QueryParam("topic")
	limit, _ := strconv.ParseInt(c.QueryParam("limit"), 10, 64)
	if limit == 0 {
		limit = defaultLimit
	}
	isBlocked, _ := strconv.ParseBool(c.QueryParam("isBlocked"))
	return userID, playerId, topic, limit, isBlocked
}
