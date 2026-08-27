package repository

import (
	"context"
	"time"

	"github.com/vance1852/waste-dispatch/internal/domain"
)

// UserFilter holds optional filters for listing users.
type UserFilter struct {
	Role   domain.Role
	Status domain.UserStatus
	Limit  int
	Offset int
}

// UserRepository defines persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter UserFilter) ([]*domain.User, int, error)
}

// VehicleFilter holds optional filters for listing vehicles.
type VehicleFilter struct {
	Status domain.VehicleStatus
	Type   domain.VehicleType
	Limit  int
	Offset int
}

// VehicleRepository defines persistence operations for vehicles.
type VehicleRepository interface {
	Create(ctx context.Context, vehicle *domain.Vehicle) error
	GetByID(ctx context.Context, id string) (*domain.Vehicle, error)
	GetByPlate(ctx context.Context, plate string) (*domain.Vehicle, error)
	UpdateWithVersion(ctx context.Context, vehicle *domain.Vehicle) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter VehicleFilter) ([]*domain.Vehicle, int, error)
	ListAvailable(ctx context.Context) ([]*domain.Vehicle, error)
}

// PointFilter holds optional filters for listing collection points.
type PointFilter struct {
	Status   domain.PointStatus
	District string
	Limit    int
	Offset   int
}

// PointRepository defines persistence operations for collection points.
type PointRepository interface {
	Create(ctx context.Context, point *domain.CollectionPoint) error
	GetByID(ctx context.Context, id string) (*domain.CollectionPoint, error)
	UpdateWithVersion(ctx context.Context, point *domain.CollectionPoint) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter PointFilter) ([]*domain.CollectionPoint, int, error)
	ListOverThreshold(ctx context.Context, threshold float64) ([]*domain.CollectionPoint, error)
}

// TaskFilter holds optional filters for listing collection tasks.
type TaskFilter struct {
	Status    domain.TaskStatus
	VehicleID string
	DriverID  string
	PointID   string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

// TaskRepository defines persistence operations for collection tasks.
type TaskRepository interface {
	Create(ctx context.Context, task *domain.CollectionTask) error
	GetByID(ctx context.Context, id string) (*domain.CollectionTask, error)
	UpdateWithVersion(ctx context.Context, task *domain.CollectionTask) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter TaskFilter) ([]*domain.CollectionTask, int, error)
	ListStaleInProgress(ctx context.Context, olderThan time.Time) ([]*domain.CollectionTask, error)
}

// IncidentFilter holds optional filters for listing incidents.
type IncidentFilter struct {
	Status   domain.IncidentStatus
	Type     domain.IncidentType
	PointID  string
	VehicleID string
	Limit    int
	Offset   int
}

// IncidentRepository defines persistence operations for incidents.
type IncidentRepository interface {
	Create(ctx context.Context, incident *domain.Incident) error
	GetByID(ctx context.Context, id string) (*domain.Incident, error)
	Update(ctx context.Context, incident *domain.Incident) error
	List(ctx context.Context, filter IncidentFilter) ([]*domain.Incident, int, error)
	// CountActiveForPoint reports how many incidents of a type are still awaiting
	// closure for a collection point.
	CountActiveForPoint(ctx context.Context, pointID string, incidentType domain.IncidentType) (int, error)
}

// CreditRepository defines persistence operations for resident credits.
type CreditRepository interface {
	GetOrCreateByResidentID(ctx context.Context, residentID string) (*domain.ResidentCredit, error)
	UpdateBalance(ctx context.Context, credit *domain.ResidentCredit) error
	RecordTransaction(ctx context.Context, tx *domain.CreditTransaction) error
	GetTransactionByIdempotencyKey(ctx context.Context, key string) (*domain.CreditTransaction, error)
	ListTransactions(ctx context.Context, residentID string, limit, offset int) ([]*domain.CreditTransaction, int, error)
}

// SessionRepository defines persistence operations for user sessions.
type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error)
	GetByID(ctx context.Context, id string) (*domain.Session, error)
	UpdateLastSeen(ctx context.Context, id string) error
	Revoke(ctx context.Context, id string) error
	RevokeAllForUser(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// AuditRepository defines persistence operations for audit logs.
type AuditRepository interface {
	Record(ctx context.Context, log *domain.AuditLog) error
	List(ctx context.Context, entityType, entityID string, limit, offset int) ([]*domain.AuditLog, int, error)
}
