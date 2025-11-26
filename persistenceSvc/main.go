package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"microservice/persistenceSvc/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "microservice/template/personpb"
)

var Client *mongo.Client

type PersistenceServer struct {
	pb.UnimplementedPersistenceServiceServer
}

func InitMongo(uri string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return err
	}

	Client = client
	log.Println("Connected to MongoDB!")
	return nil
}

func InsertUser(databaseName, collectionName string, person models.Person) (*mongo.InsertOneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := Client.Database(databaseName).Collection(collectionName)

	result, err := collection.InsertOne(ctx, person)
	if err != nil {
		return nil, err
	}
	log.Println("Inserted user with ID:", result.InsertedID)

	return result, nil
}

func (s *PersistenceServer) PostPerson(ctx context.Context, req *pb.PostPersonRequest) (*pb.PostPersonResponse, error) {
	log.Println("(s *PersistenceServer) PostPerson. Req: ", req.Person)
	_, err := InsertUser("microservice", "curriculumVitaes", models.Person{
		ID:         req.Person.Id,
		Name:       req.Person.Name,
		Email:      req.Person.Email,
		Experience: req.Person.Experience,
		Skills:     req.Person.Skills,
		CreatedAt:  time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return &pb.PostPersonResponse{Id: req.Person.Id}, nil
}

func GetUserByID(databaseName, collectionName string, id uint64) (*models.Person, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := Client.Database(databaseName).Collection(collectionName)

	var person models.Person
	err := collection.FindOne(ctx, bson.M{"id": id}).Decode(&person)
	if err != nil {
		return nil, err
	}
	fmt.Println("Retrieved user at DB, CreatedAt:", person.CreatedAt)

	return &person, nil
}

func (s *PersistenceServer) GetPerson(ctx context.Context, req *pb.GetPersonRequest) (*pb.GetPersonResponse, error) {
	person, err := GetUserByID("microservice", "curriculumVitaes", req.Id)
	if err != nil {
		return nil, err
	}
	personRes := &pb.Person{
		Id:    req.Id,
		Name:  person.Name,
		Email: person.Email,
		// Experience: req.Experience,
		CreatedAt: timestamppb.New(person.CreatedAt),
	}
	return &pb.GetPersonResponse{Person: personRes}, nil
}

func (s *PersistenceServer) ListPersons(empty *emptypb.Empty, stream pb.PersistenceService_ListPersonsServer) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := Client.Database("microservice").Collection("curriculumVitaes")

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var person models.Person
		if err := cursor.Decode(&person); err != nil {
			return err
		}

		personRes := &pb.Person{
			Id:         person.ID,
			Name:       person.Name,
			Email:      person.Email,
			Experience: person.Experience,
			Skills:     person.Skills,
			CreatedAt:  timestamppb.New(person.CreatedAt),
		}

		if err := stream.Send(&pb.GetPersonResponse{Person: personRes}); err != nil {
			return err
		}
	}

	if err := cursor.Err(); err != nil {
		return err
	}

	return nil
}

func main() {
	err := InitMongo("mongodb+srv://skylab:skylab@consultatantaimatch.ftecqos.mongodb.net/")
	if err != nil {
		log.Fatal("Failed to initialize MongoDB:", err)
	}

	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPersistenceServiceServer(grpcServer, &PersistenceServer{})

	log.Println("Product Service gRPC server running at port 9000...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	// user := models.User{
	// 	ID:        1,
	// 	Name:      "Alice",
	// 	Email:     "AliceAtBorderland.com",
	// 	CreatedAt: time.Now(),
	// }

	// result, err := InsertUser("microservice", "curriculumVitaes", user)
	// if err != nil {
	// 	log.Fatal("Failed to insert user:", err)
	// }

	// fmt.Println("Inserted user with ID:", result.InsertedID)

	// retrievedUser, err := GetUserByID("microservice", "curriculumVitaes", 1)
	// if err != nil {
	// 	log.Fatal("Failed to get user:", err)
	// }

	// fmt.Printf("Retrieved User: %+v\n", retrievedUser)
}
