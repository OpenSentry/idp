package client

import (
  "net/http"  
  "io/ioutil"
)

// FIXME: This belongs in AAP
type RevokeConsentRequest struct {
  Id string `json:"id"`
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
