begin;

  create table identity (
    id        uuid primary key not null
  );

  create table human (
    subject               uuid references identity(id),
    name                  varchar null,
    given_name            varchar null,
    family_name           varchar null,
    last_name             varchar null,
    middle_name           varchar null,
    nickname              varchar null,
    preferred_username    varchar null,
    profile               varchar null,
    picture               varchar null,
    website               varchar null,
    email                 varchar not null,
    email_verified        boolean not null,
    gender                varchar null,
    birthdate             varchar null,
    zoneinfo              varchar null,
    locale                varchar null,
    phone_number          varchar null,
    phone_number_verified boolean null,
    updated_at            timestamp not null,
    primary key(subject)
  );

  create table client (
    client_id                  uuid references identity(id),
    name                       varchar not null,
    description                text not null,
    is_public                  boolean not null,
    secret                     varchar null,
    grant_types                varchar[] null,
    response_types             varchar[] null,
    redirect_uris              varchar[] null,
    token_endpoint_auth_method varchar null,
    post_logout_redirect_uris  varchar null,
    primary key(client_id)
  );

  create table resource_provider (
    resource_provider_id uuid references identity(id),
    name                 varchar not null,
    description          text not null,
    audience             varchar not null,
    primary key(resource_provider_id)
  );

  create table provider (
    id serial,
    name varchar,
    password boolean,
    primary key(id)
  );

  create table secret (
    id serial,
    hash varchar,
    recoverable boolean,
    end_dtm timestamp null,
    primary key(id)
  );

  create table crediential (
    identity_id          uuid references identity(id),
    provider_id          integer references provider(id),
    secret_id            integer references secret(id)
  );

commit;
