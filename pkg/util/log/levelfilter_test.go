package log

import (
	"bytes"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
)

func TestLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	logger := kitlog.NewLogfmtLogger(&buf)
	logger = NewLevelFilter(2, logger)
	assert.NoError(t, logger.Log("key", "val"))
	assert.Equal(t, "key=val\n", buf.String())

	buf.Reset()
	assert.NoError(t, logger.Log("v", 2))
	assert.Equal(t, "v=2\n", buf.String())

	buf.Reset()
	assert.NoError(t, logger.Log("v", 3))
	assert.Equal(t, "", buf.String())

	buf.Reset()
	assert.NoError(t, logger.Log("key", "val", "v", 3))
	assert.Equal(t, "", buf.String())

	buf.Reset()
	assert.NoError(t, logger.Log("key", "val", "v", 3, "key2", "val2"))
	assert.Equal(t, "", buf.String())

	assert.Error(t, logger.Log("v", "noint"))

}
