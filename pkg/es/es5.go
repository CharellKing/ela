package es

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/CharellKing/ela/config"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	elasticsearch5 "github.com/elastic/go-elasticsearch/v5"
	"github.com/elastic/go-elasticsearch/v5/esapi"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type V5 struct {
	*elasticsearch5.Client
	ClusterVersion string
}

func NewESV5(esConfig *config.ESConfig, clusterVersion string) (*V5, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client, err := elasticsearch5.NewClient(elasticsearch5.Config{
		Addresses: esConfig.Addresses,
		Username:  esConfig.User,
		Password:  esConfig.Password,
		Transport: transport,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &V5{
		Client:         client,
		ClusterVersion: clusterVersion,
	}, nil
}

func (es *V5) GetClusterVersion() string {
	return es.ClusterVersion
}

type ScrollResultV5 struct {
	Took     int    `json:"took,omitempty"`
	ScrollId string `json:"_scroll_id,omitempty"`
	TimedOut bool   `json:"timed_out,omitempty"`
	Hits     struct {
		MaxScore float32       `json:"max_score,omitempty"`
		Total    int           `json:"total,omitempty"`
		Docs     []interface{} `json:"hits,omitempty"`
	} `json:"hits"`
	Shards struct {
		Total      int `json:"total,omitempty"`
		Successful int `json:"successful,omitempty"`
		Skipped    int `json:"skipped,omitempty"`
		Failed     int `json:"failed,omitempty"`
		Failures   []struct {
			Shard  int         `json:"shard,omitempty"`
			Index  string      `json:"index,omitempty"`
			Status int         `json:"status,omitempty"`
			Reason interface{} `json:"reason,omitempty"`
		} `json:"failures,omitempty"`
	} `json:"_shards,omitempty"`
}

func (es *V5) NewScroll(index string, option *ScrollOption) (*ScrollResult, error) {
	scrollSearchOptions := []func(*esapi.SearchRequest){
		es.Search.WithIndex(index),
		es.Search.WithSize(cast.ToInt(option.ScrollSize)),
		es.Search.WithScroll(cast.ToDuration(option.ScrollTime) * time.Minute),
	}

	query := make(map[string]interface{})
	for k, v := range option.Query {
		query[k] = v
	}

	if option.SliceId != nil {
		query["slice"] = map[string]interface{}{
			"id":  *option.SliceId,
			"max": *option.SliceSize,
		}
	}

	if len(query) > 0 {
		var buf bytes.Buffer
		_ = json.NewEncoder(&buf).Encode(query)
		scrollSearchOptions = append(scrollSearchOptions, es.Client.Search.WithBody(&buf))
	}

	if len(option.SortFields) > 0 {
		scrollSearchOptions = append(scrollSearchOptions, es.Client.Search.WithSort(option.SortFields...))
	}

	res, err := es.Client.Search(scrollSearchOptions...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer func() {
		_ = res.Body.Close()
	}()

	if res.IsError() {
		return nil, errors.New(res.String())
	}

	var scrollResult ScrollResultV5
	if err := json.NewDecoder(res.Body).Decode(&scrollResult); err != nil {
		return nil, errors.WithStack(err)
	}

	var hitDocs []*Doc
	for _, hit := range scrollResult.Hits.Docs {
		var hitDoc Doc
		_ = mapstructure.Decode(hit, &hitDoc)
		hitDocs = append(hitDocs, &hitDoc)
	}

	return &ScrollResult{
		Total:    uint64(scrollResult.Hits.Total),
		Docs:     hitDocs,
		ScrollId: scrollResult.ScrollId,
	}, nil
}

func (es *V5) NextScroll(ctx context.Context, scrollId string, scrollTime uint) (*ScrollResult, error) {
	res, err := es.Client.Scroll(es.Client.Scroll.WithScrollID(scrollId), es.Client.Scroll.WithScroll(time.Duration(scrollTime)*time.Minute))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer func() {
		_ = res.Body.Close()
	}()

	if res.IsError() {
		return nil, errors.New(res.String())
	}

	var scrollResult ScrollResultV5
	if err := json.NewDecoder(res.Body).Decode(&scrollResult); err != nil {
		return nil, errors.WithStack(err)
	}

	var hitDocs []*Doc
	for _, hit := range scrollResult.Hits.Docs {
		var hitDoc Doc
		_ = mapstructure.Decode(hit, &hitDoc)
		hitDocs = append(hitDocs, &hitDoc)
	}

	return &ScrollResult{
		Total:    uint64(scrollResult.Hits.Total),
		Docs:     hitDocs,
		ScrollId: scrollResult.ScrollId,
	}, nil
}

func (es *V5) ClearScroll(scrollId string) error {
	res, err := es.Client.ClearScroll(es.Client.ClearScroll.WithScrollID(scrollId))
	if err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		_ = res.Body.Close()
	}()

	if res.IsError() {
		return errors.New(res.String())
	}

	return nil
}

func (es *V5) GetIndexMappingAndSetting(index string) (IESSettings, error) {
	// Get settings
	setting, err := es.GetIndexSettings(index)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	mapping, err := es.GetIndexMapping(index)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return NewV5Settings(setting, mapping, index), nil
}

func (es *V5) GetIndexMapping(index string) (map[string]interface{}, error) {
	// Get settings
	mappingRes, err := es.Client.Indices.GetMapping(es.Client.Indices.GetMapping.WithIndex(index))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer func() {
		_ = mappingRes.Body.Close()
	}()

	if mappingRes.IsError() {
		return nil, fmt.Errorf("error: %s", mappingRes.String())
	}

	bodyBytes, err := io.ReadAll(mappingRes.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	indexMapping := make(map[string]interface{})
	if err := json.Unmarshal(bodyBytes, &indexMapping); err != nil {
		return nil, errors.WithStack(err)
	}
	return indexMapping, nil
}

func (es *V5) GetIndexSettings(index string) (map[string]interface{}, error) {
	// Get settings
	settingRes, err := es.Client.Indices.GetSettings(es.Client.Indices.GetSettings.WithIndex(index))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer func() {
		_ = settingRes.Body.Close()
	}()

	if settingRes.IsError() {
		return nil, fmt.Errorf("error: %s", settingRes.String())
	}

	var indexSetting map[string]interface{}
	if err := json.NewDecoder(settingRes.Body).Decode(&indexSetting); err != nil {
		return nil, errors.WithStack(err)
	}

	return indexSetting, nil
}

func (es *V5) BulkInsert(index string, hitDocs []*Doc) error {
	var buf bytes.Buffer
	for _, doc := range hitDocs {
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": index,
				"_id":    doc.Type,
				"_type":  doc.Type,
			},
		}
		metaBytes, _ := json.Marshal(meta)
		buf.Write(metaBytes)
		buf.WriteByte('\n')
		dataBytes, _ := json.Marshal(doc.Source)
		buf.Write(dataBytes)
		buf.WriteByte('\n')
	}

	res, err := es.Client.Bulk(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return errors.WithStack(err)
	}
	if res.IsError() {
		return errors.New(res.String())
	}
	return nil
}

func (es *V5) CreateIndex(esSetting IESSettings) error {
	indexBodyMap := lo.Assign(
		esSetting.GetSettings(),
		esSetting.GetMappings(),
	)

	indexSettingsBytes, _ := json.Marshal(indexBodyMap)

	req := esapi.IndicesCreateRequest{
		Index: esSetting.GetIndex(),
		Body:  bytes.NewBuffer(indexSettingsBytes),
	}

	res, err := req.Do(context.Background(), es)
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	// 检查响应状态
	if res.IsError() {
		return fmt.Errorf("error creating index: %s", res.String())
	}
	return nil
}

func (es *V5) IndexExisted(indexName string) (bool, error) {
	res, err := es.Client.Indices.Exists([]string{indexName})
	if err != nil {
		return false, errors.WithStack(err)
	}

	if res.StatusCode == 404 {
		return false, nil
	}

	defer func() {
		_ = res.Body.Close()
	}()

	if res.IsError() {
		return false, fmt.Errorf("error checking index existence: %s", res.String())
	}

	return res.StatusCode == 200, nil
}

func (es *V5) DeleteIndex(index string) error {
	res, err := es.Client.Indices.Delete([]string{index})
	if err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		_ = res.Body.Close()
	}()

	if res.IsError() {
		return errors.New(res.String())
	}

	return nil
}

