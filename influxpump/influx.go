package influxpump

import (
	"encoding/json"
	"fmt"

	ilpo "github.com/morfien101/influxLineProtocolOutput"
)

type JSONMeasurement struct {
	Name      string                 `json:"measurement"`
	Fields    map[string]interface{} `json:"fields"`
	Tags      map[string]string      `json:"tags"`
	TimeStamp int64                  `json:"timestamp"`
}

func convert(input []byte) (*ilpo.MetricContainer, error) {
	newMeasurement := &JSONMeasurement{}
	if err := json.Unmarshal(input, newMeasurement); err != nil {
		return nil, err
	}
	if newMeasurement.Name == "" {
		return nil, fmt.Errorf("Measurement has no name")
	}
	if len(newMeasurement.Fields) < 1 {
		return nil, fmt.Errorf("Measurement %s has no Fields", newMeasurement.Name)
	}
	if len(newMeasurement.Tags) < 1 {
		return nil, fmt.Errorf("Measurement %s has no Tags", newMeasurement.Name)
	}

	influxMeasurement := ilpo.New(newMeasurement.Name)
	influxMeasurement.Add(newMeasurement.Tags, newMeasurement.Fields)
	if newMeasurement.TimeStamp > 0 {
		influxMeasurement.SetTimeStamp(newMeasurement.TimeStamp)
	}

	return influxMeasurement, nil
}
