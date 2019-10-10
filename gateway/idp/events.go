package idp

import (
  nats "github.com/nats-io/nats.go"
  "fmt"
)

func EmitEventHumanCreated(natsConnection *nats.Conn, human Human) {
  e := fmt.Sprintf("{id:%s, name:%s, email:%s}", human.Id, human.Name, human.Email)
  natsConnection.Publish("idp.human.created", []byte(e))
}

func EmitEventIdentityAuthenticated(natsConnection *nats.Conn, i Identity, acr string) {
  e := fmt.Sprintf("{id:%s, acr:%s}", i.Id, acr)
  natsConnection.Publish("idp.identity.authenticated", []byte(e))
}

func EmitEventClientCreated(natsConnection *nats.Conn, client Client) {
  e := fmt.Sprintf("{id:%s, name:%s}", client.Id, client.Name)
  natsConnection.Publish("idp.client.created", []byte(e))
}