package interfaces

type HydraLoginResponse struct {
  Skip        bool        `json:"skip"`
  RedirectTo  string      `json:"redirect_to"`
  Subject     string      `json:"subject"`
}

type HydraLoginAcceptRequest struct {
  Subject     string      `json:"subject"`
}

type HydraLoginAcceptResponse struct {
  RedirectTo  string      `json:"redirect_to"`
}
