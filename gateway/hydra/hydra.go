package hydra

import (
  "golang-idp-be/config"
  "golang-idp-be/interfaces"
  "net/http"
  "bytes"
  "encoding/json"
  "io/ioutil"
  _ "fmt"
)

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

func GetUserInfo(accessToken string) (interfaces.HydraUserInfoResponse, error) {
  var hydraUserInfoResponse interfaces.HydraUserInfoResponse

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

func GetLogin(challenge string) (interfaces.HydraLoginResponse, error) {
  var hydraLoginResponse interfaces.HydraLoginResponse

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

func AcceptLogin(challenge string, hydraLoginAcceptRequest interfaces.HydraLoginAcceptRequest) interfaces.HydraLoginAcceptResponse {
  var hydraLoginAcceptResponse interfaces.HydraLoginAcceptResponse

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

func GetLogout(challenge string) (interfaces.HydraLogoutResponse, error) {
  var hydraLogoutResponse interfaces.HydraLogoutResponse

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

func AcceptLogout(challenge string, hydraLogoutAcceptRequest interfaces.HydraLogoutAcceptRequest) (interfaces.HydraLogoutAcceptResponse, error) {
  var hydraLogoutAcceptResponse interfaces.HydraLogoutAcceptResponse

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
