package checker

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/ytax/snapchat-username-checker/modules/requestparser"
	"google.golang.org/protobuf/proto"
)

// CheckUsername returns true if the provided username is available.
func CheckUsername(username string) (bool, error) {
	req := &requestparser.SuggestUsernameRequest{
		Username:      &requestparser.SuggestUsernameRequest_NameWrapper{Name: username},
		Locale:        "",
		SomethingFlag: 0,
		DeviceId:      "c798e85f-4511-66b0-889a-ef303fa6bfab",
		SessionId:     "6687bd20-731d-387c-e3b9-d47c5a90f410",
	}

	body, err := proto.Marshal(req)
	if err != nil {
		return false, err
	}

	payload := append([]byte{0}, uint32ToBytes(uint32(len(body)))...)
	payload = append(payload, body...)

	headers := map[string]string{
		"Content-Type":            "application/grpc",
		"TE":                      "trailers",
		"Grpc-Accept-Encoding":    "identity, deflate, gzip",
		"Grpc-Timeout":            "3S",
		"User-Agent":              "Snapchat/13.21.0.43 (moto g play (2021); Android 11#e00ca2#30; gzip) V/MUSHROOM grpc-c++/1.48.0 grpc-c/26.0.0 (android; cronet_http)",
		"Allow-Recycled-Username": "true",
		"X-Request-Id":            "63adac91-301f-46d3-a576-44c28d302153",
	}

	client := &http.Client{Timeout: 3 * time.Second}
	url := "https://aws.api.snapchat.com/snapchat.activation.api.SuggestUsernameService/SuggestUsername"
	reqHTTP, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return false, err
	}

	for k, v := range headers {
		reqHTTP.Header.Set(k, v)
	}

	resp, err := client.Do(reqHTTP)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	bodyResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	if len(bodyResp) <= 5 {
		return false, nil
	}
	bodyResp = bodyResp[5:]

	var response requestparser.SuggestUsernameResponse
	if err := proto.Unmarshal(bodyResp, &response); err != nil {
		return false, err
	}

	suggestions := response.GetSuggestions()
	return len(suggestions) > 0 && suggestions[0] == username, nil
}

// CheckUsernames checks a list of usernames using a pool of workers.
func CheckUsernames(usernames []string, workers int) (available []string, unavailable []string) {
	if workers <= 0 {
		workers = 1
	}

	sem := make(chan struct{}, workers)
	results := make(chan struct {
		name  string
		avail bool
	})

	for _, u := range usernames {
		u := u
		sem <- struct{}{}
		go func() {
			avail, _ := CheckUsername(u)
			results <- struct {
				name  string
				avail bool
			}{u, avail}
			<-sem
		}()
	}

	for range usernames {
		r := <-results
		if r.avail {
			available = append(available, r.name)
		} else {
			unavailable = append(unavailable, r.name)
		}
	}
	return
}

func uint32ToBytes(n uint32) []byte {
	return []byte{
		byte(n >> 24),
		byte(n >> 16),
		byte(n >> 8),
		byte(n),
	}
}
