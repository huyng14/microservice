package services

import (
	"microservice/authorization"
	"microservice/models"
	mongodb "microservice/mongoDB"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	databaseName      = "project"
	profileCollection = "consultants"
	jobCollection     = "jobs"
)

type PersistenceService struct {
	Repo *mongodb.MongoSvc
}

func (s *PersistenceService) ListProfiles(user authorization.UserContext) ([]models.Profile, error) {
	grant, ok := authorization.GrantFor(user.Role, authorization.ResumeView)
	if !ok {
		return nil, authorization.ErrForbidden
	}
	filter := bson.M{}
	if grant.Scope == authorization.ScopeOwn {
		filter["ownerId"] = user.UserID
	}
	return s.Repo.ListCVs(databaseName, profileCollection, filter)
}

func (s *PersistenceService) CreateProfile(user authorization.UserContext, profile *models.Profile) (*mongo.InsertOneResult, error) {
	if err := authorization.CanAccess(user, authorization.ResumeCreate, user.UserID); err != nil {
		return nil, err
	}
	profile.OwnerID = user.UserID
	return s.Repo.InsertCV(databaseName, profileCollection, profile)
}

func (s *PersistenceService) UpdateProfile(user authorization.UserContext, profile models.Profile) error {
	existing, err := s.Repo.GetCV(databaseName, profileCollection, profile.Id)
	if err != nil {
		return err
	}
	if err := authorization.CanAccess(user, authorization.ResumeUpdate, existing.OwnerID); err != nil {
		return err
	}
	profile.OwnerID = existing.OwnerID
	return s.Repo.UpdateCV(databaseName, profileCollection, profile)
}

func (s *PersistenceService) DeleteProfile(user authorization.UserContext, id string) error {
	existing, err := s.Repo.GetCV(databaseName, profileCollection, id)
	if err != nil {
		return err
	}
	if err := authorization.CanAccess(user, authorization.ResumeDelete, existing.OwnerID); err != nil {
		return err
	}
	return s.Repo.DeleteCV(databaseName, profileCollection, id)
}

func (s *PersistenceService) ListJobs(user authorization.UserContext) ([]models.Job, error) {
	grant, ok := authorization.GrantFor(user.Role, authorization.JobView)
	if !ok {
		return nil, authorization.ErrForbidden
	}
	filter := bson.M{}
	if grant.Scope == authorization.ScopeOwn {
		filter["creatorId"] = user.UserID
	}
	return s.Repo.ListJobs(databaseName, jobCollection, filter)
}

func (s *PersistenceService) CreateJob(user authorization.UserContext, job models.Job) (*mongo.InsertOneResult, error) {
	if err := authorization.CanAccess(user, authorization.JobCreate, user.UserID); err != nil {
		return nil, err
	}
	job.CreatorID = user.UserID
	return s.Repo.InsertJob(databaseName, jobCollection, job)
}

func (s *PersistenceService) UpdateJob(user authorization.UserContext, job models.Job) error {
	existing, err := s.Repo.GetJob(databaseName, jobCollection, job.Id)
	if err != nil {
		return err
	}
	if err := authorization.CanAccess(user, authorization.JobUpdate, existing.CreatorID); err != nil {
		return err
	}
	job.CreatorID = existing.CreatorID
	return s.Repo.UpdateJob(databaseName, jobCollection, job)
}

func (s *PersistenceService) DeleteJob(user authorization.UserContext, id string) error {
	existing, err := s.Repo.GetJob(databaseName, jobCollection, id)
	if err != nil {
		return err
	}
	if err := authorization.CanAccess(user, authorization.JobDelete, existing.CreatorID); err != nil {
		return err
	}
	return s.Repo.DeleteJob(databaseName, jobCollection, id)
}
