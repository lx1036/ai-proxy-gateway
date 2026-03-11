package xds

import (
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

type Watcher interface {
	DeleteWatchedResource(url string)
	GetWatchedResource(url string) *WatchedResource
	NewWatchedResource(url string, names []string)
	UpdateWatchedResource(string, func(*WatchedResource) *WatchedResource)
	// GetID identifies an xDS client. This is different from a connection ID.
	GetID() string
}

func Stream(ctx *ConnectionContext) error {
	con := ctx.XdsConnection()

	// Block until either a request is received or a push is triggered.
	// We need 2 go routines because 'read' blocks in Recv().
	go Receive(ctx)

	<-con.initialized

	for {
		select {
		case req, ok := <-con.reqChan:
			if ok {
				if err := ctx.Process(req); err != nil {
					return err
				}
			} else {
				// Remote side closed connection or error processing the request.
				return <-con.errorChan
			}
		case <-con.stop:
			return nil
		default:
		}

		select {
		case req, ok := <-con.reqChan:
			if ok {
				if err := ctx.Process(req); err != nil {
					return err
				}
			} else {
				// Remote side closed connection or error processing the request.
				return <-con.errorChan
			}
		case pushEv := <-con.pushChannel:
			err := ctx.Push(pushEv)
			if err != nil {
				return err
			}
		case <-con.stop:
			return nil
		}
	}
}

func Receive(ctx *ConnectionContext) {
	con := ctx.XdsConnection()

	firstRequest := true
	for {
		req, err := con.stream.Recv()
		if err != nil {
			//if istiogrpc.GRPCErrorType(err) != istiogrpc.UnexpectedError {
			//	klog.Infof("ADS: %q %s terminated", con.peerAddr, con.conID)
			//	return
			//}
			con.errorChan <- err
			klog.Errorf("ADS: %q %s terminated with error: %v", con.peerAddr, con.conID, err)
			//TotalXDSInternalErrors.Increment()
			return
		}

		// This should be only set for the first request. The node id may not be set - for example malicious clients.
		// check for security
		if firstRequest {

		}

		select {
		case con.reqChan <- req:
		case <-con.stream.Context().Done():
			klog.Infof("ADS: %q %s terminated with stream closed", con.peerAddr, con.conID)
			return
		}
	}
}

func Send(ctx *ConnectionContext, res *discovery.DiscoveryResponse) error {
	conn := ctx.XdsConnection()
	//start := time.Now()
	//defer func() { RecordSendTime(time.Since(start)) }()
	err := conn.stream.Send(res)
	if err == nil {
		//if res.Nonce != "" && !strings.HasPrefix(res.TypeUrl, model.DebugType) {
		//	ctx.Watcher().UpdateWatchedResource(res.TypeUrl, func(wr *WatchedResource) *WatchedResource {
		//		if wr == nil {
		//			wr = &WatchedResource{TypeUrl: res.TypeUrl}
		//		}
		//		wr.NonceSent = res.Nonce
		//		wr.LastSendTime = time.Now()
		//		return wr
		//	})
		//}
	} else if status.Convert(err).Code() == codes.DeadlineExceeded {
		//klog.Infof("Timeout writing %s: %v", conn.conID, model.GetShortType(res.TypeUrl))
		//ResponseWriteTimeouts.Increment()
	}

	return err
}
