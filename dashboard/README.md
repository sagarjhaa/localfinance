# LocalFinance Dashboard

## 🌐 Web Interface for LocalFinance

Modern, responsive web dashboard for uploading bank statements and viewing financial data.

### Features
- **Drag & Drop Upload**: Easy file upload interface
- **Real-time Statistics**: Live processing stats and transaction counts
- **Mobile Responsive**: Works on desktop, tablet, and mobile
- **Progress Tracking**: Visual feedback during file processing
- **File Management**: View processed files and history

### Files
- **app.py**: Main Flask application (20KB)
- **enhanced_web_upload.py**: Enhanced version with modern UI
- **web_upload.py**: Basic version for reference
- **templates/**: HTML templates (to be created)
- **static/**: CSS, JS, and assets (to be created)
- **config/**: Dashboard-specific configuration

### Usage

#### Development
```bash
cd dashboard
python3 app.py
# Open http://localhost:5000
```

#### Production
```bash
cd production
sudo ./scripts/installer.sh YOUR_BOT_TOKEN
sudo systemctl start hisab-web
```

### API Endpoints
- **GET /**: Main dashboard interface
- **POST /upload**: File upload endpoint
- **GET /stats**: Statistics API
- **GET /transactions**: Recent transactions API

### Dependencies
- Flask 3.1.3
- Watchdog 4.0.0
- PDFplumber 0.10.3
- Pandas 2.1.4

Built for privacy-first local financial management.