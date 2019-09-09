package client

import (
  "net/http"
  "bytes"
  "encoding/json"
  . "github.com/charmixer/idp/models"
)

func ReadChallenge(client *IdpClient, challengeUrl string, request *ChallengesReadRequest) (*ChallengesReadResponse, error) {
  var response ChallengesReadResponse

  req, err := http.NewRequest("GET", challengeUrl, nil)
  if err != nil {
    return nil, err
  }

  // TODO: Can we marshal this somehow?
  query := req.URL.Query()
  query.Add("otp_challenge", request.OtpChallenge)
  req.URL.RawQuery = query.Encode()

  res, err := client.Do(req)
  if err != nil {
    return nil, err
  }

  result, err := parseResponse(res)
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func CreateChallenge(client *IdpClient, challengeUrl string, request *ChallengeCreateRequest) (*ChallengeCreateResponse, error) {
  var response ChallengeCreateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", challengeUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func VerifyChallenge(client *IdpClient, verifyUrl string, request *ChallengeVerifyRequest) (*ChallengeVerifyResponse, error) {
  var response ChallengeVerifyResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", verifyUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}
