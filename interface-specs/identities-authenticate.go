package interfaces

type PostIdentitiesAuthenticateRequest struct {
  Id		string		`json:"id"`
  password	string		`json:"password"`
}

type PostIdentitiesAuthenticateResponse struct {
  Id		string		`json:"id"`
  Authenticated	bool		`json:"authenticated"`
}
