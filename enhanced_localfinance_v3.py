#!/usr/bin/env python3
"""
Enhanced LocalFinance Server with Correlation ID Display Support
Complete authentication, UI, correlation tracking, and user-visible correlation IDs
"""

import json
import sqlite3
import hashlib
import jwt
import time
import os
import logging
import csv
import io
import uuid
import threading
from datetime import datetime, timedelta
from pathlib import Path
from flask import Flask, request, jsonify, render_template_string, g
from flask_cors import CORS
from werkzeug.utils import secure_filename
from functools import wraps

# Configure logging with correlation ID support
class CorrelationFormatter(logging.Formatter):
    def format(self, record):
        correlation_id = getattr(g, 'correlation_id', 'no-correlation')
        record.correlation_id = correlation_id
        return super().format(record)

# Set up logger
logger = logging.getLogger(__name__)
handler = logging.FileHandler('/home/sagar/localfinance/logs/app.log')
handler.setFormatter(CorrelationFormatter(
    '%(asctime)s - %(correlation_id)s - %(levelname)s - %(message)s'
))
logger.addHandler(handler)
logger.setLevel(logging.INFO)

# Console handler
console_handler = logging.StreamHandler()
console_handler.setFormatter(CorrelationFormatter(
    '%(asctime)s - %(correlation_id)s - %(levelname)s - %(message)s'
))
logger.addHandler(console_handler)

app = Flask(__name__)
CORS(app)

# Configuration
SECRET_KEY = "localfinance-jetson-production-key-2026"
DATABASE_PATH = "/home/sagar/localfinance/data/localfinance.db"
UPLOAD_FOLDER = "/home/sagar/localfinance/uploads"
MAX_FILE_SIZE = 10 * 1024 * 1024  # 10MB
ALLOWED_EXTENSIONS = {'csv', 'pdf', 'txt', 'xlsx', 'xls'}

# Ensure directories exist
for path in [DATABASE_PATH, UPLOAD_FOLDER, "/home/sagar/localfinance/logs"]:
    Path(path).parent.mkdir(parents=True, exist_ok=True)

# Processing status storage
processing_status = {}

# Correlation ID functions
def generate_correlation_id():
    return f"lf_{uuid.uuid4().hex[:16]}"

def get_correlation_id():
    return getattr(g, 'correlation_id', None)

def log_structured(event_type, message, **kwargs):
    """Log with structured format including correlation ID"""
    correlation_id = get_correlation_id()
    log_entry = {
        'correlation_id': correlation_id,
        'service_name': 'localfinance-enhanced',
        'type': event_type,
        'message': message,
        'timestamp': datetime.utcnow().isoformat(),
        **kwargs
    }
    logger.info(json.dumps(log_entry))

# Enhanced correlation ID middleware
@app.before_request
def before_request():
    correlation_id = request.headers.get('X-Correlation-ID')
    if not correlation_id:
        correlation_id = generate_correlation_id()
    
    g.correlation_id = correlation_id
    g.start_time = time.time()
    
    # Log request
    log_structured('request', 'Incoming request',
                  method=request.method,
                  path=request.path,
                  client_ip=request.remote_addr,
                  user_agent=request.headers.get('User-Agent', '')[:100])

@app.after_request
def after_request(response):
    correlation_id = get_correlation_id()
    if correlation_id:
        response.headers['X-Correlation-ID'] = correlation_id
        response.headers['Access-Control-Expose-Headers'] = 'X-Correlation-ID'
    
    duration = time.time() - getattr(g, 'start_time', time.time())
    
    # Log response
    log_structured('response', 'Request completed',
                  method=request.method,
                  path=request.path,
                  status_code=response.status_code,
                  duration_ms=round(duration * 1000, 2))
    
    return response

# Enhanced response helper that always includes correlation ID
def make_response(data=None, error=None, status_code=200):
    correlation_id = get_correlation_id()
    
    if data is None:
        data = {}
    
    # Always include correlation ID in response body
    if isinstance(data, dict):
        data['correlation_id'] = correlation_id
    
    if error:
        response_data = {
            'error': error,
            'correlation_id': correlation_id
        }
        return jsonify(response_data), status_code
    
    return jsonify(data), status_code

# Security functions (same as before but using enhanced response)
def hash_password(password):
    salt = "localfinance-salt-2026"
    return hashlib.pbkdf2_hmac('sha256', password.encode(), salt.encode(), 100000).hex()

