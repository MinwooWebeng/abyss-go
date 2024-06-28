package anet

import (
	"encoding/json"

	"github.com/google/uuid"
)

type WorldJsonBlob struct {
	URL  string
	UUID string
}

type World struct {
	URL       string
	UUID      []byte
	jsonbytes []byte
}

func NewWorld(url string) *World {
	result := new(World)
	result.URL = url
	uuid := uuid.New().String()
	result.UUID = []byte(uuid)
	result.jsonbytes, _ = json.Marshal(WorldJsonBlob{URL: url, UUID: uuid})
	return result
}

func ParseWorldJson(data []byte) (*World, error) {
	parseres := new(WorldJsonBlob)
	err := json.Unmarshal(data, parseres)
	if err != nil {
		return nil, err
	}
	result := new(World)
	result.URL = parseres.URL
	result.UUID = []byte(parseres.UUID)
	result.jsonbytes = data
	return result, nil
}

func (w *World) GetJsonBytes() []byte { return w.jsonbytes }
func (w *World) GetUUID() string      { return string(w.UUID) }
func (w *World) GetUUIDBytes() []byte { return w.UUID }
