package hydra

import (
  "net/http"
  "bytes"
  "encoding/json"
  "io/ioutil"
  _ "fmt"

  "golang.org/x/net/context"
  "golang.org/x/oauth2/clientcredentials"
)

type HydraLoginResponse struct {
  Skip        bool        `json:"skip"`
  RedirectTo  string      `json:"redirect_to"`
  Subject     string      `json:"subject"`
}

type HydraLoginAcceptRequest struct {
  Subject     string      `json:"subject"`
  Remember    bool        `json:"remember,omitempty"`
  RememberFor int       `json:"remember_for,omitempty"`
}

type HydraLoginAcceptResponse struct {
  RedirectTo  string      `json:"redirect_to"`
}

type HydraLogoutResponse struct {
  RequestUrl string `json:"request_url"`
  RpInitiated bool `json:"rp_initiated"`
  Sid string `json:"sid"`
  Subject string `json:"subject"`
}

type HydraLogoutAcceptRequest struct {

}

type HydraLogoutAcceptResponse struct {
  RedirectTo string `json:"redirect_to"`
}

type HydraUserInfoResponse struct {
  Sub        string      `json:"sub"`
}

type HydraClient struct {
  *http.Client
}

func NewHydraClient(config *clientcredentials.Config) *HydraClient {
  ctx := context.Background()
  client := config.Client(ctx)
  return &HydraClient{client}
}

// config.Hydra.UserInfoUrl
func GetUserInfo(url string, client *HydraClient) (HydraUserInfoResponse, error) {
  var hydraUserInfoResponse HydraUserInfoResponse

  request, _ := http.NewRequest("GET", url, nil)

  response, err := client.Do(request)
  if err != nil {
    return hydraUserInfoResponse, err
  }

  responseData, err := ioutil.ReadAll(response.Body)
  if err != nil {
    return hydraUserInfoResponse, err
  }
  json.Unmarshal(responseData, &hydraUserInfoResponse)

  return hydraUserInfoResponse, nil
}

// config.Hydra.LoginRequestUrl
func GetLogin(url string, client *HydraClient, challenge string) (HydraLoginResponse, error) {
  var hydraLoginResponse HydraLoginResponse

  request, _ := http.NewRequest("GET", url, nil)

  query := request.URL.Query()
  query.Add("login_challenge", challenge)
  request.URL.RawQuery = query.Encode()

  response, err := client.Do(request)
  if err != nil {
    return hydraLoginResponse, err
  }

  responseData, err := ioutil.ReadAll(response.Body)
  if err != nil {
    return hydraLoginResponse, err
  }
  json.Unmarshal(responseData, &hydraLoginResponse)

  return hydraLoginResponse, nil
}

// config.Hydra.LoginRequestAcceptUrl
func AcceptLogin(url string, client *HydraClient, challenge string, hydraLoginAcceptRequest HydraLoginAcceptRequest) HydraLoginAcceptResponse {
  var hydraLoginAcceptResponse HydraLoginAcceptResponse

  body, _ := json.Marshal(hydraLoginAcceptRequest)

  request, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))

  query := request.URL.Query()
  query.Add("login_challenge", challenge)
  request.URL.RawQuery = query.Encode()

  response, _ := client.Do(request)
  responseData, _ := ioutil.ReadAll(response.Body)
  json.Unmarshal(responseData, &hydraLoginAcceptResponse)

  return hydraLoginAcceptResponse
}

// config.Hydra.LogoutRequestUrl
func GetLogout(url string, client *HydraClient, challenge string) (HydraLogoutResponse, error) {
  var hydraLogoutResponse HydraLogoutResponse

  request, _ := http.NewRequest("GET", url, nil)

  query := request.URL.Query()
  query.Add("logout_challenge", challenge)
  request.URL.RawQuery = query.Encode()

  response, err := client.Do(request)
  if err != nil {
    return hydraLogoutResponse, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)

  json.Unmarshal(responseData, &hydraLogoutResponse)

  return hydraLogoutResponse, nil
}

// config.Hydra.LogoutRequestAcceptUrl
func AcceptLogout(url string, client *HydraClient, challenge string, hydraLogoutAcceptRequest HydraLogoutAcceptRequest) (HydraLogoutAcceptResponse, error) {
  var hydraLogoutAcceptResponse HydraLogoutAcceptResponse

  body, _ := json.Marshal(hydraLogoutAcceptRequest)

  request, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))

  query := request.URL.Query()
  query.Add("logout_challenge", challenge)
  request.URL.RawQuery = query.Encode()

  response, _ := client.Do(request)

  responseData, _ := ioutil.ReadAll(response.Body)
  json.Unmarshal(responseData, &hydraLogoutAcceptResponse)

  return hydraLogoutAcceptResponse, nil
}
