package api

import (
	"net/http"

	"pr-reviewer-service/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{service: svc}
}

type ErrorCode string

const (
	CodeTeamExists  ErrorCode = "TEAM_EXISTS"
	CodePRExists    ErrorCode = "PR_EXISTS"
	CodePRMerged    ErrorCode = "PR_MERGED"
	CodeNotAssigned ErrorCode = "NOT_ASSIGNED"
	CodeNoCandidate ErrorCode = "NO_CANDIDATE"
	CodeNotFound    ErrorCode = "NOT_FOUND"
)

type ErrorResponse struct {
	Error struct {
		Code    ErrorCode `json:"code"`
		Message string    `json:"message"`
	} `json:"error"`
}

func sendError(c *gin.Context, status int, code ErrorCode, message string) {
	c.JSON(status, ErrorResponse{
		Error: struct {
			Code    ErrorCode `json:"code"`
			Message string    `json:"message"`
		}{
			Code:    code,
			Message: message,
		},
	})
}

func handleServiceError(c *gin.Context, err error) {
	switch err {
	case service.ErrTeamExists:
		sendError(c, http.StatusBadRequest, CodeTeamExists, "team_name already exists")
	case service.ErrPRExists:
		sendError(c, http.StatusConflict, CodePRExists, "PR id already exists")
	case service.ErrPRMerged:
		sendError(c, http.StatusConflict, CodePRMerged, "cannot reassign on merged PR")
	case service.ErrNotAssigned:
		sendError(c, http.StatusConflict, CodeNotAssigned, "reviewer is not assigned to this PR")
	case service.ErrNoCandidate:
		sendError(c, http.StatusConflict, CodeNoCandidate, "no active replacement candidate in team")
	case service.ErrNotFound:
		sendError(c, http.StatusNotFound, CodeNotFound, "resource not found")
	default:
		sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
