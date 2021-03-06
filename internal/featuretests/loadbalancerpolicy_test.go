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

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	contour_api_v1 "github.com/projectcontour/contour/apis/projectcontour/v1"
	envoy_v2 "github.com/projectcontour/contour/internal/envoy/v2"
	"github.com/projectcontour/contour/internal/fixture"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// session affinity is only available in httpproxy
func TestLoadBalancerPolicySessionAffinity(t *testing.T) {
	rh, c, done := setup(t)
	defer done()

	s1 := fixture.NewService("app").WithPorts(
		v1.ServicePort{Port: 80, TargetPort: intstr.FromInt(8080)},
		v1.ServicePort{Port: 8080, TargetPort: intstr.FromInt(8080)})
	rh.OnAdd(s1)

	// simple single service
	proxy1 := fixture.NewProxy("simple").
		WithFQDN("www.example.com").
		WithSpec(contour_api_v1.HTTPProxySpec{
			Routes: []contour_api_v1.Route{{
				Conditions: matchconditions(prefixMatchCondition("/cart")),
				LoadBalancerPolicy: &contour_api_v1.LoadBalancerPolicy{
					Strategy: "Cookie",
				},
				Services: []contour_api_v1.Service{{
					Name: s1.Name,
					Port: 80,
				}},
			}},
		})
	rh.OnAdd(proxy1)

	c.Request(routeType).Equals(&envoy_api_v2.DiscoveryResponse{
		Resources: resources(t,
			envoy_v2.RouteConfiguration("ingress_http",
				envoy_v2.VirtualHost("www.example.com",
					&envoy_api_v2_route.Route{
						Match:  routePrefix("/cart"),
						Action: withSessionAffinity(routeCluster("default/app/80/e4f81994fe")),
					},
				),
			),
		),
		TypeUrl: routeType,
	})

	// two backends
	rh.OnUpdate(
		proxy1,
		fixture.NewProxy("simple").
			WithFQDN("www.example.com").
			WithSpec(contour_api_v1.HTTPProxySpec{
				Routes: []contour_api_v1.Route{{
					Conditions: matchconditions(prefixMatchCondition("/cart")),
					LoadBalancerPolicy: &contour_api_v1.LoadBalancerPolicy{
						Strategy: "Cookie",
					},
					Services: []contour_api_v1.Service{{
						Name: s1.Name,
						Port: 80,
					}, {
						Name: s1.Name,
						Port: 8080,
					}},
				}},
			}),
	)

	c.Request(routeType).Equals(&envoy_api_v2.DiscoveryResponse{
		Resources: resources(t,
			envoy_v2.RouteConfiguration("ingress_http",
				envoy_v2.VirtualHost("www.example.com",
					&envoy_api_v2_route.Route{
						Match: routePrefix("/cart"),
						Action: withSessionAffinity(
							routeWeightedCluster(
								weightedCluster{"default/app/80/e4f81994fe", 1},
								weightedCluster{"default/app/8080/e4f81994fe", 1},
							),
						),
					},
				),
			),
		),
		TypeUrl: routeType,
	})
}
