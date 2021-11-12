package flatcar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sapcc/kubernikus/pkg/util/version"
)

func TestReleaseGrownup(t *testing.T) {
	now = func() time.Time { return time.Date(2020, 2, 9, 10, 11, 0, 0, time.UTC) }
	count := 0

	subject := &Release{}
	subject.Client = NewTestClient(t, "https://stable.release.flatcar-linux.net/amd64-usr/2303.4.0/version.txt", ReleasesStable, &count)
	t.Run("fetches versions", func(t *testing.T) {
		_, err := subject.GrownUp(version.MustParseSemantic("2303.4.0"), 7*24*time.Hour)
		assert.NoError(t, err)
	})

	subject = &Release{}
	subject.Client = NewTestClient(t, "https://stable.release.flatcar-linux.net/amd64-usr/2079.99.0/version.txt", ReleasesStable, &count)
	t.Run("unknown version", func(t *testing.T) {
		result, err := subject.GrownUp(version.MustParseSemantic("2079.99.0"), 7*24*time.Hour)
		assert.Error(t, err)
		assert.False(t, result)
	})

	subject = &Release{}
	subject.Client = NewTestClient(t, "https://stable.release.flatcar-linux.net/amd64-usr/2303.4.0/version.txt", ReleasesStable, &count)
	t.Run("holdoff time not up yet", func(t *testing.T) {
		result, err := subject.GrownUp(version.MustParseSemantic("2303.4.0"), 7*24*time.Hour)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	subject = &Release{}
	subject.Client = NewTestClient(t, "https://stable.release.flatcar-linux.net/amd64-usr/2303.4.0/version.txt", ReleasesStable, &count)
	t.Run("holdoff time up", func(t *testing.T) {
		now = func() time.Time { return time.Date(2020, 2, 15, 10, 11, 0, 0, time.UTC) }
		result, err := subject.GrownUp(version.MustParseSemantic("2303.4.0"), 7*24*time.Hour)
		assert.NoError(t, err)
		assert.True(t, result)
	})
}

const (
	ReleasesStable = `
		FLATCAR_BUILD=2303
		FLATCAR_BRANCH=4
		FLATCAR_PATCH=0
		FLATCAR_VERSION=2303.4.0
		FLATCAR_VERSION_ID=2303.4.0
		FLATCAR_BUILD_ID="2020-02-08-0830"
		FLATCAR_SDK_VERSION=2303.3.0`
)
