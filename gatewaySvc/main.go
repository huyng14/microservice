package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	loggingpb "microservice/template/logpb"
	pb "microservice/template/personpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

var grpcClient pb.PersistenceServiceClient
var logClient loggingpb.LogServiceClient

func handleGetPerson(w http.ResponseWriter, r *http.Request) {
	req := &pb.GetPersonRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Call gRPC method GetPerson
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := grpcClient.GetPerson(ctx, &pb.GetPersonRequest{Id: req.Id})
	if err != nil {
		log.Printf("error calling GetPerson: %v", err)
		http.Error(w, "Error fetching person", http.StatusInternalServerError)
		return
	}

	log.Printf("User Service received person: %+v", res.Person)

	fmt.Fprint(w, res.GetPerson())
}

func handlePostPerson(w http.ResponseWriter, r *http.Request) {
	req := &pb.PostPersonRequest{}
	err := json.NewDecoder(r.Body).Decode(&req.Person)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	log.Println("Print PostPersonRequest: ", req.Person)
	// Call gRPC method PostPerson
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := grpcClient.PostPerson(ctx, req)
	if err != nil {
		log.Printf("error calling PostPerson: %v", err)
		http.Error(w, "Error posting person", http.StatusInternalServerError)
		return
	}

	log.Printf("User Service posted person with ID: %d", res.Id)

	fmt.Fprintf(w, "Person created with ID: %d", res.Id)
	// http.Error(w, "POST method not implemented yet", http.StatusNotImplemented)
}

func handleProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetPerson(w, r)
	case http.MethodPost:
		handlePostPerson(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleListProfiles(w http.ResponseWriter, r *http.Request) {
	// Only handle GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	stream, err := grpcClient.ListPersons(ctx, &emptypb.Empty{})
	if err != nil {
		log.Printf("error calling ListPersons: %v", err)
		http.Error(w, "Error listing persons", http.StatusInternalServerError)
		return
	}

	var persons []pb.Person
	for {
		personRes, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("error receiving from stream: %v", err)
			http.Error(w, fmt.Sprintf("error receiving from stream: %v", err), http.StatusInternalServerError)
			return
		}
		persons = append(persons, *personRes.Person)
	}

	responseData, err := json.Marshal(persons)
	if err != nil {
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseData)
}

func firstPage(w http.ResponseWriter, r *http.Request) {
	// Only handle GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		// Call gRPC logging method
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var buf bytes.Buffer
		logger := log.New(&buf, "", log.LstdFlags|log.Lshortfile)
		logger.Println("firstPage: method not allowed")
		stackTrace := buf.String()
		_, err := logClient.LogEvent(ctx, &loggingpb.LogEventRequest{
			LogEntry: &loggingpb.LogEntry{
				Service:    "gatewaySvc",
				Level:      "ERROR",
				Message:    fmt.Sprintf("Method not allowed", http.StatusMethodNotAllowed),
				TraceId:    "trace001",
				SpanId:     "span001",
				StackTrace: stackTrace,
			},
		})
		if err != nil {
			log.Printf("error calling LogEvent: %v", err)
		}
		return
	}

	// Format person data as JSON-like output
	response := fmt.Sprintf("Hello stranger. \nTime: %s", time.Now())

	fmt.Fprint(w, response)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	go func() {
		log.Println("Starting HTTP server on :8080")
		http.HandleFunc("/", firstPage)
		http.HandleFunc("/profiles", handleProfiles)
		http.HandleFunc("/listprofiles", handleListProfiles)

		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	go func() {
		log.Println("Starting gRPC client connection to Persistence Service")
		// Connect to Persistence Service
		conn, err := grpc.NewClient("localhost:9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("failed to connect to persistence service: %v", err)
		}

		grpcClient = pb.NewPersistenceServiceClient(conn)
		// defer conn.Close()

	}()

	go func() {
		log.Println("Starting gRPC client connection to Logging Service")
		// Connect to Logging Service
		conn, err := grpc.NewClient("localhost:6514", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("failed to connect to persistence service: %v", err)
		}

		logClient = loggingpb.NewLogServiceClient(conn)
		// defer conn.Close()
	}()
	// Prevent main from exiting
	select {}
}
