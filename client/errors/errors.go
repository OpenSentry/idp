package errors

import (
  bulky "github.com/charmixer/bulky/errors"
)

const INPUT_VALIDATION_FAILED    = 1
const EMPTY_REQUEST_NOT_ALLOWED  = 2
const MAX_REQUESTS_EXCEEDED      = 3
const FAILED_DUE_TO_OTHER_ERRORS = 4
const INTERNAL_SERVER_ERROR      = 5

const IDENTITY_NOT_FOUND = 10

const HUMAN_NOT_FOUND = 20
const HUMAN_NOT_CREATED = 21
const HUMAN_NOT_UPDATED = 22
const HUMAN_TOTP_NOT_REQUIRED = 23
const HUMAN_ALREADY_EXISTS = 24

const CLIENT_NOT_FOUND = 50
const CLIENT_NOT_CREATED = 51

const CHALLENGE_NOT_FOUND = 30
const CHALLENGE_NOT_CREATED = 31

const USERNAME_BANNED = 80
const USERNAME_EXISTS = 81

const INVITE_NOT_FOUND = 90
const INVITE_NOT_CREATED = 91
const INVITE_EXPIRES_IN_THE_PAST = 92

const FOLLOW_NOT_FOUND = 110
const FOLLOW_NOT_CREATED = 111

func InitRestErrors() {
  bulky.AppendErrors(
    map[int]map[string]string{
      HUMAN_ALREADY_EXISTS:
        {
          "en":  "Already exists",
          "dev": "Human already exists",
        },
      HUMAN_NOT_CREATED:
        {
          "en":  "Not created",
          "dev": "Failed to create human. This requires investigation as it should never happen with validation in place.",
        },
      HUMAN_NOT_UPDATED:
        {
          "en":  "Not updated",
          "dev": "Failed to update human. This requires investigation as it should never happen with validation in place.",
        },
      HUMAN_NOT_FOUND:
        {
          "en":  "Not found",
          "dev": "Human not found",
        },
      IDENTITY_NOT_FOUND:
        {
          "en":  "Not found",
          "dev": "Identity not found",
        },
      USERNAME_BANNED:
        {
          "en":  "Username is banned",
          "dev": "Username is banned according to the banlist",
        },
      USERNAME_EXISTS:
        {
          "en":  "Username exists",
          "dev": "Username already exists",
        },
      CHALLENGE_NOT_CREATED:
        {
          "en":  "Not created",
          "dev": "Failed to create challenge. This requires investigation as it should never happen with validation in place.",
        },
      CHALLENGE_NOT_FOUND:
        {
          "en": "Not found",
          "dev": "Challenge not found",
        },
      HUMAN_TOTP_NOT_REQUIRED:
        {
          "en":  "TOTP not required",
          "dev": "TOTP not required",
        },

      CLIENT_NOT_FOUND:
        {
          "en":  "Not found",
          "dev": "Client not found",
        },
      CLIENT_NOT_CREATED:
        {
          "en":  "Not created",
          "dev": "Failed to create client. This requires investigation as it should never happen with validation in place.",
        },

      INVITE_NOT_FOUND:
        {
          "en":  "Not found",
          "dev": "Invite not found",
        },
      INVITE_NOT_CREATED:
        {
          "en":  "Not created",
          "dev": "Failed to create invite. This requires investigation as it should never happen with validation in place.",
        },
      INVITE_EXPIRES_IN_THE_PAST:
        {
          "en":  "Expires in the past",
          "dev": "Expires in the past. Hint: exp field expires in the past.",
        },

      FOLLOW_NOT_FOUND:
        {
          "en":  "Not found",
          "dev": "Follow not found",
        },
      FOLLOW_NOT_CREATED:
        {
          "en":  "Not created",
          "dev": "Failed to create follow. This requires investigation as it should never happen with validation in place.",
        },
    },
  )
}
