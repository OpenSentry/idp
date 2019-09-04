// OBS: Schema changes cannot be run in same transaction as data queries

CREATE CONSTRAINT ON (i:Identity) ASSERT i.sub IS UNIQUE;
CREATE CONSTRAINT ON (c:Challenge) ASSERT c.otp_challenge IS UNIQUE;
