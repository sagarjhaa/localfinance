# Iris 🌈

**LocalFinance Frontend Service**

Iris is the modern React-based frontend service for the LocalFinance ecosystem. Named after the Greek goddess of the rainbow and messenger, Iris provides a colorful, user-friendly gateway for managing financial data.

## Features

- **Modern UI**: React 18 with responsive design
- **File Upload**: Drag & drop support for CSV, Excel, PDF, JSON, TXT
- **Authentication**: JWT-based secure login system
- **Service Integration**: Proxy connections to all LocalFinance services
- **Real-time Status**: Live health monitoring of backend services
- **User Dashboard**: Analytics and file management interface

## Quick Start

```bash
# Install all dependencies
npm run install:all

# Start development server (both client and server)
npm run dev

# Or start individually
npm run server  # Start Node.js backend on port 3001
npm run client  # Start React frontend on port 3000
```

## Architecture

```
├── server/                 # Node.js/Express backend
│   ├── index.js           # Main server file
│   ├── routes/            # API routes
│   │   ├── auth.js        # Authentication endpoints
│   │   ├── upload.js      # File upload handling
│   │   └── proxy.js       # Service proxy routes
│   └── middleware/        # Express middleware
│       └── auth.js        # JWT authentication
│
└── client/                # React frontend
    ├── public/            # Static assets
    ├── src/
    │   ├── pages/         # Main page components
    │   │   ├── Login.jsx
    │   │   ├── Dashboard.jsx
    │   │   ├── Upload.jsx
    │   │   └── Settings.jsx
    │   ├── components/    # Reusable components
    │   │   └── Layout/
    │   │       └── Sidebar.jsx
    │   ├── api/           # API client utilities
    │   │   └── client.js
    │   ├── App.jsx        # Main app component
    │   └── index.js       # React entry point
    └── package.json
```

## Environment Setup

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Key environment variables:
- `JWT_SECRET`: Secret key for JWT tokens (change in production!)
- `HERMES_URL`: Gateway service URL (default: http://localhost:3000)
- `NODE_ENV`: Environment mode (development/production)

## API Endpoints

### Authentication
- `POST /api/auth/login` - User login
- `POST /api/auth/logout` - User logout  
- `GET /api/auth/me` - Get current user
- `POST /api/auth/refresh` - Refresh JWT token

### File Upload
- `POST /api/upload/single` - Upload single file
- `POST /api/upload/multiple` - Upload multiple files
- `GET /api/upload/files` - List uploaded files
- `DELETE /api/upload/files/:filename` - Delete file

### Service Proxy
- `GET /api/proxy/health` - Check all service health
- `GET /api/proxy/services` - List available services
- `POST /api/proxy/batch` - Batch service requests
- `*` `/api/proxy/{service}/*` - Proxy to specific service

## Demo Accounts

For development and testing:

| Username | Password  | Role  |
|----------|-----------|-------|
| demo     | demo123   | user  |
| admin    | admin123  | admin |

## Development

### Prerequisites
- Node.js 16+ 
- npm or yarn

### Installation
```bash
# Install root dependencies
npm install

# Install client dependencies  
cd client && npm install

# Return to root
cd ..
```

### Running
```bash
# Development mode (both client and server)
npm run dev

# Production mode
npm run build
npm start
```

### File Upload Testing
The upload system supports:
- **File types**: CSV, Excel (.xlsx/.xls), PDF, JSON, TXT
- **Size limit**: 50MB per file
- **Multiple files**: Up to 10 files per upload
- **Drag & drop**: Full drag and drop support

## Deployment

### Production Build
```bash
# Build React app
npm run build

# Start production server
NODE_ENV=production npm start
```

### Docker (if configured)
```bash
# Build image
docker build -t iris-service .

# Run container
docker run -p 3001:3001 iris-service
```

## Security

- **JWT Authentication**: Secure token-based authentication
- **CORS Protection**: Configurable CORS headers
- **File Validation**: Strict file type and size validation
- **Input Sanitization**: Request validation and sanitization
- **Helmet Security**: Production security headers

## Integration

Iris integrates with the LocalFinance ecosystem:

1. **Hermes** (Gateway): Main service coordination
2. **Thesaurus** (Data Processing): File processing and transformation  
3. **Logos** (Analytics): Data analysis and insights
4. **Sophia** (Intelligence): AI-powered financial insights

## Troubleshooting

### Common Issues

**Port conflicts**: Default ports are 3001 (server) and 3000 (client development)
```bash
# Change ports in package.json scripts if needed
PORT=3005 npm run server
```

**CORS errors**: Update CORS configuration in server/index.js
```js
// Add your domain to CORS origins
origin: ['http://localhost:3000', 'https://yourdomain.com']
```

**File upload fails**: Check file size and type restrictions
- Max size: 50MB per file
- Allowed types: CSV, Excel, PDF, JSON, TXT

### Logs
```bash
# Server logs
npm run server

# Enable debug logs
DEBUG=iris:* npm run server
```

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see LICENSE file for details

---

**Iris v1.0** - Built with ❤️ for LocalFinance