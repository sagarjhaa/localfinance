const express = require('express');
const multer = require('multer');
const path = require('path');
const fs = require('fs');
const { authenticateToken } = require('../middleware/auth');

const router = express.Router();

// Create uploads directory if it doesn't exist
const uploadsDir = path.join(__dirname, '../uploads');
if (!fs.existsSync(uploadsDir)) {
  fs.mkdirSync(uploadsDir, { recursive: true });
}

// Configure multer for file uploads
const storage = multer.diskStorage({
  destination: (req, file, cb) => {
    // Create user-specific directory
    const userDir = path.join(uploadsDir, req.user.username);
    if (!fs.existsSync(userDir)) {
      fs.mkdirSync(userDir, { recursive: true });
    }
    cb(null, userDir);
  },
  filename: (req, file, cb) => {
    // Generate unique filename with timestamp
    const uniqueSuffix = Date.now() + '-' + Math.round(Math.random() * 1E9);
    const extension = path.extname(file.originalname);
    const baseName = path.basename(file.originalname, extension);
    cb(null, `${baseName}-${uniqueSuffix}${extension}`);
  }
});

// File filter for allowed file types
const fileFilter = (req, file, cb) => {
  const allowedTypes = [
    'text/csv',
    'application/vnd.ms-excel',
    'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    'text/plain',
    'application/json',
    'application/pdf'
  ];

  const allowedExtensions = ['.csv', '.xlsx', '.xls', '.txt', '.json', '.pdf'];
  const fileExtension = path.extname(file.originalname).toLowerCase();

  if (allowedTypes.includes(file.mimetype) || allowedExtensions.includes(fileExtension)) {
    cb(null, true);
  } else {
    cb(new Error('Invalid file type. Only CSV, Excel, TXT, JSON, and PDF files are allowed.'), false);
  }
};

const upload = multer({
  storage: storage,
  fileFilter: fileFilter,
  limits: {
    fileSize: 50 * 1024 * 1024, // 50MB limit
    files: 10 // Max 10 files per request
  }
});

/**
 * POST /api/upload/single
 * Upload a single file
 */
router.post('/single', authenticateToken, upload.single('file'), (req, res) => {
  try {
    if (!req.file) {
      return res.status(400).json({
        message: 'No file uploaded',
        code: 'NO_FILE'
      });
    }

    const fileInfo = {
      id: Date.now().toString(),
      originalName: req.file.originalname,
      filename: req.file.filename,
      size: req.file.size,
      mimetype: req.file.mimetype,
      path: req.file.path,
      uploadedBy: req.user.username,
      uploadedAt: new Date().toISOString(),
      status: 'uploaded'
    };

    res.json({
      message: 'File uploaded successfully',
      file: fileInfo
    });

  } catch (error) {
    console.error('File upload error:', error);
    res.status(500).json({
      message: 'File upload failed',
      code: 'UPLOAD_ERROR'
    });
  }
});

/**
 * POST /api/upload/multiple
 * Upload multiple files
 */
router.post('/multiple', authenticateToken, upload.array('files', 10), (req, res) => {
  try {
    if (!req.files || req.files.length === 0) {
      return res.status(400).json({
        message: 'No files uploaded',
        code: 'NO_FILES'
      });
    }

    const filesInfo = req.files.map(file => ({
      id: Date.now().toString() + Math.random().toString(36).substr(2, 9),
      originalName: file.originalname,
      filename: file.filename,
      size: file.size,
      mimetype: file.mimetype,
      path: file.path,
      uploadedBy: req.user.username,
      uploadedAt: new Date().toISOString(),
      status: 'uploaded'
    }));

    res.json({
      message: `${req.files.length} files uploaded successfully`,
      files: filesInfo,
      count: req.files.length
    });

  } catch (error) {
    console.error('Multiple file upload error:', error);
    res.status(500).json({
      message: 'File upload failed',
      code: 'UPLOAD_ERROR'
    });
  }
});

/**
 * GET /api/upload/files
 * List uploaded files for the current user
 */
router.get('/files', authenticateToken, (req, res) => {
  try {
    const userDir = path.join(uploadsDir, req.user.username);
    
    if (!fs.existsSync(userDir)) {
      return res.json({
        files: [],
        count: 0
      });
    }

    const files = fs.readdirSync(userDir).map(filename => {
      const filePath = path.join(userDir, filename);
      const stats = fs.statSync(filePath);
      
      return {
        filename,
        size: stats.size,
        uploadedAt: stats.mtime.toISOString(),
        path: filePath
      };
    });

    res.json({
      files,
      count: files.length
    });

  } catch (error) {
    console.error('File list error:', error);
    res.status(500).json({
      message: 'Failed to retrieve file list',
      code: 'LIST_ERROR'
    });
  }
});

/**
 * DELETE /api/upload/files/:filename
 * Delete a specific file
 */
router.delete('/files/:filename', authenticateToken, (req, res) => {
  try {
    const filename = req.params.filename;
    const filePath = path.join(uploadsDir, req.user.username, filename);

    if (!fs.existsSync(filePath)) {
      return res.status(404).json({
        message: 'File not found',
        code: 'FILE_NOT_FOUND'
      });
    }

    fs.unlinkSync(filePath);

    res.json({
      message: 'File deleted successfully',
      filename
    });

  } catch (error) {
    console.error('File delete error:', error);
    res.status(500).json({
      message: 'Failed to delete file',
      code: 'DELETE_ERROR'
    });
  }
});

/**
 * Error handling for multer
 */
router.use((error, req, res, next) => {
  if (error instanceof multer.MulterError) {
    if (error.code === 'LIMIT_FILE_SIZE') {
      return res.status(400).json({
        message: 'File too large. Maximum size is 50MB.',
        code: 'FILE_TOO_LARGE'
      });
    }
    if (error.code === 'LIMIT_FILE_COUNT') {
      return res.status(400).json({
        message: 'Too many files. Maximum is 10 files per upload.',
        code: 'TOO_MANY_FILES'
      });
    }
  }

  if (error.message.includes('Invalid file type')) {
    return res.status(400).json({
      message: error.message,
      code: 'INVALID_FILE_TYPE'
    });
  }

  next(error);
});

module.exports = router;