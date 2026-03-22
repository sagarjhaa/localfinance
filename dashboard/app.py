#!/usr/bin/env python3
"""
Enhanced Hisab Web Interface - Modern Drag & Drop for Bank Statements
"""
import os
from flask import Flask, request, redirect, url_for, flash, render_template_string, jsonify
import subprocess
from pathlib import Path
import json

app = Flask(__name__)
app.secret_key = 'hisab-enhanced-upload-secret'

UPLOAD_FOLDER = '/home/sagar/localfinance/inbox/statements'
PROCESSED_FOLDER = '/home/sagar/localfinance/inbox/processed'
FAILED_FOLDER = '/home/sagar/localfinance/inbox/failed'
ALLOWED_EXTENSIONS = {'csv', 'pdf', 'txt', 'xlsx', 'xls'}

def allowed_file(filename):
    return '.' in filename and filename.rsplit('.', 1)[1].lower() in ALLOWED_EXTENSIONS

ENHANCED_HTML = '''
<!doctype html>
<html>
<head>
    <title>Hisab Finance - Document Upload</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        * { box-sizing: border-box; }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0; padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        
        .container {
            max-width: 1200px; margin: 0 auto;
            background: white; border-radius: 20px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #2196F3, #21CBF3);
            color: white; padding: 30px;
            text-align: center;
        }
        
        .header h1 { margin: 0; font-size: 2.5em; }
        .header p { margin: 10px 0 0 0; opacity: 0.9; font-size: 1.1em; }
        
        .main-content { padding: 40px; }
        
        .drop-zone {
            border: 3px dashed #ddd;
            border-radius: 15px;
            padding: 60px 20px;
            text-align: center;
            margin: 20px 0;
            background: #fafafa;
            transition: all 0.3s ease;
            cursor: pointer;
            position: relative;
        }
        
        .drop-zone.dragover {
            border-color: #2196F3;
            background: #e3f2fd;
            transform: scale(1.02);
        }
        
        .drop-zone .icon {
            font-size: 4em; color: #2196F3;
            margin-bottom: 20px; display: block;
        }
        
        .drop-zone h3 { color: #333; margin: 0 0 10px 0; }
        .drop-zone p { color: #666; margin: 0; }
        
        .file-input { display: none; }
        
        .upload-btn {
            background: linear-gradient(135deg, #4CAF50, #45a049);
            color: white; padding: 15px 30px;
            border: none; border-radius: 25px;
            font-size: 1.1em; cursor: pointer;
            margin: 20px 10px; transition: all 0.3s;
            box-shadow: 0 4px 15px rgba(76, 175, 80, 0.3);
        }
        
        .upload-btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(76, 175, 80, 0.4);
        }
        
        .upload-btn:disabled {
            background: #ccc; cursor: not-allowed;
            transform: none; box-shadow: none;
        }
        
        .file-list {
            margin: 30px 0;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
        }
        
        .file-item {
            background: white;
            border: 1px solid #eee;
            border-radius: 10px;
            padding: 20px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            transition: all 0.3s;
        }
        
        .file-item:hover { transform: translateY(-2px); }
        
        .file-preview {
            display: flex; align-items: center;
            margin-bottom: 15px;
        }
        
        .file-icon {
            font-size: 2em; margin-right: 15px;
            color: #4CAF50;
        }
        
        .file-info h4 { margin: 0; color: #333; }
        .file-info p { margin: 5px 0 0 0; color: #666; font-size: 0.9em; }
        
        .status { padding: 20px; border-radius: 10px; margin: 20px 0; }
        .status.success { background: #e8f5e8; color: #4caf50; border-left: 4px solid #4caf50; }
        .status.error { background: #ffe8e8; color: #f44336; border-left: 4px solid #f44336; }
        .status.processing { background: #e3f2fd; color: #2196f3; border-left: 4px solid #2196f3; }
        
        .progress {
            width: 100%; height: 8px;
            background: #eee; border-radius: 4px;
            overflow: hidden; margin: 10px 0;
        }
        
        .progress-bar {
            height: 100%; background: linear-gradient(90deg, #4CAF50, #45a049);
            width: 0%; transition: width 0.3s ease;
        }
        
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px; margin: 30px 0;
        }
        
        .stat-card {
            background: linear-gradient(135deg, #667eea, #764ba2);
            color: white; padding: 20px;
            border-radius: 15px; text-align: center;
        }
        
        .stat-number { font-size: 2em; font-weight: bold; }
        .stat-label { opacity: 0.9; margin-top: 5px; }
        
        .bot-commands {
            background: #f8f9fa;
            border-radius: 15px;
            padding: 30px; margin: 30px 0;
        }
        
        .command {
            background: white;
            padding: 15px; margin: 10px 0;
            border-radius: 8px;
            border-left: 4px solid #2196F3;
            font-family: 'Monaco', monospace;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
        }
        
        .loading { display: none; text-align: center; padding: 20px; }
        .spinner {
            border: 4px solid #f3f3f3;
            border-top: 4px solid #2196F3;
            border-radius: 50%; width: 40px; height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto 20px;
        }
        
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        
        .footer {
            background: #2c3e50; color: white;
            padding: 30px; text-align: center;
        }
        
        @media (max-width: 768px) {
            .container { margin: 10px; }
            .header { padding: 20px; }
            .header h1 { font-size: 2em; }
            .main-content { padding: 20px; }
            .stats { grid-template-columns: 1fr; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🏦 Hisab Finance Assistant</h1>
            <p>Drag & drop your bank statements for instant processing</p>
        </div>
        
        <div class="main-content">
            <!-- Status Messages -->
            {% with messages = get_flashed_messages(with_categories=true) %}
                {% if messages %}
                    {% for category, message in messages %}
                        <div class="status {{ 'success' if category == 'success' else 'error' }}">
                            {{ message }}
                        </div>
                    {% endfor %}
                {% endif %}
            {% endwith %}
            
            <!-- Upload Area -->
            <div class="drop-zone" id="dropZone" onclick="document.getElementById('fileInput').click()">
                <span class="icon">📄</span>
                <h3>Drop your bank statements here</h3>
                <p>Or click to browse files • Supports CSV, PDF, Excel</p>
            </div>
            
            <form id="uploadForm" method="post" enctype="multipart/form-data">
                <input type="file" id="fileInput" name="files" class="file-input" multiple accept=".csv,.pdf,.txt,.xlsx,.xls">
            </form>
            
            <!-- Selected Files Preview -->
            <div id="selectedFiles" style="display: none;">
                <h3>📋 Selected Files</h3>
                <div id="filePreview"></div>
                <button type="button" class="upload-btn" id="uploadBtn" onclick="uploadFiles()">
                    🚀 Upload & Process Files
                </button>
            </div>
            
            <!-- Processing Status -->
            <div class="loading" id="loadingDiv">
                <div class="spinner"></div>
                <p>Processing your documents...</p>
                <div class="progress">
                    <div class="progress-bar" id="progressBar"></div>
                </div>
            </div>
            
            <!-- Statistics -->
            <div class="stats">
                <div class="stat-card">
                    <div class="stat-number">{{ stats.total_files }}</div>
                    <div class="stat-label">Total Files Processed</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">{{ stats.total_transactions }}</div>
                    <div class="stat-label">Transactions Extracted</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">{{ stats.success_rate }}%</div>
                    <div class="stat-label">Success Rate</div>
                </div>
            </div>
            
            <!-- Recent Files -->
            {% if recent_files %}
            <h3>📁 Recently Processed</h3>
            <div class="file-list">
                {% for file in recent_files %}
                <div class="file-item">
                    <div class="file-preview">
                        <span class="file-icon">{{ file.icon }}</span>
                        <div class="file-info">
                            <h4>{{ file.name }}</h4>
                            <p>{{ file.date }} • {{ file.transactions }} transactions</p>
                        </div>
                    </div>
                </div>
                {% endfor %}
            </div>
            {% endif %}
            
            <!-- Bot Commands -->
            <div class="bot-commands">
                <h3>🤖 Telegram Bot Commands</h3>
                <p>After processing, use these commands with your Hisab bot:</p>
                
                <div class="command">/transactions - View recent transactions</div>
                <div class="command">/summary - Monthly spending breakdown</div>
                <div class="command">/status - Check processing status</div>
                <div class="command">💬 "What did I spend on groceries this month?"</div>
            </div>
        </div>
        
        <div class="footer">
            <p>🔒 100% Private • All processing happens locally on your Jetson</p>
            <p>Hisab Finance Assistant • Powered by Local AI</p>
        </div>
    </div>

    <script>
        const dropZone = document.getElementById('dropZone');
        const fileInput = document.getElementById('fileInput');
        const uploadForm = document.getElementById('uploadForm');
        const selectedFiles = document.getElementById('selectedFiles');
        const filePreview = document.getElementById('filePreview');
        const loadingDiv = document.getElementById('loadingDiv');
        const progressBar = document.getElementById('progressBar');

        // Drag and drop handlers
        dropZone.addEventListener('dragover', handleDragOver);
        dropZone.addEventListener('dragleave', handleDragLeave);
        dropZone.addEventListener('drop', handleDrop);
        fileInput.addEventListener('change', handleFileSelect);

        function handleDragOver(e) {
            e.preventDefault();
            dropZone.classList.add('dragover');
        }

        function handleDragLeave(e) {
            e.preventDefault();
            dropZone.classList.remove('dragover');
        }

        function handleDrop(e) {
            e.preventDefault();
            dropZone.classList.remove('dragover');
            
            const files = e.dataTransfer.files;
            fileInput.files = files;
            displaySelectedFiles(files);
        }

        function handleFileSelect(e) {
            displaySelectedFiles(e.target.files);
        }

        function displaySelectedFiles(files) {
            if (files.length === 0) return;

            filePreview.innerHTML = '';
            
            for (let file of files) {
                const fileDiv = document.createElement('div');
                fileDiv.className = 'file-item';
                
                const icon = getFileIcon(file.name);
                const size = (file.size / 1024).toFixed(1) + ' KB';
                
                fileDiv.innerHTML = `
                    <div class="file-preview">
                        <span class="file-icon">${icon}</span>
                        <div class="file-info">
                            <h4>${file.name}</h4>
                            <p>${size} • ${file.type || 'Unknown type'}</p>
                        </div>
                    </div>
                `;
                
                filePreview.appendChild(fileDiv);
            }
            
            selectedFiles.style.display = 'block';
        }

        function getFileIcon(filename) {
            const ext = filename.split('.').pop().toLowerCase();
            switch(ext) {
                case 'csv': return '📊';
                case 'pdf': return '📄';
                case 'xlsx':
                case 'xls': return '📗';
                default: return '📄';
            }
        }

        function uploadFiles() {
            const formData = new FormData();
            for (let file of fileInput.files) {
                formData.append('files', file);
            }

            selectedFiles.style.display = 'none';
            loadingDiv.style.display = 'block';
            
            // Animate progress bar
            let progress = 0;
            const progressInterval = setInterval(() => {
                progress += 10;
                progressBar.style.width = progress + '%';
                if (progress >= 90) {
                    clearInterval(progressInterval);
                }
            }, 200);

            fetch('/', {
                method: 'POST',
                body: formData
            })
            .then(response => response.text())
            .then(data => {
                clearInterval(progressInterval);
                progressBar.style.width = '100%';
                
                setTimeout(() => {
                    window.location.reload();
                }, 1000);
            })
            .catch(error => {
                console.error('Upload failed:', error);
                loadingDiv.style.display = 'none';
                alert('Upload failed. Please try again.');
            });
        }

        // Prevent default drag behaviors on the page
        ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
            document.addEventListener(eventName, preventDefaults, false);
        });

        function preventDefaults(e) {
            e.preventDefault();
            e.stopPropagation();
        }
    </script>
</body>
</html>
'''

