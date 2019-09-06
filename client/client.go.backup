package client

import (
  "errors"
  "net/http"
  "bytes"
  "encoding/json"
  "io/ioutil"
  "golang.org/x/net/context"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/clientcredentials"

  "github.com/charmixer/idp/identities"
  "github.com/charmixer/idp/challenges"
)

type IdpApiClient struct {
  *http.Client
}

func NewIdpApiClient(config *clientcredentials.Config) *IdpApiClient {
  ctx := context.Background()
  client := config.Client(ctx)
  return &IdpApiClient{client}
}

func NewIdpApiClientWithUserAccessToken(config *oauth2.Config, token *oauth2.Token) *IdpApiClient {
  ctx := context.Background()
  client := config.Client(ctx, token)
  return &IdpApiClient{client}
}

// config.IdpApi.IdentitiesUrl
func CreateIdentity(identitiesUrl string, client *IdpApiClient, identityRequest identities.IdentitiesRequest) (identities.IdentitiesResponse, error) {
  var identityResponse identities.IdentitiesResponse

  body, err := json.Marshal(identityRequest)
  if err != nil {
    return identityResponse, err
  }

  var data = bytes.NewBuffer(body)

  request, err := http.NewRequest("POST", identitiesUrl, data)
  if err != nil {
    return identityResponse, err
  }

  response, err := client.Do(request)
  if err != nil {
    return identityResponse, err
  }

  responseData, err := ioutil.ReadAll(response.Body)
  if err != nil {
    return identityResponse, err
  }

  if response.StatusCode != 200 {
    return identityResponse, errors.New(string(responseData))
  }

  err = json.Unmarshal(responseData, &identityResponse)
  if err != nil {
    return identityResponse, err
  }
  return identityResponse, nil
}

func DeleteIdentity(deleteUrl string, client *IdpApiClient, deleteProfileRequest identities.DeleteRequest) (identities.DeleteResponse, error) {
  var deleteResponse identities.DeleteResponse

  body, _ := json.Marshal(deleteProfileRequest)

  var data = bytes.NewBuffer(body)

  request, _ := http.NewRequest("DELETE", deleteUrl, data)

  response, err := client.Do(request)
  if err != nil {
    return deleteResponse, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)

  if response.StatusCode != 200 {
    return deleteResponse, errors.New(string(responseData))
  }

  err = json.Unmarshal(responseData, &deleteResponse)
  if err != nil {
    return deleteResponse, err
  }

  return deleteResponse, nil
}

func DeleteIdentityVerification(deleteVerificationUrl string, client *IdpApiClient, deleteRequest identities.DeleteVerificationRequest) (identities.DeleteVerificationResponse, error) {
  var deleteVerificationResponse identities.DeleteVerificationResponse

  body, _ := json.Marshal(deleteRequest)

  var data = bytes.NewBuffer(body)

  request, _ := http.NewRequest("POST", deleteVerificationUrl, data)

  response, err := client.Do(request)
  if err != nil {
    return deleteVerificationResponse, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)

  if response.StatusCode != 200 {
    return deleteVerificationResponse, errors.New(string(responseData))
  }

  err = json.Unmarshal(responseData, &deleteVerificationResponse)
  if err != nil {
    return deleteVerificationResponse, err
  }

  return deleteVerificationResponse, nil
}

func UpdateIdentity(identitiesUrl string, client *IdpApiClient, identityRequest identities.IdentitiesRequest) (identities.IdentitiesResponse, error) {
  var identityResponse identities.IdentitiesResponse

  body, err := json.Marshal(identityRequest)
  if err != nil {
    return identityResponse, nil
  }

  var data = bytes.NewBuffer(body)

  request, err := http.NewRequest("PUT", identitiesUrl, data)
  if err != nil {
    return identityResponse, nil
  }

  response, err := client.Do(request)
  if err != nil {
    return identityResponse, err
  }

  responseData, err := ioutil.ReadAll(response.Body)
  if err != nil {
    return identityResponse, err
  }

  if response.StatusCode != 200 {
    return identityResponse, errors.New(string(responseData))
  }

  err = json.Unmarshal(responseData, &identityResponse)
  if err != nil {
    return identityResponse, err
  }
  return identityResponse, nil
}

func UpdateTotp(identitiesUrl string, client *IdpApiClient, totpRequest identities.TotpRequest) (identities.TotpResponse, error) {
  var totpResponse identities.TotpResponse

  body, err := json.Marshal(totpRequest)
  if err != nil {
    return totpResponse, err
  }

  var data = bytes.NewBuffer(body)

  request, err := http.NewRequest("POST", identitiesUrl, data)
  if err != nil {
    return totpResponse, err
  }

  response, err := client.Do(request)
  if err != nil {
    return totpResponse, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)
  if response.StatusCode != 200 {
    return totpResponse, errors.New(string(responseData))
  }

  err = json.Unmarshal(responseData, &totpResponse)
  if err != nil {
    return totpResponse, err
  }

  return totpResponse, nil
}

func UpdatePassword(passwordUrl string, client *IdpApiClient, passwordRequest identities.PasswordRequest) (identities.PasswordResponse, error) {
  var passwordResponse identities.PasswordResponse

  body, err := json.Marshal(passwordRequest)
  if err != nil {
    return passwordResponse, err
  }

  var data = bytes.NewBuffer(body)

  request, err := http.NewRequest("POST", passwordUrl, data)
  if err != nil {
    return passwordResponse, err
  }

  response, err := client.Do(request)
  if err != nil {
    return passwordResponse, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)
  if response.StatusCode != 200 {
    return passwordResponse, errors.New(string(responseData))
  }

  err = json.Unmarshal(responseData, &passwordResponse)
  if err != nil {
    return passwordResponse, err
  }

  return passwordResponse, nil
}

