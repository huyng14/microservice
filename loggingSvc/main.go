package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"

	pb "microservice/template/logpb"

	"google.golang.org/grpc"
)

type LogServiceServer struct {
	pb.UnimplementedLogServiceServer
}

type LogEntry struct {
	Service    string      `json:"service"`
	Level      string      `json:"level"`
	Message    string      `json:"message"`
	TraceID    string      `json:"trace_id,omitempty"`
	SpanID     string      `json:"span_id,omitempty"`
	Error      string      `json:"error,omitempty"`
	OrderID    int         `json:"order_id,omitempty"`
	Attempt    int         `json:"attempt,omitempty"`
	UserID     int         `json:"user_id,omitempty"`
	LatencyMS  int64       `json:"latency_ms,omitempty"`
	StackTrace string      `json:"stack_trace,omitempty"`
	Extra      interface{} `json:"extra,omitempty"`
}

func logJSON(entry LogEntry) {
	os.MkdirAll("logfile", 0755)
	file, err := os.OpenFile("logfile/log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("failed to open log file: %v", err)
		return
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(entry)
}

func main() {
	lis, err := net.Listen("tcp", ":6514")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLogServiceServer(grpcServer, &LogServiceServer{})

	log.Println("Logging Service gRPC server running at port 6514...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

	serviceName := "loggingSvc"
	level := "INFO"
	message := "Sample log message"
	traceID := "d4c12e7f2a9c4bf3"
	spanID := "operation-specific-id"

	// INFO: business event
	logJSON(LogEntry{
		Service: serviceName,
		Level:   level,
		Message: message,
		UserID:  123,
		TraceID: traceID,
		SpanID:  spanID,
	})

	// INFO: business event with metrics
	logJSON(LogEntry{
		Service:   serviceName,
		Level:     "INFO",
		Message:   "Handled request",
		LatencyMS: 34,
		TraceID:   traceID,
		SpanID:    spanID,
	})

	// WARN: temporary issue
	logJSON(LogEntry{
		Service: serviceName,
		Level:   "WARN",
		Message: "Retrying ProductService call",
		Attempt: 2,
		TraceID: traceID,
		SpanID:  spanID,
	})

	// ERROR: with stack trace
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logJSON(LogEntry{
		Service:    serviceName,
		Level:      "ERROR",
		Message:    "Failed to charge credit card",
		Error:      "StripeTimeout",
		TraceID:    traceID,
		SpanID:     spanID,
		StackTrace: "main.go:42",
	})

	// Never log sensitive data!
}

func (s *LogServiceServer) LogEvent(ctx context.Context, req *pb.LogEventRequest) (*pb.LogEventResponse, error) {
	logJSON(LogEntry{
		Service:    req.LogEntry.Service,
		Level:      req.LogEntry.Level,
		Message:    req.LogEntry.Message,
		TraceID:    req.LogEntry.TraceId,
		SpanID:     req.LogEntry.SpanId,
		Error:      req.LogEntry.Error,
		OrderID:    int(req.LogEntry.OrderId),
		Attempt:    int(req.LogEntry.Attempt),
		UserID:     int(req.LogEntry.UserId),
		LatencyMS:  req.LogEntry.LatencyMs,
		StackTrace: req.LogEntry.StackTrace,
		Extra:      req.LogEntry.Extra,
	})
	return &pb.LogEventResponse{Success: true}, nil
}
