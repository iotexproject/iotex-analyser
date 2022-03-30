package kernel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddress(t *testing.T) {
	//specify the test case
	//ioctl action hash 91612af5b7ace680d28c9833c91706bad8953eaaade8078b35c2b8b48a366a3c
	require := require.New(t)
	tests := []struct {
		address string
		want    string
	}{

		{
			address: "iota1qp3mxh8gx8fkqmss9c6jsm979wuv6qpm0waw6vhxt0dwzze8xxzkqanqr4d",
			want:    "io1djlzhwxdqqahhwhdxtn9hkhppvnnrptqtwf2h5",
		},
	}
	for _, test := range tests {
		got, err := AddressFromString(test.address)
		require.NoError(err)
		require.Equal(got.String(), test.want)
	}
}
