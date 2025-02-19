// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opencensusexporter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/common/testutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/testdata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver"
)

func TestSendTraces(t *testing.T) {
	sink := new(consumertest.TracesSink)
	rFactory := opencensusreceiver.NewFactory()
	rCfg := rFactory.CreateDefaultConfig().(*opencensusreceiver.Config)
	endpoint := testutil.GetAvailableLocalAddress(t)
	rCfg.GRPCServerSettings.NetAddr.Endpoint = endpoint
	set := componenttest.NewNopReceiverCreateSettings()
	recv, err := rFactory.CreateTracesReceiver(context.Background(), set, rCfg, sink)
	assert.NoError(t, err)
	assert.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, recv.Shutdown(context.Background()))
	})

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: endpoint,
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	cfg.NumWorkers = 1
	exp, err := factory.CreateTracesExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))
	t.Cleanup(func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	})

	td := testdata.GenerateTracesOneSpan()
	assert.NoError(t, exp.ConsumeTraces(context.Background(), td))
	assert.Eventually(t, func() bool {
		return len(sink.AllTraces()) == 1
	}, 10*time.Second, 5*time.Millisecond)
	traces := sink.AllTraces()
	require.Len(t, traces, 1)
	assert.Equal(t, td, traces[0])

	sink.Reset()
	// Sending data no Node.
	td.ResourceSpans().At(0).Resource().Attributes().Clear()
	newData := ptrace.NewTraces()
	td.CopyTo(newData)
	assert.NoError(t, exp.ConsumeTraces(context.Background(), newData))
	assert.Eventually(t, func() bool {
		return len(sink.AllTraces()) == 1
	}, 10*time.Second, 5*time.Millisecond)
	traces = sink.AllTraces()
	require.Len(t, traces, 1)
	assert.EqualValues(t, newData, traces[0])
}

func TestSendTraces_NoBackend(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: "localhost:56569",
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	exp, err := factory.CreateTracesExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))
	t.Cleanup(func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	})

	td := testdata.GenerateTracesOneSpan()
	for i := 0; i < 10000; i++ {
		assert.Error(t, exp.ConsumeTraces(context.Background(), td))
	}
}

func TestSendTraces_AfterStop(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: "localhost:56569",
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	exp, err := factory.CreateTracesExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))
	assert.NoError(t, exp.Shutdown(context.Background()))

	td := testdata.GenerateTracesOneSpan()
	assert.Error(t, exp.ConsumeTraces(context.Background(), td))
}

func TestSendMetrics(t *testing.T) {
	sink := new(consumertest.MetricsSink)
	rFactory := opencensusreceiver.NewFactory()
	rCfg := rFactory.CreateDefaultConfig().(*opencensusreceiver.Config)
	endpoint := testutil.GetAvailableLocalAddress(t)
	rCfg.GRPCServerSettings.NetAddr.Endpoint = endpoint
	set := componenttest.NewNopReceiverCreateSettings()
	recv, err := rFactory.CreateMetricsReceiver(context.Background(), set, rCfg, sink)
	assert.NoError(t, err)
	assert.NoError(t, recv.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, recv.Shutdown(context.Background()))
	})

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: endpoint,
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	cfg.NumWorkers = 1
	exp, err := factory.CreateMetricsExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))
	t.Cleanup(func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	})

	md := testdata.GenerateMetricsOneMetric()
	assert.NoError(t, exp.ConsumeMetrics(context.Background(), md))
	assert.Eventually(t, func() bool {
		return len(sink.AllMetrics()) == 1
	}, 10*time.Second, 5*time.Millisecond)
	metrics := sink.AllMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, md, metrics[0])

	// Sending data no node.
	sink.Reset()
	md.ResourceMetrics().At(0).Resource().Attributes().Clear()
	assert.NoError(t, exp.ConsumeMetrics(context.Background(), md))
	assert.Eventually(t, func() bool {
		return len(sink.AllMetrics()) == 1
	}, 10*time.Second, 5*time.Millisecond)
	metrics = sink.AllMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, md, metrics[0])
}

func TestSendMetrics_NoBackend(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: "localhost:56569",
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	exp, err := factory.CreateMetricsExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))
	t.Cleanup(func() {
		assert.NoError(t, exp.Shutdown(context.Background()))
	})

	md := testdata.GenerateMetricsOneMetric()
	for i := 0; i < 10000; i++ {
		assert.Error(t, exp.ConsumeMetrics(context.Background(), md))
	}
}

func TestSendMetrics_AfterStop(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
		Endpoint: "localhost:56569",
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	}
	exp, err := factory.CreateMetricsExporter(context.Background(), componenttest.NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	require.NotNil(t, exp)
	host := componenttest.NewNopHost()
	require.NoError(t, exp.Start(context.Background(), host))
	assert.NoError(t, exp.Shutdown(context.Background()))

	md := testdata.GenerateMetricsOneMetric()
	assert.Error(t, exp.ConsumeMetrics(context.Background(), md))
}
