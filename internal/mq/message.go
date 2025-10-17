package mq

import "time"

type GpuMetric struct {
	Timestamp  time.Time         `json:"timestamp"`
	MetricName string            `json:"metric_name"`
	GpuID      string            `json:"gpu_id"`
	UUID       string            `json:"uuid"`
	Hostname   string            `json:"hostname"`
	ModelName  string            `json:"model_name"`
	Value      float64           `json:"value"`
	Labels     map[string]string `json:"labels"`
}

type Message struct {
	Payload []byte
	Ack     func()
}
