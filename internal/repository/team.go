package repository

import (
	"database/sql"

	"pr-reviewer-service/internal/models"
)

type TeamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(teamName string) error {
	query := `INSERT INTO teams (team_name) VALUES ($1)`
	_, err := r.db.Exec(query, teamName)
	return err
}

func (r *TeamRepository) Exists(teamName string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`

	var exists bool
	err := r.db.QueryRow(query, teamName).Scan(&exists)
	return exists, err
}

func (r *TeamRepository) Get(teamName string) (*models.Team, error) {
	exists, err := r.Exists(teamName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	query := `SELECT user_id, username, is_active FROM users WHERE team_name = $1`

	rows, err := r.db.Query(query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.TeamMember
	for rows.Next() {
		var member models.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	team := &models.Team{
		TeamName: teamName,
		Members:  members,
	}

	return team, nil
}
