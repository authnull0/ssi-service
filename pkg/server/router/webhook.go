package router

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/tbd54566975/ssi-service/pkg/server/framework"
	svcframework "github.com/tbd54566975/ssi-service/pkg/service/framework"
	"github.com/tbd54566975/ssi-service/pkg/service/webhook"
)

type WebhookRouter struct {
	service *webhook.Service
}

func NewWebhookRouter(s svcframework.Service) (*WebhookRouter, error) {
	if s == nil {
		return nil, errors.New("service cannot be nil")
	}
	webhookService, ok := s.(*webhook.Service)
	if !ok {
		return nil, fmt.Errorf("could not create webhook router with service type: %s", s.Type())
	}
	return &WebhookRouter{service: webhookService}, nil
}

// CreateWebhookRequest In the context of webhooks, it's common to use noun.verb notation to describe events,
// such as "credential.create" or "schema.delete".
type CreateWebhookRequest struct {
	// The noun (entity) for the new webhook.eg: Credential
	Noun webhook.Noun `json:"noun" validate:"required"`
	// The verb for the new webhook.eg: Create
	Verb webhook.Verb `json:"verb" validate:"required"`
	// The URLS to post the output of this request to Noun.Verb action to.
	URLS []string `json:"urls" validate:"required"`
}

type CreateWebhookResponse struct {
	ID      string          `json:"id"`
	Webhook webhook.Webhook `json:"webhook"`
}

// CreateWebhook godoc
//
// @Summary     Create Webhook
// @Description Create webhook
// @Tags        WebhookAPI
// @Accept      json
// @Produce     json
// @Param       request body     CreateWebhookRequest true "request body"
// @Success     201     {object} CreateWebhookResponse
// @Failure     400     {string} string "Bad request"
// @Failure     500     {string} string "Internal server error"
// @Router      /v1/webhooks [put]
func (wr WebhookRouter) CreateWebhook(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var request CreateWebhookRequest
	invalidCreateWebhookRequest := "invalid create webhook request"
	if err := framework.Decode(r, &request); err != nil {
		logrus.WithError(err).Error(invalidCreateWebhookRequest)
		return framework.NewRequestError(errors.Wrap(err, invalidCreateWebhookRequest), http.StatusBadRequest)
	}

	if err := framework.ValidateRequest(request); err != nil {
		errMsg := invalidCreateWebhookRequest
		logrus.WithError(err).Error(errMsg)
		return framework.NewRequestError(errors.Wrap(err, errMsg), http.StatusBadRequest)
	}

	req := webhook.CreateWebhookRequest{Noun: request.Noun, Verb: request.Verb, URLS: request.URLS}
	createWebhookResponse, err := wr.service.CreateWebhook(ctx, req)
	if err != nil {
		errMsg := "could not create webhook"
		logrus.WithError(err).Error(errMsg)
		return framework.NewRequestError(errors.Wrap(err, errMsg), http.StatusInternalServerError)
	}

	resp := CreateWebhookResponse{ID: createWebhookResponse.ID, Webhook: createWebhookResponse.Webhook}
	return framework.Respond(ctx, w, resp, http.StatusCreated)
}

type GetWebhookResponse struct {
	ID      string          `json:"id"`
	Webhook webhook.Webhook `json:"webhook"`
}

// GetWebhook godoc
//
// @Summary     Get Webhook
// @Description Get a webhook by its ID
// @Tags        WebhookAPI
// @Accept      json
// @Produce     json
// @Param       id  path     string true "ID"
// @Success     200 {object} GetWebhookResponse
// @Failure     400 {string} string "Bad request"
// @Router      /v1/webhooks/{id} [get]
func (wr WebhookRouter) GetWebhook(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id := framework.GetParam(ctx, IDParam)
	if id == nil {
		errMsg := "cannot get webhook without ID parameter"
		logrus.Error(errMsg)
		return framework.NewRequestErrorMsg(errMsg, http.StatusBadRequest)
	}

	gotWebhook, err := wr.service.GetWebhook(ctx, webhook.GetWebhookRequest{ID: *id})
	if err != nil {
		errMsg := fmt.Sprintf("could not get webhook with id: %s", *id)
		logrus.WithError(err).Error(errMsg)
		return framework.NewRequestError(errors.Wrap(err, errMsg), http.StatusInternalServerError)
	}

	resp := GetWebhookResponse{Webhook: gotWebhook.Webhook}
	return framework.Respond(ctx, w, resp, http.StatusOK)
}

