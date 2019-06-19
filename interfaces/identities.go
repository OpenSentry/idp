package interfaces

type GetIdentitiesRequest struct {
  Id            string          `json:"id"`
}

type GetIdentitiesResponse struct {
  Id            string          `json:"id"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}

type PostIdentitiesRequest struct {
  Id            string          `json:"id"`
  Password      string          `json:"password"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}

type PostIdentitiesResponse struct {
  Id            string          `json:"id"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}

type PutIdentitiesRequest struct {
  Id            string          `json:"id"`
  Password      string          `json:"password"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}

type PutIdentitiesResponse struct {
  Id            string          `json:"id"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
}
