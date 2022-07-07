package models

import "github.com/arangodb/go-driver"

type Room struct {
	Name  string            `json:"name" binding:"required"`
	Owner driver.DocumentID `json:"owner"`
}

type RoomUpdate struct {
	Name  string `json:"name,omitempty"`
	Owner string `json:"owner,omitempty"`
}