type GetWebhooksResponse struct {
	Webhooks []GetWebhookResponse `json:"webhooks,omitempty"`
}

// GetWebhooks godoc
//
// @Summary     Get Webhooks
// @Description Get webhooks
// @Tags        WebhookAPI
// @Accept      json
// @Produce     json
// @Success     200 {object} GetWebhooksResponse
// @Failure     500 {string} string "Internal server error"
// @Router      /v1/webhooks [get]
func (wr WebhookRouter) GetWebhooks(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	gotWebhooks, err := wr.service.GetWebhooks(ctx)
	if err != nil {
		errMsg := "could not get webhooks"
		logrus.WithError(err).Error(errMsg)
		return framework.NewRequestError(errors.Wrap(err, errMsg), http.StatusInternalServerError)
	}

	webhooks := make([]GetWebhookResponse, 0, len(gotWebhooks.Webhooks))
	for _, w := range gotWebhooks.Webhooks {
		webhooks = append(webhooks, GetWebhookResponse{Webhook: w})
	}

	resp := GetWebhooksResponse{Webhooks: webhooks}
	return framework.Respond(ctx, w, resp, http.StatusOK)
}

// DeleteWebhook godoc
//
// @Summary     Delete Webhook
// @Description Delete a webhook by its ID
// @Tags        WebhookAPI
// @Accept      json
// @Produce     json
// @Param       id  path     string true "ID"
// @Success     200 {string} string "OK"
// @Failure     400 {string} string "Bad request"
// @Failure     500 {string} string "Internal server error"
// @Router      /v1/webhooks/{id} [delete]
func (wr WebhookRouter) DeleteWebhook(ctx context.Context, w http.ResponseWriter, _ *http.Request) error {
	id := framework.GetParam(ctx, IDParam)
	if id == nil {
		errMsg := "cannot delete a webhook without an ID parameter"
		logrus.Error(errMsg)
		return framework.NewRequestErrorMsg(errMsg, http.StatusBadRequest)
	}

	if err := wr.service.DeleteWebhook(ctx, webhook.DeleteWebhookRequest{ID: *id}); err != nil {
		errMsg := fmt.Sprintf("could not delete webhook with id: %s", *id)
		logrus.WithError(err).Error(errMsg)
		return framework.NewRequestError(errors.Wrap(err, errMsg), http.StatusInternalServerError)
	}

	return framework.Respond(ctx, w, nil, http.StatusOK)
}

// GetSupportedNouns godoc
//
// @Summary     Get Supported Nouns
// @Description Get supported nouns for webhook generation
// @Tags        WebhookAPI
// @Accept      json
// @Produce     json
// @Success     200 {object} GetSupportedNounsResponse
// @Failure     500 {string} string "Internal server error"
// @Router      /v1/webhooks/nouns [get]
func (wr WebhookRouter) GetSupportedNouns(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	nouns := wr.service.GetSupportedNouns()
	return framework.Respond(ctx, w, nouns, http.StatusOK)
}

// GetSupportedVerbs godoc
//
// @Summary     Get Supported Verbs
// @Description Get supported verbs for webhook generation
// @Tags        WebhookAPI
// @Accept      json
// @Produce     json
// @Success     200 {object} GetSupportedVerbsResponse
// @Failure     500 {string} string "Internal server error"
// @Router      /v1/webhooks/verbs [get]
func (wr WebhookRouter) GetSupportedVerbs(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	verbs := wr.service.GetSupportedVerbs()
	return framework.Respond(ctx, w, verbs, http.StatusOK)
}