@app.route('/', methods=['GET', 'POST'])
def enhanced_upload():
    if request.method == 'POST':
        files = request.files.getlist('files')
        uploaded_count = 0
        
        for file in files:
            if file and file.filename and allowed_file(file.filename):
                filename = file.filename
                file_path = os.path.join(UPLOAD_FOLDER, filename)
                file.save(file_path)
                uploaded_count += 1
        
        if uploaded_count > 0:
            try:
                # Process the files
                result = subprocess.run(['python3', '/home/sagar/localfinance/fixed_processor.py'], 
                                     capture_output=True, text=True, cwd='/home/sagar/localfinance')
                flash(f'✅ Successfully uploaded and processed {uploaded_count} files! Check your Telegram bot.', 'success')
            except Exception as e:
                flash(f'⚠️ Uploaded {uploaded_count} files, but processing failed: {str(e)}', 'error')
        else:
            flash('❌ No valid files uploaded. Please use CSV, PDF, or Excel files.', 'error')
    
    # Get statistics and recent files
    stats = get_processing_stats()
    recent_files = get_recent_files()
    
    return render_template_string(ENHANCED_HTML, stats=stats, recent_files=recent_files)

def get_processing_stats():
    """Get processing statistics"""
    try:
        import sqlite3
        db_path = '/home/sagar/localfinance/data/finances.db'
        
        if not os.path.exists(db_path):
            return {'total_files': 0, 'total_transactions': 0, 'success_rate': 0}
        
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        
        # Get total documents
        cursor.execute('SELECT COUNT(*) FROM documents')
        total_files = cursor.fetchone()[0]
        
        # Get total transactions
        cursor.execute('SELECT COUNT(*) FROM transactions')
        total_transactions = cursor.fetchone()[0]
        
        # Get success rate
        cursor.execute('SELECT COUNT(*) FROM documents WHERE status = "processed"')
        successful_files = cursor.fetchone()[0]
        
        success_rate = int((successful_files / total_files * 100)) if total_files > 0 else 100
        
        conn.close()
        
        return {
            'total_files': total_files,
            'total_transactions': total_transactions,
            'success_rate': success_rate
        }
    except Exception as e:
        return {'total_files': 0, 'total_transactions': 0, 'success_rate': 0}

