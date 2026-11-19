package installer

import (
	"context"
	"fmt"
	"github.com/joseluisald/verman/internal/fault"
)

type offlineKey struct{}

func WithOffline(ctx context.Context) context.Context {
	return context.WithValue(ctx, offlineKey{}, true)
}
func offline(ctx context.Context) bool { value, _ := ctx.Value(offlineKey{}).(bool); return value }
func requireNetwork(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if offline(ctx) {
		return fault.New("offline", 3, "Use uma versão já instalada ou repita sem --offline.", fmt.Errorf("operação exige rede e --offline está ativo"))
	}
	return nil
}
