// OpenId Subject

CREATE CONSTRAINT ON (i:Identity) ASSERT i.sub IS UNIQUE;
