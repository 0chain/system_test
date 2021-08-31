package magma

import (
	magma "github.com/magma/augmented-networks/accounting/protos"
	"google.golang.org/grpc"
)

func Client(magmaAddress string) (magma.AccountingClient, error) {
	conn, err := grpc.Dial(magmaAddress, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return magma.NewAccountingClient(conn), nil
}
