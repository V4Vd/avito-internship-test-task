package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"pr-reviewer-service/internal/api"
	"pr-reviewer-service/internal/database"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/service"
)

var (
	testRouter *gin.Engine
	testDB     *sql.DB
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	testDbURL := "postgres://postgres:postgres@localhost:5432/pr_reviewer_test?sslmode=disable"

	if err := setupTestDB(testDbURL); err != nil {
		fmt.Printf("failed to setup database for tests: %v\n", err)
		os.Exit(1)
	}

	var err error
	testDB, err = database.Connect(testDbURL)
	if err != nil {
		fmt.Printf("Failed to connect to test database: %v\n", err)
		os.Exit(1)
	}

	if err := database.RunMigrations(testDB); err != nil {
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	userRepo := repository.NewUserRepository(testDB)
	teamRepo := repository.NewTeamRepository(testDB)
	prRepo := repository.NewPRRepository(testDB)
	svc := service.NewService(userRepo, teamRepo, prRepo)
	handler := api.NewHandler(svc)
	testRouter = api.SetupRoutes(handler)

	code := m.Run()

	// can't use defer here
	testDB.Close()

	os.Exit(code)
}

func setupTestDB(dbURL string) error {
	u, err := url.Parse(dbURL)
	if err != nil {
		return err
	}
	dbName := strings.TrimPrefix(u.Path, "/")

	u.Path = "/postgres"

	adminDB, err := sql.Open("postgres", u.String())
	if err != nil {
		return err
	}
	defer adminDB.Close()

	if _, err := adminDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %q`, dbName)); err != nil {
		return err
	}

	if _, err := adminDB.Exec(fmt.Sprintf(`CREATE DATABASE %q`, dbName)); err != nil {
		return err
	}

	return nil
}

func cleanupDB(t *testing.T) {
	tables := []string{"pull_requests", "teams", "users"}
	for _, table := range tables {
		_, err := testDB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Fatalf("Failed to truncate table %s: %v", table, err)
		}
	}
}

func TestTeamAPI(t *testing.T) {
	cleanupDB(t)

	t.Run("CreateTeam", func(t *testing.T) {
		payload := map[string]any{
			"team_name": "Backend Team",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]any
		json.Unmarshal(w.Body.Bytes(), &response)

		team, ok := response["team"].(map[string]any)
		if !ok {
			t.Fatalf("expected team object in response, got %v", response)
		}

		if team["team_name"] != "Backend Team" {
			t.Errorf("expected team_name 'Backend Team', got %v", team["team_name"])
		}
	})

	t.Run("GetTeam", func(t *testing.T) {
		payload := map[string]any{
			"team_name": "Frontend Team",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		req = httptest.NewRequest(http.MethodGet, "/team/get?team_name=Frontend%20Team", http.NoBody)
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var team map[string]any
		json.Unmarshal(w.Body.Bytes(), &team)

		if team["team_name"] != "Frontend Team" {
			t.Errorf("expected team_name 'Frontend Team', got %v", team["team_name"])
		}
	})

	t.Run("CreateTeamDuplicate", func(t *testing.T) {
		teamName := "DevOps Team"
		payload := map[string]any{
			"team_name": teamName,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		body, _ = json.Marshal(payload)
		req = httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestUserAPI(t *testing.T) {
	cleanupDB(t)

	teamPayload := map[string]any{
		"team_name": "Test Team",
		"members": []map[string]any{
			{"user_id": "user1", "username": "User One", "is_active": true},
			{"user_id": "user2", "username": "User Two", "is_active": true},
		},
	}
	body, _ := json.Marshal(teamPayload)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	t.Run("SetUserIsActive", func(t *testing.T) {
		payload := map[string]any{
			"user_id":   "user1",
			"team_name": "Test Team",
			"is_active": true,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GetUserReviews", func(t *testing.T) {
		payload := map[string]any{
			"user_id":   "user2",
			"team_name": "Test Team",
			"is_active": true,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		req = httptest.NewRequest(http.MethodGet, "/users/getReview?user_id=user2", http.NoBody)
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]any
		json.Unmarshal(w.Body.Bytes(), &response)

		if prs, ok := response["pull_requests"].([]any); !ok || len(prs) != 0 {
			t.Errorf("Expected empty pull_requests array, got %v", response["pull_requests"])
		}
	})
}

func TestPullRequestAPI(t *testing.T) {
	cleanupDB(t)

	teamPayload := map[string]any{
		"team_name": "PR Test Team",
		"members": []map[string]any{
			{"user_id": "author1", "username": "Author One", "is_active": true},
			{"user_id": "author2", "username": "Author Two", "is_active": true},
			{"user_id": "author3", "username": "Author Three", "is_active": true},
			{"user_id": "reviewer1", "username": "Reviewer One", "is_active": true},
			{"user_id": "reviewer2", "username": "Reviewer Two", "is_active": true},
			{"user_id": "reviewer3", "username": "Reviewer Three", "is_active": true},
		},
	}
	body, _ := json.Marshal(teamPayload)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	t.Run("CreatePullRequest", func(t *testing.T) {
		payload := map[string]any{
			"pull_request_id":   "pr-123",
			"pull_request_name": "Add search feature",
			"author_id":         "author1",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]any
		json.Unmarshal(w.Body.Bytes(), &response)

		prObj := response["pr"]
		if prObj == nil {
			t.Fatalf("Expected pr in response, got %v", response)
		}

		pr := prObj.(map[string]any)
		assignedReviewers, _ := pr["assigned_reviewers"].([]any)

		if len(assignedReviewers) == 0 {
			t.Error("Expected at least one reviewer to be assigned")
		}

		if len(assignedReviewers) > 2 {
			t.Errorf("Expected at most 2 reviewers, got %d", len(assignedReviewers))
		}
	})

	t.Run("MergePullRequest", func(t *testing.T) {
		createPayload := map[string]any{
			"pull_request_id":   "pr-456",
			"pull_request_name": "Fix bug",
			"author_id":         "author2",
		}
		body, _ := json.Marshal(createPayload)
		req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		mergePayload := map[string]any{
			"pull_request_id": "pr-456",
		}
		body, _ = json.Marshal(mergePayload)
		req = httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]any
		json.Unmarshal(w.Body.Bytes(), &response)

		pr := response["pr"].(map[string]any)
		if pr["status"] != "MERGED" {
			t.Errorf("Expected status 'MERGED', got %v", pr["status"])
		}
	})

	t.Run("ReassignReviewer", func(t *testing.T) {
		createPayload := map[string]any{
			"pull_request_id":   "pr-789",
			"pull_request_name": "Refactor",
			"author_id":         "author3",
		}
		body, _ := json.Marshal(createPayload)
		req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		var createResponse map[string]any
		json.Unmarshal(w.Body.Bytes(), &createResponse)
		pr := createResponse["pr"].(map[string]any)
		assignedReviewers := pr["assigned_reviewers"].([]any)

		if len(assignedReviewers) == 0 {
			t.Skip("No reviewers assigned, cannot test reassignment")
		}

		oldReviewer := assignedReviewers[0].(string)

		reassignPayload := map[string]any{
			"pull_request_id": "pr-789",
			"old_user_id":     oldReviewer,
		}
		body, _ = json.Marshal(reassignPayload)
		req = httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]any
		json.Unmarshal(w.Body.Bytes(), &response)

		updatedPR := response["pr"].(map[string]any)
		newReviewers := updatedPR["assigned_reviewers"].([]any)

		for _, reviewer := range newReviewers {
			if reviewer.(string) == oldReviewer {
				t.Errorf("Old reviewer %s should not be in the list after reassignment", oldReviewer)
			}
		}
	})
}

func TestPRIdempotency(t *testing.T) {
	cleanupDB(t)

	teamPayload := map[string]any{
		"team_name": "Idempotency Team",
		"members": []map[string]any{
			{"user_id": "idempotency_user", "username": "Idempo", "is_active": true},
		},
	}
	body, _ := json.Marshal(teamPayload)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	t.Run("MergeIdempotency", func(t *testing.T) {
		createPayload := map[string]any{
			"pull_request_id":   "idemp-999",
			"pull_request_name": "Idempotency Test",
			"author_id":         "idempotency_user",
		}
		body, _ := json.Marshal(createPayload)
		req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		mergePayload := map[string]any{
			"pull_request_id": "idemp-999",
		}
		body, _ = json.Marshal(mergePayload)
		req = httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("First merge: Expected status 200, got %d", w.Code)
		}

		body, _ = json.Marshal(mergePayload)
		req = httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Second merge: Expected status 200, got %d", w.Code)
		}

		var response map[string]any
		json.Unmarshal(w.Body.Bytes(), &response)

		pr := response["pr"].(map[string]any)
		if pr["status"] != "MERGED" {
			t.Errorf("Expected status 'MERGED', got %v", pr["status"])
		}
	})
}
