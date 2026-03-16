package types

// --- User types ---

// RegisterRequest is the request body for user registration.
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterResponse is the response body after successful registration.
type RegisterResponse struct {
	UserId   string `json:"userId"`
	Username string `json:"username"`
}

// LoginRequest is the request body for user login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is returned after a successful login and contains a JWT token.
type LoginResponse struct {
	UserId      string `json:"userId"`
	Username    string `json:"username"`
	AccessToken string `json:"accessToken"`
	ExpireAt    int64  `json:"expireAt"`
}

// --- Farm types ---

// PlantRequest asks the server to plant a crop in the given plot.
type PlantRequest struct {
	PlotId   int    `json:"plotId"`
	CropType string `json:"cropType"`
}

// PlantResponse confirms the crop was planted.
type PlantResponse struct {
	PlotId     int    `json:"plotId"`
	CropType   string `json:"cropType"`
	PlantedAt  int64  `json:"plantedAt"`
	HarvestAt  int64  `json:"harvestAt"`
}

// WaterRequest asks the server to water the crop in the given plot.
type WaterRequest struct {
	PlotId int `json:"plotId"`
}

// WaterResponse confirms the plot was watered.
type WaterResponse struct {
	PlotId    int   `json:"plotId"`
	WateredAt int64 `json:"wateredAt"`
}

// HarvestRequest asks the server to harvest the crop from a plot.
type HarvestRequest struct {
	PlotId int `json:"plotId"`
}

// HarvestResponse contains the yield from the harvest.
type HarvestResponse struct {
	PlotId   int    `json:"plotId"`
	CropType string `json:"cropType"`
	Yield    int    `json:"yield"`
}

// StatusResponse describes the current state of all farm plots.
type StatusResponse struct {
	UserId string      `json:"userId"`
	Plots  []PlotState `json:"plots"`
}

// PlotState describes one farm plot.
type PlotState struct {
	PlotId    int    `json:"plotId"`
	CropType  string `json:"cropType"`
	PlantedAt int64  `json:"plantedAt"`
	HarvestAt int64  `json:"harvestAt"`
	Watered   bool   `json:"watered"`
	Ready     bool   `json:"ready"`
}
