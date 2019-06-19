package hydra

import (
  "golang-idp-be/config"
  "golang-idp-be/interfaces"
  "net/http"
  "bytes"
  "encoding/json"
  "io/ioutil"
)

func getDefaultHeaders() map[string][]string {
  return map[string][]string{
    "Content-Type": []string{"application/json"},
    "Accept": []string{"application/json"},
  }
}

func GetLogin(challenge string) interfaces.HydraLoginResponse {

  client := &http.Client{}

  request, _ := http.NewRequest("GET", config.Hydra.LoginRequestUrl, nil)
  request.Header = getDefaultHeaders()

  query := request.URL.Query()
  query.Add("login_challenge", challenge)
  request.URL.RawQuery = query.Encode()

  response, _ := client.Do(request)

  responseData, _ := ioutil.ReadAll(response.Body)

  var hydraLoginResponse interfaces.HydraLoginResponse
  json.Unmarshal(responseData, &hydraLoginResponse)

  return hydraLoginResponse
}

func AcceptLogin(challenge string, hydraLoginAcceptRequest interfaces.HydraLoginAcceptRequest) interfaces.HydraLoginAcceptResponse {
  // call hydra with accept login request

  client := &http.Client{}

  body, _ := json.Marshal(hydraLoginAcceptRequest)

  request, _ := http.NewRequest("PUT", config.Hydra.LoginRequestAcceptUrl, bytes.NewBuffer(body))
  request.Header = getDefaultHeaders()

  query := request.URL.Query()
  query.Add("login_challenge", challenge)
  request.URL.RawQuery = query.Encode()

  response, _ := client.Do(request)

  responseData, _ := ioutil.ReadAll(response.Body)

  var hydraLoginAcceptResponse interfaces.HydraLoginAcceptResponse
  json.Unmarshal(responseData, &hydraLoginAcceptResponse)

  return hydraLoginAcceptResponse
}
