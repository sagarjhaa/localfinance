#!/usr/bin/env python3
"""
Enhanced LocalFinance Server - Complete Authentication & Advanced UI
Features: Left sidebar navigation, document upload with processing, settings
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
from datetime import datetime, timedelta
from pathlib import Path
from flask import Flask, request, jsonify, render_template_string
from flask_cors import CORS
from werkzeug.utils import secure_filename
import uuid
import threading

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('/home/sagar/localfinance/logs/app.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

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

# Processing status storage (in production, use Redis or database)
processing_status = {}

def allowed_file(filename):
    return '.' in filename and filename.rsplit('.', 1)[1].lower() in ALLOWED_EXTENSIONS

def init_database():
    """Initialize SQLite database with comprehensive schema"""
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
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
        )
    ''')
    
    # Categories table
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS categories (
            id TEXT PRIMARY KEY,
            name TEXT UNIQUE NOT NULL,
            color TEXT DEFAULT '#6366f1',
            icon TEXT DEFAULT '💰',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    ''')
    
    # Transactions table
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS transactions (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            date DATE NOT NULL,
            description TEXT NOT NULL,
            amount REAL NOT NULL,
            category_id TEXT,
            file_source TEXT,
            notes TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
            FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE SET NULL
        )
    ''')
    
    # Documents table
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
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
        )
    ''')
    
    # User settings table
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS user_settings (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            setting_key TEXT NOT NULL,
            setting_value TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
            UNIQUE(user_id, setting_key)
        )
    ''')
    
    # Create default categories
    default_categories = [
        ('food', 'Food & Dining', '#ef4444', '🍕'),
        ('transport', 'Transportation', '#3b82f6', '🚗'),
        ('shopping', 'Shopping', '#8b5cf6', '🛍️'),
        ('utilities', 'Utilities', '#f59e0b', '⚡'),
        ('healthcare', 'Healthcare', '#10b981', '🏥'),
        ('entertainment', 'Entertainment', '#ec4899', '🎬'),
        ('income', 'Income', '#059669', '💵'),
        ('other', 'Other', '#6b7280', '📦')
    ]
    
    for cat_id, name, color, icon in default_categories:
        cursor.execute('''
            INSERT OR IGNORE INTO categories (id, name, color, icon)
            VALUES (?, ?, ?, ?)
        ''', (cat_id, name, color, icon))
    
    conn.commit()
    conn.close()
    logger.info("Database initialized successfully")

# Security functions
def hash_password(password):
    """Secure password hashing with salt"""
    salt = "localfinance-salt-2026"
    return hashlib.pbkdf2_hmac('sha256', password.encode(), salt.encode(), 100000).hex()

def verify_password(password, hash_value):
    """Verify password against hash"""
    return hash_password(password) == hash_value

def generate_token(user_id, email):
    """Generate JWT token"""
    payload = {
        'user_id': user_id,
        'email': email,
        'iat': datetime.utcnow(),
        'exp': datetime.utcnow() + timedelta(days=7)
    }
    return jwt.encode(payload, SECRET_KEY, algorithm='HS256')

def verify_token(token):
    """Verify JWT token"""
    try:
        payload = jwt.decode(token, SECRET_KEY, algorithms=['HS256'])
        return payload
    except jwt.ExpiredSignatureError:
        return None
    except jwt.InvalidTokenError:
        return None

def require_auth(f):
    """Authentication decorator"""
    def decorated_function(*args, **kwargs):
        auth_header = request.headers.get('Authorization')
        if not auth_header or not auth_header.startswith('Bearer '):
            return jsonify({'error': 'No valid authorization header'}), 401
        
        token = auth_header[7:]  # Remove 'Bearer '
        payload = verify_token(token)
        if not payload:
            return jsonify({'error': 'Invalid or expired token'}), 401
        
        # Check if session is still active
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        cursor.execute('''
            SELECT is_active FROM user_sessions 
            WHERE token = ? AND expires_at > CURRENT_TIMESTAMP AND is_active = TRUE
        ''', (token,))
        session = cursor.fetchone()
        conn.close()
        
        if not session:
            return jsonify({'error': 'Session expired or invalid'}), 401
        
        request.user_id = payload['user_id']
        request.user_email = payload['email']
        return f(*args, **kwargs)
    
    decorated_function.__name__ = f.__name__
    return decorated_function

def process_csv_file(file_path, user_id, document_id):
    """Process uploaded CSV file and extract transactions"""
    try:
        transactions_count = 0
        
        with open(file_path, 'r', encoding='utf-8') as file:
            # Try to detect delimiter
            sample = file.read(1024)
            file.seek(0)
            
            # Simple delimiter detection
            delimiter = ','
            if ';' in sample and sample.count(';') > sample.count(','):
                delimiter = ';'
            
            reader = csv.DictReader(file, delimiter=delimiter)
            
            conn = sqlite3.connect(DATABASE_PATH)
            cursor = conn.cursor()
            
            # Common column name mappings
            date_cols = ['date', 'transaction_date', 'posting_date', 'effective_date']
            desc_cols = ['description', 'memo', 'payee', 'merchant', 'details']
            amount_cols = ['amount', 'debit', 'credit', 'transaction_amount']
            
            # Find the right columns
            fieldnames = [col.lower() for col in reader.fieldnames]
            
            date_col = next((col for col in date_cols if col in fieldnames), None)
            desc_col = next((col for col in desc_cols if col in fieldnames), None)
            amount_col = next((col for col in amount_cols if col in fieldnames), None)
            
            if not all([date_col, desc_col, amount_col]):
                raise ValueError("Could not find required columns (date, description, amount)")
            
            for row in reader:
                try:
                    # Parse date
                    date_str = row.get(date_col, '').strip()
                    if not date_str:
                        continue
                    
                    # Try different date formats
                    date_obj = None
                    for date_format in ['%Y-%m-%d', '%m/%d/%Y', '%d/%m/%Y', '%Y/%m/%d']:
                        try:
                            date_obj = datetime.strptime(date_str, date_format).date()
                            break
                        except ValueError:
                            continue
                    
                    if not date_obj:
                        continue
                    
                    # Parse amount
                    amount_str = row.get(amount_col, '0').strip()
                    amount_str = amount_str.replace(',', '').replace('$', '').replace('(', '-').replace(')', '')
                    
                    try:
                        amount = float(amount_str)
                    except ValueError:
                        continue
                    
                    # Get description
                    description = row.get(desc_col, '').strip()
                    if not description:
                        continue
                    
                    # Basic categorization
                    category_id = 'other'
                    desc_lower = description.lower()
                    
                    if any(word in desc_lower for word in ['grocery', 'food', 'restaurant', 'cafe', 'starbucks']):
                        category_id = 'food'
                    elif any(word in desc_lower for word in ['gas', 'fuel', 'uber', 'lyft', 'transport']):
                        category_id = 'transport'
                    elif any(word in desc_lower for word in ['amazon', 'walmart', 'target', 'shopping']):
                        category_id = 'shopping'
                    elif any(word in desc_lower for word in ['electric', 'water', 'internet', 'phone', 'utility']):
                        category_id = 'utilities'
                    elif any(word in desc_lower for word in ['salary', 'payroll', 'deposit', 'income']):
                        category_id = 'income'
                    
                    # Create transaction
                    transaction_id = f"txn_{uuid.uuid4().hex[:16]}"
                    
                    cursor.execute('''
                        INSERT INTO transactions (id, user_id, date, description, amount, category_id, file_source)
                        VALUES (?, ?, ?, ?, ?, ?, ?)
                    ''', (transaction_id, user_id, date_obj, description, amount, category_id, document_id))
                    
                    transactions_count += 1
                    
                except Exception as e:
                    logger.warning(f"Skipping row due to error: {str(e)}")
                    continue
            
            # Update document status
            cursor.execute('''
                UPDATE documents 
                SET status = 'processed', processed_at = CURRENT_TIMESTAMP, transactions_extracted = ?
                WHERE id = ?
            ''', (transactions_count, document_id))
            
            conn.commit()
            conn.close()
            
            return transactions_count
            
    except Exception as e:
        # Update document with error status
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        cursor.execute('''
            UPDATE documents 
            SET status = 'error', error_message = ?, processed_at = CURRENT_TIMESTAMP
            WHERE id = ?
        ''', (str(e), document_id))
        conn.commit()
        conn.close()
        
        raise e

def process_document_async(file_path, user_id, document_id, filename):
    """Process document in background thread"""
    try:
        processing_status[document_id] = {
            'status': 'processing',
            'progress': 0,
            'message': 'Starting document analysis...',
            'transactions_found': 0
        }
        
        processing_status[document_id]['progress'] = 25
        processing_status[document_id]['message'] = 'Analyzing file format...'
        
        # Determine file type and process accordingly
        file_ext = filename.lower().split('.')[-1]
        
        processing_status[document_id]['progress'] = 50
        processing_status[document_id]['message'] = 'Extracting transactions...'
        
        if file_ext == 'csv':
            transactions_count = process_csv_file(file_path, user_id, document_id)
        else:
            # For now, only CSV is supported
            raise ValueError(f"File type '{file_ext}' is not yet supported")
        
        processing_status[document_id]['progress'] = 75
        processing_status[document_id]['message'] = 'Categorizing transactions...'
        
        # Simulate some processing time
        time.sleep(1)
        
        processing_status[document_id] = {
            'status': 'completed',
            'progress': 100,
            'message': f'Successfully processed {transactions_count} transactions',
            'transactions_found': transactions_count
        }
        
        logger.info(f"Document {document_id} processed successfully: {transactions_count} transactions")
        
    except Exception as e:
        processing_status[document_id] = {
            'status': 'error',
            'progress': 0,
            'message': f'Error processing file: {str(e)}',
            'transactions_found': 0
        }
        logger.error(f"Document processing error for {document_id}: {str(e)}")

# Authentication endpoints (same as before)
@app.route('/api/auth/register', methods=['POST'])
def register():
    """User registration"""
    try:
        data = request.get_json()
        email = data.get('email')
        password = data.get('password')
        first_name = data.get('first_name')
        last_name = data.get('last_name', '')
        
        if not all([email, password, first_name]):
            return jsonify({'error': 'Missing required fields'}), 400
        
        if len(password) < 6:
            return jsonify({'error': 'Password must be at least 6 characters'}), 400
        
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        
        # Check if user exists
        cursor.execute('SELECT id FROM users WHERE email = ?', (email,))
        if cursor.fetchone():
            conn.close()
            return jsonify({'error': 'User with this email already exists'}), 409
        
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
            INSERT INTO user_sessions (id, user_id, token, expires_at, ip_address, user_agent)
            VALUES (?, ?, ?, ?, ?, ?)
        ''', (session_id, user_id, token, expires_at, request.remote_addr, request.headers.get('User-Agent', '')[:200]))
        
        conn.commit()
        conn.close()
        
        logger.info(f"User registered successfully: {email}")
        
        return jsonify({
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
        logger.error(f"Registration error: {str(e)}")
        return jsonify({'error': 'Registration failed'}), 500

@app.route('/api/auth/login', methods=['POST'])
def login():
    """User login"""
    try:
        data = request.get_json()
        email = data.get('email')
        password = data.get('password')
        
        if not all([email, password]):
            return jsonify({'error': 'Missing email or password'}), 400
        
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        cursor.execute('''
            SELECT id, password_hash, first_name, last_name, is_active
            FROM users WHERE email = ?
        ''', (email,))
        user = cursor.fetchone()
        
        if not user or not user[4] or not verify_password(password, user[1]):
            conn.close()
            logger.warning(f"Failed login attempt for email: {email}")
            return jsonify({'error': 'Invalid email or password'}), 401
        
        user_id, _, first_name, last_name, _ = user
        
        # Generate token and session
        token = generate_token(user_id, email)
        expires_at = datetime.utcnow() + timedelta(days=7)
        session_id = f"session_{uuid.uuid4().hex[:16]}"
        
        cursor.execute('''
            INSERT INTO user_sessions (id, user_id, token, expires_at, ip_address, user_agent)
            VALUES (?, ?, ?, ?, ?, ?)
        ''', (session_id, user_id, token, expires_at, request.remote_addr, request.headers.get('User-Agent', '')[:200]))
        
        conn.commit()
        conn.close()
        
        logger.info(f"User logged in successfully: {email}")
        
        return jsonify({
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
        logger.error(f"Login error: {str(e)}")
        return jsonify({'error': 'Login failed'}), 500

@app.route('/api/auth/logout', methods=['POST'])
@require_auth
def logout():
    """User logout"""
    try:
        auth_header = request.headers.get('Authorization')
        token = auth_header[7:]  # Remove 'Bearer '
        
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        cursor.execute('UPDATE user_sessions SET is_active = FALSE WHERE token = ?', (token,))
        conn.commit()
        conn.close()
        
        logger.info(f"User logged out: {request.user_email}")
        return jsonify({'message': 'Logged out successfully'})
        
    except Exception as e:
        logger.error(f"Logout error: {str(e)}")
        return jsonify({'error': 'Logout failed'}), 500

# Upload endpoints
@app.route('/api/v1/upload', methods=['POST'])
@require_auth
def upload_document():
    """Upload and process financial document"""
    try:
        if 'file' not in request.files:
            return jsonify({'error': 'No file provided'}), 400
        
        file = request.files['file']
        if file.filename == '':
            return jsonify({'error': 'No file selected'}), 400
        
        if not allowed_file(file.filename):
            return jsonify({'error': f'File type not allowed. Supported: {", ".join(ALLOWED_EXTENSIONS)}'}), 400
        
        # Secure filename
        filename = secure_filename(file.filename)
        if not filename:
            filename = f"upload_{int(time.time())}.csv"
        
        # Check file size
        file.seek(0, os.SEEK_END)
        file_size = file.tell()
        file.seek(0)
        
        if file_size > MAX_FILE_SIZE:
            return jsonify({'error': f'File too large. Maximum size: {MAX_FILE_SIZE // 1024 // 1024}MB'}), 400
        
        # Save file
        document_id = f"doc_{uuid.uuid4().hex[:16]}"
        file_path = os.path.join(UPLOAD_FOLDER, f"{document_id}_{filename}")
        file.save(file_path)
        
        # Record in database
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        cursor.execute('''
            INSERT INTO documents (id, user_id, filename, original_filename, file_path, file_size, mime_type, status)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        ''', (document_id, request.user_id, filename, file.filename, file_path, file_size, file.content_type, 'uploaded'))
        conn.commit()
        conn.close()
        
        # Start background processing
        thread = threading.Thread(
            target=process_document_async,
            args=(file_path, request.user_id, document_id, filename)
        )
        thread.daemon = True
        thread.start()
        
        logger.info(f"Document uploaded: {document_id} by {request.user_email}")
        
        return jsonify({
            'message': 'File uploaded successfully',
            'document_id': document_id,
            'filename': filename,
            'status': 'processing'
        })
        
    except Exception as e:
        logger.error(f"Upload error: {str(e)}")
        return jsonify({'error': 'Upload failed'}), 500

@app.route('/api/v1/processing-status/<document_id>', methods=['GET'])
@require_auth
def get_processing_status(document_id):
    """Get processing status for a document"""
    try:
        # Check if document belongs to user
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        cursor.execute('SELECT status FROM documents WHERE id = ? AND user_id = ?', (document_id, request.user_id))
        doc = cursor.fetchone()
        conn.close()
        
        if not doc:
            return jsonify({'error': 'Document not found'}), 404
        
        # Get processing status
        status = processing_status.get(document_id, {
            'status': doc[0],
            'progress': 100 if doc[0] == 'processed' else 0,
            'message': 'Processing completed' if doc[0] == 'processed' else 'Processing...',
            'transactions_found': 0
        })
        
        return jsonify(status)
        
    except Exception as e:
        logger.error(f"Status check error: {str(e)}")
        return jsonify({'error': 'Failed to get status'}), 500

# Transaction endpoints
@app.route('/api/v1/transactions', methods=['GET'])
@require_auth
def get_transactions():
    """Get user transactions with pagination"""
    try:
        page = int(request.args.get('page', 1))
        limit = min(int(request.args.get('limit', 20)), 100)
        category = request.args.get('category')
        
        offset = (page - 1) * limit
        
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        
        # Build query
        base_query = '''
            SELECT t.id, t.date, t.description, t.amount, c.name as category, 
                   c.color as category_color, c.icon as category_icon,
                   t.file_source, t.notes, t.created_at
            FROM transactions t
            LEFT JOIN categories c ON t.category_id = c.id
            WHERE t.user_id = ?
        '''
        
        params = [request.user_id]
        if category:
            base_query += ' AND c.name = ?'
            params.append(category)
        
        base_query += ' ORDER BY t.date DESC, t.created_at DESC LIMIT ? OFFSET ?'
        params.extend([limit, offset])
        
        cursor.execute(base_query, params)
        
        transactions = []
        for row in cursor.fetchall():
            transactions.append({
                'id': row[0],
                'date': row[1],
                'description': row[2],
                'amount': row[3],
                'category': row[4] or 'Other',
                'category_color': row[5] or '#6b7280',
                'category_icon': row[6] or '📦',
                'file_source': row[7],
                'notes': row[8],
                'created_at': row[9]
            })
        
        # Get total count
        count_query = 'SELECT COUNT(*) FROM transactions WHERE user_id = ?'
        count_params = [request.user_id]
        if category:
            count_query += ' AND category_id = (SELECT id FROM categories WHERE name = ?)'
            count_params.append(category)
        
        cursor.execute(count_query, count_params)
        total = cursor.fetchone()[0]
        
        conn.close()
        
        return jsonify({
            'transactions': transactions,
            'pagination': {
                'page': page,
                'limit': limit,
                'total': total,
                'pages': (total + limit - 1) // limit
            }
        })
        
    except Exception as e:
        logger.error(f"Get transactions error: {str(e)}")
        return jsonify({'error': 'Failed to fetch transactions'}), 500

@app.route('/api/v1/documents', methods=['GET'])
@require_auth
def get_documents():
    """Get user's uploaded documents"""
    try:
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        cursor.execute('''
            SELECT id, filename, original_filename, file_size, status, 
                   transactions_extracted, error_message, created_at, processed_at
            FROM documents 
            WHERE user_id = ? 
            ORDER BY created_at DESC
        ''', (request.user_id,))
        
        documents = []
        for row in cursor.fetchall():
            documents.append({
                'id': row[0],
                'filename': row[1],
                'original_filename': row[2],
                'file_size': row[3],
                'status': row[4],
                'transactions_extracted': row[5] or 0,
                'error_message': row[6],
                'created_at': row[7],
                'processed_at': row[8]
            })
        
        conn.close()
        
        return jsonify({'documents': documents})
        
    except Exception as e:
        logger.error(f"Get documents error: {str(e)}")
        return jsonify({'error': 'Failed to fetch documents'}), 500

