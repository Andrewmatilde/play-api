package meta

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ResourceStatus represents the current state of a resource
type ResourceStatus struct {
	// Phase represents the current phase of the resource
	Phase string `json:"phase,omitempty"`

	// Message provides a human-readable message indicating details about why the resource is in this phase
	Message string `json:"message,omitempty"`

	// Reason is a brief CamelCase string that describes any failure and is meant for machine parsing and tidy display in the CLI
	Reason string `json:"reason,omitempty"`

	// LastTransitionTime is the last time the condition transitioned from one status to another
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

// TypeMeta describes an individual object in an API response or request
// with strings representing the type of the object and its API schema version.
type TypeMeta struct {
	// Kind is a string value representing the REST resource this object represents.
	// Servers may infer this from the endpoint the client submits requests to.
	Kind string `json:"kind,omitempty"`

	// APIVersion defines the versioned schema of this representation of an object.
	// Servers should convert recognized schemas to the latest internal value, and
	// may reject unrecognized values.
	APIVersion string `json:"apiVersion,omitempty"`
}

// ObjectMeta is metadata that all persisted resources must have, which includes all objects
// users must create.
type ObjectMeta struct {
	// ID is the unique in time and space value for this object.
	ID uint `gorm:"primaryKey" json:"id"`

	// UID is the unique in time and space value for this object.
	UID string `gorm:"type:char(36)" json:"uid,omitempty"`

	// ResourceVersion is a string that identifies the internal version of this object
	// that can be used by clients to determine when objects have changed.
	ResourceVersion int `json:"resourceVersion,omitempty" gorm:"column:resource_version"`

	// CreationTimestamp is a timestamp representing the server time when this object was created.
	CreatedAt time.Time `json:"createdAt"`

	// UpdateTimestamp is a timestamp representing the server time when this object was last updated.
	UpdatedAt time.Time `json:"updatedAt"`

	// Labels are key/value pairs that are attached to objects and may be used to organize
	// and to select subsets of objects.
	Labels map[string]string `gorm:"serializer:json" json:"labels,omitempty"`

	// Annotations are unstructured key value data stored with a resource that may be set by
	// external tools to store and retrieve arbitrary metadata.
	Annotations map[string]string `gorm:"serializer:json" json:"annotations,omitempty"`

	// Status represents the current state of the resource
	Status ResourceStatus `json:"status,omitempty" gorm:"embedded"`
}

// BaseResource is the base type that all resources should embed
type BaseResource struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,inline"`
}

// ResourceValidator defines the interface for resource validation
type ResourceValidator interface {
	Validate() error
}

// ResourceEventHandler defines the interface for resource event handling
type ResourceEventHandler interface {
	OnCreate() error
	OnUpdate() error
	OnDelete() error
}

// GetID returns the ID of the resource
func (b *BaseResource) GetID() uint {
	return b.ID
}

// GetUID returns the UID of the resource
func (b *BaseResource) GetUID() string {
	return b.UID
}

// GetResourceVersion returns the resource version
func (b *BaseResource) GetResourceVersion() int {
	return b.ResourceVersion
}

// GetKind returns the kind of the resource
func (b *BaseResource) GetKind() string {
	return b.Kind
}

// GetAPIVersion returns the API version
func (b *BaseResource) GetAPIVersion() string {
	return b.APIVersion
}

// SetStatus updates the resource status
func (b *BaseResource) SetStatus(phase, message, reason string) {
	b.Status.Phase = phase
	b.Status.Message = message
	b.Status.Reason = reason
	b.Status.LastTransitionTime = time.Now()
}

// Validate performs basic validation of the resource
func (b *BaseResource) Validate() error {
	if b.Kind == "" {
		return errors.New("kind is required")
	}
	if b.APIVersion == "" {
		return errors.New("apiVersion is required")
	}
	return nil
}

// BeforeCreate is a GORM hook that runs before creating a resource
func (b *BaseResource) BeforeCreate(tx *gorm.DB) error {
	if b.UID == "" {
		b.UID = uuid.New().String()
	}
	if b.ResourceVersion == 0 {
		b.ResourceVersion = 1
	}

	// Set initial status
	if b.Status.Phase == "" {
		b.SetStatus("Pending", "Resource is being created", "")
	}

	// Validate the resource
	if err := b.Validate(); err != nil {
		return err
	}

	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a resource
func (b *BaseResource) BeforeUpdate(tx *gorm.DB) error {
	b.ResourceVersion++

	// Validate the resource
	if err := b.Validate(); err != nil {
		return err
	}

	return nil
}

// BeforeDelete is a GORM hook that runs before deleting a resource
func (b *BaseResource) BeforeDelete(tx *gorm.DB) error {
	return nil
}

// SetMetadata sets a metadata key-value pair
func (b *BaseResource) SetMetadata(key, value string) {
	if b.Annotations == nil {
		b.Annotations = make(map[string]string)
	}
	b.Annotations[key] = value
}

// GetMetadata gets a metadata value by key
func (b *BaseResource) GetMetadata(key string) (string, bool) {
	if b.Annotations == nil {
		return "", false
	}
	value, exists := b.Annotations[key]
	return value, exists
}

// DeleteMetadata deletes a metadata key
func (b *BaseResource) DeleteMetadata(key string) {
	if b.Annotations == nil {
		return
	}
	delete(b.Annotations, key)
}
