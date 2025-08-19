package app

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo"
	"github.com/topfreegames/mqtt-history/logger"
	"github.com/topfreegames/mqtt-history/mongoclient"
)

// BattletanksCSVHandler is the handler responsible for returning battletanks messages as CSV from August 12th
func BattletanksCSVHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		c.Set("route", "BattletanksCSV")

		logger.Logger.Debug("Request for Battletanks CSV export from August 12th (including blocked and non-blocked messages)")

		// Define the start date as August 12, current year in UTC
		currentYear := time.Now().Year()
		august12, err := time.ParseInLocation("2006-01-02", fmt.Sprintf("%d-08-12", currentYear), time.UTC)
		if err != nil {
			logger.Logger.Errorf("Error parsing date: %s", err.Error())
			return c.JSON(http.StatusInternalServerError, "Error parsing date")
		}
		fromTimestamp := august12.Unix()

		// Validate that app.Defaults is initialized
		if app.Defaults == nil {
			logger.Logger.Error("App defaults not initialized")
			return c.JSON(http.StatusInternalServerError, "Server configuration error")
		}

		// Query parameters for MongoDB
		collection := app.Defaults.MongoMessagesCollection
		if collection == "" {
			logger.Logger.Error("MongoDB collection not configured")
			return c.JSON(http.StatusInternalServerError, "Server configuration error")
		}

		queryParams := mongoclient.QueryParameters{
			Collection: collection,
			GameID:     "battletanks",
			From:       fromTimestamp,
			Limit:      0, // No limit, get all messages
		}

		// Get messages from MongoDB
		messages := mongoclient.GetMessagesByGameIDWithDateFilter(c, queryParams)

		if len(messages) == 0 {
			logger.Logger.Debug("No messages found for battletanks from August 12th")
			return c.JSON(http.StatusNotFound, "No messages found")
		}

		logger.Logger.Debugf("Found %d messages for battletanks from August 12th (including blocked and non-blocked)", len(messages))

		// Set response headers for CSV download
		filename := fmt.Sprintf("battletanks_messages_%s.csv", time.Now().Format("20060102_150405"))
		c.Response().Header().Set("Content-Type", "text/csv; charset=utf-8")
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		// Create CSV writer
		writer := csv.NewWriter(c.Response())
		defer writer.Flush()

		// Write CSV header
		header := []string{
			"id",
			"timestamp",
			"date",
			"topic",
			"player_id",
			"message",
			"game_id",
			"blocked",
			"should_moderate",
		}
		if err := writer.Write(header); err != nil {
			logger.Logger.Errorf("Error writing CSV header: %s", err.Error())
			return c.String(http.StatusInternalServerError, "Error generating CSV header")
		}

		// Write CSV data
		for _, msg := range messages {
			// Skip nil messages
			if msg == nil {
				continue
			}

			// Convert timestamp to readable date in UTC
			messageTime := time.Unix(msg.Timestamp, 0).UTC()
			dateStr := messageTime.Format("2006-01-02 15:04:05")

			// Safely extract fields, handling potential empty values
			id := ""
			if msg.Id != "" {
				id = msg.Id
			}

			playerId := ""
			if msg.PlayerId != "" {
				playerId = msg.PlayerId
			}

			message := ""
			if msg.Message != "" {
				message = msg.Message
			}

			topic := ""
			if msg.Topic != "" {
				topic = msg.Topic
			}

			gameId := ""
			if msg.GameId != "" {
				gameId = msg.GameId
			}

			record := []string{
				id,
				strconv.FormatInt(msg.Timestamp, 10),
				dateStr,
				topic,
				playerId,
				message,
				gameId,
				strconv.FormatBool(msg.Blocked),
				strconv.FormatBool(msg.ShouldModerate),
			}

			if err := writer.Write(record); err != nil {
				logger.Logger.Errorf("Error writing CSV record: %s", err.Error())
				return c.String(http.StatusInternalServerError, "Error generating CSV data")
			}
		}

		return nil
	}
}
