package processing

import (
	"context"
	"encoding/json"
)

func Filter(ctx context.Context, channel <-chan TickerForm, payload chan<- []byte) {
    for {
        select {
            case <-ctx.Done():
                return
            case data, ok := <-channel:
                if !ok {
                    return
                }
                bytesData, err := json.Marshal(data)
                if err != nil {
                    return
                }
                payload <- bytesData
        }
    }
}
