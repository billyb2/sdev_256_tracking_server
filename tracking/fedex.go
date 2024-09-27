package fedex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type trackingNumberInfo struct {
	TrackingNumber string `json:"trackingNumber"`
}

type trackBody struct {
	IncludeDetailedScans bool                 `json:"includeDetailedScans"`
	TrackingInfo         []trackingNumberInfo `json:"trackingInfo"`
}

type completeTrackResult struct {
	TrackingNumber string           `json:"trackingNumber"`
	TrackResults   []map[string]any `json:"trackResults"`
}

type trackResp struct {
	Output map[string][]any `json:"output"`
}

type DistanceToDestination struct {
	Units string  `json:"units"`
	Value float64 `json:"value"`
}

type TrackingNumberStatus struct {
	StatusDescription     string                `json:"status_description"`
	DistanceToDestination DistanceToDestination `json:"distance_to_destination"`
}

type authResp struct {
	AccessToken string `json:"access_token"`
}

var authBearer string

func getAuth() (string, error) {
	if authBearer != "" {
		return authBearer, nil
	}

	secretKey := os.Getenv("SECRET_KEY")
	publicKey := os.Getenv("PUBLIC_KEY")

	data := url.Values{}
	data.Add("client_id", publicKey)
	data.Add("client_secret", secretKey)
	data.Add("grant_type", "client_credentials")

	baseURL := "https://apis-sandbox.fedex.com/"
	url, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	url.Path = "/oauth/token"
	urlStr := url.String()
	client := &http.Client{}
	r, _ := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(data.Encode())) // URL-encoded payload
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(r)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("%s", string(body))
		return "", err
	}

	authResp := authResp{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&authResp)
	if err != nil {
		return "", err
	}

	authBearer = authResp.AccessToken

	return authResp.AccessToken, nil
}

func TrackByTrackingNumber(trackingNumbers []string) (map[string]TrackingNumberStatus, error) {
	if len(trackingNumbers) == 0 {
		return nil, nil
	}

	trackingNumberInfos := []trackingNumberInfo{}
	for _, trackingNumber := range trackingNumbers {
		trackingNumberInfos = append(trackingNumberInfos, trackingNumberInfo{
			TrackingNumber: trackingNumber,
		})
	}

	body := trackBody{
		IncludeDetailedScans: false,
		TrackingInfo:         trackingNumberInfos,
	}
	auth, err := getAuth()
	if err != nil {
		return nil, err
	}

	resp, err := sendAPIRequest("track/v1/trackingnumbers", body, auth)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("non-200 status code %d: %s", resp.StatusCode, string(respBody))
	}

	var trackResp trackResp
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&trackResp); err != nil {
		return nil, err
	}

	results := map[string]TrackingNumberStatus{}

	reqResults := trackResp.Output["completeTrackResults"]
	// the structure of fedex api responses makes me want to die.
	for _, reqResult := range reqResults {
		result := reqResult.(completeTrackResult)
		for _, trackResult := range result.TrackResults {
			latestStatusDetail := trackResult["latestStatusDetail"].(map[string]any)
			status := TrackingNumberStatus{
				StatusDescription:     latestStatusDetail["description"].(string),
				DistanceToDestination: trackResult["distanceToDestination"].(DistanceToDestination),
			}

			results[result.TrackingNumber] = status
		}

	}

	return results, nil
}

func sendAPIRequest(apiPath string, body any, auth string) (*http.Response, error) {
	httpClient := http.DefaultClient

	jsonPayload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	baseURL := "https://apis-sandbox.fedex.com/"
	url, err := url.Parse(baseURL + apiPath)
	if err != nil {
		return nil, err
	}
	headers := map[string][]string{
		"X-locale":     []string{"en_US"},
		"Content-Type": []string{"application/json"},
	}
	if auth != "" {
		headers["Authorization"] = []string{"Bearer " + auth}
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    url,
		Header: headers,
		Body:   io.NopCloser(bytes.NewReader([]byte(jsonPayload))),
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s", string(body))
	}

	return resp, nil

}
