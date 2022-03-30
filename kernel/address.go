package kernel

import (
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-address/address/bech32"
	"github.com/pkg/errors"
)

const (
	AddressLength = 20
)

func decodeBech32(encodedAddr string) ([]byte, error) {
	_, grouped, err := bech32.Decode(encodedAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode address")
	}
	// Group the payload into 8 bit groups.
	payload, err := bech32.ConvertBits(grouped, 5, 8, false)
	if err != nil {
		return nil, errors.Wrapf(err, "error when converting 5 bit groups into the payload")
	}
	return payload, nil
}

func AddressFromString(encodedAddr string) (address.Address, error) {
	payload, err := decodeBech32(encodedAddr)
	if err != nil {
		return nil, err
	}
	return address.FromBytes(payload)
}
