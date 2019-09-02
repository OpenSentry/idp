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
)

type AuthenticateRequest struct {
  Id              string            `json:"id"`
  Password        string            `json:"password"`
  Challenge       string            `json:"challenge" binding:"required"`
}

type AuthenticateResponse struct {
  Id              string            `json:"id"`
  NotFound        bool              `json:"not_found"`
  Authenticated   bool              `json:"authenticated"`
  Require2Fa      bool              `json:"require_2fa"`
  RedirectTo      string            `json:"redirect_to,omitempty"`
}

type PasscodeRequest struct {
  Id        string `json:"id" binding:"required"`
  Passcode  string `json:"passcode" binding:"required"`
  Challenge string `json:"challenge" binding:"required"`
}

type PasscodeResponse struct {
  Id         string `json:"id" binding:"required"`
  Verified   bool   `json:"verifed" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
}

type LogoutRequest struct {
  Challenge       string            `json:"challenge" binding:"required"`
}

type LogoutResponse struct {
  RedirectTo      string            `json:"redirect_to" binding:"required"`
}

type IdentityRequest struct {
  Id            string          `json:"id" binding:"required"`
  Name          string          `json:"name,omitempty"`
  Email         string          `json:"email,omitempty"`
  Password      string          `json:"password,omitempty"`
  Require2Fa    bool            `json:"require_2fa,omitempty"`
  Secret2Fa     string          `json:"secret_2fa,omitempty"`
}

type IdentityResponse struct {
  Id            string          `json:"id" binding:"required"`
  Name          string          `json:"name,omitempty"`
  Email         string          `json:"email,omitempty"`
  Password      string          `json:"password,omitempty"`
  Require2Fa    bool            `json:"require_2fa,omitempty"`
  Secret2Fa     string          `json:"secret_2fa,omitempty"`
}

type RevokeConsentRequest struct {
  Id string `json:"id"`
}

type UserInfoResponse struct {
  Sub       string      `json:"sub"`
}

type RecoverRequest struct {
  Id              string            `json:"id" binding:"required"`
  Password        string            `json:"password" binding:"required"`
}

type RecoverResponse struct {
  Id         string `json:"id" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
}

type RecoverVerificationRequest struct {
  Id               string `json:"id" binding:"required"`
  VerificationCode string `json:"verification_code" binding:"required"`
  Password         string `json:"password" binding:"required"`
  RedirectTo       string `json:"redirect_to" binding:"required"`
}

type RecoverVerificationResponse struct {
  Id         string `json:"id" binding:"required"`
  Verified   bool   `json:"verifed" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
}

type DeleteProfileRequest struct {
  Id              string            `json:"id" binding:"required"`
}

type DeleteProfileResponse struct {
  Id         string `json:"id" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
}

type DeleteProfileVerificationRequest struct {
  Id               string `json:"id" binding:"required"`
  VerificationCode string `json:"verification_code" binding:"required"`
  RedirectTo       string `json:"redirect_to" binding:"required"`
}

