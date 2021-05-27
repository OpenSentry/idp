package humans

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

func PutTotp(env *app.Environment) gin.HandlerFunc {
	fn := func(c *gin.Context) {

		log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
		log = log.WithFields(logrus.Fields{
			"func": "PutTotp",
		})

		var requests []client.UpdateHumansTotpRequest
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

			requestor := c.MustGet("sub").(string)
			var requestedBy *idp.Identity
			if requestor != "" {
				identities, err := idp.FetchIdentities(tx, []idp.Identity{{Id: requestor}})
				if err != nil {
					bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
					log.Debug(err.Error())
					return
				}
				if len(identities) > 0 {
					requestedBy = &identities[0]
				}
			}

			for _, request := range iRequests {
				r := request.Input.(client.UpdateHumansTotpRequest)

				log = log.WithFields(logrus.Fields{"id": r.Id})

				// Sanity check. Do not allow updating on anything but the access token subject
				if requestedBy.Id != r.Id {
					e := tx.Rollback()
					if e != nil {
						log.Debug(e.Error())
					}
					bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
					request.Output = bulky.NewErrorResponse(request.Index, http.StatusForbidden, E.HUMAN_TOKEN_INVALID)
					return
				}

				dbHumans, err := idp.FetchHumans(tx, []idp.Human{{Identity: idp.Identity{Id: r.Id}}})
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

				encryptedSecret, err := idp.Encrypt(r.TotpSecret, cryptoKey)
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

				updatedHuman, err := idp.UpdateTotp(tx, idp.Human{
					Identity: idp.Identity{
						Id: human.Id,
					},
					TotpRequired: r.TotpRequired,
					TotpSecret:   encryptedSecret,
				})
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

				if updatedHuman != (idp.Human{}) {
					request.Output = bulky.NewOkResponse(request.Index, client.UpdateHumansTotpResponse{
						Id:       updatedHuman.Id,
						Username: updatedHuman.Username,
						//Password: updatedHuman.Password,
						Name:         updatedHuman.Name,
						Email:        updatedHuman.Email,
						AllowLogin:   updatedHuman.AllowLogin,
						TotpRequired: updatedHuman.TotpRequired,
						TotpSecret:   updatedHuman.TotpSecret,
					})
					continue
				}

				// Deny by default
				e := tx.Rollback()
				if e != nil {
					log.Debug(e.Error())
				}
				bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
				request.Output = bulky.NewInternalErrorResponse(request.Index)
				log.Debug("Update totp failed. Hint: Maybe input validation needs to be improved.")
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
