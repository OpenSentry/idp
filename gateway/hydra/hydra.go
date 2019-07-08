package hydra

import (
  "net/http"
  "bytes"
  "encoding/json"
  "io/ioutil"
  _ "fmt"

  "golang-idp-be/config"
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

func getDefaultHeaders() map[string][]string {
  return map[string][]string{
    "Content-Type": []string{"application/json"},
    "Accept": []string{"application/json"},
  }
}

func getDefaultHeadersWithAuthentication(accessToken string) map[string][]string {
  return map[string][]string{
    "Content-Type": []string{"application/json"},
    "Accept": []string{"application/json"},
    "Authorization": []string{"Bearer " + accessToken},
  }
}

func GetUserInfo(accessToken string) (HydraUserInfoResponse, error) {
  var hydraUserInfoResponse HydraUserInfoResponse

  client := &http.Client{}

  request, _ := http.NewRequest("GET", config.Hydra.UserInfoUrl, nil)
  request.Header = getDefaultHeadersWithAuthentication(accessToken)

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

func GetLogin(challenge string) (HydraLoginResponse, error) {
  var hydraLoginResponse HydraLoginResponse

  client := &http.Client{}

  request, _ := http.NewRequest("GET", config.Hydra.LoginRequestUrl, nil)
  request.Header = getDefaultHeaders()

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

func AcceptLogin(challenge string, hydraLoginAcceptRequest HydraLoginAcceptRequest) HydraLoginAcceptResponse {
  var hydraLoginAcceptResponse HydraLoginAcceptResponse

  client := &http.Client{}

  body, _ := json.Marshal(hydraLoginAcceptRequest)

  request, _ := http.NewRequest("PUT", config.Hydra.LoginRequestAcceptUrl, bytes.NewBuffer(body))
  request.Header = getDefaultHeaders()

  query := request.URL.Query()
  query.Add("login_challenge", challenge)
  request.URL.RawQuery = query.Encode()

  response, _ := client.Do(request)
  responseData, _ := ioutil.ReadAll(response.Body)
  json.Unmarshal(responseData, &hydraLoginAcceptResponse)

  return hydraLoginAcceptResponse
}

func GetLogout(challenge string) (HydraLogoutResponse, error) {
  var hydraLogoutResponse HydraLogoutResponse

  client := &http.Client{}

  request, _ := http.NewRequest("GET", config.Hydra.LogoutRequestUrl, nil)
  request.Header = getDefaultHeaders()

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

func AcceptLogout(challenge string, hydraLogoutAcceptRequest HydraLogoutAcceptRequest) (HydraLogoutAcceptResponse, error) {
  var hydraLogoutAcceptResponse HydraLogoutAcceptResponse

  client := &http.Client{}

  body, _ := json.Marshal(hydraLogoutAcceptRequest)

  request, _ := http.NewRequest("PUT", config.Hydra.LogoutRequestAcceptUrl, bytes.NewBuffer(body))
  request.Header = getDefaultHeaders()

  query := request.URL.Query()
  query.Add("logout_challenge", challenge)
  request.URL.RawQuery = query.Encode()

  response, _ := client.Do(request)

  responseData, _ := ioutil.ReadAll(response.Body)
  json.Unmarshal(responseData, &hydraLogoutAcceptResponse)

  return hydraLogoutAcceptResponse, nil
}
