package ddog

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	go_metrics "github.com/kentik/go-metrics"
	"github.com/kentik/ktranslate/pkg/eggs/logger"
	"github.com/kentik/ktranslate/pkg/formats"
	"github.com/kentik/ktranslate/pkg/kt"
)

type DDogSink struct {
	logger.ContextL
	TargetUrl string

	client   *http.Client
	tr       *http.Transport
	registry go_metrics.Registry
	metrics  *DDogMetric
	headers  map[string]string
}

type DDogMetric struct {
	DeliveryErr go_metrics.Meter
	DeliveryWin go_metrics.Meter
}

const (
	DD_API_KEY = "DD_API_KEY"
)

var (
	TargetUrl = flag.String("ddog_url", "https://api.datadoghq.com/api/v1/series", "URL to post to")
)

func NewSink(log logger.Underlying, registry go_metrics.Registry) (*DDogSink, error) {
	ddog := DDogSink{
		ContextL: logger.NewContextLFromUnderlying(logger.SContext{S: "ddogSink"}, log),
		registry: registry,
		metrics: &DDogMetric{
			DeliveryErr: go_metrics.GetOrRegisterMeter("delivery_errors_ddog", registry),
			DeliveryWin: go_metrics.GetOrRegisterMeter("delivery_wins_ddog", registry),
		},
		headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	return &ddog, nil
}

func (s *DDogSink) Init(ctx context.Context, format formats.Format, compression kt.Compression, fmtr formats.Formatter) error {
	s.TargetUrl = *TargetUrl

	s.tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	s.client = &http.Client{Transport: s.tr}

	if compression == kt.CompressionGzip {
		s.headers["Content-Encoding"] = "GZIP"
	}

	apiKey := os.Getenv(DD_API_KEY)
	if apiKey == "" {
		return fmt.Errorf("Missing env var %s", DD_API_KEY)
	}
	s.headers["DD-API-KEY"] = apiKey

	s.Infof("Exporting to DDog at %s", s.TargetUrl)

	return nil
}

func (s *DDogSink) Send(ctx context.Context, payload []byte) {
	go s.sendHttp(ctx, payload)
}

func (s *DDogSink) Close() {}

func (s *DDogSink) HttpInfo() map[string]float64 {
	return map[string]float64{
		"DeliveryErr": s.metrics.DeliveryErr.Rate1(),
		"DeliveryWin": s.metrics.DeliveryWin.Rate1(),
	}
}

func (s *DDogSink) sendHttp(ctx context.Context, payload []byte) {
	req, err := http.NewRequestWithContext(ctx, "POST", s.TargetUrl, bytes.NewBuffer(payload))
	if err != nil {
		s.Errorf("Cannot create HTTP request: %v", err)
		return
	}

	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.Errorf("Cannot write to DDOG: %v, creating new client", err)
		s.client = &http.Client{Transport: s.tr}
	} else {
		defer resp.Body.Close()
		bdy, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			s.Errorf("Cannot get resp body from DDOG: %v", err)
			s.metrics.DeliveryErr.Mark(1)
		} else {
			if resp.StatusCode >= 400 {
				s.Errorf("Cannot write to DDOG, status code %d, bdy: %s", resp.StatusCode, string(bdy))
				s.metrics.DeliveryErr.Mark(1)
			} else {
				s.metrics.DeliveryWin.Mark(1)
			}
		}
	}
}
