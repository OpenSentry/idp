package interfaces

type PostIdentitiesRecoverRequest struct {
  Id		string		`json:"id"`
}

type PostIdentitiesRecoverResponse struct {
  Id		string		`json:"id"`
  Email		string		`json:"email"`
  RecoverMethod string		`json:"recover_method"`
}
