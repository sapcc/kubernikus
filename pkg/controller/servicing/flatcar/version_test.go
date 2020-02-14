package flatcar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	core_v1 "k8s.io/api/core/v1"
)

const (
	Stable2303_3_1 = `
		FLATCAR_BUILD=2303
		FLATCAR_BRANCH=3
		FLATCAR_PATCH=1
		FLATCAR_VERSION=2303.3.1
		FLATCAR_VERSION_ID=2303.3.1
		FLATCAR_BUILD_ID="2019-12-17-1953"
		FLATCAR_SDK_VERSION=2303.3.0`

	Stable2303_4_0 = `
		FLATCAR_BUILD=2303
		FLATCAR_BRANCH=4
		FLATCAR_PATCH=0
		FLATCAR_VERSION=2303.4.0
		FLATCAR_VERSION_ID=2303.4.0
		FLATCAR_BUILD_ID="2020-02-08-0830"
		FLATCAR_SDK_VERSION=2303.4.0`

	Beta2345_2_0 = `
		FLATCAR_BUILD=2345
		FLATCAR_BRANCH=2
		FLATCAR_PATCH=0
		FLATCAR_VERSION=2345.2.0
		FLATCAR_VERSION_ID=2345.2.0
		FLATCAR_BUILD_ID="2020-02-08-0832"
		FLATCAR_SDK_VERSION=2345.2.0`

	Alpha2387_0_0 = ` 
		FLATCAR_BUILD=2387
		FLATCAR_BRANCH=0
		FLATCAR_PATCH=0
		FLATCAR_VERSION=2387.0.0
		FLATCAR_VERSION_ID=2387.0.0
		FLATCAR_BUILD_ID="2020-01-21-0017"
		FLATCAR_SDK_VERSION=2387.0.0`

	UnexpectedResponse = `404 CoreOS is now EOL`
)

func TestVersionStable(t *testing.T) {
	now = func() time.Time { return time.Date(2019, 2, 3, 4, 0, 0, 0, time.UTC) }
	count := 0
	subject := &Version{}
	subject.Client = NewTestClient(t, "https://stable.release.flatcar-linux.net/amd64-usr/current/version.txt", Stable2303_3_1, &count)

	t.Run("fetches correct version", func(t *testing.T) {
		version, err := subject.Stable()
		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.Equal(t, "2303.3.1", version.String())
	})

	subject.Client = NewTestClient(t, "https://stable.release.flatcar-linux.net/amd64-usr/current/version.txt", Stable2303_4_0, &count)

	t.Run("caches result", func(t *testing.T) {
		version, err := subject.Stable()
		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.Equal(t, "2303.3.1", version.String())
		assert.Equal(t, 1, count)
	})

	now = func() time.Time {
		return time.Date(2019, 2, 3, 4, 0, 0, 0, time.UTC).Add(flatcarVersionFetchInterval).Add(1 * time.Minute)
	}

	t.Run("invalidates cached result", func(t *testing.T) {
		version, err := subject.Stable()
		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.Equal(t, "2303.4.0", version.String())
		assert.Equal(t, 2, count)
	})

	subject = &Version{}
	subject.Client = NewTestClient(t, "https://stable.release.flatcar-linux.net/amd64-usr/current/version.txt", UnexpectedResponse, &count)

	t.Run("garble from flatcar servers", func(t *testing.T) {
		version, err := subject.Stable()
		assert.Error(t, err)
		assert.Nil(t, version)
	})
}

func TestVersionBeta(t *testing.T) {
	now = func() time.Time { return time.Date(2019, 2, 3, 4, 0, 0, 0, time.UTC) }
	subject := &Version{}
	subject.Client = NewTestClient(t, "https://beta.release.flatcar-linux.net/amd64-usr/current/version.txt", Beta2345_2_0, nil)

	t.Run("fetches correct version", func(t *testing.T) {
		version, err := subject.Beta()
		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.Equal(t, "2345.2.0", version.String())
	})
}

func TestVersionAlpha(t *testing.T) {
	now = func() time.Time { return time.Date(2019, 2, 3, 4, 0, 0, 0, time.UTC) }
	subject := &Version{}
	subject.Client = NewTestClient(t, "https://alpha.release.flatcar-linux.net/amd64-usr/current/version.txt", Alpha2387_0_0, nil)

	t.Run("fetches correct version", func(t *testing.T) {
		version, err := subject.Alpha()
		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.Equal(t, "2387.0.0", version.String())
	})
}

func TestVersionIsNodeUptodate(t *testing.T) {
	subject := &Version{}
	subject.Client = NewTestClient(t, "https://stable.release.flatcar-linux.net/amd64-usr/current/version.txt", Stable2303_3_1, nil)

	t.Run("outdated node", func(t *testing.T) {
		node := &core_v1.Node{
			Status: core_v1.NodeStatus{
				NodeInfo: core_v1.NodeSystemInfo{
					OSImage: "Flatcar Container Linux by Kinvolk 2000.0.0 (Rhyolite)",
				},
			},
		}

		result, err := subject.IsNodeUptodate(node)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("recent node", func(t *testing.T) {
		node := &core_v1.Node{
			Status: core_v1.NodeStatus{
				NodeInfo: core_v1.NodeSystemInfo{
					OSImage: "Flatcar Container Linux by Kinvolk 2303.3.1 (Rhyolite)",
				},
			},
		}

		result, err := subject.IsNodeUptodate(node)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("unknown OS", func(t *testing.T) {
		node := &core_v1.Node{
			Status: core_v1.NodeStatus{
				NodeInfo: core_v1.NodeSystemInfo{
					OSImage: "SLES11SP3",
				},
			},
		}

		result, err := subject.IsNodeUptodate(node)
		assert.Error(t, err)
		assert.False(t, result)
	})

}
