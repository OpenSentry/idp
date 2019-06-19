package interfaces

type HydraLoginRequestResponse struct {
  Skip        bool        `json:"skip"`
  RedirectTo  string      `json:"redirect_to"`
  Subject     string      `json:"subject"`
}

type HydraLoginRequestAcceptResponse struct {
  RedirectTo  string      `json:"redirect_to"`
}
