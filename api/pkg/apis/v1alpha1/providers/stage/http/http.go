/*
MIT License

Copyright (c) Microsoft Corporation.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE
*/
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/azure/symphony/api/pkg/apis/v1alpha1/model"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/contexts"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers"
	"github.com/azure/symphony/coa/pkg/logger"
	"github.com/oliveagle/jsonpath"
)

var msLock sync.Mutex
var sLog = logger.NewLogger("coa.runtime")

type HttpStageProviderConfig struct {
	Url              string `json:"url"`
	Method           string `json:"method"`
	SuccessCodes     []int  `json:"successCodes,omitempty"`
	WaitUrl          string `json:"wait.url,omitempty"`
	WaitInterval     int    `json:"wait.interval,omitempty"`
	WaitCount        int    `json:"wait.count,omitempty"`
	WaitStartCodes   []int  `json:"wait.start,omitempty"`
	WaitSuccessCodes []int  `json:"wait.success,omitempty"`
	WaitFailedCodes  []int  `json:"wait.fail,omitempty"`
	WaitJsonPath     string `json:"wait.jsonpath,omitempty"`
}
type HttpStageProvider struct {
	Config  HttpStageProviderConfig
	Context *contexts.ManagerContext
}

func (m *HttpStageProvider) Init(config providers.IProviderConfig) error {
	msLock.Lock()
	defer msLock.Unlock()
	sLog.Debug("  P (Http Stage): initialize")

	mockConfig, err := toHttpStageProviderConfig(config)
	if err != nil {
		sLog.Errorf("  P (Http Stage): expected HttpStageProviderConfig: %+v", err)
		return err
	}
	m.Config = mockConfig
	return nil
}
func toHttpStageProviderConfig(config providers.IProviderConfig) (HttpStageProviderConfig, error) {
	ret := HttpStageProviderConfig{}
	data, err := json.Marshal(config)
	if err != nil {
		return ret, err
	}
	err = json.Unmarshal(data, &ret)
	return ret, err
}
func (i *HttpStageProvider) InitWithMap(properties map[string]string) error {
	config, err := MockStageProviderConfigFromMap(properties)
	if err != nil {
		return err
	}
	return i.Init(config)
}
func MockStageProviderConfigFromMap(properties map[string]string) (HttpStageProviderConfig, error) {
	ret := HttpStageProviderConfig{}
	if v, ok := properties["url"]; ok {
		ret.Url = v
	} else {
		return ret, v1alpha2.NewCOAError(nil, "missing required property url", v1alpha2.BadConfig)
	}
	if v, ok := properties["method"]; ok {
		ret.Method = v
	} else {
		return ret, v1alpha2.NewCOAError(nil, "missing required property method", v1alpha2.BadConfig)
	}
	if v, ok := properties["successCodes"]; ok {
		codes, err := readIntArray(v)
		if err != nil {
			return ret, err
		}
		ret.SuccessCodes = codes
	}
	if v, ok := properties["wait.success"]; ok {
		codes, err := readIntArray(v)
		if err != nil {
			return ret, err
		}
		ret.SuccessCodes = codes
	}
	if v, ok := properties["wait.start"]; ok {
		codes, err := readIntArray(v)
		if err != nil {
			return ret, err
		}
		ret.SuccessCodes = codes
	}
	if v, ok := properties["wait.fail"]; ok {
		codes, err := readIntArray(v)
		if err != nil {
			return ret, err
		}
		ret.SuccessCodes = codes
	}
	if v, ok := properties["wait.url"]; ok {
		ret.WaitUrl = v
	}
	if v, ok := properties["wait.interval"]; ok {
		interval, err := strconv.Atoi(v)
		if err != nil {
			return ret, v1alpha2.NewCOAError(err, fmt.Sprintf("failed to parse wait interval %v", v), v1alpha2.BadConfig)
		}
		ret.WaitInterval = interval
	}
	if v, ok := properties["wait.count"]; ok {
		count, err := strconv.Atoi(v)
		if err != nil {
			return ret, v1alpha2.NewCOAError(err, fmt.Sprintf("failed to parse wait count %v", v), v1alpha2.BadConfig)
		}
		ret.WaitCount = count
	}
	if v, ok := properties["wait.jsonpath"]; ok {
		ret.WaitJsonPath = v
	}
	return ret, nil
}
func readIntArray(s string) ([]int, error) {
	var codes []int
	for _, code := range strings.Split(s, ",") {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		intCode, err := strconv.Atoi(code)
		if err != nil {
			return nil, v1alpha2.NewCOAError(err, fmt.Sprintf("failed to parse code %v", code), v1alpha2.BadConfig)
		}
		codes = append(codes, intCode)
	}
	return codes, nil
}
func (i *HttpStageProvider) Process(ctx context.Context, mgrContext contexts.ManagerContext, inputs map[string]interface{}) (map[string]interface{}, bool, error) {
	sLog.Info("  P (Http Stage): Process")

	webClient := &http.Client{}
	req, err := http.NewRequest(fmt.Sprintf("%v", i.Config.Method), fmt.Sprintf("%v", i.Config.Url), nil)
	if err != nil {
		return nil, false, err
	}
	for key, input := range inputs {
		if strings.HasPrefix(key, "header.") {
			req.Header.Add(key[7:], fmt.Sprintf("%v", input))
		}
		if key == "body" {
			jData, err := json.Marshal(input)
			if err != nil {
				return nil, false, err
			}
			req.Body = ioutil.NopCloser(bytes.NewBuffer(jData))
			req.ContentLength = int64(len(jData))
		}
	}

	resp, err := webClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	outputs := make(map[string]interface{})

	for key, values := range resp.Header {
		outputs[fmt.Sprintf("header.%v", key)] = values
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, false, err
	}
	outputs["body"] = string(data) //TODO: probably not so good to assume string
	outputs["status"] = resp.StatusCode

	if i.Config.WaitUrl != "" {
		okToWait := false
		if len(i.Config.WaitStartCodes) > 0 {
			for _, code := range i.Config.WaitStartCodes {
				if code == resp.StatusCode {
					okToWait = true
					break
				}
			}
		}
		if !okToWait {
			return nil, false, v1alpha2.NewCOAError(nil, fmt.Sprintf("unexpected status code %v", resp.StatusCode), v1alpha2.BadConfig)
		}
		counter := 0
		failed := false
		succeeded := false
		sLog.Debug("  P (Http Stage): WaitCount: %v", i.Config.WaitCount)
		for counter < i.Config.WaitCount || i.Config.WaitCount == 0 {
			sLog.Info("  P (Http Stage): Start wait iteration %v", counter)
			waitReq, err := http.NewRequest("GET", i.Config.WaitUrl, nil)
			for key, input := range inputs {
				if strings.HasPrefix(key, "header.") {
					waitReq.Header.Add(key[7:], fmt.Sprintf("%v", input))
				}
			}
			if err != nil {
				return nil, false, err
			}
			waitResp, err := webClient.Do(waitReq)
			if err != nil {
				return nil, false, err
			}
			defer waitResp.Body.Close()
			if len(i.Config.WaitFailedCodes) > 0 {
				for _, code := range i.Config.WaitFailedCodes {
					if code == waitResp.StatusCode {
						failed = true
						break
					}
				}
			}
			if len(i.Config.WaitSuccessCodes) > 0 {
				for _, code := range i.Config.WaitSuccessCodes {
					if code == waitResp.StatusCode {
						succeeded = true
						break
					}
				}
			}
			if succeeded && i.Config.WaitJsonPath != "" {
				data, err := ioutil.ReadAll(waitResp.Body)
				if err != nil {
					succeeded = false
				} else {
					var obj interface{}
					err = json.Unmarshal(data, &obj)
					if err != nil {
						succeeded = false
					} else {
						result, err := jsonpath.JsonPathLookup(obj, i.Config.WaitJsonPath)
						if err != nil || result == nil {
							succeeded = false
						} else {
							switch result.(type) {
							case []interface{}:
								coll := result.([]interface{})
								succeeded = len(coll) > 0
							case map[string]interface{}:
								coll := result.(map[string]interface{})
								succeeded = len(coll) > 0
							default:
								succeeded = true
							}
						}
					}
				}
			}
			if !failed && !succeeded {
				counter++
				if i.Config.WaitInterval > 0 {
					sLog.Debug("  P (Http Stage): Sleep for wait interval")
					time.Sleep(time.Duration(i.Config.WaitInterval) * time.Second)
				}
			} else {
				break
			}
		}
		if failed {
			sLog.Errorf("  P (Http Stage): Failed to process request: %v", resp.StatusCode)
			return nil, false, v1alpha2.NewCOAError(nil, fmt.Sprintf("failed to wait for operation %v", resp.StatusCode), v1alpha2.BadConfig)
		}

	} else if len(i.Config.SuccessCodes) > 0 {
		for _, code := range i.Config.SuccessCodes {
			if code == resp.StatusCode {
				return outputs, false, nil
			}
		}
		sLog.Errorf("  P (Http Stage): Failed to process request: %v", resp.StatusCode)
		return nil, false, v1alpha2.NewCOAError(nil, fmt.Sprintf("unexpected status code %v", resp.StatusCode), v1alpha2.BadConfig)
	}

	sLog.Info("  P (Http Stage): Process request completed with: %v", resp.StatusCode)
	return outputs, false, nil
}
func (*HttpStageProvider) GetValidationRule(ctx context.Context) model.ValidationRule {
	return model.ValidationRule{
		RequiredProperties:        []string{},
		OptionalProperties:        []string{"header.*", "body"},
		RequiredComponentType:     "",
		RequiredMetadata:          []string{},
		OptionalMetadata:          []string{},
		ChangeDetectionProperties: []model.PropertyDesc{},
	}
}
