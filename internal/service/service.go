package service

import (
	"errors"
	"math/rand"
	"time"

	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repository"
)

var (
	ErrTeamExists  = errors.New("TEAM_EXISTS")
	ErrPRExists    = errors.New("PR_EXISTS")
	ErrPRMerged    = errors.New("PR_MERGED")
	ErrNotAssigned = errors.New("NOT_ASSIGNED")
	ErrNoCandidate = errors.New("NO_CANDIDATE")
	ErrNotFound    = errors.New("NOT_FOUND")
)

const (
	reviewersCount = 2
)

type Service struct {
	userRepo *repository.UserRepository
	teamRepo *repository.TeamRepository
	prRepo   *repository.PRRepository
	rand     *rand.Rand
}

func NewService(userRepo *repository.UserRepository, teamRepo *repository.TeamRepository, prRepo *repository.PRRepository) *Service {
	return &Service{
		userRepo: userRepo,
		teamRepo: teamRepo,
		prRepo:   prRepo,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *Service) CreateTeam(team *models.Team) error {
	exists, err := s.teamRepo.Exists(team.TeamName)
	if err != nil {
		return err
	}
	if exists {
		return ErrTeamExists
	}

	if err := s.teamRepo.Create(team.TeamName); err != nil {
		return err
	}

	for _, member := range team.Members {
		user := &models.User{
			UserID:   member.UserID,
			Username: member.Username,
			TeamName: team.TeamName,
			IsActive: member.IsActive,
		}
		if err := s.userRepo.Create(user); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) GetTeam(teamName string) (*models.Team, error) {
	team, err := s.teamRepo.Get(teamName)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, ErrNotFound
	}
	return team, nil
}

func (s *Service) SetUserActive(userID string, isActive bool) (*models.User, error) {
	if err := s.userRepo.UpdateIsActive(userID, isActive); err != nil {
		return nil, ErrNotFound
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) CreatePullRequest(prID, prName, authorID string) (*models.PullRequest, error) {
	exists, err := s.prRepo.Exists(prID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrPRExists
	}

	author, err := s.userRepo.GetByID(authorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, ErrNotFound
	}

	candidates, err := s.userRepo.GetActiveTeamMembers(author.TeamName, authorID)
	if err != nil {
		return nil, err
	}

	reviewers := s.selectRandomReviewers(candidates, reviewersCount)

	pr := &models.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   prName,
		AuthorID:          authorID,
		Status:            models.StatusOpen,
		AssignedReviewers: reviewers,
	}

	if err := s.prRepo.Create(pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *Service) MergePullRequest(prID string) (*models.PullRequest, error) {
	pr, err := s.prRepo.GetByID(prID)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	if pr.Status == models.StatusMerged {
		return pr, nil
	}

	if updateErr := s.prRepo.UpdateStatus(prID, models.StatusMerged); updateErr != nil {
		return nil, updateErr
	}

	pr, err = s.prRepo.GetByID(prID)
	if err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *Service) ReassignReviewer(prID, oldReviewerID string) (*models.PullRequest, string, error) {
	pr, err := s.prRepo.GetByID(prID)
	if err != nil {
		return nil, "", err
	}
	if pr == nil {
		return nil, "", ErrNotFound
	}

	if pr.Status == models.StatusMerged {
		return nil, "", ErrPRMerged
	}

	isAssigned, err := s.prRepo.IsReviewerAssigned(prID, oldReviewerID)
	if err != nil {
		return nil, "", err
	}
	if !isAssigned {
		return nil, "", ErrNotAssigned
	}

	oldReviewer, err := s.userRepo.GetByID(oldReviewerID)
	if err != nil {
		return nil, "", err
	}
	if oldReviewer == nil {
		return nil, "", ErrNotFound
	}

	candidates, err := s.userRepo.GetActiveTeamMembers(oldReviewer.TeamName, oldReviewerID)
	if err != nil {
		return nil, "", err
	}

	var availableCandidates []models.User
	for _, candidate := range candidates {
		isCurrentReviewer := false
		for _, reviewerID := range pr.AssignedReviewers {
			if candidate.UserID == reviewerID {
				isCurrentReviewer = true
				break
			}
		}
		if !isCurrentReviewer {
			availableCandidates = append(availableCandidates, candidate)
		}
	}

	if len(availableCandidates) == 0 {
		return nil, "", ErrNoCandidate
	}

	newReviewer := availableCandidates[s.rand.Intn(len(availableCandidates))]

	if replaceErr := s.prRepo.ReplaceReviewer(prID, oldReviewerID, newReviewer.UserID); replaceErr != nil {
		return nil, "", replaceErr
	}

	pr, err = s.prRepo.GetByID(prID)
	if err != nil {
		return nil, "", err
	}

	return pr, newReviewer.UserID, nil
}

func (s *Service) GetUserReviews(userID string) ([]models.PullRequestShort, error) {
	prs, err := s.prRepo.GetByReviewer(userID)
	if err != nil {
		return nil, err
	}

	if prs == nil {
		prs = []models.PullRequestShort{}
	}

	return prs, nil
}

func (s *Service) selectRandomReviewers(candidates []models.User, n int) []string {
	if len(candidates) == 0 {
		return []string{}
	}

	if n > len(candidates) {
		n = len(candidates)
	}

	shuffled := make([]models.User, len(candidates))
	copy(shuffled, candidates)
	s.rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	reviewers := make([]string, n)
	for i := 0; i < n; i++ {
		reviewers[i] = shuffled[i].UserID
	}

	return reviewers
}
