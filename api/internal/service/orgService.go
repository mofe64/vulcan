package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mofe64/vulkan/api/internal/dto"
	"github.com/mofe64/vulkan/api/internal/events"
	platformv1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OrgService interface {
	CreateOrg(ctx context.Context, req dto.CreateOrgRequest) (dto.Org, error)
	GetOrg(orgID string) (string, error)
	UpdateOrg(orgID string, name string) error
	DeleteOrg(orgID string) error
	ListOrgs() ([]string, error)
}

type orgService struct {
	db     *pgxpool.Pool
	k8s    client.Client
	logger *zap.Logger
	bus    *events.EventBus
}

func NewOrgService(db *pgxpool.Pool, k8s client.Client, logger *zap.Logger, bus *events.EventBus) OrgService {
	return &orgService{
		db:     db,
		k8s:    k8s,
		logger: logger,
		bus:    bus,
	}
}
func (s *orgService) CreateOrg(ctx context.Context, req dto.CreateOrgRequest) (dto.Org, error) {
	id := uuid.New()

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return dto.Org{}, err
	}

	if _, err := tx.Exec(ctx, "INSERT INTO orgs (id, name, owner_id, owner_email) VALUES ($1, $2, $3, $4)",
		id, req.Name, req.OwnerId, req.OwnerEmail); err != nil {
		tx.Rollback(ctx)
		return dto.Org{}, err
	}

	// create the Kubernetes CRD for the organization
	cr := &platformv1.Org{
		ObjectMeta: metav1.ObjectMeta{
			Name: id.String(),
		},
		Spec: platformv1.OrgSpec{
			DisplayName: req.Name,
			OwnerEmail:  req.OwnerEmail,
			OrgQuota: platformv1.OrgQuota{
				Clusters: 1,   // Default cluster quota
				Apps:     100, // Default app quota
			},
		},
	}

	if err := s.k8s.Create(ctx, cr); err != nil {
		tx.Rollback(ctx)
		return dto.Org{}, err
	}

	// commit DB tx  (now DB + CRD are consistent)
	if err := tx.Commit(ctx); err != nil {
		return dto.Org{}, err
	}

	// emit nats event for org creation
	s.bus.OrgCreated(ctx, id)

	// Todo increment prom metric
	// NOTE -> NATS and Prometheus setup should be done via helm charts, Idea is to do all the setup in the vulkan setup script

	return dto.Org{
		Id:         id.String(),
		Name:       req.Name,
		OwnerId:    req.OwnerId,
		OwnerEmail: req.OwnerEmail,
		CreatedAt:  time.Now().UTC(),
	}, nil

}
func (s *orgService) GetOrg(orgID string) (string, error) {
	// Implementation for retrieving an organization by ID
	s.logger.Info("Retrieving organization", zap.String("orgID", orgID))
	// Here you would add the logic to retrieve an organization from the database
	return "org-name", nil // Return the organization's name
}
func (s *orgService) UpdateOrg(orgID string, name string) error {
	// Implementation for updating an organization
	s.logger.Info("Updating organization", zap.String("orgID", orgID), zap.String("name", name))
	// Here you would add the logic to update an organization in the database
	return nil // Return nil if successful
}
func (s *orgService) DeleteOrg(orgID string) error {
	// Implementation for deleting an organization
	s.logger.Info("Deleting organization", zap.String("orgID", orgID))
	// Here you would add the logic to delete an organization from the database and Kubernetes
	return nil // Return nil if successful
}
func (s *orgService) ListOrgs() ([]string, error) {
	// Implementation for listing organizations
	s.logger.Info("Listing organizations")
	// Here you would add the logic to retrieve all organizations from the database
	return []string{"org1", "org2"}, nil // Return the list of organization names
}
