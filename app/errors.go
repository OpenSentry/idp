package app

const ERROR_INVALID_ACCESS_TOKEN = 1
const ERROR_MISSING_BEARER_TOKEN = 2
const ERROR_MISSING_REQUIRED_SCOPES = 3

type JsonError struct {
  ErrorCode int `json:"error_code" binding:"required"`
  Error     string `json:"error" binding:"required"`
}