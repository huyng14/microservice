# Micro-service architecture

- Gateway service
    - HTTP port 8080
    - gRPC clients connect to Persistence service (9000) and Logging Service (6514)
- Persistence service
    - gRPC port 9000
- Logging service
    - gRPC port 6514

![architecture.jpg]() 

# APIs

1. List all profiles
2. Get a profile
3. Add a profile