# Settings endpoints
@app.route('/api/v1/settings', methods=['GET'])
@require_auth
def get_settings():
    """Get user settings"""
    try:
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        cursor.execute('''
            SELECT setting_key, setting_value 
            FROM user_settings 
            WHERE user_id = ?
        ''', (request.user_id,))
        
        settings = {}
        for row in cursor.fetchall():
            settings[row[0]] = row[1]
        
        # Default settings
        default_settings = {
            'currency': 'USD',
            'date_format': 'MM/DD/YYYY',
            'default_category': 'other',
            'auto_categorization': 'enabled',
            'notifications': 'enabled'
        }
        
        # Merge with defaults
        for key, value in default_settings.items():
            if key not in settings:
                settings[key] = value
        
        conn.close()
        
        return jsonify({'settings': settings})
        
    except Exception as e:
        logger.error(f"Get settings error: {str(e)}")
        return jsonify({'error': 'Failed to fetch settings'}), 500

@app.route('/api/v1/settings', methods=['POST'])
@require_auth
def update_settings():
    """Update user settings"""
    try:
        data = request.get_json()
        settings = data.get('settings', {})
        
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        
        for key, value in settings.items():
            cursor.execute('''
                INSERT OR REPLACE INTO user_settings (id, user_id, setting_key, setting_value, updated_at)
                VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
            ''', (f"set_{uuid.uuid4().hex[:16]}", request.user_id, key, str(value)))
        
        conn.commit()
        conn.close()
        
        logger.info(f"Settings updated for user: {request.user_email}")
        
        return jsonify({'message': 'Settings updated successfully'})
        
    except Exception as e:
        logger.error(f"Update settings error: {str(e)}")
        return jsonify({'error': 'Failed to update settings'}), 500

