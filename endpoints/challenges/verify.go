package challenges

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"

	"github.com/opensentry/idp/app"
	"github.com/opensentry/idp/client"
	E "github.com/opensentry/idp/client/errors"
	"github.com/opensentry/idp/config"
	"github.com/opensentry/idp/gateway/idp"

	bulky "github.com/charmixer/bulky/server"
)

func PutVerify(env *app.Environment) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
		log = log.WithFields(logrus.Fields{
			"func": "PutVerify",
		})

		var requests []client.UpdateChallengesVerifyRequest
		err := c.BindJSON(&requests)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		keys := config.GetStringSlice("crypto.keys.totp")
		if len(keys) <= 0 {
			log.WithFields(logrus.Fields{"key": "crypto.keys.totp"}).Debug("Missing config")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		cryptoKey := keys[0]

		var handleRequests = func(iRequests []*bulky.Request) {

			session, tx, err := idp.BeginWriteTx(env.Driver)
			if err != nil {
				bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
				log.Debug(err.Error())
				return
			}
			defer tx.Close() // rolls back if not already committed/rolled back
			defer session.Close()

			// requestor := c.MustGet("sub").(string)
			// var requestedBy *idp.Identity
			// if requestor != "" {
			//  identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
			//  if err != nil {
			//    bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
			//    log.Debug(err.Error())
			//    return
			//  }
			//  if len(identities) > 0 {
			//    requestedBy = &identities[0]
			//  }
			// }

			for _, request := range iRequests {
				r := request.Input.(client.UpdateChallengesVerifyRequest)

				log = log.WithFields(logrus.Fields{"otp_challenge": r.OtpChallenge})

				// Sanity check. Challenge must exists
				var aChallenge []idp.Challenge
				aChallenge = append(aChallenge, idp.Challenge{Id: r.OtpChallenge})
				dbChallenges, err := idp.FetchChallenges(tx, aChallenge)
				if err != nil {
					e := tx.Rollback()
					if e != nil {
						log.Debug(e.Error())
					}

					bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
					request.Output = bulky.NewInternalErrorResponse(request.Index)     // Specify error on failed one
					log.WithFields(logrus.Fields{"otp_challenge": r.OtpChallenge}).Debug(err.Error())
					return
				}

				cnt := len(dbChallenges)
				if cnt <= 0 {
					e := tx.Rollback()
					if e != nil {
						log.Debug(e.Error())
					}

					bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
					request.Output = bulky.NewClientErrorResponse(request.Index, E.CHALLENGE_NOT_FOUND)
					return
				}

				var challenge idp.Challenge = dbChallenges[0]
				var valid bool = false

				if client.OTPType(challenge.CodeType) == client.TOTP {

					humans, err := idp.FetchHumans(tx, []idp.Human{{Identity: idp.Identity{Id: challenge.Subject}}})
					if err != nil {
						e := tx.Rollback()
						if e != nil {
							log.Debug(e.Error())
						}
						bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
						request.Output = bulky.NewInternalErrorResponse(request.Index)     // Specify error on failed one
						log.WithFields(logrus.Fields{"otp_challenge": challenge.Id, "id": challenge.Subject}).Debug(err.Error())
						return
					}

					cnt := len(humans)
					if cnt <= 0 {
						e := tx.Rollback()
						if e != nil {
							log.Debug(e.Error())
						}
						bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
						request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
						return
					}
					var human idp.Human = humans[0]

					if human.TotpRequired != true {
						e := tx.Rollback()
						if e != nil {
							log.Debug(e.Error())
						}
						bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
						request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_TOTP_NOT_REQUIRED)
						return
					}

					decryptedSecret, err := idp.Decrypt(human.TotpSecret, cryptoKey)
					if err != nil {
						e := tx.Rollback()
						if e != nil {
							log.Debug(e.Error())
						}
						bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
						request.Output = bulky.NewInternalErrorResponse(request.Index)     // Specify error on failed one
						log.WithFields(logrus.Fields{"otp_challenge": challenge.Id, "id": human.Id}).Debug(err.Error())
						return
					}

					valid, _ = idp.ValidateOtp(r.Code, decryptedSecret)

				} else {

					valid, _ = idp.ValidatePassword(challenge.Code, r.Code)

				}

				if valid == true {

					verifiedChallenge, err := idp.VerifyChallenge(tx, challenge)
					if err != nil {
						e := tx.Rollback()
						if e != nil {
							log.Debug(e.Error())
						}
						bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
						request.Output = bulky.NewInternalErrorResponse(request.Index)     // Specify error on failed one
						log.WithFields(logrus.Fields{"otp_challenge": challenge.Id}).Debug(err.Error())
						return
					}

					request.Output = bulky.NewOkResponse(request.Index, client.UpdateChallengesVerifyResponse{
						OtpChallenge: verifiedChallenge.Id,
						Verified:     true,
						RedirectTo:   verifiedChallenge.RedirectTo,
					})
					continue
				}

				// Deny by default
				request.Output = bulky.NewOkResponse(request.Index, client.UpdateChallengesVerifyResponse{
					OtpChallenge: r.OtpChallenge,
					Verified:     false,
					RedirectTo:   "",
				})
			}

			err = bulky.OutputValidateRequests(iRequests)
			if err == nil {
				tx.Commit()
				return
			}
			tx.Rollback() // deny by default
		}

		responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
		c.JSON(http.StatusOK, responses)
	}
	return gin.HandlerFunc(fn)
}
