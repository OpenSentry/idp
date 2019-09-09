package client

import (
  "net/http"
  "bytes"
  "encoding/json"
  . "github.com/charmixer/idp/models"
)

func CreateIdentity(client *IdpClient, identitiesUrl string, request *IdentitiesCreateRequest) (*IdentitiesCreateResponse, error) {
  var response IdentitiesCreateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", identitiesUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func ReadIdentity(client *IdpClient, identitiesUrl string, request *IdentitiesReadRequest) (*IdentitiesReadResponse, error) {
  var response IdentitiesReadResponse

  req, err := http.NewRequest("GET", identitiesUrl, nil)
  if err != nil {
    return nil, err
  }

  // TODO: Can we marshal this somehow?
  query := req.URL.Query()
  query.Add("id", request.Id)
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

func UpdateIdentity(client *IdpClient, identitiesUrl string, request *IdentitiesUpdateRequest) (*IdentitiesUpdateResponse, error) {
  var response IdentitiesUpdateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "PUT", identitiesUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func DeleteIdentity(client *IdpClient, identitiesUrl string, request *IdentitiesDeleteRequest) (*IdentitiesDeleteResponse, error) {
  var response IdentitiesDeleteResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "DELETE", identitiesUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func DeleteIdentityVerification(client *IdpClient, verificationUrl string, request *IdentitiesDeleteVerificationRequest) (*IdentitiesDeleteVerificationResponse, error) {
  var response IdentitiesDeleteVerificationResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", verificationUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func UpdateIdentityPassword(client *IdpClient, passwordUrl string, request *IdentitiesPasswordRequest) (*IdentitiesPasswordResponse, error) {
  var response IdentitiesPasswordResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "PUT", passwordUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func UpdateIdentityTotp(client *IdpClient, totpUrl string, request *IdentitiesTotpRequest) (*IdentitiesTotpResponse, error) {
  var response IdentitiesTotpResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "PUT", totpUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func AuthenticateIdentity(client *IdpClient, authenticateUrl string, request *IdentitiesAuthenticateRequest) (*IdentitiesAuthenticateResponse, error) {
  var response IdentitiesAuthenticateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", authenticateUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func RecoverIdentity(client *IdpClient, recoverUrl string, request *IdentitiesRecoverRequest) (*IdentitiesRecoverResponse, error) {
  var response IdentitiesRecoverResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", recoverUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func RecoverIdentityVerification(client *IdpClient, verificationUrl string, request *IdentitiesRecoverVerificationRequest) (*IdentitiesRecoverVerificationResponse, error) {
  var response IdentitiesRecoverVerificationResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", verificationUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func LogoutIdentity(client *IdpClient, logoutUrl string, request *IdentitiesLogoutRequest) (*IdentitiesLogoutResponse, error) {
  var response IdentitiesLogoutResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", logoutUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}
