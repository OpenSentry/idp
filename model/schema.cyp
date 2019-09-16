// OBS: Schema changes cannot be run in same transaction as data queries

CREATE CONSTRAINT ON (i:Identity) ASSERT i.id IS UNIQUE;
CREATE CONSTRAINT ON (i:Identity) ASSERT i.username IS UNIQUE;

CREATE CONSTRAINT ON (c:Client) ASSERT c.client_id IS UNIQUE;
CREATE CONSTRAINT ON (a:ResourceServer) ASSERT a.aud IS UNIQUE;
CREATE CONSTRAINT ON (i:Human) ASSERT i.email IS UNIQUE;

CREATE CONSTRAINT ON (c:Challenge) ASSERT c.otp_challenge IS UNIQUE;
CREATE CONSTRAINT ON (i:Invite) ASSERT i.id IS UNIQUE;
