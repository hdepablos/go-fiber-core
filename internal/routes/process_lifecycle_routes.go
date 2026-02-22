package routes

import (
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/handlers"
	"go-fiber-core/internal/utils"

	fiber "github.com/gofiber/fiber/v2"
)

func RegisterProcessLifecycleRoutes(router fiber.Router, handler handlers.ProcessLifecycleHandler) {
	group := router.Group("/process-lifecycle")

	group.Post(
		"/replicate",
		utils.Validate(new(requests.ReplicateScenarioRequest)),
		handler.ReplicateScenario,
	)

	group.Post(
		"/promote",
		utils.Validate(new(requests.PromoteScenarioRequest)),
		handler.PromoteScenario,
	)

	group.Post(
		"/resolve",
		utils.Validate(new(requests.ResolveScenarioRequest)),
		handler.ResolveScenario,
	)

	group.Post(
		"/to-test",
		utils.Validate(new(requests.MoveToTestScenarioRequest)),
		handler.MoveToTestScenario,
	)
}
