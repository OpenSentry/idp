package identities

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"

	"github.com/opensentry/idp/app"
	"github.com/opensentry/idp/client"
	"github.com/opensentry/idp/gateway/idp"

	bulky "github.com/charmixer/bulky/server"
)

func GetIdentities(env *app.Environment) gin.HandlerFunc {
	fn := func(c *gin.Context) {

		log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
		log = log.WithFields(logrus.Fields{
			"func": "GetIdentities",
		})

		var requests []client.ReadIdentitiesRequest
		err := c.BindJSON(&requests)
		if err != nil {
			log.Debug(err.Error())
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var handleRequests = func(iRequests []*bulky.Request) {

			session, tx, err := idp.BeginReadTx(env.Driver)
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

				var dbIdentities []idp.Identity
				var err error
				var ok client.ReadIdentitiesResponse

				if request.Input == nil {
					dbIdentities, err = idp.FetchIdentities(tx, nil)
				} else {
					r := request.Input.(client.ReadIdentitiesRequest)
					if r.Id != "" {
						dbIdentities, err = idp.FetchIdentities(tx, []idp.Identity{{Id: r.Id}})
					} else {
						dbIdentities, err = idp.SearchIdentities(tx, r.Search)
					}
				}
				if err != nil {
					log.Debug(err.Error())
					request.Output = bulky.NewInternalErrorResponse(request.Index)
					continue
				}

				if len(dbIdentities) > 0 {
					for _, i := range dbIdentities {
						ok = append(ok, client.Identity{
							Id:     i.Id,
							Labels: strings.Split(i.Labels, ":"),
						})
					}
					request.Output = bulky.NewOkResponse(request.Index, ok)
					continue
				}

				// Deny by default - no results
				request.Output = bulky.NewOkResponse(request.Index, []client.ReadIdentitiesResponse{})
			}

			err = bulky.OutputValidateRequests(iRequests)
			if err == nil {
				tx.Commit()
				return
			}

			// Deny by default
			tx.Rollback()
		}

		responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{EnableEmptyRequest: true})
		c.JSON(http.StatusOK, responses)
	}
	return gin.HandlerFunc(fn)
}
