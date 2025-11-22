package api

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(h *Handler) *gin.Engine {
	r := gin.Default()

	team := r.Group("/team")
	team.POST("/add", h.CreateTeam)
	team.GET("/get", h.GetTeam)

	users := r.Group("/users")
	users.POST("/setIsActive", h.SetUserActive)
	users.GET("/getReview", h.GetUserReviews)

	pr := r.Group("/pullRequest")
	pr.POST("/create", h.CreatePullRequest)
	pr.POST("/merge", h.MergePullRequest)
	pr.POST("/reassign", h.ReassignReviewer)

	return r
}
