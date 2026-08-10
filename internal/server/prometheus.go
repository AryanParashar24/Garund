package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type PrometheusClient struct {
	BaseURL string
	Client  *http.Client
}

type PrometheusResponse struct {
	Status string `json:"status"`

	Data struct {
		ResultType string `json:"resultType"`

		Result []struct {
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func NewPrometheusClient(
	baseURL string,
) *PrometheusClient {

	return &PrometheusClient{
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

func (p *PrometheusClient) Query(
	query string,
) (float64, error) {

	endpoint, err :=
		url.Parse(
			p.BaseURL + "/api/v1/query",
		)

	if err != nil {
		return 0, err
	}

	params := endpoint.Query()

	params.Set(
		"query",
		query,
	)

	endpoint.RawQuery =
		params.Encode()

	response, err :=
		p.Client.Get(
			endpoint.String(),
		)

	if err != nil {
		return 0, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf(
			"prometheus returned %s",
			response.Status,
		)
	}

	var data PrometheusResponse

	if err :=
		json.NewDecoder(
			response.Body,
		).Decode(&data); err != nil {

		return 0, err
	}

	if data.Status != "success" {
		return 0, fmt.Errorf(
			"prometheus query failed",
		)
	}

	if len(data.Data.Result) == 0 {
		return 0, nil
	}

	if len(data.Data.Result[0].Value) < 2 {
		return 0, nil
	}

	value, ok :=
		data.Data.Result[0].Value[1].(string)

	if !ok {
		return 0, fmt.Errorf(
			"unexpected prometheus value type",
		)
	}

	var result float64

	if _, err :=
		fmt.Sscanf(
			value,
			"%f",
			&result,
		); err != nil {

		return 0, err
	}

	return result, nil
}

func (p *PrometheusClient) QueryOptional(
	query string,
) (float64, bool, error) {

	endpoint, err := url.Parse(
		p.BaseURL + "/api/v1/query",
	)

	if err != nil {
		return 0, false, err
	}

	params := endpoint.Query()

	params.Set("query", query)

	endpoint.RawQuery = params.Encode()

	response, err := p.Client.Get(
		endpoint.String(),
	)

	if err != nil {
		return 0, false, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf(
			"prometheus returned %s",
			response.Status,
		)
	}

	var data PrometheusResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&data); err != nil {
		return 0, false, err
	}

	if data.Status != "success" {
		return 0, false, fmt.Errorf(
			"prometheus query failed",
		)
	}

	if len(data.Data.Result) == 0 {
		return 0, false, nil
	}

	if len(data.Data.Result[0].Value) < 2 {
		return 0, false, nil
	}

	value, ok :=
		data.Data.Result[0].Value[1].(string)

	if !ok {
		return 0, false, fmt.Errorf(
			"unexpected prometheus value type",
		)
	}

	var result float64

	if _, err := fmt.Sscanf(
		value,
		"%f",
		&result,
	); err != nil {
		return 0, false, err
	}

	return result, true, nil
}
