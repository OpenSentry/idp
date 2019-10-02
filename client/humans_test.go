package client

import (
  "testing"
  "net/http"
  "net/url"
  "crypto/tls"
  "golang.org/x/net/context"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/clientcredentials"
  oidc "github.com/coreos/go-oidc"
  "fmt"
)

func newIdpTestClient() (*IdpClient, error)  {
  tr := &http.Transport{ TLSClientConfig: &tls.Config{InsecureSkipVerify: true} }
  c := &http.Client{Transport: tr}
  ctx := context.WithValue(context.Background(), oauth2.HTTPClient, c)

  provider, err := oidc.NewProvider(ctx, "https://oauth.localhost/")
  if err != nil {
    return nil, err
  }
  endpoint := provider.Endpoint()
  endpoint.AuthStyle = 2 // Force basic secret, so token exchange does not auto to post which we did not allow.

  config := clientcredentials.Config{
    ClientID:  "idpui",
    ClientSecret: "Z61hRd4e6PSosJk+vfiIOgrYvpw5eLBIg+VqFWqN5WOvpGfUnexqQOmh0AfwM8KCMGG90Oqln45NpkMBBSINCw==",
    TokenURL: provider.Endpoint().TokenURL,
    Scopes: []string{"read:identity"},
    EndpointParams: url.Values{"audience": {"idp"}},
    AuthStyle: 2, // https://godoc.org/golang.org/x/oauth2#AuthStyle
  }

  client := config.Client(ctx)
  return &IdpClient{client}, nil
}

func TestReadHumansWithEmptyRequest(t *testing.T) {
  const wantStatus = 200

  client, err := newIdpTestClient()
  if err != nil {
    t.Errorf(err.Error())
    return
  }
  
  gotStatus, res, err := ReadHumans(client, "https://id.localhost/api/humans", nil)
  if err != nil {
    t.Errorf("http request got error = %s", err.Error())
  } else {

    if gotStatus == wantStatus {

      idx := 0
      reqStatus, reqObj, reqErrors := UnmarshalResponse(idx, res)
      if reqStatus == wantStatus {

        if reqObj == nil {
          t.Errorf("request@idx(%d) got no obj", idx)
        } else if len(reqErrors) > 0 {
          t.Errorf("request@idx(%d) got errors", idx)
          fmt.Println(reqErrors)
        } else {

          h := reqObj.([]Human)
          human := h[0]

          fmt.Println(human)

        }

      } else {
        t.Errorf("request@%d got status = %d; want %d", idx, reqStatus, wantStatus)
      }

    } else {
      t.Errorf("http request got status = %d; want %d", gotStatus, wantStatus)
    }

  }

}
/*
func TestReadHumans(t *testing.T) {

  const wantStatus = 200

  client, err := newIdpTestClient()
  if err != nil {
    t.Errorf(err.Error())
    return
  }

  //req := []ReadHumansRequest{ {Id: "9574710c-d661-4dbb-b100-623c3b656a52"} }
  req := []ReadHumansRequest{ {Id: "asdasdasdasd"} }
  //req := []ReadHumansRequest{ {Id: ""} }
  //req := []ReadHumansRequest{}
  gotStatus, res, err := ReadHumans(client, "https://id.localhost/api/humans", req)
  if err != nil {
    t.Errorf("http request got error = %s", err.Error())
  } else {

    if gotStatus == wantStatus {

      idx := 0
      reqStatus, reqObj, reqErrors := UnmarshalResponse(idx, res)
      if reqStatus == wantStatus {

        if reqObj == nil {
          t.Errorf("request@idx(%d) got no obj", idx)
        } else if len(reqErrors) > 0 {
          t.Errorf("request@idx(%d) got errors", idx)
          fmt.Println(reqErrors)
        } else {

          h := reqObj.([]Human)
          human := h[0]

          fmt.Println(human)

        }

      } else {
        t.Errorf("request@%d got status = %d; want %d", idx, reqStatus, wantStatus)
      }

    } else {
      t.Errorf("http request got status = %d; want %d", gotStatus, wantStatus)
    }

  }

}*/