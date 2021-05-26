package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"io/ioutil"
	"net/http"
	"time"
)

type IdpClient struct {
	*http.Client
}

func NewIdpClient(config *clientcredentials.Config) *IdpClient {
	ctx := context.Background()
	client := config.Client(ctx)
	return &IdpClient{client}
}

func NewIdpClientWithUserAccessToken(config *oauth2.Config, token *oauth2.Token) *IdpClient {
	ctx := context.Background()
	client := config.Client(ctx, token)
	return &IdpClient{client}
}

func handleRequest(client *IdpClient, request interface{}, method string, url string, response interface{}) (status int, err error) {
	body, err := json.Marshal(request)
	if err != nil {
		return 999, err
	}

	status, responseData, err := callService(client, method, url, bytes.NewBuffer(body))

	if err != nil {
		return status, err
	}

	if status == 200 {
		err = json.Unmarshal(responseData, &response)
		if err != nil {
			return 666, err
		}
	}

	return status, nil
}

func callService(client *IdpClient, method string, url string, data *bytes.Buffer) (int, []byte, error) {
	if client == nil {
		return http.StatusInternalServerError, nil, errors.New("Missing client")
	}

	// for logging only
	start := time.Now()
	reqData := (*data).Bytes()

	req, err := http.NewRequest("POST", url, data)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	req.Header.Set("X-HTTP-Method-Override", method)

	res, err := client.Do(req)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	defer res.Body.Close()

	resData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, err
	}

	err = parseStatusCode(res.StatusCode)
	if err != nil {
		return res.StatusCode, nil, err
	}

	logRequestResponse(method, url, reqData, res.Status, resData, err, time.Since(start))

	return res.StatusCode, resData, nil
}

func logRequestResponse(method string, url string, reqData []byte, resStatus string, resData []byte, err error, duration time.Duration) {
	var prettyJsonRequest bytes.Buffer
	e := json.Indent(&prettyJsonRequest, reqData, "", "  ")

	if e != nil {
		fmt.Println(e.Error())
	}

	var response string
	if err == nil {
		var prettyJsonResponse bytes.Buffer
		json.Indent(&prettyJsonResponse, resData, "", "  ")
		response = string(prettyJsonResponse.Bytes())
	} else {
		response = "Error: " + err.Error()
	}

	request := string(prettyJsonRequest.Bytes())

	fmt.Printf("\n============== REST DEBUGGING ===============\n%s %s (%s) %s -> [%s] %s\n\n", method, url, duration, request, resStatus, response)
}

func parseStatusCode(statusCode int) error {
	switch statusCode {
	case http.StatusOK,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable:
		return nil
	}
	return errors.New(fmt.Sprintf("Unsupported status code: '%d'", statusCode))
}
