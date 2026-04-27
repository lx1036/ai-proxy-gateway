package translator

import (
	"fmt"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/lx1036/gateway/pkg/envoygateway/ir"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/proto"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	httpConnectionManagerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

func (translator *Translator)  addHTTPConnectionManagerToXdsListener(xdsListener *listenerv3.Listener, irListener *ir.HTTPListener,) error  {


	httpConnectionManager := &httpConnectionManagerv3.HttpConnectionManager{
		CodecType:                                  httpConnectionManagerv3.HttpConnectionManager_AUTO,
		StatPrefix:                                 "",
		RouteSpecifier:                             nil,
		HttpFilters:                                nil,
		AddUserAgent:                               nil,
		Tracing:                                    nil,
		CommonHttpProtocolOptions:                  nil,
		Http1SafeMaxConnectionDuration:             false,
		HttpProtocolOptions:                        nil,
		Http2ProtocolOptions:                       nil,
		Http3ProtocolOptions:                       nil,
		ServerName:                                 "",
		// Hide the Envoy proxy in the Server header by default
		ServerHeaderTransformation:                 httpConnectionManagerv3.HttpConnectionManager_PASS_THROUGH,
		SchemeHeaderTransformation:                 nil,
		MaxRequestHeadersKb:                        nil,
		StreamIdleTimeout:                          nil,
		StreamFlushTimeout:                         nil,
		RequestTimeout:                             nil,
		RequestHeadersTimeout:                      nil,
		DrainTimeout:                               nil,
		DelayedCloseTimeout:                        nil,
		AccessLog:                                  nil,
		AccessLogOptions:                           nil,
		UseRemoteAddress:                           nil,
		XffNumTrustedHops:                          0,
		OriginalIpDetectionExtensions:              nil,
		EarlyHeaderMutationExtensions:              nil,
		InternalAddressConfig:                      nil,
		SkipXffAppend:                              false,
		Via:                                        "",
		GenerateRequestId:                          nil,
		PreserveExternalRequestId:                  false,
		AlwaysSetRequestIdInResponse:               false,
		ForwardClientCertDetails:                   0,
		SetCurrentClientCertDetails:                nil,
		ForwardClientCertMatcher:                   nil,
		Proxy_100Continue:                          false,
		RepresentIpv4RemoteAddressAsIpv4MappedIpv6: false,
		UpgradeConfigs:                             nil,
		NormalizePath:                              nil,
		MergeSlashes:                               false,
		PathWithEscapedSlashesAction:               0,
		RequestIdExtension:                         nil,
		LocalReplyConfig:                           nil,
		StripMatchingHostPort:                      false,
		StripPortMode:                              nil,
		StreamErrorOnInvalidHttpMessage:            nil,
		PathNormalizationOptions:                   nil,
		StripTrailingHostDot:                       false,
		ProxyStatusConfig:                          nil,
		TypedHeaderValidationConfig:                nil,
		AppendXForwardedPort:                       false,
		AppendLocalOverload:                        false,
		AddProxyProtocolConnectionState:            nil,
		ForwardProtoConfig:                         nil,
	}

	var filters []*listenerv3.Filter


	filterAny, err := anypb.New(httpConnectionManager)
	if err != nil {
		return err
	}
	filters = append(filters,
		&listenerv3.Filter{
			Name: wellknown.HTTPConnectionManager, // "envoy.filters.network.http_connection_manager",
			ConfigType: &listenerv3.Filter_TypedConfig{
				TypedConfig: filterAny,
			},
		},
	)




	filterChain := &listenerv3.FilterChain{
		Name: ,
		Filters: filters,
	}

	// Add the HTTP filter chain as the default filter chain
	// Make sure one does not exist
	if xdsListener.DefaultFilterChain != nil {
		return fmt.Errorf("default filter chain already exists")
	}
	xdsListener.DefaultFilterChain = filterChain

	return nil
}

