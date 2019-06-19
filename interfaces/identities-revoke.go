package interfaces

type PostIdentitiesRevokeRequest struct {
  Id		string		`json:"id"`
}

type PostIdentitiesRevokeResponse struct {
  Id		string		`json:"id"`
  Revoked	bool		`json:"revoked"`
}
