package relayer

import (
	"sync"

	"github.com/rs/zerolog/log"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	types "github.com/scalarorg/relayers/pkg/types"
)

// Store all evm recovering redeem sessions
type CustodiansRecoverRedeemSessions struct {
	lock            sync.RWMutex
	RecoverSessions map[string]*types.GroupRedeemSessions
}

func (s *CustodiansRecoverRedeemSessions) AddRecoverSessions(chainId string, chainRedeemSessions *types.ChainRedeemSessions) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.RecoverSessions == nil {
		s.RecoverSessions = make(map[string]*types.GroupRedeemSessions)
	}
	for groupUid, switchPhaseEvent := range chainRedeemSessions.SwitchPhaseEvents {
		if len(switchPhaseEvent) == 0 {
			continue
		}
		groupSession, ok := s.RecoverSessions[groupUid]
		if !ok {
			groupSession = &types.GroupRedeemSessions{
				GroupUid:          groupUid,
				SwitchPhaseEvents: make(map[string][]*contracts.IScalarGatewaySwitchPhase),
				RedeemTokenEvents: make(map[string][]*contracts.IScalarGatewayRedeemToken),
			}
		}
		groupSession.SwitchPhaseEvents[chainId] = switchPhaseEvent
		s.RecoverSessions[groupUid] = groupSession
	}
	for groupUid, redeemTokenEvent := range chainRedeemSessions.RedeemTokenEvents {
		if len(redeemTokenEvent) == 0 {
			continue
		}
		groupSession, ok := s.RecoverSessions[groupUid]
		if !ok {
			log.Warn().Msgf("[Relayer] [AddRecoverSessions] no recover session found for group %s", groupUid)
			groupSession = &types.GroupRedeemSessions{
				GroupUid:          groupUid,
				SwitchPhaseEvents: make(map[string][]*contracts.IScalarGatewaySwitchPhase),
				RedeemTokenEvents: make(map[string][]*contracts.IScalarGatewayRedeemToken),
			}
		}
		groupSession.RedeemTokenEvents[chainId] = redeemTokenEvent
		s.RecoverSessions[groupUid] = groupSession
	}
}

func (s *CustodiansRecoverRedeemSessions) ConstructSessions() {
	log.Info().Msg("[Relayer] [ConstructSessions] start construct sessions")
	for _, groupSession := range s.RecoverSessions {
		groupSession.Construct()
	}
}

func (s *CustodiansRecoverRedeemSessions) GroupByChain() map[string]*types.ChainRedeemSessions {
	mapChainRedeemSessions := make(map[string]*types.ChainRedeemSessions)
	for groupUid, groupSession := range s.RecoverSessions {
		for chainId, switchPhaseEvent := range groupSession.SwitchPhaseEvents {
			mapChainRedeemSessions[chainId] = &types.ChainRedeemSessions{
				SwitchPhaseEvents: map[string][]*contracts.IScalarGatewaySwitchPhase{groupUid: switchPhaseEvent},
				RedeemTokenEvents: map[string][]*contracts.IScalarGatewayRedeemToken{},
			}
		}
		for chainId, redeemTokenEvent := range groupSession.RedeemTokenEvents {
			chainRedeemSessions, ok := mapChainRedeemSessions[chainId]
			if !ok {
				mapChainRedeemSessions[chainId] = &types.ChainRedeemSessions{
					SwitchPhaseEvents: map[string][]*contracts.IScalarGatewaySwitchPhase{},
					RedeemTokenEvents: map[string][]*contracts.IScalarGatewayRedeemToken{groupUid: redeemTokenEvent},
				}
			} else {
				chainRedeemSessions.RedeemTokenEvents[groupUid] = redeemTokenEvent
			}
		}
	}
	return mapChainRedeemSessions
}