# Dashboard endpoint
@app.route('/api/v1/dashboard', methods=['GET'])
@require_auth
def get_dashboard():
    """Get dashboard data"""
    try:
        conn = sqlite3.connect(DATABASE_PATH)
        cursor = conn.cursor()
        
        # Get recent transactions
        cursor.execute('''
            SELECT t.id, t.date, t.description, t.amount, c.name as category, c.color, c.icon
            FROM transactions t
            LEFT JOIN categories c ON t.category_id = c.id
            WHERE t.user_id = ?
            ORDER BY t.created_at DESC LIMIT 5
        ''', (request.user_id,))
        recent_transactions = [
            {
                'id': r[0], 'date': r[1], 'description': r[2], 'amount': r[3], 
                'category': r[4] or 'Other', 'color': r[5] or '#6b7280', 'icon': r[6] or '📦'
            }
            for r in cursor.fetchall()
        ]
        
        # Get spending by category (current month)
        cursor.execute('''
            SELECT c.name, SUM(t.amount), c.color, c.icon
            FROM transactions t
            LEFT JOIN categories c ON t.category_id = c.id
            WHERE t.user_id = ? AND t.amount < 0 
            AND date(t.date) >= date('now', 'start of month')
            GROUP BY c.id, c.name
            ORDER BY SUM(t.amount)
        ''', (request.user_id,))
        spending_by_category = [
            {
                'category': r[0] or 'Other', 
                'amount': abs(r[1]), 
                'color': r[2] or '#6b7280',
                'icon': r[3] or '📦'
            }
            for r in cursor.fetchall()
        ]
        
        # Get monthly summary
        cursor.execute('''
            SELECT 
                SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END) as income,
                SUM(CASE WHEN amount < 0 THEN ABS(amount) ELSE 0 END) as expenses,
                COUNT(*) as transaction_count
            FROM transactions 
            WHERE user_id = ? AND date(date) >= date('now', 'start of month')
        ''', (request.user_id,))
        summary = cursor.fetchone()
        
        # Get recent documents
        cursor.execute('''
            SELECT id, filename, status, transactions_extracted, created_at
            FROM documents 
            WHERE user_id = ? 
            ORDER BY created_at DESC LIMIT 3
        ''', (request.user_id,))
        recent_documents = [
            {
                'id': r[0], 'filename': r[1], 'status': r[2], 
                'transactions_extracted': r[3] or 0, 'created_at': r[4]
            }
            for r in cursor.fetchall()
        ]
        
        conn.close()
        
        return jsonify({
            'recent_transactions': recent_transactions,
            'spending_by_category': spending_by_category,
            'monthly_summary': {
                'income': summary[0] or 0,
                'expenses': summary[1] or 0,
                'transaction_count': summary[2] or 0,
                'net': (summary[0] or 0) - (summary[1] or 0)
            },
            'recent_documents': recent_documents
        })
        
    except Exception as e:
        logger.error(f"Dashboard error: {str(e)}")
        return jsonify({'error': 'Failed to load dashboard'}), 500