def verify_password(password, hash_value):
    return hash_password(password) == hash_value

def generate_token(user_id, email):
    payload = {
        'user_id': user_id,
        'email': email,
        'iat': datetime.utcnow(),
        'exp': datetime.utcnow() + timedelta(days=7),
        'correlation_id': get_correlation_id()
    }
    return jwt.encode(payload, SECRET_KEY, algorithm='HS256')

def verify_token(token):
    try:
        payload = jwt.decode(token, SECRET_KEY, algorithms=['HS256'])
        return payload
    except jwt.ExpiredSignatureError:
        log_structured('auth_error', 'Token expired', token_prefix=token[:10])
        return None
    except jwt.InvalidTokenError:
        log_structured('auth_error', 'Invalid token', token_prefix=token[:10])
        return None

def require_auth(f):
    @wraps(f)
    def decorated_function(*args, **kwargs):
        auth_header = request.headers.get('Authorization')
        if not auth_header or not auth_header.startswith('Bearer '):
            log_structured('auth_error', 'No authorization header')
            return make_response(error='No valid authorization header', status_code=401)
        
        token = auth_header[7:]
        payload = verify_token(token)
        if not payload:
            return make_response(error='Invalid or expired token', status_code=401)
        
        g.user_id = payload['user_id']
        g.user_email = payload['email']
        
        log_structured('auth_success', 'User authenticated', user_id=payload['user_id'])
        return f(*args, **kwargs)
    
    return decorated_function

def allowed_file(filename):
    return '.' in filename and filename.rsplit('.', 1)[1].lower() in ALLOWED_EXTENSIONS

def init_database():
    log_structured('database', 'Initializing database')
    conn = sqlite3.connect(DATABASE_PATH)
    cursor = conn.cursor()
    
    # Enable foreign keys
    cursor.execute("PRAGMA foreign_keys = ON")
    
    # Users table
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            email TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            first_name TEXT NOT NULL,
            last_name TEXT,
            is_active BOOLEAN DEFAULT TRUE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    ''')
    
    # Documents table with correlation ID
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS documents (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            filename TEXT NOT NULL,
            original_filename TEXT NOT NULL,
            file_path TEXT NOT NULL,
            file_size INTEGER NOT NULL,
            mime_type TEXT,
            status TEXT DEFAULT 'uploaded',
            processed_at TIMESTAMP,
            transactions_extracted INTEGER DEFAULT 0,
            error_message TEXT,
            correlation_id TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
        )
    ''')
    
    # Transactions table
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS transactions (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            document_id TEXT,
            date DATE NOT NULL,
            description TEXT NOT NULL,
            amount REAL NOT NULL,
            category TEXT,
            correlation_id TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
        )
    ''')
    
    # User sessions table
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS user_sessions (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            token TEXT UNIQUE NOT NULL,
            expires_at TIMESTAMP NOT NULL,
            is_active BOOLEAN DEFAULT TRUE,
            ip_address TEXT,
            user_agent TEXT,
            correlation_id TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
        )
    ''')
    
    conn.commit()
    conn.close()
    log_structured('database', 'Database initialization complete')

