package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	loggingpb "microservice/template/logpb"
	pb "microservice/template/personpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

var persistenceClient pb.PersistenceServiceClient
var logClient loggingpb.LogServiceClient
var buf bytes.Buffer
var logger = log.New(&buf, "", log.LstdFlags|log.Lshortfile)

func handleGetPerson(w http.ResponseWriter, r *http.Request) {
	req := &pb.GetPersonRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		// Call gRPC logging method
		// Only keep the latest log entry
		buf.Reset()
		// Write log to the buffer
		logger.Println("Invalid request body", http.StatusBadRequest)
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintf("Invalid request body", http.StatusBadRequest),
		)
		return
	}

	// Call gRPC method GetPerson
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := persistenceClient.GetPerson(ctx, &pb.GetPersonRequest{Id: req.Id})
	if err != nil {
		log.Printf("error calling GetPerson: %v", err)
		http.Error(w, "Error fetching person", http.StatusInternalServerError)

		// Call gRPC logging method
		// Only keep the latest log entry
		buf.Reset()
		// Write log to the buffer
		logger.Printf("error calling GetPerson: %v", err)
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintf("Error fetching person", http.StatusInternalServerError),
		)
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

		// Call gRPC logging method
		// Only keep the latest log entry
		buf.Reset()
		// Write log to the buffer
		logger.Println("Invalid request body", http.StatusBadRequest)
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintf("Invalid request body", http.StatusBadRequest),
		)
		return
	}
	log.Println("Print PostPersonRequest: ", req.Person)
	// Call gRPC method PostPerson
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := persistenceClient.PostPerson(ctx, req)
	if err != nil {
		log.Printf("error calling PostPerson: %v", err)
		http.Error(w, "Error posting person", http.StatusInternalServerError)

		// Call gRPC logging method
		// Only keep the latest log entry
		buf.Reset()
		// Write log to the buffer
		logger.Printf("error calling PostPerson: %v", err)
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintf("Error posting person", http.StatusInternalServerError),
		)
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

		// Call gRPC logging method
		// Only keep the latest log entry
		buf.Reset()
		// Write log to the buffer
		logger.Println("Method not allowed", http.StatusMethodNotAllowed)
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintf("Method not allowed", http.StatusMethodNotAllowed),
		)
	}
}

func handleListProfiles(w http.ResponseWriter, r *http.Request) {
	// Only handle GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		// Call gRPC logging method
		// Only keep the latest log entry
		buf.Reset()
		// Write log to the buffer
		logger.Printf("Method not allowed", http.StatusMethodNotAllowed)
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintf("Method not allowed", http.StatusMethodNotAllowed),
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	stream, err := persistenceClient.ListPersons(ctx, &emptypb.Empty{})
	if err != nil {
		log.Printf("error calling ListPersons: %v", err)
		http.Error(w, "Error listing persons", http.StatusInternalServerError)

		// Call gRPC logging method
		// Only keep the latest log entry
		buf.Reset()
		// Write log to the buffer
		logger.Printf("error calling ListPersons: %v", err)
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintf("Error listing persons", http.StatusInternalServerError),
		)
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

			// Call gRPC logging method
			// Only keep the latest log entry
			buf.Reset()
			// Write log to the buffer
			logger.Printf("error receiving from stream: %v", err)
			SendLogMessage(
				logger,
				&buf,
				"ERROR",
				fmt.Sprintf("error receiving from stream: %v, %v", err, http.StatusInternalServerError),
			)
			return
		}
		persons = append(persons, *personRes.Person)
	}

	responseData, err := json.Marshal(persons)
	if err != nil {
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)

		// Call gRPC logging method
		// Only keep the latest log entry
		buf.Reset()
		// Write log to the buffer
		logger.Println("Error marshalling response", http.StatusInternalServerError)
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintf("Error marshalling response", http.StatusInternalServerError),
		)
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
		// Only keep the latest log entry
		buf.Reset()
		// Write log to the buffer
		logger.Println("firstPage: method not allowed", http.StatusMethodNotAllowed)
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintf("firstPage: method not allowed", http.StatusMethodNotAllowed),
		)

		return
	}

	// Format person data as JSON-like output
	response := fmt.Sprintf("Hello stranger. \nTime: %s", time.Now())

	fmt.Fprint(w, response)
}

func SendLogMessage(
	logger *log.Logger,
	buf *bytes.Buffer,
	level string,
	message string,
) error {
	// Call gRPC logging method
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Extract stack trace from buffer
	stackTrace := buf.String()

	// Send log via gRPC
	_, err := logClient.LogEvent(ctx, &loggingpb.LogEventRequest{
		LogEntry: &loggingpb.LogEntry{
			Service:    "gatewaySvc",
			Level:      level,
			Message:    message,
			StackTrace: stackTrace,
		},
	})

	if err != nil {
		log.Printf("error calling LogEvent: %v", err)
		return err
	}

	return nil
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

		retryPolicy := backoff.Config{
			BaseDelay:  1 * time.Second,  // initial backoff
			Multiplier: 1.6,              // backoff multiplier
			MaxDelay:   10 * time.Second, // max delay
		}
		// Connect to Persistence Service
		persistenceAddr := os.Getenv("PERSISTENCE_GRPC_ADDR")
		if persistenceAddr == "" {
			log.Println("PERSISTENCE_GRPC_ADDR not set, defaulting to persistenceSvc:9000")
			persistenceAddr = "persistenceSvc:9000"
		}
		conn, err := grpc.NewClient(persistenceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithConnectParams(grpc.ConnectParams{
				Backoff:           retryPolicy,
				MinConnectTimeout: 5 * time.Second,
			}))
		if err != nil {
			log.Fatalf("failed to connect to persistence service: %v", err)
		}

		persistenceClient = pb.NewPersistenceServiceClient(conn)
		// defer conn.Close()

	}()

	go func() {
		log.Println("Starting gRPC client connection to Logging Service")

		retryPolicy := backoff.Config{
			BaseDelay:  1 * time.Second,  // initial backoff
			Multiplier: 1.6,              // backoff multiplier
			MaxDelay:   10 * time.Second, // max delay
		}
		// Connect to Logging Service
		loggingAddr := os.Getenv("LOGGING_GRPC_ADDR")
		if loggingAddr == "" {
			log.Println("LOGGING_GRPC_ADDR not set, defaulting to loggingSvc:6514")
			loggingAddr = "loggingSvc:6514"
		}
		conn, err := grpc.NewClient(loggingAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           retryPolicy,
			MinConnectTimeout: 5 * time.Second,
		}))
		if err != nil {
			log.Fatalf("failed to connect to persistence service: %v", err)
		}

		logClient = loggingpb.NewLogServiceClient(conn)
		// defer conn.Close()
	}()
	// Prevent main from exiting
	select {}
}
