package domain

import (
	"errors"
	"time"
)

// PointStatus represents the operational state of a collection point.
type PointStatus string

const (
	PointStatusActive    PointStatus = "active"
	PointStatusFull      PointStatus = "full"
	PointStatusSuspended PointStatus = "suspended"
	PointStatusClosed    PointStatus = "closed"
)

// IsValid checks if a PointStatus is known.
func (s PointStatus) IsValid() bool {
	switch s {
	case PointStatusActive, PointStatusFull, PointStatusSuspended, PointStatusClosed:
		return true
	}
	return false
}

// WasteCategory classifies the type of waste accepted at a collection point.
type WasteCategory string

const (
	WasteCategoryGeneral    WasteCategory = "general"
	WasteCategoryRecyclable WasteCategory = "recyclable"
	WasteCategoryHazardous  WasteCategory = "hazardous"
	WasteCategoryOrganic    WasteCategory = "organic"
	WasteCategoryBulky      WasteCategory = "bulky"
)

// CollectionPoint represents a physical waste collection location.
type CollectionPoint struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Address         string        `json:"address"`
	Latitude        float64       `json:"latitude"`
	Longitude       float64       `json:"longitude"`
	District        string        `json:"district"`
	WasteCategories []WasteCategory `json:"waste_categories"`
	CapacityKg      float64       `json:"capacity_kg"`
	CurrentLoadKg   float64       `json:"current_load_kg"`
	Status          PointStatus   `json:"status"`
	Notes           string        `json:"notes"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Version         int           `json:"version"`
}

// FillRatio returns current load as a fraction of capacity (0.0 – 1.0).
func (p *CollectionPoint) FillRatio() float64 {
	if p.CapacityKg <= 0 {
		return 0
	}
	return p.CurrentLoadKg / p.CapacityKg
}

// IsOverThreshold returns true when fill ratio exceeds the given threshold.
func (p *CollectionPoint) IsOverThreshold(threshold float64) bool {
	return p.FillRatio() >= threshold
}

// UpdateStatus recalculates status based on fill level.
func (p *CollectionPoint) UpdateStatus() {
	if p.Status == PointStatusSuspended || p.Status == PointStatusClosed {
		return
	}
	if p.FillRatio() >= 1.0 {
		p.Status = PointStatusFull
	} else {
		p.Status = PointStatusActive
	}
}

// Errors related to collection points.
var (
	ErrPointNotFound      = errors.New("collection point not found")
	ErrPointAlreadyExists = errors.New("collection point already exists")
	ErrPointVersionConflict = errors.New("collection point was modified by another request, please retry")
)