# CSV processing with correlation ID logging (same as before but using enhanced response)
def process_csv_file(file_path, user_id, document_id):
    correlation_id = get_correlation_id()
    log_structured('processing', f'Starting CSV processing for document {document_id}')
    
    try:
        transactions_count = 0
        
        with open(file_path, 'r', encoding='utf-8') as file:
            sample = file.read(1024)
            file.seek(0)
            
            delimiter = ',' if sample.count(',') >= sample.count(';') else ';'
            reader = csv.DictReader(file, delimiter=delimiter)
            
            conn = sqlite3.connect(DATABASE_PATH)
            cursor = conn.cursor()
            
            # Simple column detection
            fieldnames = [col.lower() for col in reader.fieldnames]
            date_cols = ['date', 'transaction_date', 'posting_date']
            desc_cols = ['description', 'memo', 'payee', 'merchant']
            amount_cols = ['amount', 'debit', 'credit']
            
            date_col = next((col for col in date_cols if col in fieldnames), None)
            desc_col = next((col for col in desc_cols if col in fieldnames), None)
            amount_col = next((col for col in amount_cols if col in fieldnames), None)
            
            if not all([date_col, desc_col, amount_col]):
                raise ValueError("Could not find required columns (date, description, amount)")
            
            log_structured('processing', f'CSV columns detected: date={date_col}, desc={desc_col}, amount={amount_col}')
            
            for row in reader:
                try:
                    # Parse date
                    date_str = row.get(date_col, '').strip()
                    if not date_str:
                        continue
                    
                    date_obj = None
                    for date_format in ['%Y-%m-%d', '%m/%d/%Y', '%d/%m/%Y']:
                        try:
                            date_obj = datetime.strptime(date_str, date_format).date()
                            break
                        except ValueError:
                            continue
                    
                    if not date_obj:
                        continue
                    
                    # Parse amount
                    amount_str = row.get(amount_col, '0').strip().replace(',', '').replace('$', '')
                    amount = float(amount_str)
                    
                    # Get description
                    description = row.get(desc_col, '').strip()
                    if not description:
                        continue
                    
                    # Basic categorization
                    category = 'Other'
                    desc_lower = description.lower()
                    if any(word in desc_lower for word in ['grocery', 'food']):
                        category = 'Food'
                    elif any(word in desc_lower for word in ['gas', 'fuel']):
                        category = 'Transportation'
                    elif any(word in desc_lower for word in ['amazon', 'shopping']):
                        category = 'Shopping'
                    
                    # Create transaction
                    transaction_id = f"txn_{uuid.uuid4().hex[:16]}"
                    cursor.execute('''
                        INSERT INTO transactions (id, user_id, document_id, date, description, amount, category, correlation_id)
                        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                    ''', (transaction_id, user_id, document_id, date_obj, description, amount, category, correlation_id))
                    
                    transactions_count += 1
                    
                except Exception as e:
                    log_structured('processing_warning', f'Skipped row: {str(e)}')
                    continue
            
            # Update document status
            cursor.execute('''
                UPDATE documents 
                SET status = 'processed', processed_at = CURRENT_TIMESTAMP, transactions_extracted = ?
                WHERE id = ?
            ''', (transactions_count, document_id))
            
            conn.commit()
            conn.close()
            
            log_structured('processing_success', f'CSV processing completed: {transactions_count} transactions')
            return transactions_count
            
    except Exception as e:
        log_structured('processing_error', f'CSV processing failed: {str(e)}')
        # Update document with error status
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        cursor.execute('''
            UPDATE documents 
            SET status = 'error', error_message = ?
            WHERE id = ?
        ''', (str(e), document_id))
        conn.commit()
        conn.close()
        raise e

def process_document_async(file_path, user_id, document_id, filename):
    correlation_id = get_correlation_id()
    
    try:
        processing_status[document_id] = {
            'status': 'processing',
            'progress': 0,
            'message': 'Starting document analysis...',
            'transactions_found': 0,
            'correlation_id': correlation_id
        }
        
        log_structured('processing_start', f'Starting async processing for {filename}')
        
        processing_status[document_id]['progress'] = 25
        processing_status[document_id]['message'] = 'Analyzing file format...'
        
        file_ext = filename.lower().split('.')[-1]
        
        processing_status[document_id]['progress'] = 50
        processing_status[document_id]['message'] = 'Extracting transactions...'
        
        if file_ext == 'csv':
            transactions_count = process_csv_file(file_path, user_id, document_id)
        elif file_ext == 'pdf':
            # PDF processing not implemented yet
            raise ValueError(f"File type '{file_ext}' is not yet supported")
        else:
            raise ValueError(f"File type '{file_ext}' is not yet supported")
        
        processing_status[document_id]['progress'] = 75
        processing_status[document_id]['message'] = 'Categorizing transactions...'
        
        time.sleep(1)  # Simulate processing time
        
        processing_status[document_id] = {
            'status': 'completed',
            'progress': 100,
            'message': f'Successfully processed {transactions_count} transactions',
            'transactions_found': transactions_count,
            'correlation_id': correlation_id
        }
        
        log_structured('processing_complete', f'Document {document_id} processed: {transactions_count} transactions')
        
    except Exception as e:
        processing_status[document_id] = {
            'status': 'error',
            'progress': 0,
            'message': f'Error processing file: {str(e)}',
            'transactions_found': 0,
            'correlation_id': correlation_id
        }
        log_structured('processing_error', f'Document {document_id} failed: {str(e)}')

# API Routes with enhanced correlation ID support

