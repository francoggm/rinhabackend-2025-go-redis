package models

type HealthCheck struct {
	IsFailing       bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}