# Health check
@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'service': 'localfinance-enhanced',
        'version': '1.1.0',
        'timestamp': datetime.utcnow().isoformat(),
        'database': 'connected' if os.path.exists(DATABASE_PATH) else 'not_found',
        'upload_folder': 'available' if os.path.exists(UPLOAD_FOLDER) else 'not_found'
    })

# Enhanced Frontend with Left Sidebar
@app.route('/')
def serve_frontend():
    """Serve the enhanced frontend with left sidebar navigation"""
    return render_template_string(ENHANCED_FRONTEND_TEMPLATE)

# Frontend template constant
ENHANCED_FRONTEND_TEMPLATE = '''
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🏛️ LocalFinance - Privacy-First Personal Finance</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8fafc; }
        
        /* Authentication styles */
        .auth-container { 
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); 
            min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 20px; 
        }
        .auth-card { 
            background: white; border-radius: 16px; padding: 32px; max-width: 400px; width: 100%; 
            box-shadow: 0 20px 50px rgba(0,0,0,0.15); 
        }
        .auth-header { text-align: center; margin-bottom: 32px; }
        .auth-header h1 { font-size: 2rem; color: #1f2937; margin-bottom: 8px; }
        .auth-header p { color: #6b7280; }
        
        .form-group { margin-bottom: 20px; }
        .form-group label { display: block; margin-bottom: 8px; font-weight: 600; color: #374151; }
        .form-group input { 
            width: 100%; padding: 14px 16px; border: 2px solid #e5e7eb; border-radius: 10px; 
            font-size: 16px; transition: border-color 0.3s, box-shadow 0.3s; 
        }
        .form-group input:focus { 
            outline: none; border-color: #667eea; 
            box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1); 
        }
        
        .btn { 
            width: 100%; padding: 14px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); 
            color: white; border: none; border-radius: 10px; font-size: 16px; font-weight: 600; 
            cursor: pointer; transition: transform 0.2s, box-shadow 0.2s; 
        }
        .btn:hover { transform: translateY(-2px); box-shadow: 0 8px 25px rgba(102, 126, 234, 0.3); }
        .btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none; }
        .btn-secondary { 
            background: #f3f4f6; color: #374151; margin-top: 12px; 
            box-shadow: none; 
        }
        .btn-secondary:hover { background: #e5e7eb; transform: translateY(-1px); }
        
        .alert { padding: 14px 16px; border-radius: 10px; margin-bottom: 20px; font-size: 14px; }
        .alert-success { background: #d1fae5; color: #065f46; border-left: 4px solid #10b981; }
        .alert-error { background: #fee2e2; color: #991b1b; border-left: 4px solid #ef4444; }
        
        /* Main app layout */
        .app-container { display: none; min-height: 100vh; }
        .sidebar { 
            width: 280px; background: #1f2937; color: white; position: fixed; 
            height: 100vh; overflow-y: auto; box-shadow: 2px 0 10px rgba(0,0,0,0.1); 
        }
        .sidebar-header { 
            padding: 24px 20px; border-bottom: 1px solid #374151; 
            text-align: center; 
        }
        .sidebar-header h2 { font-size: 1.5rem; margin-bottom: 4px; }
        .sidebar-header p { font-size: 0.875rem; color: #9ca3af; }
        
        .nav-menu { padding: 20px 0; }
        .nav-item { 
            display: flex; align-items: center; padding: 12px 24px; 
            color: #d1d5db; cursor: pointer; transition: all 0.2s; 
            border-left: 3px solid transparent; 
        }
        .nav-item:hover { background: #374151; color: white; }
        .nav-item.active { 
            background: #1e40af; color: white; 
            border-left-color: #3b82f6; 
        }
        .nav-item-icon { margin-right: 12px; font-size: 1.2rem; }
        .nav-item-text { font-weight: 500; }
        
        .user-info { 
            position: absolute; bottom: 0; left: 0; right: 0; 
            padding: 20px; border-top: 1px solid #374151; 
        }
        .user-info-header { 
            display: flex; align-items: center; margin-bottom: 12px; 
        }
        .user-avatar { 
            width: 40px; height: 40px; border-radius: 50%; 
            background: linear-gradient(135deg, #667eea, #764ba2); 
            display: flex; align-items: center; justify-content: center; 
            color: white; font-weight: bold; margin-right: 12px; 
        }
        .user-details h4 { font-size: 14px; margin-bottom: 2px; }
        .user-details p { font-size: 12px; color: #9ca3af; }
        .logout-btn { 
            width: 100%; padding: 8px; background: #374151; color: #d1d5db; 
            border: 1px solid #4b5563; border-radius: 6px; font-size: 13px; 
            cursor: pointer; transition: all 0.2s; 
        }
        .logout-btn:hover { background: #4b5563; color: white; }
        
        /* Main content */
        .main-content { 
            margin-left: 280px; padding: 32px; min-height: 100vh; 
            background: #f8fafc; 
        }
        .page-header { 
            margin-bottom: 32px; padding-bottom: 20px; 
            border-bottom: 1px solid #e5e7eb; 
        }
        .page-header h1 { 
            font-size: 2rem; color: #1f2937; margin-bottom: 8px; 
        }
        .page-header p { color: #6b7280; font-size: 1rem; }
        
        /* Content sections */
        .content-section { 
            background: white; border-radius: 12px; padding: 24px; 
            margin-bottom: 24px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); 
        }
        
        /* Upload section */
        .upload-area { 
            border: 3px dashed #d1d5db; border-radius: 12px; padding: 48px 24px; 
            text-align: center; transition: all 0.3s; cursor: pointer; 
        }
        .upload-area:hover { border-color: #667eea; background: #f8fafc; }
        .upload-area.dragover { 
            border-color: #667eea; background: #eff6ff; 
            transform: scale(1.02); 
        }
        .upload-icon { font-size: 3rem; margin-bottom: 16px; color: #6b7280; }
        .upload-text { color: #374151; margin-bottom: 16px; }
        .upload-text h3 { font-size: 1.25rem; margin-bottom: 8px; }
        .upload-text p { color: #6b7280; }
        .file-input { display: none; }
        .upload-btn { 
            background: #667eea; color: white; border: none; 
            padding: 12px 24px; border-radius: 8px; font-weight: 600; 
            cursor: pointer; margin-top: 16px; 
        }
        
        /* Processing status */
        .processing-status { 
            background: #eff6ff; border: 1px solid #bfdbfe; 
            border-radius: 8px; padding: 20px; margin-top: 24px; 
        }
        .status-header { 
            display: flex; align-items: center; margin-bottom: 16px; 
        }
        .status-icon { margin-right: 12px; font-size: 1.5rem; }
        .progress-bar { 
            width: 100%; height: 8px; background: #e5e7eb; 
            border-radius: 4px; overflow: hidden; margin-bottom: 12px; 
        }
        .progress-fill { 
            height: 100%; background: linear-gradient(90deg, #667eea, #764ba2); 
            transition: width 0.3s ease; 
        }
        .status-message { color: #374151; font-size: 14px; }
        
        /* Documents list */
        .documents-grid { 
            display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); 
            gap: 20px; margin-top: 24px; 
        }
        .document-card { 
            background: white; border: 1px solid #e5e7eb; border-radius: 8px; 
            padding: 20px; transition: all 0.2s; 
        }
        .document-card:hover { 
            transform: translateY(-2px); 
            box-shadow: 0 4px 12px rgba(0,0,0,0.1); 
        }
        .document-header { 
            display: flex; align-items: center; justify-content: between; margin-bottom: 12px; 
        }
        .document-icon { margin-right: 12px; font-size: 1.5rem; }
        .document-status { 
            padding: 4px 8px; border-radius: 4px; font-size: 12px; 
            font-weight: 600; margin-left: auto; 
        }
        .status-processed { background: #d1fae5; color: #065f46; }
        .status-processing { background: #fef3c7; color: #92400e; }
        .status-error { background: #fee2e2; color: #991b1b; }
        
        /* Settings form */
        .settings-grid { 
            display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); 
            gap: 24px; 
        }
        .setting-group h3 { 
            font-size: 1.125rem; color: #374151; margin-bottom: 16px; 
        }
        .setting-item { margin-bottom: 20px; }
        .setting-item label { 
            display: block; margin-bottom: 8px; font-weight: 500; color: #374151; 
        }
        .setting-item select, .setting-item input { 
            width: 100%; padding: 10px 12px; border: 2px solid #e5e7eb; 
            border-radius: 6px; font-size: 14px; 
        }
        .setting-item select:focus, .setting-item input:focus { 
            outline: none; border-color: #667eea; 
        }
        
        /* Utility classes */
        .hidden { display: none !important; }
        .text-center { text-align: center; }
        .mb-4 { margin-bottom: 16px; }
        .mt-4 { margin-top: 16px; }
        
        /* Responsive design */
        @media (max-width: 768px) {
            .sidebar { 
                width: 100%; height: auto; position: relative; 
            }
            .main-content { 
                margin-left: 0; padding: 16px; 
            }
            .nav-menu { 
                display: flex; overflow-x: auto; padding: 0; 
            }
            .nav-item { 
                min-width: 120px; text-align: center; flex-direction: column; 
                padding: 12px 8px; 
            }
            .nav-item-icon { margin-right: 0; margin-bottom: 4px; }
            .user-info { position: relative; }
        }
    </style>
</head>
<body>
    <!-- Authentication Section -->
    <div id="authContainer" class="auth-container">
        <div class="auth-card">
            <div class="auth-header">
                <h1>🏛️ LocalFinance</h1>
                <p>Privacy-First Personal Finance</p>
            </div>
            
            <div id="alertContainer"></div>
            
            <!-- Login Form -->
            <div id="loginForm">
                <div class="form-group">
                    <label>Email</label>
                    <input type="email" id="loginEmail" placeholder="Enter your email">
                </div>
                <div class="form-group">
                    <label>Password</label>
                    <input type="password" id="loginPassword" placeholder="Enter your password">
                </div>
                <button class="btn" onclick="login()">Sign In</button>
                <button class="btn btn-secondary" onclick="showRegister()">Create Account</button>
            </div>
            
            <!-- Register Form -->
            <div id="registerForm" class="hidden">
                <div class="form-group">
                    <label>First Name</label>
                    <input type="text" id="regFirstName" placeholder="Enter your first name">
                </div>
                <div class="form-group">
                    <label>Last Name</label>
                    <input type="text" id="regLastName" placeholder="Enter your last name">
                </div>
                <div class="form-group">
                    <label>Email</label>
                    <input type="email" id="regEmail" placeholder="Enter your email">
                </div>
                <div class="form-group">
                    <label>Password</label>
                    <input type="password" id="regPassword" placeholder="Create a password (min 6 chars)">
                </div>
                <button class="btn" onclick="register()">Create Account</button>
                <button class="btn btn-secondary" onclick="showLogin()">Back to Sign In</button>
            </div>
        </div>
    </div>
    
    <!-- Main Application -->
    <div id="appContainer" class="app-container">
        <!-- Sidebar -->
        <div class="sidebar">
            <div class="sidebar-header">
                <h2>🏛️ LocalFinance</h2>
                <p>Privacy-First Finance</p>
            </div>
            
            <nav class="nav-menu">
                <div class="nav-item active" onclick="showSection('dashboard')">
                    <span class="nav-item-icon">📊</span>
                    <span class="nav-item-text">Dashboard</span>
                </div>
                <div class="nav-item" onclick="showSection('upload')">
                    <span class="nav-item-icon">📤</span>
                    <span class="nav-item-text">Upload Statement</span>
                </div>
                <div class="nav-item" onclick="showSection('transactions')">
                    <span class="nav-item-icon">💰</span>
                    <span class="nav-item-text">Transactions</span>
                </div>
                <div class="nav-item" onclick="showSection('documents')">
                    <span class="nav-item-icon">📄</span>
                    <span class="nav-item-text">Documents</span>
                </div>
                <div class="nav-item" onclick="showSection('settings')">
                    <span class="nav-item-icon">⚙️</span>
                    <span class="nav-item-text">Settings</span>
                </div>
            </nav>
            
            <div class="user-info">
                <div class="user-info-header">
                    <div class="user-avatar" id="userAvatar">U</div>
                    <div class="user-details">
                        <h4 id="userName">User</h4>
                        <p id="userEmail">user@example.com</p>
                    </div>
                </div>
                <button class="logout-btn" onclick="logout()">Sign Out</button>
            </div>
        </div>
        
        <!-- Main Content -->
        <div class="main-content">
            <!-- Dashboard Section -->
            <div id="dashboardSection" class="content-section">
                <div class="page-header">
                    <h1>Dashboard</h1>
                    <p>Overview of your financial activity</p>
                </div>
                
                <div id="dashboardContent">
                    <p>Loading dashboard...</p>
                </div>
            </div>
            
            <!-- Upload Section -->
            <div id="uploadSection" class="content-section hidden">
                <div class="page-header">
                    <h1>Upload Statement</h1>
                    <p>Upload bank statements, credit card statements, or transaction files</p>
                </div>
                
                <div class="upload-area" id="uploadArea" onclick="document.getElementById('fileInput').click()">
                    <div class="upload-icon">📄</div>
                    <div class="upload-text">
                        <h3>Drop files here or click to browse</h3>
                        <p>Supports CSV, PDF, Excel files (max 10MB)</p>
                    </div>
                    <button class="upload-btn">Choose File</button>
                </div>
                
                <input type="file" id="fileInput" class="file-input" 
                       accept=".csv,.pdf,.xlsx,.xls,.txt" 
                       onchange="uploadFile(this.files[0])">
                
                <div id="processingContainer"></div>
                
                <div class="content-section">
                    <h3>Recent Uploads</h3>
                    <div id="recentUploads">
                        <p>No recent uploads</p>
                    </div>
                </div>
            </div>
            
            <!-- Transactions Section -->
            <div id="transactionsSection" class="content-section hidden">
                <div class="page-header">
                    <h1>Transactions</h1>
                    <p>View and manage your financial transactions</p>
                </div>
                
                <div id="transactionsContent">
                    <p>Loading transactions...</p>
                </div>
            </div>
            
            <!-- Documents Section -->
            <div id="documentsSection" class="content-section hidden">
                <div class="page-header">
                    <h1>Documents</h1>
                    <p>Manage uploaded financial documents</p>
                </div>
                
                <div id="documentsContent">
                    <p>Loading documents...</p>
                </div>
            </div>
            
            <!-- Settings Section -->
            <div id="settingsSection" class="content-section hidden">
                <div class="page-header">
                    <h1>Settings</h1>
                    <p>Configure your LocalFinance preferences</p>
                </div>
                
                <div class="settings-grid">
                    <div class="setting-group">
                        <h3>General Settings</h3>
                        <div class="setting-item">
                            <label>Currency</label>
                            <select id="currencySetting">
                                <option value="USD">USD - US Dollar</option>
                                <option value="EUR">EUR - Euro</option>
                                <option value="GBP">GBP - British Pound</option>
                                <option value="CAD">CAD - Canadian Dollar</option>
                            </select>
                        </div>
                        <div class="setting-item">
                            <label>Date Format</label>
                            <select id="dateFormatSetting">
                                <option value="MM/DD/YYYY">MM/DD/YYYY</option>
                                <option value="DD/MM/YYYY">DD/MM/YYYY</option>
                                <option value="YYYY-MM-DD">YYYY-MM-DD</option>
                            </select>
                        </div>
                    </div>
                    
                    <div class="setting-group">
                        <h3>Transaction Settings</h3>
                        <div class="setting-item">
                            <label>Default Category</label>
                            <select id="defaultCategorySetting">
                                <option value="other">Other</option>
                                <option value="food">Food & Dining</option>
                                <option value="transport">Transportation</option>
                                <option value="shopping">Shopping</option>
                                <option value="utilities">Utilities</option>
                            </select>
                        </div>
                        <div class="setting-item">
                            <label>Auto-Categorization</label>
                            <select id="autoCategorySetting">
                                <option value="enabled">Enabled</option>
                                <option value="disabled">Disabled</option>
                            </select>
                        </div>
                    </div>
                </div>
                
                <div class="mt-4">
                    <button class="btn" onclick="saveSettings()">Save Settings</button>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        let currentUser = null;
        let currentSection = 'dashboard';
        
        // Check if already logged in
        if (localStorage.getItem('token') && localStorage.getItem('user')) {
            currentUser = JSON.parse(localStorage.getItem('user'));
            showApp();
        }
        
        function showAlert(message, type = 'success') {
            const container = document.getElementById('alertContainer');
            container.innerHTML = `<div class="alert alert-${type}">${message}</div>`;
            setTimeout(() => container.innerHTML = '', 5000);
        }
        
        function showLogin() {
            document.getElementById('loginForm').classList.remove('hidden');
            document.getElementById('registerForm').classList.add('hidden');
        }
        
        function showRegister() {
            document.getElementById('loginForm').classList.add('hidden');
            document.getElementById('registerForm').classList.remove('hidden');
        }
        
        function showApp() {
            document.getElementById('authContainer').style.display = 'none';
            document.getElementById('appContainer').style.display = 'block';
            
            if (currentUser) {
                document.getElementById('userName').textContent = currentUser.first_name;
                document.getElementById('userEmail').textContent = currentUser.email;
                document.getElementById('userAvatar').textContent = currentUser.first_name.charAt(0).toUpperCase();
            }
            
            // Load initial section
            showSection('dashboard');
        }
        
        function showSection(section) {
            // Hide all sections
            const sections = ['dashboard', 'upload', 'transactions', 'documents', 'settings'];
            sections.forEach(s => {
                document.getElementById(s + 'Section').classList.add('hidden');
                document.querySelector(`[onclick="showSection('${s}')"]`).classList.remove('active');
            });
            
            // Show selected section
            document.getElementById(section + 'Section').classList.remove('hidden');
            document.querySelector(`[onclick="showSection('${section}')"]`).classList.add('active');
            
            currentSection = section;
            
            // Load section content
            switch(section) {
                case 'dashboard':
                    loadDashboard();
                    break;
                case 'upload':
                    loadRecentUploads();
                    break;
                case 'transactions':
                    loadTransactions();
                    break;
                case 'documents':
                    loadDocuments();
                    break;
                case 'settings':
                    loadSettings();
                    break;
            }
        }
        
        async function login() {
            const email = document.getElementById('loginEmail').value;
            const password = document.getElementById('loginPassword').value;
            
            if (!email || !password) {
                showAlert('Please fill in all fields', 'error');
                return;
            }
            
            try {
                const response = await fetch('/api/auth/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ email, password })
                });
                
                const data = await response.json();
                if (response.ok) {
                    localStorage.setItem('token', data.token);
                    localStorage.setItem('user', JSON.stringify(data.user));
                    currentUser = data.user;
                    showApp();
                    showAlert('Login successful!');
                } else {
                    showAlert(data.error, 'error');
                }
            } catch (error) {
                showAlert('Login failed. Please try again.', 'error');
            }
        }
        
        async function register() {
            const firstName = document.getElementById('regFirstName').value;
            const lastName = document.getElementById('regLastName').value;
            const email = document.getElementById('regEmail').value;
            const password = document.getElementById('regPassword').value;
            
            if (!firstName || !email || !password) {
                showAlert('Please fill in required fields', 'error');
                return;
            }
            
            if (password.length < 6) {
                showAlert('Password must be at least 6 characters', 'error');
                return;
            }
            
            try {
                const response = await fetch('/api/auth/register', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ first_name: firstName, last_name: lastName, email, password })
                });
                
                const data = await response.json();
                if (response.ok) {
                    localStorage.setItem('token', data.token);
                    localStorage.setItem('user', JSON.stringify(data.user));
                    currentUser = data.user;
                    showApp();
                    showAlert('Account created successfully!');
                } else {
                    showAlert(data.error, 'error');
                }
            } catch (error) {
                showAlert('Registration failed. Please try again.', 'error');
            }
        }
        
        async function logout() {
            try {
                const token = localStorage.getItem('token');
                if (token) {
                    await fetch('/api/auth/logout', {
                        method: 'POST',
                        headers: { 'Authorization': `Bearer ${token}` }
                    });
                }
            } catch (error) {
                console.error('Logout error:', error);
            }
            
            localStorage.removeItem('token');
            localStorage.removeItem('user');
            currentUser = null;
            document.getElementById('authContainer').style.display = 'block';
            document.getElementById('appContainer').style.display = 'none';
            showLogin();
            showAlert('Logged out successfully');
        }
        
        // Upload functionality
        function setupUploadArea() {
            const uploadArea = document.getElementById('uploadArea');
            
            uploadArea.addEventListener('dragover', (e) => {
                e.preventDefault();
                uploadArea.classList.add('dragover');
            });
            
            uploadArea.addEventListener('dragleave', () => {
                uploadArea.classList.remove('dragover');
            });
            
            uploadArea.addEventListener('drop', (e) => {
                e.preventDefault();
                uploadArea.classList.remove('dragover');
                const files = e.dataTransfer.files;
                if (files.length > 0) {
                    uploadFile(files[0]);
                }
            });
        }
        
        async function uploadFile(file) {
            if (!file) return;
            
            const allowedTypes = ['text/csv', 'application/pdf', 'application/vnd.ms-excel', 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'];
            
            if (!allowedTypes.some(type => type === file.type) && !file.name.toLowerCase().endsWith('.csv')) {
                showAlert('Please upload a CSV, PDF, or Excel file', 'error');
                return;
            }
            
            if (file.size > 10 * 1024 * 1024) {
                showAlert('File size must be less than 10MB', 'error');
                return;
            }
            
            const formData = new FormData();
            formData.append('file', file);
            
            try {
                const response = await fetch('/api/v1/upload', {
                    method: 'POST',
                    headers: {
                        'Authorization': `Bearer ${localStorage.getItem('token')}`
                    },
                    body: formData
                });
                
                const data = await response.json();
                if (response.ok) {
                    showAlert('File uploaded successfully! Processing...', 'success');
                    showProcessingStatus(data.document_id, file.name);
                    startStatusPolling(data.document_id);
                } else {
                    showAlert(data.error, 'error');
                }
            } catch (error) {
                showAlert('Upload failed. Please try again.', 'error');
            }
        }
        
        function showProcessingStatus(documentId, filename) {
            const container = document.getElementById('processingContainer');
            container.innerHTML = `
                <div class="processing-status" id="status-${documentId}">
                    <div class="status-header">
                        <span class="status-icon">⚡</span>
                        <div>
                            <h4>Processing: ${filename}</h4>
                            <p class="status-message">Starting analysis...</p>
                        </div>
                    </div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: 0%"></div>
                    </div>
                </div>
            `;
        }
        
        async function startStatusPolling(documentId) {
            const pollStatus = async () => {
                try {
                    const response = await fetch(`/api/v1/processing-status/${documentId}`, {
                        headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
                    });
                    
                    const status = await response.json();
                    updateProcessingStatus(documentId, status);
                    
                    if (status.status === 'processing') {
                        setTimeout(pollStatus, 1000);
                    } else if (status.status === 'completed') {
                        loadRecentUploads();
                        setTimeout(() => {
                            document.getElementById('processingContainer').innerHTML = '';
                        }, 3000);
                    }
                } catch (error) {
                    console.error('Status polling error:', error);
                }
            };
            
            pollStatus();
        }
        
        function updateProcessingStatus(documentId, status) {
            const statusEl = document.getElementById(`status-${documentId}`);
            if (!statusEl) return;
            
            const progressFill = statusEl.querySelector('.progress-fill');
            const message = statusEl.querySelector('.status-message');
            
            progressFill.style.width = `${status.progress}%`;
            message.textContent = status.message;
            
            if (status.status === 'completed') {
                statusEl.style.background = '#d1fae5';
                statusEl.style.borderColor = '#10b981';
            } else if (status.status === 'error') {
                statusEl.style.background = '#fee2e2';
                statusEl.style.borderColor = '#ef4444';
            }
        }
        
        // Load functions for each section
        async function loadDashboard() {
            const container = document.getElementById('dashboardContent');
            container.innerHTML = '<p>Loading dashboard...</p>';
            
            try {
                const response = await fetch('/api/v1/dashboard', {
                    headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
                });
                const data = await response.json();
                
                const summary = data.monthly_summary;
                const transactions = data.recent_transactions;
                
                container.innerHTML = `
                    <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 32px;">
                        <div class="content-section">
                            <h3>💰 Monthly Income</h3>
                            <p style="font-size: 2rem; color: #059669; font-weight: bold;">$${summary.income.toFixed(2)}</p>
                        </div>
                        <div class="content-section">
                            <h3>💸 Monthly Expenses</h3>
                            <p style="font-size: 2rem; color: #dc2626; font-weight: bold;">$${summary.expenses.toFixed(2)}</p>
                        </div>
                        <div class="content-section">
                            <h3>📊 Net Income</h3>
                            <p style="font-size: 2rem; color: ${summary.net >= 0 ? '#059669' : '#dc2626'}; font-weight: bold;">$${summary.net.toFixed(2)}</p>
                        </div>
                        <div class="content-section">
                            <h3>📝 Transactions</h3>
                            <p style="font-size: 2rem; color: #6366f1; font-weight: bold;">${summary.transaction_count}</p>
                        </div>
                    </div>
                    
                    <div class="content-section">
                        <h3>Recent Transactions</h3>
                        <div style="margin-top: 16px;">
                            ${transactions.map(t => `
                                <div style="display: flex; justify-content: space-between; align-items: center; padding: 12px 0; border-bottom: 1px solid #e5e7eb;">
                                    <div>
                                        <div style="font-weight: 600; color: #374151;">${t.description}</div>
                                        <div style="font-size: 0.875rem; color: #6b7280;">${t.date} • ${t.category}</div>
                                    </div>
                                    <div style="font-weight: bold; color: ${t.amount >= 0 ? '#059669' : '#dc2626'};">
                                        ${t.amount >= 0 ? '+' : '-'}$${Math.abs(t.amount).toFixed(2)}
                                    </div>
                                </div>
                            `).join('')}
                        </div>
                    </div>
                `;
            } catch (error) {
                container.innerHTML = '<p>Error loading dashboard</p>';
            }
        }
        
        async function loadTransactions() {
            const container = document.getElementById('transactionsContent');
            container.innerHTML = '<p>Loading transactions...</p>';
            
            try {
                const response = await fetch('/api/v1/transactions?limit=50', {
                    headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
                });
                const data = await response.json();
                
                container.innerHTML = `
                    <div style="margin-bottom: 20px;">
                        <p>Total: ${data.pagination.total} transactions</p>
                    </div>
                    ${data.transactions.map(t => `
                        <div style="background: white; border: 1px solid #e5e7eb; border-radius: 8px; padding: 16px; margin-bottom: 12px;">
                            <div style="display: flex; justify-content: space-between; align-items: center;">
                                <div>
                                    <div style="font-weight: 600; color: #374151;">${t.description}</div>
                                    <div style="font-size: 0.875rem; color: #6b7280; margin-top: 4px;">
                                        ${t.date} • ${t.category} ${t.category_icon}
                                    </div>
                                </div>
                                <div style="font-weight: bold; font-size: 1.125rem; color: ${t.amount >= 0 ? '#059669' : '#dc2626'};">
                                    ${t.amount >= 0 ? '+' : '-'}$${Math.abs(t.amount).toFixed(2)}
                                </div>
                            </div>
                        </div>
                    `).join('')}
                `;
            } catch (error) {
                container.innerHTML = '<p>Error loading transactions</p>';
            }
        }
        
        async function loadDocuments() {
            const container = document.getElementById('documentsContent');
            container.innerHTML = '<p>Loading documents...</p>';
            
            try {
                const response = await fetch('/api/v1/documents', {
                    headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
                });
                const data = await response.json();
                
                if (data.documents.length === 0) {
                    container.innerHTML = '<p>No documents uploaded yet</p>';
                    return;
                }
                
                container.innerHTML = `
                    <div class="documents-grid">
                        ${data.documents.map(d => `
                            <div class="document-card">
                                <div class="document-header">
                                    <span class="document-icon">📄</span>
                                    <div style="flex: 1;">
                                        <h4>${d.original_filename}</h4>
                                        <p style="color: #6b7280; font-size: 0.875rem;">
                                            ${(d.file_size / 1024).toFixed(1)} KB • ${d.created_at}
                                        </p>
                                    </div>
                                    <span class="document-status status-${d.status}">${d.status}</span>
                                </div>
                                ${d.status === 'processed' ? `
                                    <p style="color: #059669; font-weight: 600; margin-top: 8px;">
                                        ✅ ${d.transactions_extracted} transactions extracted
                                    </p>
                                ` : ''}
                                ${d.error_message ? `
                                    <p style="color: #dc2626; font-size: 0.875rem; margin-top: 8px;">
                                        ❌ ${d.error_message}
                                    </p>
                                ` : ''}
                            </div>
                        `).join('')}
                    </div>
                `;
            } catch (error) {
                container.innerHTML = '<p>Error loading documents</p>';
            }
        }
        
        async function loadRecentUploads() {
            const container = document.getElementById('recentUploads');
            
            try {
                const response = await fetch('/api/v1/documents', {
                    headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
                });
                const data = await response.json();
                
                const recent = data.documents.slice(0, 3);
                
                if (recent.length === 0) {
                    container.innerHTML = '<p>No recent uploads</p>';
                    return;
                }
                
                container.innerHTML = recent.map(d => `
                    <div style="display: flex; justify-content: space-between; align-items: center; padding: 12px 0; border-bottom: 1px solid #e5e7eb;">
                        <div>
                            <div style="font-weight: 600;">${d.original_filename}</div>
                            <div style="font-size: 0.875rem; color: #6b7280;">${d.created_at}</div>
                        </div>
                        <span class="document-status status-${d.status}">${d.status}</span>
                    </div>
                `).join('');
            } catch (error) {
                container.innerHTML = '<p>Error loading recent uploads</p>';
            }
        }
        
        async function loadSettings() {
            try {
                const response = await fetch('/api/v1/settings', {
                    headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
                });
                const data = await response.json();
                const settings = data.settings;
                
                document.getElementById('currencySetting').value = settings.currency || 'USD';
                document.getElementById('dateFormatSetting').value = settings.date_format || 'MM/DD/YYYY';
                document.getElementById('defaultCategorySetting').value = settings.default_category || 'other';
                document.getElementById('autoCategorySetting').value = settings.auto_categorization || 'enabled';
            } catch (error) {
                console.error('Error loading settings:', error);
            }
        }
        
        async function saveSettings() {
            const settings = {
                currency: document.getElementById('currencySetting').value,
                date_format: document.getElementById('dateFormatSetting').value,
                default_category: document.getElementById('defaultCategorySetting').value,
                auto_categorization: document.getElementById('autoCategorySetting').value
            };
            
            try {
                const response = await fetch('/api/v1/settings', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${localStorage.getItem('token')}`
                    },
                    body: JSON.stringify({ settings })
                });
                
                if (response.ok) {
                    showAlert('Settings saved successfully!');
                } else {
                    showAlert('Error saving settings', 'error');
                }
            } catch (error) {
                showAlert('Error saving settings', 'error');
            }
        }
        
        // Setup event listeners
        document.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                if (!document.getElementById('loginForm').classList.contains('hidden')) {
                    login();
                } else if (!document.getElementById('registerForm').classList.contains('hidden')) {
                    register();
                }
            }
        });
        
        // Initialize upload area when app loads
        document.addEventListener('DOMContentLoaded', () => {
            setupUploadArea();
        });
    </script>
</body>
</html>
'''

if __name__ == '__main__':
    try:
        init_database()
        logger.info("🏛️ LocalFinance Enhanced Server Starting...")
        logger.info("🗄️ Database initialized")
        logger.info("📤 Upload functionality ready")
        logger.info("🌐 Server available at: http://10.0.0.16:8080")
        
        app.run(
            host='0.0.0.0', 
            port=8080,
            debug=False,
            threaded=True
        )
    except Exception as e:
        logger.error(f"Failed to start server: {str(e)}")
        print(f"Error: {str(e)}")