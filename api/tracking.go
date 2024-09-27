package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/billyb2/tracking_server/auth"
	"github.com/billyb2/tracking_server/db"
	fedex "github.com/billyb2/tracking_server/tracking"
	"github.com/gin-gonic/gin"
)

type trackingNumberGroup struct {
	GroupName       string   `json:"group_name,omitempty"`
	TrackingNumbers []string `json:"tracking_numbers"`
}

type startTracking struct {
	TrackingNumberGroups []trackingNumberGroup `json:"tracking_number_groups"`
	Token                string                `json:"token"`
}

type startTrackingResp struct {
	Error string `json:"error,omitempty"`
}

// Register godoc
//
//	@Summary	Starts tracking the package tracking numbers given by the user
//	@ID			start-tracking-groups
//	@Accept		json
//	@Produce	json
//	@Param		startTrackingInfo	body		startTracking	true	"Tracking Info"
//	@Success	201					{object}	startTrackingResp
//	@Failure	403					{object}	startTrackingResp
//	@Failure	500					{object}	startTrackingResp
//	@Router		/start_tracking [post]
func StartTrackingGroups(c *gin.Context) {
	startTracking := startTracking{}
	if err := c.BindJSON(&startTracking); err != nil {
		err = fmt.Errorf("error parsing StartTracking: %w")
	}

	userID, err := auth.UserIDFromToken(c, startTracking.Token)
	if err != nil {
		err = fmt.Errorf("auth error: %w", err)
		switch {
		case errors.Is(err, auth.InvalidToken):
			c.JSON(http.StatusForbidden, startTrackingResp{
				Error: err.Error(),
			})
			return
		default:
			c.JSON(http.StatusInternalServerError, startTrackingResp{
				Error: err.Error(),
			})
			return
		}
	}

	for _, group := range startTracking.TrackingNumberGroups {
		err := createTrackingNumberGroup(c, &group, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, startTrackingResp{
				Error: err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusCreated, startTrackingResp{
		Error: "",
	})

}

type getTracking struct {
	TrackingNumbers []string `json:"tracking_numbers"`
	Token           string   `json:"token"`
}

type trackingInfo struct {
	TrackingNumberStatuses map[string]string `json:"tracking_info"`
}

// Register godoc
//
//	@Summary	Gets the status of tracking numbers
//	@ID			get-tracking-numbers
//	@Accept		json
//	@Produce	json
//	@Param		getTrackingInfo	body		getTracking	true	"Tracking numbers"
//	@Success	200				{object}	trackingInfo
//	@Failure	403				{object}	trackingInfo
//	@Failure	500				{object}	trackingInfo
//	@Router		/get_tracking [post]
func GetTrackingNumbers(c *gin.Context) {
	getTracking := getTracking{}
	if err := c.BindJSON(&getTracking); err != nil {
		err = fmt.Errorf("error parsing StartTracking: %w")
	}

	userID, err := auth.UserIDFromToken(c, getTracking.Token)
	if err != nil {
		err = fmt.Errorf("auth error: %w", err)
		switch {
		case errors.Is(err, auth.InvalidToken):
			c.JSON(http.StatusForbidden, err.Error())
			return
		default:
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
	}

	db := db.FromGinContext(c)

	rows, err := db.Query("select tracking_number, status from tracking where tracking_number in (?) and user_id = ?", strings.Join(getTracking.TrackingNumbers, "?"), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	trackingInfo := trackingInfo{
		TrackingNumberStatuses: map[string]string{},
	}

	for rows.Next() {
		var trackingNumber string
		var status string
		if err := rows.Scan(&trackingNumber, &status); err != nil {
			return
		}

		trackingInfo.TrackingNumberStatuses[trackingNumber] = status
	}

	c.JSON(http.StatusOK, trackingInfo)
}

func createTrackingNumberGroup(c *gin.Context, trackingNumberGroup *trackingNumberGroup, userID int32) error {
	db := db.FromGinContext(c)
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	groupName := &trackingNumberGroup.GroupName
	if *groupName == "" {
		groupName = nil
	}
	resp, err := fedex.TrackByTrackingNumber(trackingNumberGroup.TrackingNumbers)
	if err != nil {
		panic(err)
	}

	for trackingNumber, trackingStatus := range resp {
		_, err := tx.Exec(
			"insert into tracking (tracking_number, status, group_name, status_last_updated, user_id) values ( ?, ?, ?, datetime('now'), ? );",
			trackingNumber, trackingStatus.StatusDescription, trackingNumberGroup.GroupName, userID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
