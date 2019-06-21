package interfaces

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
