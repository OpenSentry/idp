package idp

import (
  nats "github.com/nats-io/nats.go"
  "fmt"
)

func EmitEventHumanCreated(natsConnection *nats.Conn, human Human) {
  e := fmt.Sprintf("{id:%s, name:%s, email:%s}", human.Id, human.Name, human.Email)
  natsConnection.Publish("idp", []byte(e))
}