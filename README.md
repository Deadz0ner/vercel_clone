# Vercel Clone - Multi-Language Implementation

This project contains parallel implementations of Vercel Clone in Node.js and Golang.

## Directory Structure

```
├── node/                           # Node.js/TypeScript implementation
│   ├── deploy_service/             # Deployment service
│   ├── request_handler/            # Request handling service
│   ├── vercel_clone/               # Main application
│   ├── frontend/                   # Frontend (Next.js/TypeScript)
│   └── .gitignore
│
├── golang/                         # Golang implementation
│   ├── app/                        # Main application
│   ├── deploy-service/             # Deployment service
│   ├── request-handler/            # Request handling service
│   ├── go.mod                      # Go module definition
│   └── .gitignore
│
└── README.md
```

## Getting Started

### Node.js Version
```bash
cd node
# Follow setup instructions in respective service directories
```

### Golang Version
```bash
cd golang
go mod tidy
# Build and run individual services
```

## Development Notes

Both implementations maintain the same service structure for easy comparison and parallel development.
