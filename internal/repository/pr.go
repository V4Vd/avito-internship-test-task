package repository

import (
	"database/sql"
	"time"

	"pr-reviewer-service/internal/models"
)

type PRRepository struct {
	db *sql.DB
}

func NewPRRepository(db *sql.DB) *PRRepository {
	return &PRRepository{db: db}
}

func (r *PRRepository) Create(pr *models.PullRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback() // nolint:errcheck // rollback is safe to ignore in defer
	}()

	query := `INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at) 
		VALUES ($1, $2, $3, $4, $5)`

	now := time.Now()
	_, err = tx.Exec(query, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status, now)
	if err != nil {
		return err
	}

	if len(pr.AssignedReviewers) > 0 {
		for _, reviewerID := range pr.AssignedReviewers {
			reviewerQuery := `INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)`
			_, err = tx.Exec(reviewerQuery, pr.PullRequestID, reviewerID)
			if err != nil {
				return err
			}
		}
	}

	pr.CreatedAt = &now
	return tx.Commit()
}

func (r *PRRepository) GetByID(prID string) (*models.PullRequest, error) {
	query := `SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at 
		FROM pull_requests WHERE pull_request_id = $1`

	pr := &models.PullRequest{}
	err := r.db.QueryRow(query, prID).Scan(
		&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID,
		&pr.Status, &pr.CreatedAt, &pr.MergedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	reviewersQuery := `SELECT user_id FROM pr_reviewers WHERE pull_request_id = $1`
	rows, err := r.db.Query(reviewersQuery, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, reviewerID)
	}

	pr.AssignedReviewers = reviewers
	return pr, rows.Err()
}

func (r *PRRepository) Exists(prID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)`

	var exists bool
	err := r.db.QueryRow(query, prID).Scan(&exists)
	return exists, err
}

func (r *PRRepository) UpdateStatus(prID, status string) error {
	now := time.Now()
	query := `UPDATE pull_requests SET status = $1, merged_at = $2 WHERE pull_request_id = $3`

	var mergedAt *time.Time
	if status == models.StatusMerged {
		mergedAt = &now
	}

	_, err := r.db.Exec(query, status, mergedAt, prID)
	return err
}

func (r *PRRepository) ReplaceReviewer(prID, oldReviewerID, newReviewerID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback() // nolint:errcheck // rollback is safe to ignore in defer
	}()

	deleteQuery := `DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2`
	_, err = tx.Exec(deleteQuery, prID, oldReviewerID)
	if err != nil {
		return err
	}

	insertQuery := `INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)`
	_, err = tx.Exec(insertQuery, prID, newReviewerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PRRepository) IsReviewerAssigned(prID, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2)`

	var exists bool
	err := r.db.QueryRow(query, prID, userID).Scan(&exists)
	return exists, err
}

func (r *PRRepository) GetByReviewer(userID string) ([]models.PullRequestShort, error) {
	query := `SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		INNER JOIN pr_reviewers rev ON pr.pull_request_id = rev.pull_request_id
		WHERE rev.user_id = $1
		ORDER BY pr.created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []models.PullRequestShort
	for rows.Next() {
		var pr models.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	return prs, rows.Err()
}
