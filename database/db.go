package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

const (
	UsersCollection = "users"
	RoomsCollection = "rooms"
	EdgeCollection  = "edgeCollection"
)

// cols is an array of the used collections
var cols = []string{UsersCollection, RoomsCollection}

// GetClient returns a new driver instance for the arango database
func GetClient() driver.Client {

	// Fetch environment variables
	uri, found := os.LookupEnv("ARANGO_URI")
	if !found {
		log.Fatal("ARANGO_URI environment variable not found")
	}
	username, found := os.LookupEnv("ARANGO_USERNAME")
	if !found {
		log.Fatal("ARANGO_USERNAME environment variable not found")
	}
	password, found := os.LookupEnv("ARANGO_PASSWORD")
	if !found {
		log.Fatal("ARANGO_PASSWORD environment variable not found")
	}

	// Connect to the database
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{uri},
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP connection: %v", err)

	}
	client, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication(username, password),
	})
	if err != nil {
		log.Fatalf("Failed to create arango client: %v", err)
	}

	// Wait for the database to be ready
	_, err = client.Version(context.TODO())
	for err != nil {
		fmt.Printf("Waiting for the database to be ready\n")
		time.Sleep(time.Second * 5)
		_, err = client.Version(context.TODO())
	}

	return client
}

// GetDatabase returns a database instance
// It creates the database or reuses an existing database if it already exists
func GetDatabase(name string) (db driver.Database) {
	client := GetClient()

	// Check if the database exists and create it if it does not
	dbExists, err := client.DatabaseExists(context.TODO(), name)
	if err != nil {
		log.Fatalf(err.Error())
	}

	if dbExists {
		fmt.Printf("%s db already exists\n", name)
		db, err = client.Database(context.TODO(), name)
		if err != nil {
			log.Fatalf("Failed to open %s database: %v", name, err)
		}
	} else {
		db, err = client.CreateDatabase(context.TODO(), name, nil)
		if err != nil {
			log.Fatalf("Failed to create %s database: %v", name, err)
		}
	}

	// Setup all needed collections before returning the database
	SetupCollections(db, cols)

	return db
}

// SetupCollections takes an array of collections name and creates them if they do not exist
func SetupCollections(db driver.Database, cols []string) {
	for _, collection := range cols {
		collExists, err := db.CollectionExists(context.TODO(), collection)
		if err != nil {
			log.Fatal(err)
		}

		if collExists {
			fmt.Printf("%s collection exists already\n", collection)
		} else {

			var col driver.Collection
			col, err = db.CreateCollection(context.TODO(), collection, nil)

			if err != nil {
				log.Fatalf("Failed to create %s collection: %v", collection, err)
			}

			fmt.Printf("Created collection '%s' in database '%s'\n", col.Name(), db.Name())
		}
	}
}

// GetGraph returns a graph instance
func GetGraph(db driver.Database, graphName string) (graph driver.Graph) {

	graphExists, err := db.GraphExists(context.TODO(), graphName)
	if err != nil {
		log.Fatalf(err.Error())
	}

	if graphExists {
		fmt.Printf("%s graph exists already\n", graphName)
		graph, err = db.Graph(context.TODO(), graphName)
		if err != nil {
			log.Fatalf(err.Error())
		}
	} else {
		graph = SetupGraph(db, graphName, cols)
	}

	return graph
}

// SetupGraph creates the edgeDefinition and the corresponding graph
func SetupGraph(db driver.Database, graphName string, cols []string) driver.Graph {

	var edgeDefinition driver.EdgeDefinition

	edgeDefinition.Collection = EdgeCollection

	// define a set of collections where an edge is going out...
	edgeDefinition.From = []string{UsersCollection}

	// repeat this for the collections where an edge is going into
	edgeDefinition.To = cols

	var options driver.CreateGraphOptions
	options.EdgeDefinitions = []driver.EdgeDefinition{edgeDefinition}

	graph, err := db.CreateGraphV2(context.TODO(), graphName, &options)
	if err != nil {
		fmt.Printf("Failed to create graph: %v", err)
	} else {
		fmt.Printf("Created graph '%s' in database '%s'\n", graphName, db.Name())
	}

	return graph
}
