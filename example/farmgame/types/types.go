// Package types defines the domain model and request/response types for the
// farm game example.
package types

// CropKind identifies a type of crop that can be planted.
type CropKind string

const (
	CropWheat  CropKind = "wheat"
	CropCorn   CropKind = "corn"
	CropCarrot CropKind = "carrot"
)

// PlotState represents the current state of a farm plot.
type PlotState string

const (
	PlotEmpty   PlotState = "empty"
	PlotGrowing PlotState = "growing"
	PlotReady   PlotState = "ready"
)

// CropInfo describes a crop type available for planting.
type CropInfo struct {
	Kind        CropKind `json:"kind"`
	Name        string   `json:"name"`
	GrowSeconds int64    `json:"grow_seconds"` // time to mature in seconds
}

// Plot holds the state of a single farm plot.
type Plot struct {
	ID        int       `json:"id"`
	State     PlotState `json:"state"`
	Crop      CropKind  `json:"crop,omitempty"`
	PlantedAt int64     `json:"planted_at,omitempty"` // Unix timestamp
	ReadyAt   int64     `json:"ready_at,omitempty"`   // Unix timestamp
}

// Farm is the full state of one player's farm.
type Farm struct {
	PlayerID string `json:"player_id"`
	Coins    int64  `json:"coins"`
	Plots    []Plot `json:"plots"`
}

// ----- request / response DTOs -----

// PlantRequest is the JSON body for the plant endpoint.
type PlantRequest struct {
	PlotID int      `json:"plot_id"`
	Crop   CropKind `json:"crop"`
}

// PlantResponse is returned after successfully planting a crop.
type PlantResponse struct {
	Plot Plot `json:"plot"`
}

// WaterResponse is returned after watering a plot.
type WaterResponse struct {
	Plot Plot `json:"plot"`
}

// HarvestResponse is returned after harvesting a ready plot.
type HarvestResponse struct {
	Plot   Plot  `json:"plot"`
	Earned int64 `json:"earned"` // coins earned from the harvest
}
