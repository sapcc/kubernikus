package flatcar

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"github.com/pkg/errors"

	"github.com/sapcc/kubernikus/pkg/util/version"
)

var (
	versionURL   = "https://%s.release.flatcar-linux.net/amd64-usr/%s/version.txt"
	timeREString = `%s\s*FLATCAR_BUILD_ID="(.+)"`
	holdoff      = 7 * 24 * time.Hour
)

type Release struct {
	Client       *http.Client
	ReleaseDates map[channel]*releaseDate
}

type releaseDate struct {
	Releases map[string]time.Time
}

func (r *Release) releasedAt(c channel, v *version.Version) (time.Time, error) {
	if r.Client == nil {
		r.Client = &http.Client{
			Timeout: time.Second * 10,
		}
	}

	if r.ReleaseDates == nil {
		r.ReleaseDates = make(map[channel]*releaseDate)
	}

	if _, ok := r.ReleaseDates[c]; !ok {
		r.ReleaseDates[c] = &releaseDate{
			Releases: map[string]time.Time{},
		}
	}

	_, ok := r.ReleaseDates[c].Releases[v.String()]
	if !ok {
		result, err := r.fetch(c, v)
		if err != nil {
			return now(), errors.Wrapf(err, "Couldn't fetch release %s/%s", c, v.String())
		}
		r.ReleaseDates[c].Releases[v.String()] = result
	}

	return r.ReleaseDates[c].Releases[v.String()], nil
}

func (r *Release) fetch(c channel, v *version.Version) (time.Time, error) {
	result, err := r.Client.Get(fmt.Sprintf(versionURL, c, v.String()))
	if err != nil {
		return now(), fmt.Errorf("Couldn't fetch flatcar version.txt: %s", err)
	}

	defer result.Body.Close()

	body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return now(), fmt.Errorf("Couldn't read flatcar version.txt: %s", err)
	}

	timeRE := regexp.MustCompile(fmt.Sprintf(timeREString, v.String()))
	ts := timeRE.FindSubmatch(body)
	if len(ts) < 2 {
		return now(), fmt.Errorf("Couldn't parse flatcar timestamp %v", ts)
	}

	t, err := time.Parse("2006-01-02", string(ts[1])[0:10])
	if err != nil {
		return now(), fmt.Errorf("Couldn't extract flatcar timestamp: %v", t)
	}

	return t, nil
}

func (r *Release) GrownUp(v *version.Version) (bool, error) {
	released, err := r.releasedAt(stable, v)
	if err != nil {
		return false, err
	}
	return released.Add(holdoff).Before(now()), nil
}
