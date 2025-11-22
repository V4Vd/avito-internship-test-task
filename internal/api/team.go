package api

import (
	"net/http"

	"pr-reviewer-service/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateTeam(c *gin.Context) {
	var team models.Team
	if err := c.ShouldBindJSON(&team); err != nil {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.service.CreateTeam(&team); err != nil {
		handleServiceError(c, err)
		return
	}

	createdTeam, err := h.service.GetTeam(team.TeamName)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"team": createdTeam,
	})
}

func (h *Handler) GetTeam(c *gin.Context) {
	teamName := c.Query("team_name")
	if teamName == "" {
		sendError(c, http.StatusBadRequest, "INVALID_REQUEST", "team_name is required")
		return
	}

	team, err := h.service.GetTeam(teamName)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, team)
}
