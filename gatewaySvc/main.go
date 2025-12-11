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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
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
			fmt.Sprintln("Invalid request body", http.StatusBadRequest),
		)
		return
	}

	// Call gRPC method GetPerson
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := persistenceClient.GetPerson(ctx, &pb.GetPersonRequest{Id: req.Id})
	if err != nil {
		log.Printf("error calling GetPerson: %v", handleRpcError(err))
		http.Error(w, "Error fetching person", http.StatusInternalServerError)

		// Call gRPC logging method
		// Only keep the latest log entry
		buf.Reset()
		// Write log to the buffer
		logger.Printf("error calling GetPerson: %v", handleRpcError(err))
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintln("Error fetching person", http.StatusInternalServerError),
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
			fmt.Sprintln("Invalid request body", http.StatusBadRequest),
		)
		return
	}
	log.Println("Print PostPersonRequest: ", req.Person)
	// Call gRPC method PostPerson
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := persistenceClient.PostPerson(ctx, req)
	if err != nil {
		log.Printf("error calling PostPerson: %v", handleRpcError(err))
		http.Error(w, "Error posting person", http.StatusInternalServerError)

		// Call gRPC logging method
		// Only keep the latest log entry
		buf.Reset()
		// Write log to the buffer
		logger.Printf("error calling PostPerson: %v", handleRpcError(err))
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintln("Error posting person", http.StatusInternalServerError),
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
			fmt.Sprintln("Method not allowed", http.StatusMethodNotAllowed),
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
		logger.Println("Method not allowed", http.StatusMethodNotAllowed)
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintln("Method not allowed", http.StatusMethodNotAllowed),
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	stream, err := persistenceClient.ListPersons(ctx, &emptypb.Empty{})
	if err != nil {
		log.Printf("error calling ListPersons: %v", handleRpcError(err))
		http.Error(w, "Error listing persons", http.StatusInternalServerError)

		// Call gRPC logging method
		// Only keep the latest log entry
		buf.Reset()
		// Write log to the buffer
		logger.Printf("error calling ListPersons: %v", handleRpcError(err))
		SendLogMessage(
			logger,
			&buf,
			"ERROR",
			fmt.Sprintln("Error listing persons", http.StatusInternalServerError),
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
			fmt.Sprintln("Error marshalling response", http.StatusInternalServerError),
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
			fmt.Sprintln("firstPage: method not allowed", http.StatusMethodNotAllowed),
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

	// Channels to signal gRPC client readiness
	persistenceReady := make(chan struct{})
	loggingReady := make(chan struct{})

	retryPolicy := backoff.Config{
		BaseDelay:  1 * time.Second,  // initial backoff
		Multiplier: 1.6,              // backoff multiplier
		MaxDelay:   10 * time.Second, // max delay
	}
	ka := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}

	go func() {
		log.Println("Starting gRPC client connection to Persistence Service")

		// Connect to Persistence Service
		persistenceAddr := os.Getenv("PERSISTENCE_GRPC_ADDR")
		if persistenceAddr == "" {
			log.Println("PERSISTENCE_GRPC_ADDR not set, defaulting to localhost:9000")
			persistenceAddr = "localhost:9000"
		}
		conn, err := grpc.NewClient(persistenceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithConnectParams(grpc.ConnectParams{
				Backoff:           retryPolicy,
				MinConnectTimeout: 5 * time.Second,
			}),
			grpc.WithKeepaliveParams(ka),
		)
		if err != nil {
			log.Fatalf("failed to connect to persistence service: %v", err)
		}

		persistenceClient = pb.NewPersistenceServiceClient(conn)
		close(persistenceReady)
	}()

	go func() {
		log.Println("Starting gRPC client connection to Logging Service")

		// Connect to Logging Service
		loggingAddr := os.Getenv("LOGGING_GRPC_ADDR")
		if loggingAddr == "" {
			log.Println("LOGGING_GRPC_ADDR not set, defaulting to localhost:6514")
			loggingAddr = "localhost:6514"
		}
		conn, err := grpc.NewClient(loggingAddr, grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithConnectParams(grpc.ConnectParams{
				Backoff:           retryPolicy,
				MinConnectTimeout: 5 * time.Second,
			}),
			grpc.WithKeepaliveParams(ka),
		)
		if err != nil {
			log.Fatalf("failed to connect to persistence service: %v", err)
		}

		logClient = loggingpb.NewLogServiceClient(conn)
		close(loggingReady)
	}()

	go func() {
		<-persistenceReady
		<-loggingReady
		log.Println("Both gRPC clients initialized successfully")

		log.Println("Starting HTTP server on :8080")
		http.HandleFunc("/", firstPage)
		http.HandleFunc("/profiles", handleProfiles)
		http.HandleFunc("/listprofiles", handleListProfiles)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		pongPersistence, err := persistenceClient.Ping(ctx, &pb.PingRequest{})
		if err != nil {
			log.Fatalf("Error pinging Persistence Service: %v", err)
		}
		pongLogging, err := logClient.Ping(ctx, &loggingpb.PingRequest{})
		if err != nil {
			log.Fatalf("Error pinging Logging Service: %v", err)
		}
		if pongPersistence.GetStatus() == "OK" && pongLogging.GetStatus() == "OK" {
			if err := http.ListenAndServe(":8080", nil); err != nil {
				log.Fatalf("HTTP server error: %v", err)
			}
		}
	}()
	// Prevent main from exiting
	select {}
}

func handleRpcError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("non-gRPC error: %v", err)
	}

	switch st.Code() {

	case codes.DeadlineExceeded:
		return fmt.Errorf("timeout, server did not respond")

	case codes.Unavailable:
		return fmt.Errorf("server unavailable or broken connection")

	case codes.InvalidArgument:
		return fmt.Errorf("bad client request: %v", st.Message())

	case codes.Internal:
		return fmt.Errorf("server internal error: %v", st.Message())

	default:
		return fmt.Errorf("gRPC error [%s]: %s", st.Code(), st.Message())
	}
}
