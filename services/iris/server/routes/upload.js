const express = require('express');
const multer = require('multer');
const axios = require('axios');
const FormData = require('form-data');
const fs = require('fs');
const path = require('path');
const { authenticateToken } = require('../middleware/auth');

const router = express.Router();
const THESAURUS_URL = process.env.THESAURUS_URL || 'http://localhost:3001';

// Temp storage — file is immediately forwarded to Thesaurus then deleted
const upload = multer({ dest: path.join(__dirname, '../uploads') });

/**
 * POST /api/upload/single
 * Proxy file upload to Thesaurus. Returns document_id for polling.
 * Iris does NO parsing — that's Logos' job.
 */
router.post('/single', authenticateToken, upload.single('file'), async (req, res) => {
  try {
    if (!req.file) {
      return res.status(400).json({ message: 'No file uploaded' });
    }

    // Forward the file to Thesaurus as multipart
    const form = new FormData();
    form.append('file', fs.createReadStream(req.file.path), req.file.originalname);

    const response = await axios.post(`${THESAURUS_URL}/api/v1/upload`, form, {
      headers: {
        ...form.getHeaders(),
        'Authorization': req.headers.authorization,
        'X-Correlation-ID': req.correlationId || '',
      },
      timeout: 30000,
      maxContentLength: 50 * 1024 * 1024,
    });

    // Clean up temp file
    fs.unlink(req.file.path, () => {});

    res.status(response.status).json(response.data);
  } catch (error) {
    // Clean up temp file on error
    if (req.file) fs.unlink(req.file.path, () => {});

    if (error.response) {
      return res.status(error.response.status).json(error.response.data);
    }
    console.error('Upload proxy error:', error.message);
    res.status(503).json({ message: 'Upload service unavailable' });
  }
});

/**
 * GET /api/upload/status/:documentId
 * Proxy document status check to Thesaurus
 */
router.get('/status/:documentId', authenticateToken, async (req, res) => {
  try {
    const response = await axios.get(
      `${THESAURUS_URL}/api/v1/documents/${req.params.documentId}`,
      { timeout: 5000 }
    );
    res.json(response.data);
  } catch (error) {
    if (error.response) {
      return res.status(error.response.status).json(error.response.data);
    }
    res.status(503).json({ message: 'Service unavailable' });
  }
});

/**
 * GET /api/upload/transactions
 * Proxy transaction query by document_id to Thesaurus
 */
router.get('/transactions', authenticateToken, async (req, res) => {
  try {
    const response = await axios.get(
      `${THESAURUS_URL}/api/v1/transactions/by-document`,
      {
        params: { document_id: req.query.document_id },
        timeout: 10000,
      }
    );
    res.json(response.data);
  } catch (error) {
    if (error.response) {
      return res.status(error.response.status).json(error.response.data);
    }
    res.status(503).json({ message: 'Service unavailable' });
  }
});

module.exports = router;
