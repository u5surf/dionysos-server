//nolint:typecheck
package routes

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Brawdunoir/dionysos-server/database"
	"github.com/Brawdunoir/dionysos-server/models"
	"github.com/Brawdunoir/dionysos-server/utils"
	"github.com/arangodb/go-driver"
	"github.com/gin-gonic/gin"
)

type Connection struct {
	From string `json:"_from"`
	To   string `json:"_to"`
}

type RoomRequest struct {
	Room    models.Room `json:"room"`
	OwnerID string      `json:"owner" binding:"required"`
}

type ConnectUserRequest struct {
	UserID string `json:"user" binding:"required"`
}

// CreateRoom creates a room in the aganro database
func CreateRoom(c *gin.Context) {
	var roomRequest RoomRequest
	var owner models.User

	ctx, cancelCtx := context.WithTimeout(c, 1000*time.Millisecond)
	defer cancelCtx()

	if err := c.ShouldBindJSON(&roomRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Printf("Failed to bind JSON: %v", err)
		return
	}

	// Get Owner
	col, err := db.Collection(ctx, database.UsersCollection)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Cannot access database collection"})
		log.Printf("Failed to access collection: %v", err)
		return
	}

	metaOwner, err := col.ReadDocument(ctx, roomRequest.OwnerID, &owner)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Owner not found"})
		log.Printf("Failed to find document: %v", err)
		return
	}

	// Create Room
	col, err = db.Collection(ctx, database.RoomsCollection)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Cannot access database collection"})
		log.Printf("Failed to access collection: %v", err)
		return
	}

	roomRequest.Room.Owner = metaOwner.ID
	metaRoom, err := col.CreateDocument(ctx, roomRequest.Room)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Room not created"})
		log.Printf("Failed to create room: %v", err)
		return
	}

	// add edge
	edgeCollection, _, err := graph.EdgeCollection(ctx, database.EdgeCollection)
	if err != nil {
		log.Fatalf("Failed to select edge collection: %v", err)
	}

	edge := Connection{From: string(metaOwner.ID), To: string(metaRoom.ID)}
	fmt.Println(edge)
	_, err = edgeCollection.CreateDocument(ctx, edge)
	if err != nil {
		log.Fatalf("Failed to create edge document: %v", err)
	}

	c.JSON(http.StatusCreated, gin.H{"id": metaRoom.Key})
}

// GetRoom returns a room from the aganro database
func GetRoom(c *gin.Context) {
	var result models.Room

	ctx, cancelCtx := context.WithTimeout(c, 1000*time.Millisecond)
	defer cancelCtx()

	id := c.Param("id")

	col, err := db.Collection(ctx, database.RoomsCollection)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Cannot access database collection"})
		log.Printf("Failed to access collection: %v", err)
		return
	}

	_, err = col.ReadDocument(ctx, id, &result)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Room not found"})
		log.Printf("Failed to get document: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"room": result})
}

// UpdateRoom updates a room in the aganro database
func UpdateRoom(c *gin.Context) {
	var roomUpdate models.RoomUpdate
	var patchedRoom models.Room

	ctx, cancelCtx := context.WithTimeout(c, 1000*time.Millisecond)
	defer cancelCtx()

	id := c.Param("id")

	if err := c.ShouldBindJSON(&roomUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Printf("Failed to bind JSON: %v", err)
		return
	}

	if isNil := roomUpdate == (models.RoomUpdate{}); isNil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No data to update"})
		log.Printf("Failed to bind JSON: No data to update")
		return
	}

	col, err := db.Collection(ctx, database.RoomsCollection)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Cannot access database collection"})
		log.Printf("Failed to access collection: %v", err)
		return
	}

	connnectedUsers := utils.GetUsersFromRoom(id, db, graph)
	fmt.Println("Connected users: ", connnectedUsers)

	// Check if roomUpdate.Owner is in connnectedUsers
	found := false

	for _, v := range connnectedUsers {
		// check if the strings match
		if string(v) == roomUpdate.Owner {
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not modified"})
		log.Printf("Failed to modify room: %v, new owner %s doesn't seem to be connected", err, roomUpdate.Owner)
		return
	} else {
		_, err = col.UpdateDocument(driver.WithReturnNew(ctx, &patchedRoom), id, roomUpdate)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Room not modified"})
			log.Printf("Failed to modify room: %v", err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"room": patchedRoom})
}

// DeleteRoom deletes a room in the aganro database
func DeleteRoom(c *gin.Context) {
	ctx, cancelCtx := context.WithTimeout(c, 1000*time.Millisecond)
	defer cancelCtx()

	id := c.Param("id")

	col, err := db.Collection(ctx, database.RoomsCollection)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Cannot access database collection"})
		log.Printf("Failed to access collection: %v", err)
		return
	}

	_, err = col.RemoveDocument(ctx, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Room not deleted"})
		log.Printf("Failed to delete document: %v", err)
		return
	}

	c.JSON(http.StatusOK, nil)
}

// ConnectUserToRoom creates a room in the aganro database
func ConnectUserToRoom(c *gin.Context) {
	var userRequest ConnectUserRequest
	var user models.User
	var room models.Room

	ctx, cancelCtx := context.WithTimeout(c, 1000*time.Millisecond)
	defer cancelCtx()

	if err := c.ShouldBindJSON(&userRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Printf("Failed to bind JSON: %v", err)
		return
	}

	id := c.Param("id")

	colRooms, err := db.Collection(ctx, database.RoomsCollection)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot access database collection"})
		log.Printf("Failed to access collection: %v", err)
		return
	}

	metaRoom, err := colRooms.ReadDocument(ctx, id, &room)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		log.Printf("Failed to get document: %v", err)
		return
	}

	colUsers, err := db.Collection(ctx, database.UsersCollection)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot access database collection"})
		log.Printf("Failed to access collection: %v", err)
		return
	}
	metaUser, err := colUsers.ReadDocument(ctx, userRequest.UserID, &user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		log.Printf("Failed to find document: %v", err)
		return
	}

	// add edge
	edgeCollection, _, err := graph.EdgeCollection(ctx, database.EdgeCollection)
	if err != nil {
		log.Fatalf("Failed to select edge collection: %v", err)
	}

	edge := Connection{From: string(metaUser.ID), To: string(metaRoom.ID)}
	fmt.Println(edge)
	_, err = edgeCollection.CreateDocument(ctx, edge)
	if err != nil {
		log.Fatalf("Failed to create edge document: %v", err)
	}

	c.JSON(http.StatusCreated, gin.H{"id": metaUser.ID})
}
