package services

import (
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/rs/zerolog"
	"github.com/vapusdata-ecosystem/vapusai/aigateway/pkgs"
	aicore "github.com/vapusdata-ecosystem/vapusai/core/aistudio/core"
	aidmstore "github.com/vapusdata-ecosystem/vapusai/core/app/datarepo/aistudio"
	apperr "github.com/vapusdata-ecosystem/vapusai/core/app/errors"
	dmerrors "github.com/vapusdata-ecosystem/vapusai/core/pkgs/errors"
	"github.com/vapusdata-ecosystem/vapusai/core/types"
)

type AgentReverseProxyServices struct {
	dmstores  *aidmstore.AIStudioDMStore
	logger    zerolog.Logger
	validator *validator.Validate
}

var AgentReverseProxyManager *AgentReverseProxyServices

func NewAgentReverseProxyServices(dmstores *aidmstore.AIStudioDMStore) *AgentReverseProxyServices {
	AgentReverseProxyManager = &AgentReverseProxyServices{
		dmstores:  dmstores,
		logger:    pkgs.DmLogger,
		validator: validator.New(),
	}

	return AgentReverseProxyManager
}

func (p *AgentReverseProxyServices) Do(c *fiber.Ctx) error {
	p.logger.Info().Msg("Proxy called")

	nodeKey := c.Request().Header.Peek(types.AIMODEL_HEADER_KEY)

	if len(nodeKey) == 0 {
		p.logger.Error().Msg("Node key not found in the request")
		return SendAIGatewayError(c, fiber.StatusBadRequest, &aicore.AiGatewayError{
			Error: aicore.AiGatewayErrorDetail{
				Code:    strconv.Itoa(fiber.StatusBadRequest),
				Message: "Model Node key not found in the request",
				Param:   types.AIMODEL_HEADER_KEY,
			},
		})
	}

	p.logger.Info().Msgf("Node key: %s", nodeKey)

	model, err := pkgs.AIModelNodeConnectionPoolManager.GetorSetNodeObject(string(nodeKey), nil, true)

	if err != nil {
		p.logger.Error().Err(err).Msg("error while getting model node")
		return dmerrors.DMError(apperr.ErrAIModelNode404, nil)
	}

	formattedUrl := strings.Replace(c.OriginalURL(), "/agent/v1", "", 1)
	targetURL := model.NetworkParams.Url + formattedUrl

	c.Request().Header.Del("Authorization")
	c.Request().Header.Del("x-aimodelnode")

	conn := pkgs.AIModelNodeConnectionPoolManager.GetConnectionById(string(nodeKey))
	headers := conn.BuildReverseProxyHeaders()

	for key, value := range headers {
		c.Request().Header.Set(key, value)
	}

	return proxy.Do(c, targetURL)
}
