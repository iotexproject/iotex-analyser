package kernel

import (
	"fmt"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/v2/action/protocol/rolldpos"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
)

func Init(cfg *config.Config) {
	g := cfg.Genesis
	fmt.Printf("genesis %+v\n", g)
	rolldposProtocol = rolldpos.NewProtocol(
		g.NumCandidateDelegates,
		g.NumDelegates,
		g.NumSubEpochs,
		rolldpos.EnableDardanellesSubEpoch(g.DardanellesBlockHeight, g.DardanellesNumSubEpochs),
		rolldpos.EnableWakeSubEpoch(g.WakeBlockHeight, g.WakeNumSubEpochs),
	)
}

func Genesis() *genesis.Genesis {
	return &config.Default.Genesis
}
