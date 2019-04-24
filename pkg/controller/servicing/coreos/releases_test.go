package coreos

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sapcc/kubernikus/pkg/util/version"
)

func TestReleaseGrownup(t *testing.T) {
	now = func() time.Time { return time.Date(2019, 4, 23, 20, 21, 0, 0, time.UTC) }
	count := 0
	subject := &Release{}
	subject.Client = NewTestClient(t, "https://coreos.com/releases/releases-stable.json", ReleasesStable, &count)

	t.Run("fetches versions", func(t *testing.T) {
		_, err := subject.GrownUp(version.MustParseSemantic("2079.3.0"))
		assert.NoError(t, err)
	})

	t.Run("unknown version", func(t *testing.T) {
		result, err := subject.GrownUp(version.MustParseSemantic("2079.99.0"))
		assert.Error(t, err)
		assert.False(t, result)
	})

	t.Run("holdoff time not up yet", func(t *testing.T) {
		result, err := subject.GrownUp(version.MustParseSemantic("2079.3.0"))
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("holdoff time up", func(t *testing.T) {
		now = func() time.Time { return time.Date(2019, 5, 23, 20, 21, 0, 0, time.UTC) }
		result, err := subject.GrownUp(version.MustParseSemantic("2079.3.0"))
		assert.NoError(t, err)
		assert.True(t, result)
	})
}

const (
	ReleasesStable = `
	{
		"2079.3.0": {
		  "release_date": "2019-04-23 20:20:00 +0000",
		  "major_software": {
			"kernel": ["4.19.34"],
			"docker": ["18.06.3"],
			"etcd": ["3.3.12"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"2023.5.0": {
		  "release_date": "2019-03-11 23:13:36 +0000",
		  "major_software": {
			"kernel": ["4.19.25"],
			"docker": ["18.06.1"],
			"etcd": ["3.3.12"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"2023.4.0": {
		  "release_date": "2019-02-26 16:45:45 +0000",
		  "major_software": {
			"kernel": ["4.19.23"],
			"docker": ["18.06.1"],
			"etcd": ["3.3.12"],
			"fleet": [""]
		  },
		  "release_notes": "No updates" 
		},
		
		"1967.6.0": {
		  "release_date": "2019-02-12 23:15:19 +0000",
		  "major_software": {
			"kernel": ["4.14.96"],
			"docker": ["18.06.1"],
			"etcd": ["3.3.10"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1967.5.0": {
		  "release_date": "2019-02-11 20:11:19 +0000",
		  "major_software": {
			"kernel": ["4.14.96"],
			"docker": ["18.06.1"],
			"etcd": ["3.3.10"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1967.4.0": {
		  "release_date": "2019-01-29 15:08:12 +0000",
		  "major_software": {
			"kernel": ["4.14.96"],
			"docker": ["18.06.1"],
			"etcd": ["3.3.10"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1967.3.0": {
		  "release_date": "2019-01-08 19:31:47 +0000",
		  "major_software": {
			"kernel": ["4.14.88"],
			"docker": ["18.06.1"],
			"etcd": ["3.3.10"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1911.5.0": {
		  "release_date": "2018-12-19 16:11:02 +0000",
		  "major_software": {
			"kernel": ["4.14.84"],
			"docker": ["18.06.1"],
			"etcd": ["3.3.9"],
			"fleet": [""]
		  },
		  "release_notes":""
		},
		
		"1911.4.0": {
		  "release_date": "2018-11-26 21:49:53 +0000",
		  "major_software": {
			"kernel": ["4.14.81"],
			"docker": ["18.06.1"],
			"etcd": ["3.3.9"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1911.3.0": {
		  "release_date": "2018-11-06 17:54:59 +0000",
		  "major_software": {
			"kernel": ["4.14.78"],
			"docker": ["18.06.1"],
			"etcd": ["3.3.9"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1855.5.0": {
		  "release_date": "2018-10-24 13:25:03 +0000",
		  "major_software": {
			"kernel": ["4.14.74"],
			"docker": ["18.06.1"],
			"etcd": ["3.3.9"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1855.4.0": {
		  "release_date": "2018-09-11 18:48:31 +0000",
		  "major_software": {
			"kernel": ["4.14.67"],
			"docker": ["18.06.1"],
			"etcd": ["3.3.9"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1800.7.0": {
		  "release_date": "2018-08-16 16:40:56 +0000",
		  "major_software": {
			"kernel": ["4.14.63"],
			"docker": ["18.03.1"],
			"etcd": ["3.3.6"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1800.6.0": {
		  "release_date": "2018-08-06 19:33:31 +0000",
		  "major_software": {
			"kernel": ["4.14.59"],
			"docker": ["18.03.1"],
			"etcd": ["3.3.6"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1800.5.0": {
		  "release_date": "2018-07-29 23:40:35 +0000",
		  "major_software": {
			"kernel": ["4.14.59"],
			"docker": ["18.03.1"],
			"etcd": ["3.3.6"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1800.4.0": {
		  "release_date": "2018-07-25 20:37:11 +0000",
		  "major_software": {
			"kernel": ["4.14.55"],
			"docker": ["18.03.1"],
			"etcd": ["3.3.6"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1745.7.0": {
		  "release_date": "2018-06-14 17:42:52 +0000",
		  "major_software": {
			"kernel": ["4.14.48"],
			"docker": ["18.03.1"],
			"etcd": ["3.3.3"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1745.6.0": {
		  "release_date": "2018-06-11 20:02:42 +0000",
		  "major_software": {
			"kernel": ["4.14.48"],
			"docker": ["18.03.1"],
			"etcd": ["3.3.3"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1745.5.0": {
		  "release_date": "2018-05-31 15:22:08 +0000",
		  "major_software": {
			"kernel": ["4.14.44"],
			"docker": ["18.03.1"],
			"etcd": ["3.3.3"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1745.4.0": {
		  "release_date": "2018-05-24 23:35:28 +0000",
		  "major_software": {
			"kernel": ["4.14.42"],
			"docker": ["18.03.1"],
			"etcd": ["3.3.3"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1745.3.1": {
		  "release_date": "2018-05-23 20:52:29 +0000",
		  "major_software": {
			"kernel": ["4.14.42"],
			"docker": ["18.03.1"],
			"etcd": ["3.3.3"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1688.5.3": {
		  "release_date": "2018-04-03 17:11:51 +0000",
		  "major_software": {
			"kernel": ["4.14.32"],
			"docker": ["17.12.1"],
			"etcd": ["3.2.15"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1688.4.0": {
		  "release_date": "2018-03-27 19:48:42 +0000",
		  "major_software": {
			"kernel": ["4.14.30"],
			"docker": ["17.12.1"],
			"etcd": ["3.2.15"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1632.3.0": {
		  "release_date": "2018-02-15 00:57:27 +0000",
		  "major_software": {
			"kernel": ["4.14.19"],
			"docker": ["17.09.1"],
			"etcd": ["3.2.11"],
			"fleet": [""]
		  },
		  "release_notes": "No updates" 
		},
		
		"1632.2.1": {
		  "release_date": "2018-02-01 22:17:31 +0000",
		  "major_software": {
			"kernel": ["4.14.16"],
			"docker": ["17.09.1"],
			"etcd": ["3.2.11"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1576.5.0": {
		  "release_date": "2018-01-05 14:13:41 +0000",
		  "major_software": {
			"kernel": ["4.14.11"],
			"docker": ["17.09.0"],
			"etcd": ["3.2.9"],
			"fleet": [""]
		  },
		  "release_notes": "No updates" 
		},
		
		"1576.4.0": {
		  "release_date": "2017-12-06 21:14:28 +0000",
		  "major_software": {
			"kernel": ["4.13.16"],
			"docker": ["17.09.0"],
			"etcd": ["3.2.9"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1520.9.0": {
		  "release_date": "2017-11-30 21:34:26 +0000",
		  "major_software": {
			"kernel": ["4.13.16"],
			"docker": ["1.12.6"],
			"etcd": ["3.1.10"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1520.8.0": {
		  "release_date": "2017-10-26 16:25:30 +0000",
		  "major_software": {
			"kernel": ["4.13.9"],
			"docker": ["1.12.6"],
			"etcd": ["3.1.10"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1520.7.0": {
		  "release_date": "2017-10-24 21:23:40 +0000",
		  "major_software": {
			"kernel": ["4.13.9"],
			"docker": ["1.12.6"],
			"etcd": ["3.1.10"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1520.6.0": {
		  "release_date": "2017-10-12 19:45:37 +0000",
		  "major_software": {
			"kernel": ["4.13.5"],
			"docker": ["1.12.6"],
			"etcd": ["3.1.10"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1520.5.0": {
		  "release_date": "2017-10-11 15:34:30 +0000",
		  "major_software": {
			"kernel": ["4.13.5"],
			"docker": ["1.12.6"],
			"etcd": ["3.1.10"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1465.8.0": {
		  "release_date": "2017-09-21 01:06:11 +0000",
		  "major_software": {
			"kernel": ["4.12.14"],
			"docker": ["1.12.6"],
			"etcd": ["3.1.8"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1465.7.0": {
		  "release_date": "2017-09-06 18:22:34 +0000",
		  "major_software": {
			"kernel": ["4.12.10"],
			"docker": ["1.12.6"],
			"etcd": ["3.1.8"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1465.6.0": {
		  "release_date": "2017-08-16 21:23:30 +0000",
		  "major_software": {
			"kernel": ["4.12.7"],
			"docker": ["1.12.6"],
			"etcd": ["2.3.7"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1409.9.0": {
		  "release_date": "2017-08-14 18:54:14 +0000",
		  "major_software": {
			"kernel": ["4.11.12"],
			"docker": ["1.12.6"],
			"etcd": ["2.3.7"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1409.8.0": {
		  "release_date": "2017-08-10 21:32:03 +0000",
		  "major_software": {
			"kernel": ["4.11.12"],
			"docker": ["1.12.6"],
			"etcd": ["2.3.7"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1409.7.0": {
		  "release_date": "2017-07-19 01:52:37 +0000",
		  "major_software": {
			"kernel": ["4.11.11"],
			"docker": ["1.12.6"],
			"etcd": ["2.3.7"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1409.6.0": {
		  "release_date": "2017-07-06 01:57:12 +0000",
		  "major_software": {
			"kernel": ["4.11.9"],
			"docker": ["1.12.6"],
			"etcd": ["2.3.7"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1409.5.0": {
		  "release_date": "2017-06-23 00:21:55 +0000",
		  "major_software": {
			"kernel": ["4.11.6"],
			"docker": ["1.12.6"],
			"etcd": ["2.3.7"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1409.2.0": {
		  "release_date": "2017-06-20 01:31:54 +0000",
		  "major_software": {
			"kernel": ["4.11.6"],
			"docker": ["1.12.6"],
			"etcd": ["2.3.7"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1353.8.0": {
		  "release_date": "2017-05-31 00:11:55 +0000",
		  "major_software": {
			"kernel": ["4.9.24"],
			"docker": ["1.12.6"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": [""]
		  },
		  "release_notes": "No updates"
		},
		
		"1353.7.0": {
		  "release_date": "2017-04-26 23:39:29 +0000",
		  "major_software": {
			"kernel": ["4.9.24"],
			"docker": ["1.12.6"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1353.6.0": {
		  "release_date": "2017-04-25 15:07:03 +0000",
		  "major_software": {
			"kernel": ["4.9.24"],
			"docker": ["1.12.6"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1298.7.0": {
		  "release_date": "2017-03-31 22:19:13 +0000",
		  "major_software": {
			"kernel": ["4.9.16"],
			"docker": ["1.12.6"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1298.6.0": {
		  "release_date": "2017-03-15 17:24:28 +0000",
		  "major_software": {
			"kernel": ["4.9.9"],
			"docker": ["1.12.6"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1298.5.0": {
		  "release_date": "2017-02-28 19:19:47 +0000",
		  "major_software": {
			"kernel": ["4.9.9"],
			"docker": ["1.12.6"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1235.12.0": {
		  "release_date": "2017-02-23 04:40:09 +0000",
		  "major_software": {
			"kernel": ["4.7.3"],
			"docker": ["1.12.6"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1235.9.0": {
		  "release_date": "2017-02-02 05:26:47 +0000",
		  "major_software": {
			"kernel": ["4.7.3"],
			"docker": ["1.12.6"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1235.8.0": {
		  "release_date": "2017-01-31 21:37:38 +0000",
		  "major_software": {
			"kernel": ["4.7.3"],
			"docker": ["1.12.6"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1235.6.0": {
		  "release_date": "2017-01-11 01:57:20 +0000",
		  "major_software": {
			"kernel": ["4.7.3"],
			"docker": ["1.12.3"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1235.5.0": {
		  "release_date": "2017-01-08 22:40:31 +0000",
		  "major_software": {
			"kernel": ["4.7.3"],
			"docker": ["1.12.3"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1235.4.0": {
		  "release_date": "2017-01-04 18:36:08 +0000",
		  "major_software": {
			"kernel": ["4.7.3"],
			"docker": ["1.12.3"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1185.5.0": {
		  "release_date": "2016-12-07 16:50:38 +0000",
		  "major_software": {
			"kernel": ["4.7.3"],
			"docker": ["1.11.2"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1185.3.0": {
		  "release_date": "2016-11-01 17:55:29 +0000",
		  "major_software": {
			"kernel": ["4.7.3"],
			"docker": ["1.11.2"],
			"etcd": ["0.4.9","2.3.7"],
			"fleet": ["0.11.8"]
		  },
		  "release_notes": "No updates"
		},
		
		"1122.3.0": {
		  "release_date": "2016-10-20 22:15:18 +0000",
		  "major_software": {
			"kernel": ["4.7.0"],
			"docker": ["1.10.3"],
			"etcd": ["0.4.9","2.3.2"],
			"fleet": ["0.11.7"]
		  },
		  "release_notes": "No updates"
		},
		
		"1122.2.0": {
		  "release_date": "2016-09-06 16:17:57 +0000",
		  "major_software": {
			"kernel": ["4.7.0"],
			"docker": ["1.10.3"],
			"etcd": ["0.4.9","2.3.2"],
			"fleet": ["0.11.7"]
		  },
		  "release_notes": "No updates"
		},
		
		"1068.10.0": {
		  "release_date": "2016-08-23 15:30:28 +0000",
		  "major_software": {
			"kernel": ["4.6.3"],
			"docker": ["1.10.3"],
			"etcd": ["0.4.9","2.3.2"],
			"fleet": ["0.11.7"]
		  },
		  "release_notes": "No updates"
		},
		
		"1068.9.0": {
		  "release_date": "2016-08-09 22:51:46 +0000",
		  "major_software": {
			"kernel": ["4.6.3"],
			"docker": ["1.10.3"],
			"etcd": ["0.4.9","2.3.2"],
			"fleet": ["0.11.7"]
		  },
		  "release_notes": "No updates"
		},
		
		"1068.8.0": {
		  "release_date": "2016-07-18 19:32:58 +0000",
		  "major_software": {
			"kernel": ["4.6.3"],
			"docker": ["1.10.3"],
			"etcd": ["0.4.9","2.3.2"],
			"fleet": ["0.11.7"]
		  },
		  "release_notes": "No updates"
		},
		
		"1068.6.0": {
		  "release_date": "2016-07-12 19:54:15 +0000",
		  "major_software": {
			"kernel": ["4.6.3"],
			"docker": ["1.10.3"],
			"etcd": ["0.4.9","2.3.2"],
			"fleet": ["0.11.7"]
		  },
		  "release_notes": "No updates"
		},
		
		"1010.6.0": {
		  "release_date": "2016-06-28 23:19:30 +0000",
		  "major_software": {
			"kernel": ["4.5.7"],
			"docker": ["1.10.3"],
			"etcd": ["0.4.9","2.3.1"],
			"fleet": ["0.11.7"]
		  },
		  "release_notes": "No updates"
		},
		
		"1010.5.0": {
		  "release_date": "2016-05-27 00:05:30 +0000",
		  "major_software": {
			"kernel": ["4.5.0"],
			"docker": ["1.10.3"],
			"etcd": ["0.4.9","2.3.1"],
			"fleet": ["0.11.7"]
		  },
		  "release_notes": "No updates"
		},
		
		"899.17.0": {
		  "release_date": "2016-05-03 23:30:01 +0000",
		  "major_software": {
			"kernel": ["4.3.6"],
			"docker": ["1.9.1"],
			"etcd": ["0.4.9","2.2.3"],
			"fleet": ["0.11.7"]
		  },
		  "release_notes": "No updates"
		},
		
		"899.15.0": {
		  "release_date": "2016-04-05 17:12:31 +0000",
		  "major_software": {
			"kernel": ["4.3.6"],
			"docker": ["1.9.1"],
			"etcd": ["0.4.9","2.2.3"],
			"fleet": ["0.11.7"]
		  },
		  "release_notes": "No updates"
		},
		
		"899.13.0": {
		  "release_date": "2016-03-23 02:45:55 +0000",
		  "major_software": {
			"kernel": ["4.3.6"],
			"docker": ["1.9.1"],
			"etcd": ["0.4.9","2.2.3"],
			"fleet": ["0.11.5"]
		  },
		  "release_notes": "No updates"
		},
		
		"835.13.0": {
		  "release_date": "2016-02-18 07:07:19 +0000",
		  "major_software": {
			"kernel": ["4.2.2"],
			"docker": ["1.8.3"],
			"etcd": ["0.4.9","2.2.0"],
			"fleet": ["0.11.5"]
		  },
		  "release_notes": "No updates"
		},
		
		"835.12.0": {
		  "release_date": "2016-02-01 21:57:35 +0000",
		  "major_software": {
			"kernel": ["4.2.2"],
			"docker": ["1.8.3"],
			"etcd": ["0.4.9","2.2.0"],
			"fleet": ["0.11.5"]
		  },
		  "release_notes": "No updates"
		},
		
		"835.11.0": {
		  "release_date": "2016-01-22 20:25:46 +0000",
		  "major_software": {
			"kernel": ["4.2.2"],
			"docker": ["1.8.3"],
			"etcd": ["0.4.9","2.2.0"],
			"fleet": ["0.11.5"]
		  },
		  "release_notes": "No updates"
		},
		
		"835.10.0": {
		  "release_date": "2016-01-20 17:46:55 +0000",
		  "major_software": {
			"kernel": ["4.2.2"],
			"docker": ["1.8.3"],
			"etcd": ["0.4.9","2.2.0"],
			"fleet": ["0.11.5"]
		  },
		  "release_notes": "No updates"
		},
		
		"835.9.0": {
		  "release_date": "2015-12-08 16:32:50 +0000",
		  "major_software": {
			"kernel": ["4.2.2"],
			"docker": ["1.8.3"],
			"etcd": ["0.4.9","2.2.0"],
			"fleet": ["0.11.5"]
		  },
		  "release_notes": "No updates"
		},
		
		"835.8.0": {
		  "release_date": "2015-12-01 23:03:20 +0000",
		  "major_software": {
			"kernel": ["4.2.2"],
			"docker": ["1.8.3"],
			"etcd": ["0.4.9","2.2.0"],
			"fleet": ["0.11.5"]
		  },
		  "release_notes": "No updates"
		},
		
		"766.5.0": {
		  "release_date": "2015-11-05 20:53:12 +0000",
		  "major_software": {
			"kernel": ["4.1.7"],
			"docker": ["1.7.1"],
			"etcd": ["0.4.9","2.1.2"],
			"fleet": ["0.10.2"]
		  },
		  "release_notes": "No updates"
		},
		
		"766.4.0": {
		  "release_date": "2015-09-16 20:23:53 +0000",
		  "major_software": {
			"kernel": ["4.1.7"],
			"docker": ["1.7.1"],
			"etcd": ["0.4.9","2.1.2"],
			"fleet": ["0.10.2"]
		  },
		  "release_notes": "No updates"
		},
		
		"766.3.0": {
		  "release_date": "2015-09-02 18:14:20 +0000",
		  "major_software": {
			"kernel": ["4.1.6"],
			"docker": ["1.7.1"],
			"etcd": ["0.4.9","2.1.2"],
			"fleet": ["0.10.2"]
		  },
		  "release_notes": "No updates"
		},
		
		"717.3.0": {
		  "release_date": "2015-07-10 00:50:39 +0000",
		  "major_software": {
			"kernel": ["4.0.5"],
			"docker": ["1.6.2"],
			"etcd": ["0.4.9","2.0.10"],
			"fleet": ["0.10.2"]
		  },
		  "release_notes": "No updates"
		},
		
		"723.3.0": {
		  "release_date": "2015-07-09 20:30:21 +0000",
		  "major_software": {
			"kernel": ["4.0.5"],
			"docker": ["1.6.2"],
			"etcd": ["0.4.9","2.0.12"],
			"fleet": ["0.10.2"]
		  },
		  "release_notes": "No updates"
		},
		
		"717.1.0": {
		  "release_date": "2015-06-24 17:11:24 +0000",
		  "major_software": {
			"kernel": ["4.0.5"],
			"docker": ["1.6.2"],
			"etcd": ["0.4.9","2.0.10"],
			"fleet": ["0.10.2"]
		  },
		  "release_notes": "No updates"
		},
		
		"681.2.0": {
		  "release_date": "2015-06-18 15:19:35 +0000",
		  "major_software": {
			"kernel": ["4.0.5"],
			"docker": ["1.6.2"],
			"etcd": ["0.4.9","2.0.10"],
			"fleet": ["0.10.2"]
		  },
		  "release_notes": "No updates"
		},
		
		"681.1.0": {
		  "release_date": "2015-06-17 18:33:42 +0000",
		  "major_software": {
			"kernel": ["4.0.5"],
			"docker": ["1.6.2"],
			"etcd": ["0.4.9","2.0.10"],
			"fleet": ["0.10.1"]
		  },
		  "release_notes": "No updates"
		},
		
		"647.2.0": {
		  "release_date": "2015-05-26 22:32:06 +0000",
		  "major_software": {
			"kernel": ["4.0.1"],
			"docker": ["1.5.0"],
			"etcd": ["0.4.9"],
			"fleet": ["0.9.2"]
		  },
		  "release_notes": "No updates"
		},
		
		"681.0.0": {
		  "release_date": "2015-05-14 16:49:27 +0000",
		  "major_software": {
			"kernel": ["4.0.3"],
			"docker": ["1.6.2"],
			"etcd": ["0.4.9","2.0.10"],
			"fleet": ["0.10.1"]
		  },
		  "release_notes": "No updates"
		},
		
		"647.0.0": {
		  "release_date": "2015-04-09 16:34:27 +0000",
		  "major_software": {
			"kernel": ["3.19.3"],
			"docker": ["1.5.0"],
			"etcd": ["0.4.9"],
			"fleet": ["0.9.2"]
		  },
		  "release_notes": "No updates"
		},
		
		"633.1.0": {
		  "release_date": "2015-03-26 17:00:41 +0000",
		  "major_software": {
			"kernel": ["3.19"],
			"docker": ["1.5.0"],
			"etcd": ["0.4.8"],
			"fleet": ["0.9.1"]
		  },
		  "release_notes": "No updates"
		},
		
		"607.0.0": {
		  "release_date": "2015-02-28 19:49:51 +0000",
		  "major_software": {
			"kernel": ["3.18.6"],
			"docker": ["1.5.0"],
			"etcd": ["0.4.7"],
			"fleet": ["0.9.1"]
		  },
		  "release_notes": "No updates"
		},
		
		"557.2.0": {
		  "release_date": "2015-02-04 18:19:14 +0000",
		  "major_software": {
			"kernel": ["3.18.1"],
			"docker": ["1.4.1"],
			"etcd": ["0.4.6"],
			"fleet": ["0.9.0"]
		  },
		  "release_notes": "No updates"
		},
		
		"522.6.0": {
		  "release_date": "2015-01-28 17:48:19 +0000",
		  "major_software": {
			"kernel": ["3.17.8"],
			"docker": ["1.3.3"],
			"etcd": ["0.4.6"],
			"fleet": ["0.8.3"]
		  },
		  "release_notes": "No updates"
		},
		
		"522.5.0": {
		  "release_date": "2015-01-12 21:58:31 +0000",
		  "major_software": {
			"kernel": ["3.17.8"],
			"docker": ["1.3.3"],
			"etcd": ["0.4.6"],
			"fleet": ["0.8.3"]
		  },
		  "release_notes": "No updates"
		},
		
		"522.4.0": {
		  "release_date": "2015-01-06 21:27:42 +0000",
		  "major_software": {
			"kernel": ["3.17.7"],
			"docker": ["1.3.3"],
			"etcd": ["0.4.6"],
			"fleet": ["0.8.3"]
		  },
		  "release_notes": "No updates"
		}
	  }`
)
