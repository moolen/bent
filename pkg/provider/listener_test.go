package provider

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/moolen/bent/envoy/api/v2/core"
	"github.com/moolen/bent/envoy/api/v2/listener"
	hcm "github.com/moolen/bent/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/moolen/bent/pkg/util"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
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
	assert.Equal(t, res.Name, "ingress")
	assert.Equal(t, res.Address.Address.(*core.Address_SocketAddress).SocketAddress.Address, "127.0.0.1")
	assert.Equal(t, res.Address.Address.(*core.Address_SocketAddress).SocketAddress.PortSpecifier.(*core.SocketAddress_PortValue).PortValue, uint32(8125))
	assert.Assert(t, is.Len(res.FilterChains, 1))
	assert.Assert(t, is.Len(res.FilterChains[0].Filters, 1))
	assert.Equal(t, res.FilterChains[0].Filters[0].Name, util.HTTPConnectionManager)

	filters, err := getHTTPFilters(res.FilterChains[0].Filters[0])
	if err != nil {
		t.Error(err)
	}
	assertHTTPFilters(t, filters, util.Router)

	// modify listener: prepend authz
	// [authz] -> [router]
	l.InjectAuthz(AuthzConfig{
		Cluster: "foo",
	})

	res = l.Resource()
	assert.Equal(t, res.Name, "ingress")
	assert.Assert(t, is.Len(res.FilterChains, 1))
	assert.Assert(t, is.Len(res.FilterChains[0].Filters, 1))
	assert.Equal(t, res.FilterChains[0].Filters[0].Name, util.HTTPConnectionManager)
	filters, err = getHTTPFilters(res.FilterChains[0].Filters[0])
	if err != nil {
		t.Error(err)
	}
	assertHTTPFilters(t, filters, util.HTTPExternalAuthorization, util.Router)

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
	assert.Equal(t, res.Name, "ingress")
	assert.Assert(t, is.Len(res.FilterChains, 1))
	assert.Assert(t, is.Len(res.FilterChains[0].Filters, 1))
	assert.Equal(t, res.FilterChains[0].Filters[0].Name, util.HTTPConnectionManager)
	filters, err = getHTTPFilters(res.FilterChains[0].Filters[0])
	if err != nil {
		t.Error(err)
	}
	assertHTTPFilters(t, filters, util.Fault, util.HTTPExternalAuthorization, util.Router)

}

func assertHTTPFilters(t *testing.T, filters []*hcm.HttpFilter, filterTypes ...string) {
	assert.Assert(t, is.Len(filters, len(filterTypes)))
	for i, filter := range filters {
		assert.Equal(t, filter.Name, filterTypes[i])
	}
}

func getHTTPFilters(f listener.Filter) ([]*hcm.HttpFilter, error) {
	var hcm hcm.HttpConnectionManager
	err := types.UnmarshalAny(f.ConfigType.(*listener.Filter_TypedConfig).TypedConfig, &hcm)
	if err != nil {
		return nil, err
	}
	return hcm.HttpFilters, nil
}
