package paperminer

import (
	"context"

	"github.com/hansmi/dossier"
	"github.com/hansmi/staticplug"
	"go.uber.org/zap"
)

type DocumentFacterOptions struct {
	Logger   *zap.Logger
	Document *dossier.Document
}

type DocumentFacter interface {
	staticplug.Plugin

	// DocumentFacts is invoked after a document has been parsed into
	// structured text. The return value can be nil to report that no suitable
	// facts were found.
	DocumentFacts(context.Context, DocumentFacterOptions) (*Facts, error)
}
