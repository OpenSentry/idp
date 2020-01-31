package idp

import (
  nats "github.com/nats-io/nats.go"
  "fmt"
)

func EmitEventHumanCreated(natsConnection *nats.Conn, human Human) {
  e := fmt.Sprintf("{\"id\":\"%s\"}", human.Id)
  natsConnection.Publish("idp.human.created", []byte(e))
}

func EmitEventIdentityAuthenticated(natsConnection *nats.Conn, i Identity, acr string) {
  e := fmt.Sprintf("{\"id\":\"%s\", \"acr\":\"%s\"}", i.Id, acr)
  natsConnection.Publish("idp.identity.authenticated", []byte(e))
}

func EmitEventHumanPasswordChanged(natsConnection *nats.Conn, human Human) {
  e := fmt.Sprintf("{\"id\":\"%s\"}", human.Id)
  natsConnection.Publish("idp.human.password.changed", []byte(e))
}

func EmitEventHumanEmailChanged(natsConnection *nats.Conn, human Human) {
  e := fmt.Sprintf("{\"id\":\"%s\"}", human.Id)
  natsConnection.Publish("idp.human.email.changed", []byte(e))
}

func EmitEventClientCreated(natsConnection *nats.Conn, client Client) {
  e := fmt.Sprintf("{\"id\":\"%s\"}", client.Id)
  natsConnection.Publish("idp.client.created", []byte(e))
}

func EmitEventClientUpdated(natsConnection *nats.Conn, client Client) {
  e := fmt.Sprintf("{\"id\":\"%s\"}", client.Id)
  natsConnection.Publish("idp.client.updated", []byte(e))
}

func EmitEventResourceServerCreated(natsConnection *nats.Conn, resourceServer ResourceServer) {
  e := fmt.Sprintf("{\"id\":\"%s\"}", resourceServer.Id)
  natsConnection.Publish("idp.resourceserver.created", []byte(e))
}

func EmitEventInviteCreated(natsConnection *nats.Conn, invite Invite) {
  e := fmt.Sprintf("{\"id\":\"%s\"}", invite.Id)
  natsConnection.Publish("idp.invite.created", []byte(e))
}

func EmitEventInviteSent(natsConnection *nats.Conn, invite Invite) {
  e := fmt.Sprintf("{\"id\":\"%s\"}", invite.Id)
  natsConnection.Publish("idp.invite.sent", []byte(e))
}
