#!/usr/bin/env python3
"""
Simple web interface for uploading documents to Hisab
"""
import os
from flask import Flask, request, redirect, url_for, flash, render_template_string
import subprocess
from pathlib import Path

app = Flask(__name__)
app.secret_key = 'hisab-upload-secret'

UPLOAD_FOLDER = '/home/sagar/localfinance/inbox/statements'
ALLOWED_EXTENSIONS = {'csv', 'pdf', 'txt'}

def allowed_file(filename):
    return '.' in filename and filename.rsplit('.', 1)[1].lower() in ALLOWED_EXTENSIONS

UPLOAD_HTML = '''
<!doctype html>
<html>
<head>
    <title>Hisab Document Upload</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .upload-box { border: 2px dashed #ccc; padding: 40px; text-align: center; margin: 20px 0; }
        .btn { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 5px; cursor: pointer; }
        .success { color: green; } .error { color: red; }
        .file-list { margin: 20px 0; } .file-item { padding: 5px; border-bottom: 1px solid #eee; }
    </style>
</head>
<body>
    <h1>🏦 Hisab Document Upload</h1>
    <p>Upload your bank statements, receipts, or financial documents for processing.</p>
    
    {% with messages = get_flashed_messages(with_categories=true) %}
        {% if messages %}
            {% for category, message in messages %}
                <div class="{{ 'success' if category == 'success' else 'error' }}">{{ message }}</div>
            {% endfor %}
        {% endif %}
    {% endwith %}
    
    <form method="post" enctype="multipart/form-data" class="upload-box">
        <h3>📁 Select Documents</h3>
        <input type="file" name="files" multiple accept=".csv,.pdf,.txt">
        <br><br>
        <button type="submit" class="btn">Upload & Process</button>
    </form>
    
    <h3>📊 Recent Uploads</h3>
    <div class="file-list">
        {% for file in recent_files %}
            <div class="file-item">📄 {{ file }}</div>
        {% endfor %}
    </div>
    
    <h3>🤖 Bot Status</h3>
    <p>Enhanced Hisab bot is running. Use Telegram commands:</p>
    <ul>
        <li><code>/transactions</code> - View processed transactions</li>
        <li><code>/summary</code> - Spending breakdown</li>
        <li><code>/status</code> - Check processing status</li>
    </ul>
</body>
</html>
'''

@app.route('/', methods=['GET', 'POST'])
def upload_file():
    if request.method == 'POST':
        if 'files' not in request.files:
            flash('No files selected', 'error')
            return redirect(request.url)
        
        files = request.files.getlist('files')
        uploaded_count = 0
        
        for file in files:
            if file and file.filename and allowed_file(file.filename):
                filename = file.filename
                file_path = os.path.join(UPLOAD_FOLDER, filename)
                file.save(file_path)
                uploaded_count += 1
        
        if uploaded_count > 0:
            # Process the files
            try:
                result = subprocess.run(['python3', '/home/sagar/localfinance/fixed_processor.py'], 
                                     capture_output=True, text=True)
                flash(f'✅ Uploaded {uploaded_count} files and processed successfully!', 'success')
            except Exception as e:
                flash(f'⚠️ Uploaded {uploaded_count} files, but processing failed: {str(e)}', 'error')
        else:
            flash('❌ No valid files uploaded. Use CSV or PDF format.', 'error')
    
    # Get recent files
    recent_files = []
    try:
        processed_dir = Path('/home/sagar/localfinance/inbox/processed')
        if processed_dir.exists():
            recent_files = [f.name for f in processed_dir.glob('*') if f.is_file()][-10:]
    except:
        pass
    
    return render_template_string(UPLOAD_HTML, recent_files=recent_files)

if __name__ == '__main__':
    os.makedirs(UPLOAD_FOLDER, exist_ok=True)
    print(f"🌐 Web upload interface starting...")
    print(f"📁 Upload folder: {UPLOAD_FOLDER}")
    print(f"🔗 Access at: http://10.0.0.16:5000")
    app.run(host='0.0.0.0', port=5000, debug=True)
