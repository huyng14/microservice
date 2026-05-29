package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"microservice/httpServerSvc"
	"microservice/models"
	mongodb "microservice/mongoDB"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (s *PersistenceServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Status: "OK"}, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	err := InitMongo("mongodb+srv://skylab:skylab@consultatantaimatch.ftecqos.mongodb.net/")
	if err != nil {
		log.Fatal("Failed to initialize MongoDB:", err)
	}

	// ==== Initialize services ====
	mongoSvc := &mongodb.MongoSvc{Client: Client}
	httpSvc := &httpServerSvc.HttpSvc{MongoSvc: mongoSvc}

	// // ==== Keepalive parameters ====
	// var kaPolicy = keepalive.EnforcementPolicy{
	// 	MinTime:             5 * time.Second, // Min time between pings from client
	// 	PermitWithoutStream: true,            // Allow keepalive pings even with no RPCs
	// }

	// var kaParams = keepalive.ServerParameters{
	// 	Time:                  10 * time.Second, // Ping clients every 10 seconds
	// 	Timeout:               3 * time.Second,  // Disconnect if no pong within 3 seconds
	// 	MaxConnectionIdle:     30 * time.Second, // Disconnect idle connections
	// 	MaxConnectionAge:      2 * time.Minute,  // Force reconnect every 2 minutes
	// 	MaxConnectionAgeGrace: 10 * time.Second, // Extra time after age expiration
	// }

	// Start gRPC server in a separate goroutine
	// go func() {
	// 	log.Println("Starting gRPC server for Persistence Service")
	// 	lis, err := net.Listen("tcp", ":9010")
	// 	if err != nil {
	// 		log.Fatalf("failed to listen: %v", err)
	// 	}

	// 	grpcServer := grpc.NewServer(
	// 		grpc.KeepaliveEnforcementPolicy(kaPolicy),
	// 		grpc.KeepaliveParams(kaParams),
	// 	)
	// 	pb.RegisterPersistenceServiceServer(grpcServer, &PersistenceServer{})

	// 	log.Println("Persistence Service gRPC server running at port 9000...")
	// 	if err := grpcServer.Serve(lis); err != nil {
	// 		log.Fatalf("failed to serve: %v", err)
	// 	}
	// }()

	// Start HTTP server (blocking call)
	go httpServer(httpSvc)
	// Prevent main from exiting
	select {}
}

func httpServer(svc *httpServerSvc.HttpSvc) {
	r := gin.Default()

	log.Println("Starting HTTP server on :9000")
	// Enable CORS so Vue (port 5173) can call Go (port 9000)
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://s3-demo-web-497559249788-ap-southeast-1-an.s3-website-ap-southeast-1.amazonaws.com",
			"http://localhost:5173"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type"},
	}))

	// Simple API endpoint
	r.GET("/api/hello", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello from Go backend 🚀",
		})
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	r.GET("/profiles/:id", svc.HandleGetPerson)
	r.GET("/listprofiles", svc.HandleListProfiles)
	r.POST("/profile", svc.HandleCreateProfile)
	r.PUT("/profile/:id", svc.HandleUpdateProfile)
	r.DELETE("/profile/:id", svc.HandleDeleteProfile)

	// ==== Job endpoints ====
	r.GET("/listjobs", svc.HandleListJobs)
	r.POST("/job", svc.HandleCreateJob)
	r.DELETE("/job/:id", svc.HandleDeleteJob)
	r.PUT("/job/:id", svc.HandleUpdateJob)

	r.Run("0.0.0.0:9000") // Run API on port 9000
}
