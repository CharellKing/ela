package es

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/CharellKing/ela/config"
	elasticsearch6 "github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type V6 struct {
	*elasticsearch6.Client
	ClusterVersion string
	Settings       IESSettings
}

func NewESV6(esConfig *config.ESConfig, clusterVersion string) (*V6, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client, err := elasticsearch6.NewClient(elasticsearch6.Config{
		Addresses: esConfig.Addresses,
		Username:  esConfig.User,
		Password:  esConfig.Password,
		Transport: transport,
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &V6{
		Client:         client,
		ClusterVersion: clusterVersion,
	}, nil
}

func (es *V6) GetClusterVersion() string {
	return es.ClusterVersion
}

func (es *V6) NewScroll(ctx context.Context, index string, option *ScrollOption) (*ScrollResult, error) {
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

func (es *V6) NextScroll(ctx context.Context, scrollId string, scrollTime uint) (*ScrollResult, error) {
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

func (es *V6) GetIndexMappingAndSetting(index string) (IESSettings, error) {
	// Get settings
	setting, err := es.GetIndexSettings(index)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	mapping, err := es.GetIndexMapping(index)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return NewV6Settings(setting, mapping, index), nil
}

func (es *V6) ClearScroll(scrollId string) error {
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

func (es *V6) GetIndexMapping(index string) (map[string]interface{}, error) {
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

func (es *V6) GetIndexSettings(index string) (map[string]interface{}, error) {
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

func (es *V6) BulkBody(index string, buf *bytes.Buffer, doc *Doc) error {
	action := ""
	var body map[string]interface{}

	switch doc.Op {
	case OperationCreate:
		action = "index"
		body = doc.Source
	case OperationUpdate:
		action = "update"
		body = map[string]interface{}{
			doc.Type: doc.Source,
		}
	case OperationDelete:
		action = "delete"
	default:
		return fmt.Errorf("unknow action %+v", doc.Op)
	}

	meta := map[string]interface{}{
		action: map[string]interface{}{
			"_index": index,
			"_id":    doc.ID,
			"_type":  doc.Type,
		},
	}

	metaBytes, _ := json.Marshal(meta)
	buf.Write(metaBytes)
	buf.WriteByte('\n')

	if len(body) > 0 {
		dataBytes, _ := json.Marshal(body)
		buf.Write(dataBytes)
		buf.WriteByte('\n')
	}
	return nil
}

func (es *V6) Bulk(buf *bytes.Buffer) error {
	// Execute the bulk request
	res, err := es.Client.Bulk(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	// Handle the response
	if res.IsError() {
		return errors.WithStack(fmt.Errorf("error executing bulk update: %s", res.String()))
	}
	return nil
}

func (es *V6) CreateIndex(esSetting IESSettings) error {
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

func (es *V6) IndexExisted(indexName string) (bool, error) {
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

func (es *V6) DeleteIndex(index string) error {
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

func (es *V6) GetIndexes() ([]string, error) {
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
