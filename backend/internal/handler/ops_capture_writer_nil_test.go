package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpsCaptureWriterNilInnerWriterDoesNotPanic(t *testing.T) {
	w := &opsCaptureWriter{}

	assert.Equal(t, 0, w.Status())
	assert.Equal(t, -1, w.Size())
	assert.False(t, w.Written())
	assert.NotNil(t, w.Header())
	assert.NotPanics(t, func() {
		w.WriteHeader(200)
		w.WriteHeaderNow()
		w.Flush()
	})

	n, err := w.Write([]byte("test"))
	assert.Zero(t, n)
	assert.NoError(t, err)
	n, err = w.WriteString("test")
	assert.Zero(t, n)
	assert.NoError(t, err)

	conn, rw, err := w.Hijack()
	assert.Nil(t, conn)
	assert.Nil(t, rw)
	assert.Error(t, err)
	assert.NotNil(t, w.CloseNotify())
	assert.Nil(t, w.Pusher())
}