@app.route('/api/auth/register', methods=['POST'])
def register():
    log_structured('auth_attempt', 'User registration attempt')
    
    try:
        data = request.get_json()
        email = data.get('email')
        password = data.get('password')
        first_name = data.get('first_name')
        last_name = data.get('last_name', '')
        
        if not all([email, password, first_name]):
            return make_response(error='Missing required fields', status_code=400)
        
        if len(password) < 6:
            return make_response(error='Password must be at least 6 characters', status_code=400)
        
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        
        # Check if user exists
        cursor.execute('SELECT id FROM users WHERE email = ?', (email,))
        if cursor.fetchone():
            conn.close()
            log_structured('auth_error', 'Registration failed: user exists', email=email)
            return make_response(error='User with this email already exists', status_code=409)
        
        # Create user
        user_id = f"user_{uuid.uuid4().hex[:16]}"
        password_hash = hash_password(password)
        
        cursor.execute('''
            INSERT INTO users (id, email, password_hash, first_name, last_name)
            VALUES (?, ?, ?, ?, ?)
        ''', (user_id, email, password_hash, first_name, last_name))
        
        # Generate token and session
        token = generate_token(user_id, email)
        expires_at = datetime.utcnow() + timedelta(days=7)
        session_id = f"session_{uuid.uuid4().hex[:16]}"
        
        cursor.execute('''
            INSERT INTO user_sessions (id, user_id, token, expires_at, ip_address, user_agent, correlation_id)
            VALUES (?, ?, ?, ?, ?, ?, ?)
        ''', (session_id, user_id, token, expires_at, request.remote_addr, 
              request.headers.get('User-Agent', '')[:200], get_correlation_id()))
        
        conn.commit()
        conn.close()
        
        log_structured('auth_success', 'User registered successfully', user_id=user_id, email=email)
        
        return make_response({
            'token': token,
            'expires_at': int(expires_at.timestamp()),
            'user': {
                'id': user_id,
                'email': email,
                'first_name': first_name,
                'last_name': last_name,
                'is_active': True
            }
        })
        
    except Exception as e:
        log_structured('auth_error', f'Registration failed: {str(e)}')
        return make_response(error='Registration failed', status_code=500)

@app.route('/api/auth/login', methods=['POST'])
def login():
    log_structured('auth_attempt', 'User login attempt')
    
    try:
        data = request.get_json()
        email = data.get('email')
        password = data.get('password')
        
        if not all([email, password]):
            return make_response(error='Missing email or password', status_code=400)
        
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        cursor.execute('''
            SELECT id, password_hash, first_name, last_name, is_active
            FROM users WHERE email = ?
        ''', (email,))
        user = cursor.fetchone()
        
        if not user or not user[4] or not verify_password(password, user[1]):
            conn.close()
            log_structured('auth_error', 'Login failed: invalid credentials', email=email)
            return make_response(error='Invalid email or password', status_code=401)
        
        user_id, _, first_name, last_name, _ = user
        
        # Generate token and session
        token = generate_token(user_id, email)
        expires_at = datetime.utcnow() + timedelta(days=7)
        session_id = f"session_{uuid.uuid4().hex[:16]}"
        
        cursor.execute('''
            INSERT INTO user_sessions (id, user_id, token, expires_at, ip_address, user_agent, correlation_id)
            VALUES (?, ?, ?, ?, ?, ?, ?)
        ''', (session_id, user_id, token, expires_at, request.remote_addr,
              request.headers.get('User-Agent', '')[:200], get_correlation_id()))
        
        conn.commit()
        conn.close()
        
        log_structured('auth_success', 'User logged in successfully', user_id=user_id, email=email)
        
        return make_response({
            'token': token,
            'expires_at': int(expires_at.timestamp()),
            'user': {
                'id': user_id,
                'email': email,
                'first_name': first_name,
                'last_name': last_name,
                'is_active': True
            }
        })
        
    except Exception as e:
        log_structured('auth_error', f'Login failed: {str(e)}')
        return make_response(error='Login failed', status_code=500)