func (es *V5) BulkUpdate(index string, hitDocs []*Doc) error {
	var buf bytes.Buffer

	for _, doc := range hitDocs {
		// Prepare the metadata for the update action
		meta := map[string]interface{}{
			"update": map[string]interface{}{
				"_index": index,
				"_id":    doc.ID,
				"_type":  doc.Type,
			},
		}
		metaBytes, err := json.Marshal(meta)
		if err != nil {
			return errors.WithStack(err)
		}
		buf.Write(metaBytes)
		buf.WriteByte('\n')

		// Prepare the document data for update
		docData := map[string]interface{}{
			doc.Type: doc.Source,
		}
		docBytes, err := json.Marshal(docData)
		if err != nil {
			return errors.WithStack(err)
		}
		buf.Write(docBytes)
		buf.WriteByte('\n')
	}

	// Execute the bulk request
	res, err := es.Client.Bulk(bytes.NewReader(buf.Bytes()), es.Client.Bulk.WithIndex(index))
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	// Handle the response
	if res.IsError() {
		return fmt.Errorf("error executing bulk update: %s", res.String())
	}

	return nil
}

func (es *V5) BulkDelete(index string, hitDocs []*Doc) error {
	var buf bytes.Buffer

	for _, doc := range hitDocs {
		meta := map[string]interface{}{
			"delete": map[string]interface{}{
				"_index": index,
				"_id":    doc.ID,
				"_type":  doc.Type,
			},
		}
		metaBytes, err := json.Marshal(meta)
		if err != nil {
			return errors.WithStack(err)
		}
		buf.Write(metaBytes)
		buf.WriteByte('\n')
	}

	// Execute the bulk request
	res, err := es.Client.Bulk(bytes.NewReader(buf.Bytes()), es.Client.Bulk.WithIndex(index))
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	// Handle the response
	if res.IsError() {
		return fmt.Errorf("error executing bulk delete: %s", res.String())
	}

	return nil
}

func (es *V5) GetIndexes() ([]string, error) {
	res, err := es.Client.Cat.Indices()
	if err != nil {
		log.Fatalf("Error getting indices: %s", err)
		return nil, err
	}

	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %s", err)
		}
	}()

	if res.IsError() {
		return nil, fmt.Errorf("error: %s", res.String())
	}

	var indices []string
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		value := scanner.Text()
		segments := strings.Split(value, " ")
		indices = append(indices, segments[2])
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.WithStack(err)
	}

	return indices, nil
}