def get_recent_files():
    """Get recent processed files"""
    recent_files = []
    try:
        processed_dir = Path(PROCESSED_FOLDER)
        if processed_dir.exists():
            files = sorted(processed_dir.glob('*'), key=lambda f: f.stat().st_mtime, reverse=True)[:6]
            
            for file_path in files:
                if file_path.is_file():
                    # Get file icon
                    ext = file_path.suffix.lower()
                    icon = '📊' if ext == '.csv' else '📄' if ext == '.pdf' else '📗' if ext in ['.xlsx', '.xls'] else '📄'
                    
                    # Try to get transaction count from database
                    transactions = get_file_transaction_count(file_path.name)
                    
                    recent_files.append({
                        'name': file_path.name,
                        'icon': icon,
                        'date': 'Recently processed',
                        'transactions': transactions
                    })
    except Exception as e:
        pass
    
    return recent_files

def get_file_transaction_count(filename):
    """Get transaction count for a specific file"""
    try:
        import sqlite3
        db_path = '/home/sagar/localfinance/data/finances.db'
        
        if not os.path.exists(db_path):
            return 0
        
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        
        cursor.execute('SELECT COUNT(*) FROM transactions WHERE file_source = ?', (filename,))
        count = cursor.fetchone()[0]
        
        conn.close()
        return count
    except:
        return 0

if __name__ == '__main__':
    # Ensure directories exist
    for folder in [UPLOAD_FOLDER, PROCESSED_FOLDER, FAILED_FOLDER]:
        os.makedirs(folder, exist_ok=True)
    
    print(f"🌐 Enhanced Hisab Web Interface starting...")
    print(f"📁 Upload folder: {UPLOAD_FOLDER}")
    print(f"🔗 Access at: http://10.0.0.16:5000")
    print(f"🎨 Features: Drag & drop, progress tracking, statistics")
    
    app.run(host='0.0.0.0', port=5000, debug=False)
