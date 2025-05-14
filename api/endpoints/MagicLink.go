package endpoints

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/auth0login"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

func PostMagicLinkLoginInfo(params apiRouter.ApiHandlerGenericPublicParams) error {
	body := params.Request.Body
	req := &protos.ReviewerMagicLinkLoginReq{}
	defer body.Close()

	if err := json.NewDecoder(body).Decode(req); err != nil {
		return err
	}

	workspaceId := string(req.MagicLink)
	coll := params.Svcs.MongoDB.Collection(dbCollections.ScreenConfigurationName)
	workspace := &protos.ScreenConfiguration{}
	err := coll.FindOne(params.Request.Context(), bson.M{"_id": workspaceId}).Decode(workspace)
	if err != nil {
		return errors.New("workspace not found")
	}

	reviewerId := workspace.ReviewerId
	if reviewerId == "" {
		return errors.New("no reviewer found for workspace")
	}

	user, err := wsHelpers.GetDBUser(reviewerId, params.Svcs.MongoDB)
	if err != nil {
		return err
	}

	currentTime := time.Now().Unix()
	if user.Info.ExpirationDateUnixSec > 0 && user.Info.ExpirationDateUnixSec < currentTime {
		return errors.New("user access has expired")
	}

	auth0API, err := auth0login.InitAuth0ManagementAPI(params.Svcs.Config)
	if err != nil {
		return errors.New("failed to initialize Auth0 API: " + err.Error())
	}

	auth0User, err := auth0API.User.Read(reviewerId)
	if err != nil {
		return errors.New("failed to fetch Auth0 user: " + err.Error())
	}

	hasReviewerRole := false
	roles, err := auth0API.User.Roles(reviewerId)
	if err != nil {
		return errors.New("failed to fetch roles for user: " + err.Error())
	}

	for _, role := range roles.Roles {
		if *role.Name == "Reviewer" {
			hasReviewerRole = true
			break
		}
	}
	if !hasReviewerRole {
		return errors.New("user does not have the 'Reviewer' role")
	}

	//clientSecret := params.Svcs.Config.Auth0ClientSecret
	clientID := req.ClientId
	redirectUri := req.RedirectURI
	audience := req.Audience
	domain := req.Domain
	scope := "openid profile email"

	jwt, err := auth0login.GetJWT(*auth0User.Email, user.Info.NonSecretPassword, clientID /*clientSecret*/, domain, redirectUri, audience, scope)
	if err != nil {
		return errors.New("failed to get JWT: " + err.Error())
	}

	result := &protos.ReviewerMagicLinkLoginResp{
		UserId:            reviewerId,
		Email:             *auth0User.Email,
		NonSecretPassword: user.Info.NonSecretPassword,
		Token:             jwt,
	}

	utils.SendProtoJSON(params.Writer, result)
	return nil
}
