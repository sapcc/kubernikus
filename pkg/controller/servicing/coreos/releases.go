package coreos

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/sapcc/kubernikus/pkg/util/version"
)

var (
	baseURL       = "https://coreos.com/releases/releases-%s.json"
	holdoff       = 7 * 24 * time.Hour
	fetchInterval = 1 * time.Hour
)

type Release struct {
	Client       *http.Client
	ReleaseDates map[channel]*releaseDate
}

type releaseDate struct {
	Releases  map[string]time.Time
	FetchedAt time.Time
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

	if r.ReleaseDates[c] == nil || now().After(r.ReleaseDates[c].FetchedAt.Add(fetchInterval)) {
		result, err := r.fetch(c)
		if err != nil {
			return now(), errors.Wrapf(err, "Couldn't fetch %s releases", c)
		}
		r.ReleaseDates[c] = result
	}

	released, ok := r.ReleaseDates[c].Releases[v.String()]
	if !ok {
		return now(), errors.Errorf("Version %s not found in %s releases", v, c)
	}

	return released, nil
}

func (r *Release) fetch(c channel) (*releaseDate, error) {
	result, err := r.Client.Get(fmt.Sprintf(baseURL, c))
	if err != nil {
		return nil, fmt.Errorf("Couldn't fetch %s releases: %s", c, err)
	}

	defer result.Body.Close()

	body, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't read body")
	}

	var rdates map[string]interface{}

	err = json.Unmarshal(body, &rdates)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't parse response")
	}

	releaseDates := &releaseDate{
		Releases:  map[string]time.Time{},
		FetchedAt: now(),
	}

	for k, v := range rdates {
		version, err := version.ParseSemantic(k)
		if err != nil {
			return nil, errors.Wrap(err, "Couldn't parse version")
		}

		t, err := time.Parse("2006-01-02 15:04:05 +0000", v.(map[string]interface{})["release_date"].(string))
		if err != nil {
			return nil, errors.Wrap(err, "Couldn't parse release date")
		}

		releaseDates.Releases[version.String()] = t

	}

	return releaseDates, nil
}

func (r *Release) GrownUp(v *version.Version) (bool, error) {
	released, err := r.releasedAt(stable, v)
	if err != nil {
		return false, err
	}
	return released.Add(holdoff).Before(now()), nil
}
