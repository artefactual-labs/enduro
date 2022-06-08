package a3m

import (
	context "context"

	a3m_transferservice "go.buf.build/grpc/go/artefactual/a3m/a3m/api/transferservice/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a client of a3m that remembers and reuses the underlying gPRC client.
type Client struct {
	TransferClient a3m_transferservice.TransferServiceClient
}

var currClient *Client

func NewClient(ctx context.Context, addr string) (*Client, error) {
	if currClient != nil {
		// Do we need to call conn.Connect()?
		return currClient, nil
	}

	c := &Client{}

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	currClient = c
	c.TransferClient = a3m_transferservice.NewTransferServiceClient(conn)

	return c, nil
}
