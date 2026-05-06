# 🔐 BoundedAuthT (BAT) - Zero-Knowledge Multi-Tenant Auth Gateway

A production-ready authentication gateway implementing Domain-Driven Design (DDD) with clean architecture principles, demonstrating delegated authentication with OTP flows.

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   User Client   │───▶│   APISIX GW     │───▶│   BAT API       │
│                 │    │   (9080)        │    │   (8080)        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Mock Tenant   │◀───│   Redis Store   │◀───│   OTP Service   │
│   (8081)        │    │   (6379)        │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for local development)

### 1. Start Infrastructure
```bash
# Start all services
docker-compose up -d

# Check service status
docker-compose ps
```

### 2. Access Services
- **BAT API Dashboard**: http://localhost:8080/dashboard
- **Mock Tenant Dashboard**: http://localhost:8081/dashboard
- **APISIX Admin**: http://localhost:9091
- **Jaeger Tracing**: http://localhost:16686
- **Redis**: localhost:6379

### 3. Test Authentication Flow

#### Method 1: Via Dashboard (Recommended)
1. Visit http://localhost:8080/dashboard
2. Use test credentials: `admin` / `123456`
3. Follow the flow: Login → Generate OTP → Exchange OTP

#### Method 2: Direct API Calls
```bash
# Step 1: User Login (via tenant subdomain)
curl -X POST http://localhost:9080/api/auth/login \
  -H "Host: test-tenant.localhost:9080" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin","password":"123456"}'

# Step 2: Generate OTP
curl -X POST http://localhost:9080/api/auth/otp/generate \
  -H "Host: test-tenant.localhost:9080" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"uuid-123"}'

# Step 3: Exchange OTP for JWT
curl -X POST http://localhost:8080/internal/exchange \
  -H "Content-Type: application/json" \
  -d '{"code":"123456"}'
```

## 🏛️ Architecture Layers

### Domain Layer (`internal/domain/`)
Pure business logic with zero external dependencies:
- **Entities**: User aggregate root
- **Value Objects**: TenantID, PlainTextPassword, OTPCode
- **Interfaces**: Repository contracts, Domain services
- **Services**: AuthDomainService with zero-knowledge pattern

### Application Layer (`internal/application/`)
Use cases and application orchestration:
- **Commands**: Request/response DTOs
- **Use Cases**: AuthUseCase coordinating domain services
- **Validation**: Input validation and business rules

### Infrastructure Layer (`internal/infrastructure/`)
External concerns and technical implementations:
- **API**: Gin handlers, middleware, routing
- **Cache**: Redis OTP store implementation
- **Clients**: HTTP client for tenant communication
- **Configuration**: Service wiring and dependencies

## 🔄 Authentication Flow

### 1. Tenant Identification
```
User Request → APISIX Gateway → Tenant Middleware → Extract subdomain
```
- Subdomain extraction: `company-a.app.com` → `company-a`
- Tenant validation and context injection

### 2. Credential Verification
```
BAT API → Tenant Internal API → User Verification → Response
```
- Delegated authentication to tenant service
- Zero-knowledge pattern (no password handling)
- User entity creation and validation

### 3. OTP Generation & Exchange
```
Valid User → Generate OTP → Store in Redis → Return to User
User with OTP → Exchange Endpoint → Validate & Delete → Return JWT
```
- Time-limited OTP codes (5 minutes)
- Atomic get-and-delete operations
- JWT token generation

## 🛠️ Development

### Local Development
```bash
# Install dependencies
go mod tidy

# Run BAT API locally
go run cmd/bat-api/main.go

# Run Mock Tenant locally
go run cmd/mock-tenant/main.go
```

### Testing
```bash
# Run unit tests
go test ./...

# Run integration tests
go test -tags=integration ./...

# Test with coverage
go test -cover ./...
```

### Configuration
Environment variables:
- `REDIS_ADDR`: Redis server address (default: `redis:6379`)
- `GIN_MODE`: Gin mode (default: `release`)
- `OTEL_EXPORTER_JAEGER_ENDPOINT`: Jaeger collector endpoint

