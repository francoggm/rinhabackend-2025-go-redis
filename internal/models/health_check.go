package models

type HealthCheck struct {
	IsFailing       bool  `json:"failing"`
	MinResponseTime int32 `json:"minResponseTime"`
}
