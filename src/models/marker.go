package models

import (
    "context"
    "database/sql"
    "time"
    
    "github.com/volatiletech/null/v8"
    "github.com/volatiletech/sqlboiler/v4/boil"
    "github.com/volatiletech/sqlboiler/v4/queries/qm"
)

// Marker represents a marker in the database
type Marker struct {
    ID             int       `boil:"id" json:"id" toml:"id" yaml:"id"`
    Name           string    `boil:"name" json:"name" toml:"name" yaml:"name"`
    ImagePath      string    `boil:"image_path" json:"image_path" toml:"image_path" yaml:"image_path"`
    RequiredPoints int       `boil:"required_points" json:"required_points" toml:"required_points" yaml:"required_points"`
    CreatedAt      time.Time `boil:"created_at" json:"created_at" toml:"created_at" yaml:"created_at"`
    UpdatedAt      time.Time `boil:"updated_at" json:"updated_at" toml:"updated_at" yaml:"updated_at"`
}

// TableName returns the name of the table
func (*Marker) TableName() string {
    return "markers"
}

// Insert inserts the Marker to the database
func (m *Marker) Insert(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
    return m.insert(ctx, exec, columns)
}

// Update updates the Marker in the database
func (m *Marker) Update(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
    return m.update(ctx, exec, columns)
}

// Delete deletes the Marker from the database
func (m *Marker) Delete(ctx context.Context, exec boil.ContextExecutor) error {
    return m.delete(ctx, exec)
}

// Markers retrieves all markers from the database
func Markers(mods ...qm.QueryMod) markerQuery {
    mods = append(mods, qm.From("markers"))
    return markerQuery{NewQuery(mods...)}
}

// GetAvailableMarkers retrieves markers that are available for the given points
func GetAvailableMarkers(ctx context.Context, exec boil.ContextExecutor, points int) ([]*Marker, error) {
    return Markers(
        qm.Where("required_points <= ?", points),
        qm.OrderBy("required_points ASC"),
    ).All(ctx, exec)
} 