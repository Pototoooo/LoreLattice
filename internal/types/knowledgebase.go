package types

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// KnowledgeBase represents a knowledge base
type KnowledgeBase struct {
	// Unique identifier of the knowledge base
	ID string `yaml:"id" json:"id" gorm:"type:varchar(36);primaryKey"`
	// Name of the knowledge base
	Name string `yaml:"name" json:"name"`
	// Description of the knowledge base
	Description string `yaml:"description" json:"description"`
	// Tenant ID
	TenantID uint `yaml:"tenant_id" json:"tenant_id"`
	// Chunking configuration
	ChunkingConfig ChunkingConfig `yaml:"chunking_config" json:"chunking_config" gorm:"type:json"`
	// Image processing configuration
	ImageProcessingConfig ImageProcessingConfig `yaml:"image_processing_config" json:"image_processing_config" gorm:"type:json"`
	// ID of the embedding model
	EmbeddingModelID string `yaml:"embedding_model_id" json:"embedding_model_id"`
	// Summary model ID
	SummaryModelID string `yaml:"summary_model_id" json:"summary_model_id"`
	// Creation time of the knowledge base
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
	// Last updated time of the knowledge base
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
	// Deletion time of the knowledge base
	DeletedAt gorm.DeletedAt `yaml:"deleted_at" json:"deleted_at" gorm:"index"`
}

// KnowledgeBaseConfig represents the knowledge base configuration
type KnowledgeBaseConfig struct {
	// Chunking configuration
	ChunkingConfig ChunkingConfig `yaml:"chunking_config" json:"chunking_config"`
	// Image processing configuration
	ImageProcessingConfig ImageProcessingConfig `yaml:"image_processing_config" json:"image_processing_config"`
}

// ChunkingConfig represents the document splitting configuration
type ChunkingConfig struct {
	// Chunk size
	ChunkSize int `yaml:"chunk_size" json:"chunk_size"`
	// Chunk overlap
	ChunkOverlap int `yaml:"chunk_overlap" json:"chunk_overlap"`
	// Separators
	Separators []string `yaml:"separators" json:"separators"`
	// Enable multimodal
	EnableMultimodal bool `yaml:"enable_multimodal" json:"enable_multimodal"`
}

// ImageProcessingConfig represents the image processing configuration
type ImageProcessingConfig struct {
	// Model ID
	ModelID string `yaml:"model_id" json:"model_id"`
}

// Value implements the driver.Valuer interface, used to convert ChunkingConfig to database value
func (c ChunkingConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface, used to convert database value to ChunkingConfig
func (c *ChunkingConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, c)
}

// Value implements the driver.Valuer interface, used to convert ImageProcessingConfig to database value
func (c ImageProcessingConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface, used to convert database value to ImageProcessingConfig
func (c *ImageProcessingConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, c)
}
