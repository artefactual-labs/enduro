package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	ssclient "go.artefactual.dev/ssclient"
	"go.artefactual.dev/ssclient/kiota/models"
)

var ErrStoragePackageNotFound = errors.New("storage package not found")

type StoragePackage struct {
	UUID            string
	Status          string
	StoredDate      *time.Time
	CurrentFullPath string
	CurrentLocation StorageLocation
	Replicas        []StorageReplica
}

type StorageLocation struct {
	UUID        string
	Description string
	Purpose     string
}

type StorageReplica struct {
	UUID            string
	Status          string
	StoredDate      *time.Time
	CurrentFullPath string
	CurrentLocation StorageLocation
}

func (p *Pipeline) GetStoragePackage(ctx context.Context, aipID string) (*StoragePackage, error) {
	return p.getStoragePackage(ctx, aipID, true)
}

// GetStoragePackageForReconciliation returns only the storage details needed
// by the configured recovery policy. Primary-only policies do not need replica
// hydration, so unrelated replica problems should not block reconciliation.
func (p *Pipeline) GetStoragePackageForReconciliation(ctx context.Context, aipID string, cfg RecoveryConfig) (*StoragePackage, error) {
	return p.getStoragePackage(ctx, aipID, len(cfg.RequiredLocations) > 0)
}

func (p *Pipeline) getStoragePackage(ctx context.Context, aipID string, loadReplicas bool) (*StoragePackage, error) {
	if p.storageServiceClient == nil {
		return nil, errors.New("storage service client is not configured")
	}

	packageID, err := parseUUID(aipID)
	if err != nil {
		return nil, fmt.Errorf("invalid storage package UUID %q: %w", aipID, err)
	}

	pkg, err := p.storageServiceClient.Packages().Get(ctx, packageID)
	if err != nil {
		if ssclient.IsNotFound(err) {
			return nil, ErrStoragePackageNotFound
		}
		return nil, fmt.Errorf("error querying storage service: %w", err)
	}
	if pkg == nil {
		return nil, errors.New("storage service returned an empty package response")
	}

	return p.normalizeStoragePackage(ctx, pkg, loadReplicas)
}

func (p *Pipeline) DownloadStoragePackage(ctx context.Context, aipID string) (*ssclient.FileStream, error) {
	if p.storageServiceClient == nil {
		return nil, errors.New("storage service client is not configured")
	}

	packageID, err := parseUUID(aipID)
	if err != nil {
		return nil, fmt.Errorf("invalid storage package UUID %q: %w", aipID, err)
	}

	stream, err := p.storageServiceClient.Packages().DownloadPackage(ctx, packageID)
	if err != nil {
		if ssclient.IsNotFound(err) {
			return nil, ErrStoragePackageNotFound
		}
		return nil, fmt.Errorf("error downloading storage package: %w", err)
	}
	if stream == nil {
		return nil, errors.New("storage service returned an empty download response")
	}

	return stream, nil
}

func (p *Pipeline) normalizeStoragePackage(ctx context.Context, pkg *models.PackageEscaped, loadReplicas bool) (*StoragePackage, error) {
	out := &StoragePackage{
		UUID:            derefUUID(pkg.GetUuid()),
		Status:          derefString(pkg.GetStatus()),
		StoredDate:      pkg.GetStoredDate(),
		CurrentFullPath: derefString(pkg.GetCurrentFullPath()),
		Replicas:        make([]StorageReplica, 0, len(pkg.GetReplicas())),
	}
	if pkg.GetCurrentLocation() != nil {
		locationID, err := parseResourceUUID(*pkg.GetCurrentLocation(), "location")
		if err != nil {
			return nil, fmt.Errorf("parse storage service package location: %w", err)
		}
		out.CurrentLocation = StorageLocation{UUID: locationID}
	}

	if !loadReplicas {
		return out, nil
	}

	for _, replicaURI := range pkg.GetReplicas() {
		replicaID, err := parseResourceUUID(replicaURI, "file")
		if err != nil {
			return nil, fmt.Errorf("parse storage service replica URI: %w", err)
		}

		replicaUUID, err := parseUUID(replicaID)
		if err != nil {
			return nil, fmt.Errorf("parse storage service replica URI: invalid UUID %q: %w", replicaID, err)
		}

		replicaPkg, err := p.storageServiceClient.Packages().Get(ctx, replicaUUID)
		if err != nil {
			return nil, fmt.Errorf("error querying storage service replica %q: %w", replicaID, err)
		}
		if replicaPkg == nil {
			return nil, fmt.Errorf("storage service replica %q was not found", replicaID)
		}

		out.Replicas = append(out.Replicas, StorageReplica{
			UUID:            derefUUID(replicaPkg.GetUuid()),
			Status:          derefString(replicaPkg.GetStatus()),
			StoredDate:      replicaPkg.GetStoredDate(),
			CurrentFullPath: derefString(replicaPkg.GetCurrentFullPath()),
		})
		if replicaPkg.GetCurrentLocation() != nil {
			locationID, err := parseResourceUUID(*replicaPkg.GetCurrentLocation(), "location")
			if err != nil {
				return nil, fmt.Errorf("parse storage service replica %q location: %w", replicaID, err)
			}
			out.Replicas[len(out.Replicas)-1].CurrentLocation = StorageLocation{UUID: locationID}
		}
	}

	return out, nil
}

func parseResourceUUID(resourceURI, expectedResource string) (string, error) {
	if strings.TrimSpace(resourceURI) == "" {
		return "", nil
	}

	resource, uuid, err := ssclient.ParseResourceURI(resourceURI)
	if err != nil {
		return "", err
	}
	if expectedResource != "" && resource != expectedResource {
		return "", fmt.Errorf("unexpected resource %q", resource)
	}

	return uuid, nil
}

func parseUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(value))
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func derefUUID(value *uuid.UUID) string {
	if value == nil {
		return ""
	}

	return value.String()
}
