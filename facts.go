package paperminer

import (
	"encoding/json"
	"time"
)

type Facts struct {
	Reporter *string `json:"reporter"`

	Title         *string    `json:"title,omitempty"`
	Created       *time.Time `json:"created,omitempty"`
	DocumentType  *string    `json:"document_type,omitempty"`
	Correspondent *string    `json:"correspondent,omitempty"`
	StoragePath   *string    `json:"storage_path,omitempty"`

	SetTags   []string `json:"set_tags,omitempty"`
	UnsetTags []string `json:"unset_tags,omitempty"`

	// TODO: Support custom fields
}

func (f *Facts) String() string {
	buf, err := json.Marshal(f)
	if err != nil {
		return err.Error()
	}

	return string(buf)
}

// IsEmpty returns whether at least one fact property has been set.
func (f *Facts) IsEmpty() bool {
	return (f.Title == nil &&
		f.DocumentType == nil &&
		f.Correspondent == nil &&
		f.StoragePath == nil &&
		f.Created == nil &&
		len(f.SetTags) == 0 &&
		len(f.UnsetTags) == 0)
}
