package store

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"time"
)

type DocumentTaskAttempt struct {
	Begin   time.Time
	End     time.Time
	Success bool
	Message string
}

type DocumentTask struct {
	// Document ID
	ID int64

	// Time at which the document was added to or edited in Paperless.
	Added    time.Time
	Modified time.Time

	// Document file checksums.
	OriginalChecksum string
	ArchiveChecksum  string

	RecordCreated time.Time
	RecordUpdated time.Time `boltholdIndex:""`

	RetryCount int
	RetryAfter time.Time

	Attempts []DocumentTaskAttempt
}

// Key produces the primary key for the database record.
func (t *DocumentTask) Key() ([]byte, error) {
	var buf bytes.Buffer

	if err := binary.Write(&buf, binary.LittleEndian, []int64{
		t.ID,
		t.Added.UnixMicro(),
		t.Modified.UnixMicro(),
	}); err != nil {
		return nil, err
	}

	buf.WriteByte('\x00')

	for _, i := range []string{
		t.OriginalChecksum,
		t.ArchiveChecksum,
	} {
		buf.WriteString(strconv.QuoteToASCII(i))
	}

	return buf.Bytes(), nil
}
