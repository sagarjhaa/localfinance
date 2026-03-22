package processors

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/sagarjhaa/localfinance/services/logos/models"
)

// DocumentProcessor interface for different file type processors
type DocumentProcessor interface {
	CanProcess(fileType string) bool
	ProcessDocument(file io.Reader, filename string) ([]models.Transaction, error)
	ValidateFormat(file io.Reader) error
	GetSupportedTypes() []string
}

// Manager manages all document processors
type Manager struct {
	processors map[string]DocumentProcessor
}

// NewManager creates a new processor manager
func NewManager() *Manager {
	manager := &Manager{
		processors: make(map[string]DocumentProcessor),
	}

	// Register available processors
	manager.RegisterProcessor(&CSVProcessor{})
	manager.RegisterProcessor(&PDFProcessor{})
	manager.RegisterProcessor(&ExcelProcessor{})

	return manager
}

// RegisterProcessor registers a new document processor
func (m *Manager) RegisterProcessor(processor DocumentProcessor) {
	for _, fileType := range processor.GetSupportedTypes() {
		m.processors[fileType] = processor
	}
}

// ProcessDocument processes a document and returns transactions
func (m *Manager) ProcessDocument(file io.Reader, filename string) (models.ProcessingResult, error) {
	// Determine file type
	fileType := m.getFileType(filename)
	
	// Get appropriate processor
	processor, exists := m.processors[fileType]
	if !exists {
		return models.ProcessingResult{}, fmt.Errorf("unsupported file type: %s", fileType)
	}

	// Validate file format
	if err := processor.ValidateFormat(file); err != nil {
		return models.ProcessingResult{}, fmt.Errorf("invalid file format: %w", err)
	}

	// Process document to extract transactions
	transactions, err := processor.ProcessDocument(file, filename)
	if err != nil {
		return models.ProcessingResult{}, fmt.Errorf("failed to process document: %w", err)
	}

	// Create processing result
	result := models.ProcessingResult{
		Filename:          filename,
		FileType:          fileType,
		TransactionsFound: len(transactions),
		Transactions:      transactions,
		Status:           "success",
	}

	return result, nil
}

// GetSupportedTypes returns all supported file types
func (m *Manager) GetSupportedTypes() []string {
	types := make([]string, 0, len(m.processors))
	for fileType := range m.processors {
		types = append(types, fileType)
	}
	return types
}

// getFileType determines the file type from filename
func (m *Manager) getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv":
		return "csv"
	case ".pdf":
		return "pdf"
	case ".xlsx", ".xls":
		return "excel"
	default:
		return "unknown"
	}
}

// CanProcessFile checks if a file can be processed
func (m *Manager) CanProcessFile(filename string) bool {
	fileType := m.getFileType(filename)
	_, exists := m.processors[fileType]
	return exists
}