package humans

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"time"

	"github.com/opensentry/idp/app"
	"github.com/opensentry/idp/client"
	E "github.com/opensentry/idp/client/errors"
	"github.com/opensentry/idp/config"
	"github.com/opensentry/idp/gateway/idp"

	bulky "github.com/charmixer/bulky/server"
)

type RecoverTemplateData struct {
	Name             string
	VerificationCode string
	Sender           string
}

func PostRecover(env *app.Environment) gin.HandlerFunc {
	fn := func(c *gin.Context) {

		log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
		log = log.WithFields(logrus.Fields{
			"func": "PostRecover",
		})

		var requests []client.CreateHumansRecoverRequest
		err := c.BindJSON(&requests)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		controllerConfirm := config.GetString("idpui.public.url") + config.GetString("idpui.public.endpoints.recoverconfirm")
		redirectToConfirm, err := url.Parse(controllerConfirm)
		if err != nil {
			log.WithFields(logrus.Fields{"url": controllerConfirm}).Debug(err.Error())
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		var sender idp.SMTPSender = idp.SMTPSender{Name: config.GetString("provider.name"), Email: config.GetString("provider.email")}
		var templateFile string = config.GetString("templates.recover.email.templatefile")
		var emailSubject string = config.GetString("templates.recover.email.subject")

		smtpConfig := idp.SMTPConfig{
			Host:          config.GetString("mail.smtp.host"),
			Username:      config.GetString("mail.smtp.user"),
			Password:      config.GetString("mail.smtp.password"),
			Sender:        sender,
			SkipTlsVerify: config.GetInt("mail.smtp.skip_tls_verify"),
		}

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
			//   identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
			//   if err != nil {
			//     bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
			//     log.Debug(err.Error())
			//     return
			//   }
			//   if len(identities) > 0 {
			//     requestedBy = &identities[0]
			//   }
			// }

			for _, request := range iRequests {
				r := request.Input.(client.CreateHumansRecoverRequest)

				log = log.WithFields(logrus.Fields{"id": r.Id})

				// // Sanity check. Do not allow updating on anything but the access token subject
				// if requestedBy.Id != r.Id {
				//   e := tx.Rollback()
				//   if e != nil {
				//     log.Debug(e.Error())
				//   }
				//   bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
				//   request.Output = bulky.NewErrorResponse(request.Index, http.StatusForbidden, E.HUMAN_TOKEN_INVALID)
				//   return
				// }

				dbHumans, err := idp.FetchHumans(tx, []idp.Human{{Identity: idp.Identity{Id: r.Id}}})
				if err != nil {
					e := tx.Rollback()
					if e != nil {
						log.Debug(e.Error())
					}
					bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
					request.Output = bulky.NewInternalErrorResponse(request.Index)
					log.Debug(err.Error())
					return
				}

				if len(dbHumans) <= 0 {
					e := tx.Rollback()
					if e != nil {
						log.Debug(e.Error())
					}
					bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
					request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
					return
				}
				human := dbHumans[0]

				if human != (idp.Human{}) {

					// Require email confirmation challenge

					newChallenge := idp.Challenge{
						JwtRegisteredClaims: idp.JwtRegisteredClaims{
							Subject:   human.Id,
							Issuer:    config.GetString("idp.public.issuer"),
							Audience:  config.GetString("idp.public.url") + config.GetString("idp.public.endpoints.challenges.verify"),
							ExpiresAt: time.Now().Unix() + 900, // 15 min,  FIXME: Should be configurable
						},
						RedirectTo: r.RedirectTo, // Requested success url redirect.
						CodeType:   int64(client.OTP),
					}
					challenge, otpCode, err := idp.CreateChallengeUsingOtp(tx, idp.ChallengeRecover, newChallenge)
					if err != nil {
						e := tx.Rollback()
						if e != nil {
							log.Debug(e.Error())
						}
						bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
						request.Output = bulky.NewInternalErrorResponse(request.Index)     // Specify error on failed one
						log.Debug(err.Error())
						return
					}

					if challenge != (idp.Challenge{}) {

						if otpCode.Code != "" && human.Email != "" {

							var data = ConfirmTemplateData{
								Challenge: challenge.Id,
								Sender:    sender.Name,
								Id:        challenge.Subject,
								Email:     human.Email,
								Code:      otpCode.Code, // Note this is the clear text generated code and not the hashed one stored in DB.
							}
							_, err = idp.SendEmailUsingTemplate(smtpConfig, human.Email, human.Email, emailSubject, templateFile, data)
							if err != nil {
								e := tx.Rollback()
								if e != nil {
									log.Debug(e.Error())
								}
								bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
								request.Output = bulky.NewInternalErrorResponse(request.Index)     // Specify error on failed one
								log.Debug(err.Error())
								return
							}

						}

						q := redirectToConfirm.Query()
						q.Add("recover_challenge", challenge.Id)
						redirectToConfirm.RawQuery = q.Encode()

						request.Output = bulky.NewOkResponse(request.Index, client.CreateHumansRecoverResponse{
							Id:         human.Id,
							RedirectTo: redirectToConfirm.String(),
						})
						continue
					}

				}

				// Deny by default
				e := tx.Rollback()
				if e != nil {
					log.Debug(e.Error())
				}
				bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
				request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
				log.Debug("Recover human failed. Hint: Maybe input validation needs to be improved.")
				return
			}

			err = bulky.OutputValidateRequests(iRequests)
			if err == nil {
				tx.Commit()
				return
			}

			// Deny by default
			tx.Rollback()
		}

		responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
		c.JSON(http.StatusOK, responses)
	}
	return gin.HandlerFunc(fn)
}