type DeleteProfileVerificationResponse struct {
  Id         string `json:"id" binding:"required"`
  Verified   bool   `json:"verifed" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
}

// App structs

type TwoFactor struct {
  Required bool
  Secret string
}

type Profile struct {
  Id              string
  Name            string
  Email           string
  Password        string
  TwoFactor       TwoFactor
}

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

// config.AapApi.AuthorizationsUrl
func RevokeConsent(url string, client *IdpApiClient, revokeConsentRequest RevokeConsentRequest) (bool, error) {

  // FIXME: Call hydra directly. This should not be allowed! (idpui does not have hydra scope)
  // It should call aap instead. But for testing this was faster.
  u := "https://oauth.localhost/admin/oauth2/auth/sessions/consent?subject=" + revokeConsentRequest.Id
  consentRequest, err := http.NewRequest("DELETE", u, nil)
  if err != nil {
    return false, err
  }

  response, err := client.Do(consentRequest)
  if err != nil {
    return false, err
  }

  _ /* responseData */, err = ioutil.ReadAll(response.Body)
  if err != nil {
    return false, err
  }

  return true, nil
}

// config.IdpApi.IdentitiesUrl
func CreateProfile(identitiesUrl string, client *IdpApiClient, profile Profile) (Profile, error) {
  var identityResponse IdentityResponse
  var newProfile Profile

  identityRequest := IdentityRequest{
    Id: profile.Id,
    Name: profile.Name,
    Email: profile.Email,
    Password: profile.Password,
  }
  body, _ := json.Marshal(identityRequest)

  var data = bytes.NewBuffer(body)

  request, _ := http.NewRequest("POST", identitiesUrl, data)

  response, err := client.Do(request)
  if err != nil {
    return newProfile, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)
  if response.StatusCode != 200 {
    return newProfile, errors.New("status: " + string(response.StatusCode) + ", error="+string(responseData))
  }

  err = json.Unmarshal(responseData, &identityResponse)
  if err != nil {
    return newProfile, err
  }

  newProfile = Profile{
    Id: identityResponse.Id,
    Name: identityResponse.Name,
    Email: identityResponse.Email,
    Password: identityResponse.Password,
    TwoFactor: TwoFactor{
      Required: identityResponse.Require2Fa,
      Secret: identityResponse.Secret2Fa,
    },
  }
  return newProfile, nil
}

func DeleteProfile(deleteUrl string, client *IdpApiClient, deleteProfileRequest DeleteProfileRequest) (DeleteProfileResponse, error) {
  var deleteResponse DeleteProfileResponse

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

func DeleteProfileVerification(deleteVerificationUrl string, client *IdpApiClient, deleteRequest DeleteProfileVerificationRequest) (DeleteProfileVerificationResponse, error) {
  var deleteVerificationResponse DeleteProfileVerificationResponse

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

func UpdateProfile(identitiesUrl string, client *IdpApiClient, profile Profile) (Profile, error) {
  var identityResponse IdentityResponse
  var updatedProfile Profile

  identityRequest := IdentityRequest{
    Id: profile.Id,
    Name: profile.Name,
    Email: profile.Email,
  }
  body, _ := json.Marshal(identityRequest)

  var data = bytes.NewBuffer(body)

  request, _ := http.NewRequest("PUT", identitiesUrl, data)

  response, err := client.Do(request)
  if err != nil {
    return updatedProfile, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)
  if response.StatusCode != 200 {
    return updatedProfile, errors.New("status: " + string(response.StatusCode) + ", error="+string(responseData))
  }

  err = json.Unmarshal(responseData, &identityResponse)
  if err != nil {
    return updatedProfile, err
  }

  updatedProfile = Profile{
    Id: identityResponse.Id,
    Name: identityResponse.Name,
    Email: identityResponse.Email,
    Password: identityResponse.Password,
    TwoFactor: TwoFactor{
      Required: identityResponse.Require2Fa,
      Secret: identityResponse.Secret2Fa,
    },
  }
  return updatedProfile, nil
}

func UpdateTwoFactor(identitiesUrl string, client *IdpApiClient, profile Profile) (Profile, error) {
  var identityResponse IdentityResponse
  var updatedProfile Profile

  identityRequest := IdentityRequest{
    Id: profile.Id,
    Require2Fa: profile.TwoFactor.Required,
    Secret2Fa: profile.TwoFactor.Secret,

  }
  body, _ := json.Marshal(identityRequest)

  var data = bytes.NewBuffer(body)

  request, _ := http.NewRequest("POST", identitiesUrl, data)

  response, err := client.Do(request)
  if err != nil {
    return updatedProfile, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)
  if response.StatusCode != 200 {
    return updatedProfile, errors.New("status: " + string(response.StatusCode) + ", error="+string(responseData))
  }

  err = json.Unmarshal(responseData, &identityResponse)
  if err != nil {
    return updatedProfile, err
  }

  updatedProfile = Profile{
    Id: identityResponse.Id,
    Name: identityResponse.Name,
    Email: identityResponse.Email,
    Password: identityResponse.Password,
    TwoFactor: TwoFactor{
      Required: identityResponse.Require2Fa,
      Secret: identityResponse.Secret2Fa,
    },
  }
  return updatedProfile, nil
}

func UpdatePassword(identitiesUrl string, client *IdpApiClient, profile Profile) (Profile, error) {
  var identityResponse IdentityResponse
  var updatedProfile Profile

  identityRequest := IdentityRequest{
    Id: profile.Id,
    Password: profile.Password,
  }
  body, _ := json.Marshal(identityRequest)

  var data = bytes.NewBuffer(body)

  request, _ := http.NewRequest("POST", identitiesUrl, data)

  response, err := client.Do(request)
  if err != nil {
    return updatedProfile, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)
  if response.StatusCode != 200 {
    return updatedProfile, errors.New("status: " + string(response.StatusCode) + ", error="+string(responseData))
  }

  err = json.Unmarshal(responseData, &identityResponse)
  if err != nil {
    return updatedProfile, err
  }

  updatedProfile = Profile{
    Id: identityResponse.Id,
    Name: identityResponse.Name,
    Email: identityResponse.Email,
    Password: identityResponse.Password,
    TwoFactor: TwoFactor{
      Required: identityResponse.Require2Fa,
      Secret: identityResponse.Secret2Fa,
    },
  }
  return updatedProfile, nil
}

// config.IdpApi.IdentitiesUrl
func FetchProfile(url string, client *IdpApiClient, identityRequest IdentityRequest) (Profile, error) {
  var profile Profile
  var identityResponse IdentityResponse
  var userInfoResponse UserInfoResponse

  id := identityRequest.Id
  if id == "" {
    // Ask hydra for user from access token in client.
    userInfoRequest, err := http.NewRequest("GET", url, nil)
    if err != nil {
      return profile, err
    }

    response, err := client.Do(userInfoRequest)
    if err != nil {
      return profile, err
    }

    responseData, err := ioutil.ReadAll(response.Body)
    if err != nil {
      return profile, err
    }

    json.Unmarshal(responseData, &userInfoResponse)
    id = userInfoResponse.Sub
  }

  request, err := http.NewRequest("GET", url, nil)
  if err != nil {
    return profile, err
  }

  query := request.URL.Query()
  query.Add("id", id)
  request.URL.RawQuery = query.Encode()

  response, err := client.Do(request)
  if err != nil {
    return profile, err
  }

  responseData, err := ioutil.ReadAll(response.Body)
  if err != nil {
    return profile, err
  }

  err = json.Unmarshal(responseData, &identityResponse)
  if err != nil {
    return profile, err
  }

  profile = Profile{
    Id: identityResponse.Id,
    Name: identityResponse.Name,
    Email: identityResponse.Email,
    Password: identityResponse.Password,
    TwoFactor: TwoFactor{
      Required: identityResponse.Require2Fa,
      Secret: identityResponse.Secret2Fa,
    },
  }
  return profile, nil
}

// config.IdpApi.AuthenticateUrl
func Authenticate(authenticateUrl string, client *IdpApiClient, authenticateRequest AuthenticateRequest) (AuthenticateResponse, error) {
  var authenticateResponse AuthenticateResponse

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

func VerifyPasscode(passcodeUrl string, client *IdpApiClient, passcodeRequest PasscodeRequest) (PasscodeResponse, error) {
  var passcodeResponse PasscodeResponse

  body, _ := json.Marshal(passcodeRequest)

  var data = bytes.NewBuffer(body)

  request, _ := http.NewRequest("POST", passcodeUrl, data)

  response, err := client.Do(request)
  if err != nil {
    return passcodeResponse, err
  }

  responseData, _ := ioutil.ReadAll(response.Body)

  if response.StatusCode != 200 {
    return passcodeResponse, errors.New(string(responseData))
  }

  err = json.Unmarshal(responseData, &passcodeResponse)
  if err != nil {
    return passcodeResponse, err
  }

  return passcodeResponse, nil
}

func Recover(recoverUrl string, client *IdpApiClient, recoverRequest RecoverRequest) (RecoverResponse, error) {
  var recoverResponse RecoverResponse

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

func RecoverVerification(recoverVerificationUrl string, client *IdpApiClient, recoverRequest RecoverVerificationRequest) (RecoverVerificationResponse, error) {
  var recoverResponse RecoverVerificationResponse

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
func Logout(logoutUrl string, client *IdpApiClient, logoutRequest LogoutRequest) (LogoutResponse, error) {
  var logoutResponse LogoutResponse

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
