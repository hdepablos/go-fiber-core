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
		"/current-version",
		utils.Validate(new(requests.ResolveScenarioRequest)),
		handler.ResolveCurrentVersion,
	)

	group.Post(
		"/to-test",
		utils.Validate(new(requests.MoveToTestScenarioRequest)),
		handler.MoveToTestScenario,
	)

	group.Post(
		"/versions/paginated",
		handler.ListProcessVersions,
	)

	group.Get(
		"/versions/:id",
		handler.GetProcessVersion,
	)

	group.Post(
		"/run",
		utils.Validate(new(requests.RunProcessRequest)),
		handler.RunLoanRiskLifecycle,
	)

	group.Post(
		"/export-preview",
		utils.Validate(new(requests.PreviewExportRequest)),
		handler.PreviewExport,
	)

	group.Post(
		"/batch-preview",
		utils.Validate(new(requests.PreviewBatchRequest)),
		handler.PreviewBatch,
	)
}
