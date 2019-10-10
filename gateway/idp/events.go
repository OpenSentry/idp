package idp

import (
  nats "github.com/nats-io/nats.go"
  "fmt"
)

func EmitEventHumanCreated(natsConnection *nats.Conn, human Human) {
  e := fmt.Sprintf("{id:%s, name:%s, email:%s}", human.Id, human.Name, human.Email)
  natsConnection.Publish("idp.human.created", []byte(e))
}

func EmitEventHumanAuthenticated(natsConnection *nats.Conn, human Human) {
  e := fmt.Sprintf("{id:%s}", human.Id)
  natsConnection.Publish("idp.human.authenticated", []byte(e))
}

func EmitEventClientCreated(natsConnection *nats.Conn, client Client) {
  e := fmt.Sprintf("{id:%s, name:%s}", client.Id, client.Name)
  natsConnection.Publish("idp.client.created", []byte(e))
}