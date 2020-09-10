package xhrg

import (
	"context"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"time"
)

func Dial(ctx context.Context, target string) func() *grpc.ClientConn {
	dial := func() *grpc.ClientConn {
		dialTime := 10 * time.Second
		if *connectTimeout > 0 {
			dialTime = time.Duration(*connectTimeout * float64(time.Second))
		}
		ctx, cancel := context.WithTimeout(ctx, dialTime)
		defer cancel()
		var opts []grpc.DialOption
		if *keepaliveTime > 0 {
			timeout := time.Duration(*keepaliveTime * float64(time.Second))
			opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:    timeout,
				Timeout: timeout,
			}))
		}
		if *maxMsgSz > 0 {
			opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(*maxMsgSz)))
		}
		var creds credentials.TransportCredentials
		if !*plaintext {
			var err error
			creds, err = grpcurl.ClientTransportCredentials(*insecure, *cacert, *cert, *key)
			if err != nil {
				fail(err, "Failed to configure transport credentials")
			}

			// can use either -servername or -authority; but not both
			if *serverName != "" && *authority != "" {
				if *serverName == *authority {
					warn("Both -servername and -authority are present; prefer only -authority.")
				} else {
					fail(nil, "Cannot specify different values for -servername and -authority.")
				}
			}
			overrideName := *serverName
			if overrideName == "" {
				overrideName = *authority
			}

			if overrideName != "" {
				if err := creds.OverrideServerName(overrideName); err != nil {
					fail(err, "Failed to override server name as %q", overrideName)
				}
			}
		} else if *authority != "" {
			opts = append(opts, grpc.WithAuthority(*authority))
		}

		grpcurlUA := "grpcurl/" + version
		if version == no_version {
			grpcurlUA = "grpcurl/dev-build (no version set)"
		}
		if *userAgent != "" {
			grpcurlUA = *userAgent + " " + grpcurlUA
		}
		opts = append(opts, grpc.WithUserAgent(grpcurlUA))

		network := "tcp"
		if isUnixSocket != nil && isUnixSocket() {
			network = "unix"
		}
		cc, err := grpcurl.BlockingDial(ctx, network, target, creds, opts...)
		if err != nil {
			fail(err, "Failed to dial target host %q", target)
		}
		return cc
	}
	return dial
}

func ListS() {

	ctx := context.Background()
	md := grpcurl.MetadataFromHeaders(append(addlHeaders, reflHeaders...))
	refCtx := metadata.NewOutgoingContext(ctx, md)
	cc := Dial(ctx, "127.0.0.1:1535")
	refClient := grpcreflect.NewClient(refCtx, reflectpb.NewServerReflectionClient(cc()))
	reflSource := grpcurl.DescriptorSourceFromServer(ctx, refClient)
	svcs, _ := grpcurl.ListServices(reflSource)
	for index,s := range svcs {
		println(index)
		println(s)
	}
}