// config.IdpApi.IdentitiesUrl
func FetchIdentity(identitiesUrl string, client *IdpApiClient, identityRequest identities.IdentitiesRequest) (identities.IdentitiesResponse, error) {
  var identityResponse identities.IdentitiesResponse

  request, err := http.NewRequest("GET", identitiesUrl, nil)
  if err != nil {
    return identityResponse, err
  }

  query := request.URL.Query()
  query.Add("id", identityRequest.Id)
  request.URL.RawQuery = query.Encode()

  response, err := client.Do(request)
  if err != nil {
    return identityResponse, err
  }

  responseData, err := ioutil.ReadAll(response.Body)
  if err != nil {
    return identityResponse, err
  }

  err = json.Unmarshal(responseData, &identityResponse)
  if err != nil {
    return identityResponse, err
  }
  return identityResponse, nil
}

// config.IdpApi.AuthenticateUrl
func Authenticate(authenticateUrl string, client *IdpApiClient, authenticateRequest identities.AuthenticateRequest) (identities.AuthenticateResponse, error) {
  var authenticateResponse identities.AuthenticateResponse

  body, _ := json.Marshal(authenticateRequest)

  var data = bytes.NewBuffer(body)

  request, _ := http.NewRequest("POST", authenticateUrl, data)

  response, err := client.Do(request)
  if err != nil {
    return authenticateResponse, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)

  if response.StatusCode != 200 {
    return authenticateResponse, errors.New(string(responseData))
  }

  err = json.Unmarshal(responseData, &authenticateResponse)
  if err != nil {
    return authenticateResponse, err
  }

  return authenticateResponse, nil
}

func FetchChallenge(challengeUrl string, client *IdpApiClient, challengeRequest challenges.OtpChallengeRequest) (challenges.OtpChallengeResponse, error) {
  var challengeResponse challenges.OtpChallengeResponse

  request, err := http.NewRequest("GET", challengeUrl, nil)
  if err != nil {
    return challengeResponse, err
  }

  query := request.URL.Query()
  query.Add("otp_challenge", challengeRequest.OtpChallenge)
  request.URL.RawQuery = query.Encode()

  response, err := client.Do(request)
  if err != nil {
    return challengeResponse, err
  }

  responseData, err := ioutil.ReadAll(response.Body)
  if err != nil {
    return challengeResponse, err
  }

  if response.StatusCode != 200 {
    return challengeResponse, errors.New(string(responseData))
  }

  err = json.Unmarshal(responseData, &challengeResponse)
  if err != nil {
    return challengeResponse, err
  }
  return challengeResponse, nil
}

func VerifyChallenge(verifyUrl string, client *IdpApiClient, verifyRequest challenges.VerifyRequest) (challenges.VerifyResponse, error) {
  var verifyResponse challenges.VerifyResponse

  body, _ := json.Marshal(verifyRequest)

  var data = bytes.NewBuffer(body)

  request, _ := http.NewRequest("POST", verifyUrl, data)

  response, err := client.Do(request)
  if err != nil {
    return verifyResponse, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)

  if response.StatusCode != 200 {
    return verifyResponse, errors.New(string(responseData))
  }

  err = json.Unmarshal(responseData, &verifyResponse)
  if err != nil {
    return verifyResponse, err
  }

  return verifyResponse, nil
}

func Recover(recoverUrl string, client *IdpApiClient, recoverRequest identities.RecoverRequest) (identities.RecoverResponse, error) {
  var recoverResponse identities.RecoverResponse

  body, _ := json.Marshal(recoverRequest)

  var data = bytes.NewBuffer(body)

  request, _ := http.NewRequest("POST", recoverUrl, data)

  response, err := client.Do(request)
  if err != nil {
    return recoverResponse, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)

  if response.StatusCode != 200 {
    return recoverResponse, errors.New(string(responseData))
  }

  err = json.Unmarshal(responseData, &recoverResponse)
  if err != nil {
    return recoverResponse, err
  }

  return recoverResponse, nil
}

func RecoverVerification(recoverVerificationUrl string, client *IdpApiClient, recoverRequest identities.RecoverVerificationRequest) (identities.RecoverVerificationResponse, error) {
  var recoverResponse identities.RecoverVerificationResponse

  body, _ := json.Marshal(recoverRequest)

  var data = bytes.NewBuffer(body)

  request, _ := http.NewRequest("POST", recoverVerificationUrl, data)

  response, err := client.Do(request)
  if err != nil {
    return recoverResponse, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)

  if response.StatusCode != 200 {
    return recoverResponse, errors.New(string(responseData))
  }

  err = json.Unmarshal(responseData, &recoverResponse)
  if err != nil {
    return recoverResponse, err
  }

  return recoverResponse, nil
}

// config.IdpApi.LogoutUrl
func Logout(logoutUrl string, client *IdpApiClient, logoutRequest identities.LogoutRequest) (identities.LogoutResponse, error) {
  var logoutResponse identities.LogoutResponse

  body, _ := json.Marshal(logoutRequest)

  var data = bytes.NewBuffer(body)

  request, _ := http.NewRequest("POST", logoutUrl, data)

  response, err := client.Do(request)
  if err != nil {
    return logoutResponse, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)

  err = json.Unmarshal(responseData, &logoutResponse)
  if err != nil {
    return logoutResponse, err
  }

  return logoutResponse, nil
}