## 📊 Monitoring & Observability

### Tracing
- **Jaeger**: Distributed tracing across services
- **OpenTelemetry**: Standardized instrumentation
- **Service Maps**: Request flow visualization

### Metrics
- **Prometheus**: Metrics collection
- **Grafana**: Visualization (optional)
- **APISIX**: Gateway metrics

### Health Checks
All services expose `/health` endpoints:
```bash
curl http://localhost:8080/health  # BAT API
curl http://localhost:8081/health  # Mock Tenant
curl http://localhost:9080/healthz  # APISIX
```

## 🔧 Configuration Files

### APISIX Configuration
- `configs/apisix/config.yaml`: Core APISIX settings
- `configs/apisix/apisix.yml`: Routes and upstreams

### Observability
- `configs/otel-collector-config.yaml`: OpenTelemetry collector
- `configs/prometheus.yml`: Prometheus scraping rules

### Docker
- `docker-compose.yml`: Complete service orchestration
- `cmd/*/Dockerfile`: Multi-stage builds for services

## 🏢 Multi-Tenancy

### Tenant Isolation
- **Subdomain-based**: `tenant.domain.com` routing
- **Data Separation**: Tenant-scoped Redis keys
- **Context Propagation**: Tenant ID in request context

### Tenant Registry
```go
type RegistryURLResolver interface {
    Resolve(tenantID auth.TenantID) (string, error)
}
```

### Mock Tenant Implementation
- Hardcoded user: `admin` / `123456`
- Simulated tenant service responses
- Dashboard for testing flows

## 🔐 Security Features

### Zero-Knowledge Pattern
- Domain layer never handles password hashes
- Delegated verification to tenant services
- Clean separation of concerns

### OTP Security
- Time-limited codes (5 minutes)
- Atomic get-and-delete operations
- Single-use tokens

### Infrastructure Security
- Non-root Docker containers
- Network isolation with Docker networks
- Health checks and graceful shutdowns

## 📈 Scalability

### Horizontal Scaling
- Stateless services
- Redis for distributed state
- Load balancing via APISIX

### Performance
- Connection pooling
- Request timeouts
- Circuit breaker patterns

### Reliability
- Health checks and restarts
- Graceful shutdowns
- Error handling and logging

## 🧪 Testing Strategy

### Unit Tests
- Domain layer business logic
- Value object validation
- Service contracts

### Integration Tests
- API endpoint testing
- Database operations
- External service calls

### E2E Tests
- Complete authentication flows
- Multi-tenant scenarios
- Error conditions

## 📝 API Documentation

### Authentication Endpoints

#### Login
```http
POST /api/auth/login
Host: {tenant}.localhost:9080
Content-Type: application/json

{
  "email": "admin",
  "password": "123456"
}
```

#### Generate OTP
```http
POST /api/auth/otp/generate
Host: {tenant}.localhost:9080
Content-Type: application/json

{
  "user_id": "uuid-123"
}
```

#### Exchange OTP
```http
POST /internal/exchange
Content-Type: application/json

{
  "code": "123456"
}
```

## 🚀 Production Deployment

### Environment Variables
```bash
# Production settings
GIN_MODE=release
REDIS_ADDR=redis-cluster:6379
OTEL_EXPORTER_JAEGER_ENDPOINT=http://jaeger:14268/api/traces
```

### Docker Production
```bash
# Production deployment
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### Kubernetes
Helm charts and K8s manifests can be generated from the Docker Compose configuration.

## 🤝 Contributing

1. Fork the repository
2. Create feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🔗 Related Projects

- [APISIX](https://github.com/apache/apisix) - API Gateway
- [Jaeger](https://github.com/jaegertracing/jaeger) - Distributed Tracing
- [OpenTelemetry](https://github.com/open-telemetry/opentelemetry-go) - Observability
- [Gin](https://github.com/gin-gonic/gin) - HTTP Framework
- [Redis](https://github.com/redis/redis) - In-Memory Database

---

**Built with ❤️ using Go, Docker, and Clean Architecture principles**
