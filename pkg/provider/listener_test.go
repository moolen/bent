package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/moolen/bent/envoy/api/v2/core"
	"github.com/moolen/bent/envoy/api/v2/listener"
	hcm "github.com/moolen/bent/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/moolen/bent/pkg/util"
)

func TestListenerDefaults(t *testing.T) {
	l := NewListener(ListenerConfig{
		Name:             "ingress",
		Address:          "127.0.0.1",
		Port:             8125,
		TargetRoute:      "myroute",
		TracingOperation: hcm.INGRESS,
	})

	res := l.Resource()
	if res.Name != "ingress" {
		t.Errorf("listener has wrong name; expected %s, got %s", "ingress", res.Name)
	}
	if res.Address.Address.(*core.Address_SocketAddress).SocketAddress.Address != "127.0.0.1" {
		t.Errorf("listener has wrong address; expected %s, got %s", "127.0.0.1", res.Address.Address.(*core.Address_SocketAddress).SocketAddress.Address)
	}
	if res.Address.Address.(*core.Address_SocketAddress).SocketAddress.PortSpecifier.(*core.SocketAddress_PortValue).PortValue != 8125 {
		t.Errorf("listener has wrong port; expected %d, got %d", 8125, res.Address.Address.(*core.Address_SocketAddress).SocketAddress.PortSpecifier.(*core.SocketAddress_PortValue).PortValue)
	}
	if len(res.FilterChains) == 0 {
		t.Errorf("listener is missing a filter chain; expected 1 filter, got %d", len(res.FilterChains))
	}
	if len(res.FilterChains[0].Filters) == 0 {
		t.Errorf("listener is missing a filter; expected 1 filter, got %d", len(res.FilterChains[0].Filters))
	}
	if res.FilterChains[0].Filters[0].Name != util.HTTPConnectionManager {
		t.Errorf("expected hcm as first filter but found: %s", res.FilterChains[0].Filters[0].Name)
	}
	filters, err := getHTTPFilters(res.FilterChains[0].Filters[0])
	if err != nil {
		t.Error(err)
	}
	if err := assertHTTPFilters(filters, util.Router); err != nil {
		t.Error(err)
	}

	// modify listener: prepend authz
	// [authz] -> [router]
	l.InjectAuthz(AuthzConfig{
		Cluster: "foo",
	})

	res = l.Resource()
	if res.Name != "ingress" {
		t.Errorf("listener has wrong address; expected %s, got %s", "ingress", res.Name)
	}
	if len(res.FilterChains) == 0 {
		t.Errorf("listener is missing a filter chain; expected 1 filter, got %d", len(res.FilterChains))
	}
	if len(res.FilterChains[0].Filters) == 0 {
		t.Errorf("listener is missing a filter; expected 1 filter, got %d", len(res.FilterChains[0].Filters))
	}
	if res.FilterChains[0].Filters[0].Name != util.HTTPConnectionManager {
		t.Errorf("expected hcm as first filter but found: %s", res.FilterChains[0].Filters[0].Name)
	}
	filters, err = getHTTPFilters(res.FilterChains[0].Filters[0])
	if err != nil {
		t.Error(err)
	}
	if err := assertHTTPFilters(filters, util.HTTPExternalAuthorization, util.Router); err != nil {
		t.Error(err)
	}

	// modify listener: prepend fault
	// [fault] -> [authz] -> [router]
	l.InjectFault(FaultConfig{
		Enabled:       true,
		AbortChance:   10,
		AbortCode:     418,
		DelayChance:   20,
		DelayDuration: time.Millisecond * 100,
	})

	res = l.Resource()
	if res.Name != "ingress" {
		t.Errorf("listener has wrong address; expected %s, got %s", "ingress", res.Name)
	}
	if len(res.FilterChains) == 0 {
		t.Errorf("listener is missing a filter chain; expected 1 filter, got %d", len(res.FilterChains))
	}
	if len(res.FilterChains[0].Filters) == 0 {
		t.Errorf("listener is missing a filter; expected 1 filter, got %d", len(res.FilterChains[0].Filters))
	}
	if res.FilterChains[0].Filters[0].Name != util.HTTPConnectionManager {
		t.Errorf("expected hcm as first filter but found: %s", res.FilterChains[0].Filters[0].Name)
	}
	filters, err = getHTTPFilters(res.FilterChains[0].Filters[0])
	if err != nil {
		t.Error(err)
	}
	if err := assertHTTPFilters(filters, util.Fault, util.HTTPExternalAuthorization, util.Router); err != nil {
		t.Error(err)
	}

}

func assertHTTPFilters(filters []*hcm.HttpFilter, filterTypes ...string) error {
	if len(filters) != len(filterTypes) {
		return fmt.Errorf("wrong number of filters: expected %d, found %d", len(filterTypes), len(filters))
	}
	for i, filter := range filters {
		if filter.Name != filterTypes[i] {
			return fmt.Errorf("filter #%d has wrong type. expected %s, found: %s", i, filterTypes[i], filter.Name)
		}
	}
	return nil
}

func getHTTPFilters(f listener.Filter) ([]*hcm.HttpFilter, error) {
	var hcm hcm.HttpConnectionManager
	err := types.UnmarshalAny(f.ConfigType.(*listener.Filter_TypedConfig).TypedConfig, &hcm)
	if err != nil {
		return nil, err
	}
	return hcm.HttpFilters, nil
}
