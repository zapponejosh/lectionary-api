# Lectionary API

A RESTful API for accessing daily lectionary readings from the Presbyterian Book of Common Worship two-year cycle.

## Project Structure

```
lectionary-api/
├── cmd/
│   ├── api/                    # Main API server
│   │   └── main.go
│   └── import/                 # PDF import tool
│       └── main.go
│
├── internal/
│   ├── api/                    # HTTP handlers and routes
│   │   ├── handlers.go         # Request handlers
│   │   ├── middleware.go       # Auth, logging, CORS
│   │   ├── routes.go           # Route definitions
│   │   └── response.go         # Standard response formats
│   │
│   ├── database/               # Database layer
│   │   ├── db.go              # Connection management
│   │   ├── queries.go         # SQL queries
│   │   ├── models.go          # Data models
│   │   └── migrations.go      # Schema migrations
│   │
│   ├── parser/                 # PDF parsing
│   │   ├── parser.go          # Main parser logic
│   │   ├── extractor.go       # PDF text extraction
│   │   ├── patterns.go        # Regex patterns
│   │   └── validator.go       # Data validation
│   │
│   ├── logger/                 # Structured logging
│   │   └── logger.go          # Logging setup (slog)
│   │
│   └── config/                 # Configuration
│       └── config.go          # Environment config
│
├── data/                       # Data directory
│   ├── lectionary.db          # SQLite database (gitignored)
│   └── pdfs/                  # Source PDFs (gitignored)
│
├── migrations/                 # SQL migration files
│   ├── 001_initial_schema.sql
│   └── 002_add_indexes.sql
│
├── scripts/                    # Utility scripts
│   ├── import.sh              # Import helper
│   └── backup.sh              # Database backup
│
├── .github/
│   └── workflows/
│       └── ci.yml             # CI/CD pipeline
│
├── Dockerfile                  # Multi-stage build
├── .dockerignore
├── fly.toml                    # Fly.io configuration
├── .env.example               # Example environment vars
├── .gitignore
├── go.mod
├── go.sum
├── Makefile                    # Common tasks
└── README.md
```

## Quick Start

### Prerequisites
- Go 1.22+
- SQLite3
- Fly CLI (for deployment)

### Local Development

```bash
# Clone repository
git clone <repo-url>
cd lectionary-api

# Install dependencies
go mod download

# Copy environment template
cp .env.example .env

# Run migrations
make migrate

# Import lectionary data
make import PDF=./data/pdfs/2025_Daily_Full_Year.pdf

# Run API server
make run

# API available at http://localhost:8080
```

### Import Lectionary Data

```bash
# Download PDF to data/pdfs/
# Then run import
go run cmd/import/main.go -pdf ./data/pdfs/2025_Daily_Full_Year.pdf
```

## API Endpoints

### Public (No Authentication)

```
GET  /health                           # Health check
GET  /api/v1/readings/today            # Today's readings
GET  /api/v1/readings/date/{YYYY-MM-DD} # Specific date
GET  /api/v1/readings/range            # Date range
     ?start=YYYY-MM-DD&end=YYYY-MM-DD
```

### Authenticated (Requires `X-API-Key` header)

```
GET    /api/v1/progress                # Reading history
       ?limit=50&offset=0
POST   /api/v1/progress                # Mark reading complete
       Body: {"reading_id": 123, "notes": "optional"}
DELETE /api/v1/progress/{reading_id}   # Unmark reading
GET    /api/v1/progress/stats          # Statistics
```

## Environment Variables

```bash
# Server
PORT=8080
ENV=production  # development, production

# Database
DATABASE_PATH=./data/lectionary.db

# Authentication
API_KEY=your-secret-api-key-here

# Logging
LOG_LEVEL=info  # debug, info, warn, error
LOG_FORMAT=json # json, text

# Fly.io (production)
FLY_APP_NAME=lectionary-api
```

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Build binary
make build

# Run with hot reload (requires air)
air
```

## Deployment to Fly.io

```bash
# Login to Fly
fly auth login

# Create app
fly apps create lectionary-api

# Create volume for SQLite
fly volumes create lectionary_data --size 1

# Set secrets
fly secrets set API_KEY=your-secret-key

# Deploy
fly deploy

# Check status
fly status

# View logs
fly logs
```

## Database Schema

### liturgical_days
- Complete daily lectionary calendar
- Includes liturgical seasons and special days
- Two-year cycle tracking

### readings
- Morning/evening psalms
- Scripture readings (OT, Epistle, Gospel)
- Alternative readings marked

### reading_progress
- User reading completion tracking
- Notes and timestamps
- Per-user, per-reading basis

## Best Practices Implemented

### Logging
- Structured logging with `log/slog`
- Request ID tracing
- Performance metrics
- Error context preservation

### Security
- API key authentication
- Rate limiting
- CORS configuration
- Input validation

### Reliability
- Graceful shutdown
- Health checks
- Database connection pooling
- Panic recovery middleware

### Observability
- Request/response logging
- Metrics endpoints
- Error tracking
- Performance monitoring

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Lectionary readings from the Presbyterian Book of Common Worship (2018)
- Published by Westminster John Knox Press