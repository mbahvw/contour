// Copyright Project Contour Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package featuretests

import (
	"testing"
	"time"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	contour_api_v1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	envoy_v2 "github.com/projectcontour/contour/internal/envoy/v2"
	"github.com/projectcontour/contour/internal/fixture"
	"github.com/projectcontour/contour/internal/timeout"
	xdscache_v2 "github.com/projectcontour/contour/internal/xdscache/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTimeoutsNotSpecified(t *testing.T) {
	// the contour.EventHandler.ListenerConfig has no timeout values specified
	rh, c, done := setup(t)
	defer done()

	s1 := fixture.NewService("backend").
		WithPorts(v1.ServicePort{Name: "http", Port: 80})
	rh.OnAdd(s1)

	hp1 := &contour_api_v1.HTTPProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: s1.Namespace,
		},
		Spec: contour_api_v1.HTTPProxySpec{
			VirtualHost: &contour_api_v1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []contour_api_v1.Route{{
				Conditions: matchconditions(prefixMatchCondition("/")),
				Services: []contour_api_v1.Service{{
					Name: s1.Name,
					Port: 80,
				}},
			}},
		},
	}
	rh.OnAdd(hp1)

	c.Request(listenerType, xdscache_v2.ENVOY_HTTP_LISTENER).Equals(&envoy_api_v2.DiscoveryResponse{
		TypeUrl: listenerType,
		Resources: resources(t,
			&envoy_api_v2.Listener{
				Name:          xdscache_v2.ENVOY_HTTP_LISTENER,
				Address:       envoy_v2.SocketAddress("0.0.0.0", 8080),
				SocketOptions: envoy_v2.TCPKeepaliveSocketOptions(),
				FilterChains: envoy_v2.FilterChains(envoy_v2.HTTPConnectionManagerBuilder().
					RouteConfigName(xdscache_v2.ENVOY_HTTP_LISTENER).
					MetricsPrefix(xdscache_v2.ENVOY_HTTP_LISTENER).
					AccessLoggers(envoy_v2.FileAccessLogEnvoy(xdscache_v2.DEFAULT_HTTP_ACCESS_LOG)).
					DefaultFilters().
					Get(),
				),
			}),
	})
}

func TestNonZeroTimeoutsSpecified(t *testing.T) {
	withTimeouts := func(conf *xdscache_v2.ListenerConfig) {
		conf.ConnectionIdleTimeout = timeout.DurationSetting(7 * time.Second)
		conf.StreamIdleTimeout = timeout.DurationSetting(70 * time.Second)
		conf.MaxConnectionDuration = timeout.DurationSetting(700 * time.Second)
		conf.ConnectionShutdownGracePeriod = timeout.DurationSetting(7000 * time.Second)
	}

	rh, c, done := setup(t, withTimeouts)
	defer done()

	s1 := fixture.NewService("backend").
		WithPorts(v1.ServicePort{Name: "http", Port: 80})
	rh.OnAdd(s1)

	hp1 := &contour_api_v1.HTTPProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: s1.Namespace,
		},
		Spec: contour_api_v1.HTTPProxySpec{
			VirtualHost: &contour_api_v1.VirtualHost{
				Fqdn: "example.com",
			},
			Routes: []contour_api_v1.Route{{
				Conditions: matchconditions(prefixMatchCondition("/")),
				Services: []contour_api_v1.Service{{
					Name: s1.Name,
					Port: 80,
				}},
			}},
		},
	}
	rh.OnAdd(hp1)

	c.Request(listenerType, xdscache_v2.ENVOY_HTTP_LISTENER).Equals(&envoy_api_v2.DiscoveryResponse{
		TypeUrl: listenerType,
		Resources: resources(t,
			&envoy_api_v2.Listener{
				Name:          xdscache_v2.ENVOY_HTTP_LISTENER,
				Address:       envoy_v2.SocketAddress("0.0.0.0", 8080),
				SocketOptions: envoy_v2.TCPKeepaliveSocketOptions(),
				FilterChains: envoy_v2.FilterChains(envoy_v2.HTTPConnectionManagerBuilder().
					RouteConfigName(xdscache_v2.ENVOY_HTTP_LISTENER).
					MetricsPrefix(xdscache_v2.ENVOY_HTTP_LISTENER).
					AccessLoggers(envoy_v2.FileAccessLogEnvoy(xdscache_v2.DEFAULT_HTTP_ACCESS_LOG)).
					DefaultFilters().
					ConnectionIdleTimeout(timeout.DurationSetting(7 * time.Second)).
					StreamIdleTimeout(timeout.DurationSetting(70 * time.Second)).
					MaxConnectionDuration(timeout.DurationSetting(700 * time.Second)).
					ConnectionShutdownGracePeriod(timeout.DurationSetting(7000 * time.Second)).
					Get(),
				),
			}),
	})
}
