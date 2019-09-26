package client

import (
  "errors"
  "bytes"
  "io/ioutil"
  "net/http"
  "golang.org/x/net/context"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/clientcredentials"
  "fmt"
  "encoding/json"
  "reflect"
)

type IdpClient struct {
  *http.Client
}

func NewIdpClient(config *clientcredentials.Config) *IdpClient {
  ctx := context.Background()
  client := config.Client(ctx)
  return &IdpClient{client}
}

func NewIdpClientWithUserAccessToken(config *oauth2.Config, token *oauth2.Token) *IdpClient {
  ctx := context.Background()
  client := config.Client(ctx, token)
  return &IdpClient{client}
}


type ErrorResponse struct {
  Code  int    `json:"code" binding:"required"`
  Error string `json:"error" binding:"required"`
}

type BulkResponse struct {
  Index  int             `json:"index" binding:"required"`
  Status int             `json:"status" binding:"required"`
  Errors []ErrorResponse `json:"errors,omitempty"`
}

func callService(client *IdpClient, method string, url string, data *bytes.Buffer) (int, []byte, error) {
  // for logging only
  reqData := (*data).Bytes()

  req, err := http.NewRequest("POST", url, data)
  if err != nil {
    return http.StatusBadRequest, nil, err
  }

  req.Header.Set("X-HTTP-Method-Override", method)

  res, err := client.Do(req)

  defer res.Body.Close()

  if err != nil {
    return res.StatusCode, nil, err
  }

  resData, err := ioutil.ReadAll(res.Body)
  if err != nil {
    return http.StatusInternalServerError, nil, err
  }

  err = parseStatusCode(res.StatusCode)
  if err != nil {
    return res.StatusCode, nil, err
  }

  logRequestResponse(method, url, reqData, res.Status, resData, err)

  return res.StatusCode, resData, nil
}

func logRequestResponse(method string, url string, reqData []byte, resStatus string, resData []byte, err error) {
  var prettyJsonRequest bytes.Buffer
  e := json.Indent(&prettyJsonRequest, reqData, "", "  ")

  if e != nil {
    fmt.Println(e.Error())
  }

  var response string
  if err == nil {
    var prettyJsonResponse bytes.Buffer
    json.Indent(&prettyJsonResponse, resData, "", "  ")
    response = string(prettyJsonResponse.Bytes())
  } else {
    response = "Error: " + err.Error()
  }

  request := string(prettyJsonRequest.Bytes())

  fmt.Println("\n============== REST DEBUGGING ===============\n" + method + " " + url + " " + request + " -> [" + resStatus + "] " + response + "\n\n")
}

func parseStatusCode(statusCode int) (error) {
  switch statusCode {
    case http.StatusOK,
         http.StatusBadRequest,
         http.StatusUnauthorized,
         http.StatusForbidden,
         http.StatusNotFound,
         http.StatusInternalServerError,
         http.StatusServiceUnavailable:
         return nil
  }
  return errors.New(fmt.Sprintf("Unsupported status code: '%d'", statusCode))
}


func UnmarshalResponse(iIndex int, iResponses interface{}) (rStatus int, rOk interface{}, rErr []ErrorResponse) {
  responses := reflect.ValueOf(iResponses)
  for index := 0; index < responses.Len(); index++ {
    response := responses.Index(index)

    i := response.FieldByName("Index").Interface().(int)

    if i == iIndex {
      // found response with given index

      rStatus := response.FieldByName("Status").Interface().(int)
      err    := response.FieldByName("Errors")
      ok     := response.FieldByName("Ok")

      if ok.CanInterface() {
        rOk = ok.Interface()
      }

      if err.CanInterface() {
        rErr = err.Interface().([]ErrorResponse)
      }

      return rStatus, rOk, rErr
    }

  }

  panic("Given index not found")
}
