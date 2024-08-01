package es

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/CharellKing/ela/config"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

type ScrollResultYield struct {
	Total uint64
	Docs  []Doc
}

type Doc struct {
	Type   string                 `mapstructure:"_type"`
	ID     string                 `mapstructure:"_id"`
	Source map[string]interface{} `mapstructure:"_source"`
}

type ES interface {
	GetClusterVersion() string
	IndexExisted(index string) (bool, error)
	GetIndexes() ([]string, error)
	SearchByScroll(ctx context.Context, index string, query map[string]interface{},
		sort string, scrollSize uint, scrollTime uint, yield func(*ScrollResultYield)) error

	BulkInsert(index string, hitDocs []Doc) error
	BulkUpdate(index string, hitDocs []Doc) error
	BulkDelete(index string, hitDocs []Doc) error
	GetIndexMappingAndSetting(index string) (IESSettings, error)

	CreateIndex(esSetting IESSettings) error
	DeleteIndex(index string) error
}

type V0 struct {
	Config *config.ESConfig
}

type ClusterVersion struct {
	Name        string `json:"name,omitempty"`
	ClusterName string `json:"cluster_name,omitempty"`
	Version     struct {
		Number        string `json:"number,omitempty"`
		LuceneVersion string `json:"lucene_version,omitempty"`
	} `json:"version,omitempty"`
}

func NewESV0(config *config.ESConfig) *V0 {
	return &V0{
		Config: config,
	}
}

func (es *V0) GetES() (ES, error) {
	clusterVersion, err := es.GetVersion()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if strings.HasPrefix(clusterVersion.Version.Number, "8.") {
		return NewESV8(es.Config, clusterVersion.Version.Number)
	} else if strings.HasPrefix(clusterVersion.Version.Number, "7.") {
		return NewESV7(es.Config, clusterVersion.Version.Number)
	} else if strings.HasPrefix(clusterVersion.Version.Number, "6.") {
		return NewESV6(es.Config, clusterVersion.Version.Number)
	} else if strings.HasPrefix(clusterVersion.Version.Number, "5.") {
		return NewESV5(es.Config, clusterVersion.Version.Number)
	}

	return nil, errors.Errorf("unsupported version: %s", clusterVersion.Version.Number)
}

func (es *V0) GetVersion() (*ClusterVersion, error) {
	byteBuf, err := es.Get(es.Config.Addresses[0])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	version := &ClusterVersion{}
	err = json.Unmarshal(byteBuf, version)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return version, nil
}

func (es *V0) Get(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	transport := &http.Transport{
		DisableKeepAlives:  true,
		DisableCompression: false,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	}

	if es.Config.User != "" && es.Config.Password != "" {
		req.SetBasicAuth(es.Config.User, es.Config.Password)
	}

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return bodyBytes, nil
}
