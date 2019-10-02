package errors

const DEFAULT_LANG = 1
const DEV = 0
const EN = 1

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

const CHALLENGE_NOT_FOUND = 30

const USERNAME_BANNED = 80

const INVITE_NOT_FOUND = 90
const INVITE_NOT_CREATED = 91

const CLIENT_NOT_FOUND = 100

const FOLLOW_NOT_FOUND = 110
const FOLLOW_NOT_CREATED = 111


// keep em' static
var ERRORS = map[int]map[int]string{
  INPUT_VALIDATION_FAILED:
    {
      EN:  "Input validation failed",
      DEV: "Struct validations failed on tags for input",
    },
  EMPTY_REQUEST_NOT_ALLOWED:
    {
      EN:  "Empty request not allowed",
      DEV: "This endpoint does not allow the empty request - each request must be defined separately",
    },
  MAX_REQUESTS_EXCEEDED:
    {
      EN:  "Max number of requests exceeded",
      DEV: "MaxRequest parameter has been set for endpoint and is exceeded by the number of request-objects given in the input",
    },
  FAILED_DUE_TO_OTHER_ERRORS:
    {
      EN:  "Failed due to other errors",
      DEV: "Other request has already been invalidated, no reason to continue until those have been fixed",
    },
  INTERNAL_SERVER_ERROR:
    {
      EN:  "Internal server error occured. Please wait until it has been fixed, before you try again",
      DEV: "Internal server error occured. Please wait until it has been fixed, before you try again",
    },

  HUMAN_NOT_CREATED:
    {
      EN:  "Not created",
      DEV: "Failed to create human. This requires investigation as it should never happen with validation in place.",
    },
  HUMAN_NOT_UPDATED:
    {
      EN:  "Not updated",
      DEV: "Failed to update human. This requires investigation as it should never happen with validation in place.",
    },
  HUMAN_NOT_FOUND:
    {
      EN:  "Not found",
      DEV: "Human not found",
    },
  IDENTITY_NOT_FOUND:
    {
      EN:  "Not found",
      DEV: "Identity not found",
    },
  USERNAME_BANNED:
    {
      EN:  "Username is banned",
      DEV: "Username is banned according to the banlist",
    },
  CHALLENGE_NOT_FOUND:
    {
      EN:  "Not found",
      DEV: "Challenge not found",
    },
  HUMAN_TOTP_NOT_REQUIRED:
    {
      EN:  "TOTP not required",
      DEV: "TOTP not required",
    },
  INVITE_NOT_FOUND:
    {
      EN:  "Not found",
      DEV: "Invite not found",
    },
  INVITE_NOT_CREATED:
    {
      EN:  "Not created",
      DEV: "Failed to create invite. This requires investigation as it should never happen with validation in place.",
    },
  CLIENT_NOT_FOUND:
    {
      EN:  "Not found",
      DEV: "Client not found",
    },
  FOLLOW_NOT_FOUND:
    {
      EN:  "Not found",
      DEV: "Follow not found",
    },
  FOLLOW_NOT_CREATED:
    {
      EN:  "Not created",
      DEV: "Failed to create follow. This requires investigation as it should never happen with validation in place.",
    },

}

// E[INPUT_VALIDATION_FAILED][EN]
// client.E[client.INPUT_VALIDATION_FAILED][client.EN]