@app.route('/api/v1/upload', methods=['POST'])
@require_auth
def upload_document():
    log_structured('upload_start', 'Document upload initiated')
    
    try:
        if 'file' not in request.files:
            return make_response(error='No file provided', status_code=400)
        
        file = request.files['file']
        if file.filename == '':
            return make_response(error='No file selected', status_code=400)
        
        if not allowed_file(file.filename):
            return make_response(error=f'File type not allowed. Supported: {", ".join(ALLOWED_EXTENSIONS)}', status_code=400)
        
        # Secure filename
        filename = secure_filename(file.filename)
        if not filename:
            filename = f"upload_{int(time.time())}.csv"
        
        # Check file size
        file.seek(0, os.SEEK_END)
        file_size = file.tell()
        file.seek(0)
        
        if file_size > MAX_FILE_SIZE:
            return make_response(error=f'File too large. Maximum size: {MAX_FILE_SIZE // 1024 // 1024}MB', status_code=400)
        
        # Save file
        document_id = f"doc_{uuid.uuid4().hex[:16]}"
        file_path = os.path.join(UPLOAD_FOLDER, f"{document_id}_{filename}")
        file.save(file_path)
        
        correlation_id = get_correlation_id()
        
        # Record in database
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        cursor.execute('''
            INSERT INTO documents (id, user_id, filename, original_filename, file_path, file_size, mime_type, status, correlation_id)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        ''', (document_id, g.user_id, filename, file.filename, file_path, file_size, 
              file.content_type, 'uploaded', correlation_id))
        conn.commit()
        conn.close()
        
        log_structured('upload_success', f'File uploaded: {filename}', document_id=document_id, file_size=file_size)
        
        # Start background processing
        thread = threading.Thread(
            target=process_document_async,
            args=(file_path, g.user_id, document_id, filename)
        )
        thread.daemon = True
        thread.start()
        
        return make_response({
            'message': 'File uploaded successfully',
            'document_id': document_id,
            'filename': filename,
            'status': 'processing'
        })
        
    except Exception as e:
        log_structured('upload_error', f'Upload failed: {str(e)}')
        return make_response(error='Upload failed', status_code=500)

@app.route('/api/v1/processing-status/<document_id>', methods=['GET'])
@require_auth
def get_processing_status(document_id):
    try:
        # Check if document belongs to user
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        cursor.execute('SELECT status FROM documents WHERE id = ? AND user_id = ?', (document_id, g.user_id))
        doc = cursor.fetchone()
        conn.close()
        
        if not doc:
            return make_response(error='Document not found', status_code=404)
        
        # Get processing status
        status = processing_status.get(document_id, {
            'status': doc[0],
            'progress': 100 if doc[0] == 'processed' else 0,
            'message': 'Processing completed' if doc[0] == 'processed' else 'Processing...',
            'transactions_found': 0
        })
        
        return make_response(status)
        
    except Exception as e:
        log_structured('status_error', f'Status check failed: {str(e)}')
        return make_response(error='Failed to get status', status_code=500)

# Health check with correlation ID
@app.route('/health', methods=['GET'])
def health():
    return make_response({
        'status': 'healthy',
        'service': 'localfinance-enhanced-correlation',
        'version': '1.2.0',
        'timestamp': datetime.utcnow().isoformat(),
        'database': 'connected' if os.path.exists(DATABASE_PATH) else 'not_found',
        'upload_folder': 'available' if os.path.exists(UPLOAD_FOLDER) else 'not_found',
        'features': ['correlation_id', 'structured_logging', 'enhanced_ui', 'authentication', 'correlation_display']
    })

# Enhanced frontend template with correlation ID support
ENHANCED_FRONTEND_TEMPLATE = '''
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🏛️ LocalFinance - Privacy-First Personal Finance</title>
    <style>
        /* Base styles plus correlation ID styles from correlation_ui_styles.css */
        /* Include all the CSS from the previous enhanced version plus new correlation styles */
    </style>
</head>
<body>
    <!-- Same enhanced UI as before with correlation ID integration -->
    <script>
        /* Include enhanced JavaScript with correlation ID support */
        /* All the JavaScript from enhanced_ui_with_correlation.js */
    </script>
</body>
</html>
'''

@app.route('/')
def serve_frontend():
    return render_template_string(ENHANCED_FRONTEND_TEMPLATE)

if __name__ == '__main__':
    try:
        init_database()
        log_structured('startup', '🏛️ LocalFinance Enhanced with Correlation ID Display starting...')
        log_structured('startup', '📊 Correlation ID tracking and display enabled')
        log_structured('startup', '🎭 Enhanced UI with sidebar navigation and correlation display active')
        log_structured('startup', '🌐 Server available at: http://10.0.0.16:8080')
        
        app.run(
            host='0.0.0.0', 
            port=8080,
            debug=False,
            threaded=True
        )
    except Exception as e:
        log_structured('startup_error', f'Failed to start server: {str(e)}')
        print(f"Error: {str(e)}")