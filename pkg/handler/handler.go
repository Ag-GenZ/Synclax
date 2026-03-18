package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/wibus-wee/synclax/pkg/symphony/control"
	"github.com/wibus-wee/synclax/pkg/zcore/model"
	"github.com/wibus-wee/synclax/pkg/zgen/apigen"
	"github.com/wibus-wee/synclax/pkg/zgen/taskgen"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	model      model.ModelInterface
	taskrunner taskgen.TaskRunner
	symphony   *control.Manager
}

func NewHandler(model model.ModelInterface, taskrunner taskgen.TaskRunner, symphony *control.Manager) (apigen.ServerInterface, error) {
	return &Handler{model: model, taskrunner: taskrunner, symphony: symphony}, nil
}

func (h *Handler) GetCounter(c *fiber.Ctx) error {
	count, err := h.model.GetCounter(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(apigen.Counter{Count: count.Value})
}

func (h *Handler) IncrementCounter(c *fiber.Ctx) error {
	_, err := h.taskrunner.RunIncrementCounter(c.Context(), &taskgen.IncrementCounterParameters{
		Amount: 1,
	})
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusAccepted).SendString("Incremented")
}

func (h *Handler) GetHealth(c *fiber.Ctx) error {
	health := control.Health{}
	if h.symphony != nil {
		health = h.symphony.Health()
	}

	var lastErr *string
	if health.LastError != nil {
		s := health.LastError.Error()
		lastErr = &s
	}
	var workflowPath *string
	if strings.TrimSpace(health.WorkflowPath) != "" {
		p := health.WorkflowPath
		workflowPath = &p
	}

	var httpPort *int32
	if health.HTTPPort != nil {
		p := int32(*health.HTTPPort)
		httpPort = &p
	}

	var activeWorkflowID *string
	if strings.TrimSpace(health.ActiveWorkflowID) != "" {
		id := health.ActiveWorkflowID
		activeWorkflowID = &id
	}

	var workflows *[]apigen.SymphonyWorkflow
	if len(health.Workflows) > 0 {
		out := make([]apigen.SymphonyWorkflow, 0, len(health.Workflows))
		for _, w := range health.Workflows {
			var we *string
			if w.LastError != nil {
				s := w.LastError.Error()
				we = &s
			}
			var wp *int32
			if w.HTTPPort != nil {
				p := int32(*w.HTTPPort)
				wp = &p
			}
			out = append(out, apigen.SymphonyWorkflow{
				Id:           w.ID,
				WorkflowPath: w.WorkflowPath,
				Running:      w.Running,
				LastError:    we,
				HttpPort:     wp,
			})
		}
		workflows = &out
	}

	return c.JSON(apigen.HealthResponse{
		Status:                   "ok",
		SymphonyRunning:          health.Running,
		SymphonyActiveWorkflowId: activeWorkflowID,
		SymphonyWorkflowPath:     workflowPath,
		SymphonyLastError:        lastErr,
		SymphonyHttpPort:         httpPort,
		SymphonyWorkflows:        workflows,
	})
}

func (h *Handler) GetSymphonySnapshot(c *fiber.Ctx, params apigen.GetSymphonySnapshotParams) error {
	if h.symphony == nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString("symphony manager not configured")
	}
	workflowID := ""
	if params.WorkflowId != nil {
		workflowID = strings.TrimSpace(*params.WorkflowId)
	}
	snap, err := h.symphony.Snapshot(workflowID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString(err.Error())
	}
	return c.JSON(snap)
}

func (h *Handler) StartSymphony(c *fiber.Ctx) error {
	if h.symphony == nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString("symphony manager not configured")
	}

	var req apigen.StartSymphonyRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		}
	}

	workflowPath := ""
	if req.WorkflowPath != nil {
		workflowPath = strings.TrimSpace(*req.WorkflowPath)
	}
	workflowID := ""
	if req.WorkflowId != nil {
		workflowID = strings.TrimSpace(*req.WorkflowId)
	}

	var httpPort *int
	if req.HttpPort != nil {
		p := int(*req.HttpPort)
		httpPort = &p
	}

	if err := h.symphony.Start(c.Context(), workflowID, workflowPath, httpPort); err != nil {
		// Conflict is a client error; others are server errors.
		if strings.Contains(strings.ToLower(err.Error()), "already running") {
			return c.Status(fiber.StatusConflict).SendString(err.Error())
		}
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	health := h.symphony.Health()
	var wp *string
	if strings.TrimSpace(health.WorkflowPath) != "" {
		p := health.WorkflowPath
		wp = &p
	}
	var wid *string
	if strings.TrimSpace(health.ActiveWorkflowID) != "" {
		id := health.ActiveWorkflowID
		wid = &id
	}
	return c.JSON(apigen.StartSymphonyResult{Running: health.Running, WorkflowId: wid, WorkflowPath: wp})
}

func (h *Handler) StopSymphony(c *fiber.Ctx, params apigen.StopSymphonyParams) error {
	if h.symphony == nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString("symphony manager not configured")
	}

	workflowID := ""
	if params.WorkflowId != nil {
		workflowID = strings.TrimSpace(*params.WorkflowId)
	}

	if err := h.symphony.Stop(c.Context(), workflowID); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return c.Status(fiber.StatusRequestTimeout).SendString(err.Error())
		}
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	health := h.symphony.Health()
	var wid *string
	if strings.TrimSpace(workflowID) != "" {
		id := workflowID
		wid = &id
	}
	return c.JSON(apigen.StopSymphonyResult{Running: health.Running, WorkflowId: wid})
}

func (h *Handler) GetSymphonyWorkflows(c *fiber.Ctx) error {
	if h.symphony == nil {
		return c.Status(fiber.StatusServiceUnavailable).SendString("symphony manager not configured")
	}
	health := h.symphony.Health()
	workflows := make([]apigen.SymphonyWorkflow, 0, len(health.Workflows))
	for _, w := range health.Workflows {
		var we *string
		if w.LastError != nil {
			s := w.LastError.Error()
			we = &s
		}
		var wp *int32
		if w.HTTPPort != nil {
			p := int32(*w.HTTPPort)
			wp = &p
		}
		workflows = append(workflows, apigen.SymphonyWorkflow{
			Id:           w.ID,
			WorkflowPath: w.WorkflowPath,
			Running:      w.Running,
			LastError:    we,
			HttpPort:     wp,
		})
	}

	var active *string
	if strings.TrimSpace(health.ActiveWorkflowID) != "" {
		id := health.ActiveWorkflowID
		active = &id
	}

	return c.JSON(apigen.SymphonyWorkflowsResponse{
		ActiveWorkflowId: active,
		Workflows:        workflows,
	})
}
