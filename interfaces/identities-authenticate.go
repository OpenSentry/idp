package interfaces

type PostIdentitiesAuthenticateRequest struct {
  Id              string            `json:"id" binding:"required"`
  Password        string            `json:"password" binding:"required"`
  Challenge       string            `json:"challenge" binding:"required"`
}

type PostIdentitiesAuthenticateResponse struct {
  Id              string            `json:"id"`
  Authenticated   bool              `json:"authenticated"`
}
