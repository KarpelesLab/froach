package froach

import (
	"context"

	"github.com/KarpelesLab/fleet"
)

func init() {
	fleet.SetRpcEndpoint("froach:address", func(any) (any, error) {
		addr := fleet.Self().IP + ":36257"
		return addr, nil
	})
}

func getAddrs() []string {
	res, _ := fleet.Self().AllRPC(context.Background(), "froach:address", nil)
	var final []string

	for _, v := range res {
		final = append(final, v.(string))
	}

	return final
}
