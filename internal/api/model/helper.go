package model

import "time"

func DefaultBlobberRequirements(id string, publicKey string) BlobberRequirements {
	return BlobberRequirements{
		Size:           10000,
		DataShards:     1,
		ParityShards:   1,
		ExpirationDate: time.Now().Add(time.Minute * 20).Unix(),
		ReadPriceRange: PriceRange{
			Min: 0,
			Max: 9223372036854775807,
		},
		WritePriceRange: PriceRange{
			Min: 0,
			Max: 9223372036854775807,
		},
		OwnerId:        id,
		OwnerPublicKey: publicKey,
	}
}
