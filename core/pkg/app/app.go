package application

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mofe64/vulcan/pkg/common"
	"github.com/mofe64/vulcan/pkg/models"
)

// TODO: Ability for root user to bypass owner restrictions
// TODO: custom error types

// CreateNewApplication creates a new application record in the database.
// The application name must be unique.
func CreateNewApplication(ctx context.Context, forge common.Forge, applicationName string, ownerId int64) error {
	database := forge.GetDB()
	logger := forge.GetLogger()

	exists, err := ApplicationNameExists(ctx, forge, applicationName)
	if err != nil {
		logger.Error("Failed to check if application name exists")
		return err
	}
	if exists {
		err := fmt.Errorf("application name %s already exists", applicationName)
		logger.Error(err)
		return err
	}

	query := "insert into application (name, ownerId) values (?)"
	_, err = database.ExecContext(ctx, query, applicationName, ownerId)
	if err != nil {
		logger.Error("Failed to create a new application")
		return err
	}
	logger.Printf("application %s created successfully", applicationName)
	return nil
}

// ListApplications retrieves all applications owned by a user.
func ListApplications(ctx context.Context, forge common.Forge, ownerId int64) ([]*models.Application, error) {
	database := forge.GetDB()
	logger := forge.GetLogger()

	query := "select * from application where ownerId = ?"
	rows, err := database.QueryContext(ctx, query)
	if err != nil {
		logger.Errorf("Failed to list applications, error: %v", err)
		return nil, err
	}
	defer rows.Close()

	var applications []*models.Application
	for rows.Next() {
		var application models.Application
		var sourceCodeLink models.SourceCodeLink
		var repoName sql.NullString
		var repoUrl sql.NullString
		var repoType sql.NullString
		var webhookIdentifier sql.NullString

		err := rows.Scan(
			&application.Id,
			&application.OwnerId,
			&application.Name,
			&repoName,
			&repoUrl,
			&repoType,
			&webhookIdentifier,
		)
		if err != nil {
			logger.Errorf("Failed to list applications - error during scan operation, error: %v", err)
			return nil, err
		}

		if repoName.Valid {
			sourceCodeLink.RepoName = repoName.String
		} else {
			sourceCodeLink.RepoName = "NOT SET"
		}
		if repoUrl.Valid {
			sourceCodeLink.RepoUrl = repoUrl.String
		} else {
			sourceCodeLink.RepoUrl = "NOT SET"
		}
		if repoType.Valid {
			sourceCodeLink.RepoType = repoType.String
		} else {
			sourceCodeLink.RepoType = "NOT SET"
		}
		if webhookIdentifier.Valid {
			sourceCodeLink.WebhookIdentifier = webhookIdentifier.String
		} else {
			sourceCodeLink.WebhookIdentifier = "NOT SET"
		}

		application.SourceCodeLink = sourceCodeLink
		applications = append(applications, &application)

	}

	if err = rows.Err(); err != nil {
		logger.Errorf("Failed to list applications - error during scan operation, error: %v", err)
		return nil, err
	}

	return applications, nil

}

// DeleteApplication removes an application record by its id.
// Only the application owner can delete an application.
func DeleteApplication(ctx context.Context, forge common.Forge, applicationId int64, requesterOwnerId int64) error {
	database := forge.GetDB()
	logger := forge.GetLogger()

	err := VerifyApplicationOwner(ctx, forge, applicationId, requesterOwnerId)
	if err != nil {
		logger.Error(err)
		return err
	}

	query := "delete from application where id = ?"
	_, err = database.ExecContext(ctx, query, applicationId)
	if err != nil {
		logger.Errorf("Failed to delete application with id %d: %v", applicationId, err)
		return err
	}
	logger.Printf("Application with id %d deleted successfully", applicationId)
	return nil
}

// ChangeApplicationName updates the name of an application.
// Only the application owner can change the name.
func ChangeApplicationName(ctx context.Context, forge common.Forge, applicationId int64, newName string, requesterOwnerId int64) error {
	database := forge.GetDB()
	logger := forge.GetLogger()

	err := VerifyApplicationOwner(ctx, forge, applicationId, requesterOwnerId)
	if err != nil {
		logger.Error(err)
		return err
	}

	query := "update application set name = ? where id = ?"
	_, err = database.ExecContext(ctx, query, newName, applicationId)
	if err != nil {
		logger.Errorf("Failed to change name for application with id %d: %v", applicationId, err)
		return err
	}
	logger.Printf("Application with id %d renamed to %s successfully", applicationId, newName)
	return nil
}

// SetApplicationOwner updates the owner of an application.
// Only the current owner can change the owner.
func SetApplicationOwner(ctx context.Context, forge common.Forge, applicationId int64, newOwnerId int64, requesterOwnerId int64) error {
	database := forge.GetDB()
	logger := forge.GetLogger()

	err := VerifyApplicationOwner(ctx, forge, applicationId, requesterOwnerId)
	if err != nil {
		logger.Error(err)
		return err
	}

	query := "update application set ownerId = ? where id = ?"
	_, err = database.ExecContext(ctx, query, newOwnerId, applicationId)
	if err != nil {
		logger.Errorf("Failed to update owner for application with id %d: %v", applicationId, err)
		return err
	}
	logger.Printf("Application with id %d owner updated to %d successfully", applicationId, newOwnerId)
	return nil
}

// SetApplicationSourceCodeLink updates the repository link details for an application.
// Only the application owner can update the source code link.
func SetApplicationSourceCodeLink(ctx context.Context, forge common.Forge, applicationId int64, repoUrl, repoName, repoType, webhookIdentifier string, requesterId int64) error {
	database := forge.GetDB()
	logger := forge.GetLogger()

	err := VerifyApplicationOwner(ctx, forge, applicationId, requesterId)
	if err != nil {
		logger.Error(err)
	}

	query := "update application set repoUrl = ?, repoName = ?, repoType = ?, webhookIdentifier = ? where id = ?"
	_, err = database.ExecContext(ctx, query, repoUrl, repoName, repoType, webhookIdentifier, applicationId)
	if err != nil {
		logger.Errorf("Failed to update source code link for application with id %d: %v", applicationId, err)
		return err
	}
	logger.Printf("Application with id %d source code link updated successfully", applicationId)
	return nil
}

// VerifyApplicationOwner checks if the requester is the owner of the application.
// Returns an error if the requester is not the owner.
func VerifyApplicationOwner(ctx context.Context, forge common.Forge, applicationId int64, requesterOwnerId int64) error {
	// verify ownership
	database := forge.GetDB()
	logger := forge.GetLogger()
	var ownerId int64
	err := database.QueryRowContext(ctx, "select ownerId from application where id = ?", applicationId).Scan(&ownerId)
	if err != nil {
		logger.Errorf("Failed to verify application owner for application id %d: %v", applicationId, err)
		return err
	}
	if ownerId != requesterOwnerId {
		err := fmt.Errorf("unauthorized: only the owner can delete the application")
		return err
	}
	return nil
}

// ApplicationNameExists checks if an application name already exists in the database.
// Returns true if the name exists, false otherwise.
func ApplicationNameExists(ctx context.Context, forge common.Forge, applicationName string) (bool, error) {
	database := forge.GetDB()
	logger := forge.GetLogger()

	var count int
	err := database.QueryRowContext(ctx, "select count(*) from application where name = ?", applicationName).Scan(&count)
	if err != nil {
		logger.Errorf("Failed to check if application name exists: %v", err)
		return false, err
	}
	return count > 0, nil
}
