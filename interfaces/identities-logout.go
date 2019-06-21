package interfaces

type PostIdentitiesLogoutRequest struct {
  Challenge       string            `json:"challenge" binding:"required"`
}

type PostIdentitiesLogoutResponse struct {

}
