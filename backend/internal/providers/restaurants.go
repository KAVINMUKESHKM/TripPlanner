package providers

type Restaurant struct {
	Name         string  `json:"name"`
	Amenity      string  `json:"amenity"`
	Cuisine      string  `json:"cuisine,omitempty"`
	Address      string  `json:"address,omitempty"`
	OpeningHours string  `json:"openingHours,omitempty"`
	Phone        string  `json:"phone,omitempty"`
	Website      string  `json:"website,omitempty"`
	SourceURL    string  `json:"sourceUrl"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	DistanceKM   float64 `json:"distanceKm"`
}
