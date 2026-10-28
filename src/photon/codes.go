package photon

import "strconv"

type OpCode byte

const (
	OperationInitEncryption            OpCode = 0
	OperationPing                      OpCode = 1
	OperationGetRoomList               OpCode = 217
	OperationServerSettings            OpCode = 218
	OperationRPC                       OpCode = 219
	OperationGetRegions                OpCode = 220
	OperationLobbyStats                OpCode = 221
	OperationFindFriends               OpCode = 222
	OperationJoinRandomRoom            OpCode = 225
	OperationJoinRoom                  OpCode = 226
	OperationCreateRoom                OpCode = 227
	OperationLeaveLobby                OpCode = 228
	OperationJoinLobby                 OpCode = 229
	OperationAuthenticate              OpCode = 230
	OperationAuthOnce                  OpCode = 231
	OperationChangeGroups              OpCode = 248
	OperationExchangeKeysForEncryption OpCode = 250
	OperationGetProperties             OpCode = 251
	OperationSetProperties             OpCode = 252
	OperationRaiseEvent                OpCode = 253
	OperationLeave                     OpCode = 254
	OperationJoin                      OpCode = 255
)

type EvCode byte

const (
	EventBattleEnter                        EvCode = 1
	EventBattleWaveStart                    EvCode = 2
	EventBattleWaveEnd                      EvCode = 3
	EventBattleExit                         EvCode = 4
	EventBattleRetire                       EvCode = 5
	EventBattleContinue                     EvCode = 6
	EventBattleBackgrondStart               EvCode = 7
	EventResumeEntry                        EvCode = 11
	EventIntrudeEntry                       EvCode = 12
	EventIntrudeRequest                     EvCode = 13
	EventIntrudeAccept                      EvCode = 14
	EventIntrudeJoin                        EvCode = 15
	EventMidwayReady                        EvCode = 16
	EventMidwayStart                        EvCode = 17
	EventSkillInput                         EvCode = 21
	EventSkillMoveEndInput                  EvCode = 22
	EventSkillReturnEndInput                EvCode = 23
	EventPotionInput                        EvCode = 24
	EventTargetInput                        EvCode = 25
	EventAutoModeInput                      EvCode = 26
	EventSpeedInput                         EvCode = 27
	EventPotionSettingInput                 EvCode = 28
	EventTimeControlInput                   EvCode = 29
	EventSwitchInput                        EvCode = 30
	EventSkillActivate                      EvCode = 31
	EventSkillMove                          EvCode = 32
	EventSkillInvoke                        EvCode = 33
	EventSkillCool                          EvCode = 34
	EventSkillSeal                          EvCode = 35
	EventSkillCancel                        EvCode = 36
	EventPotionInvoke                       EvCode = 37
	EventReturnPhase                        EvCode = 38
	EventHitEffect                          EvCode = 39
	EventDropReward                         EvCode = 40
	EventPossessionRelease                  EvCode = 41
	EventSwitch                             EvCode = 42
	EventPassiveInvoke                      EvCode = 43
	EventCharacterPropertyChanged           EvCode = 51
	EventCharacterPropertyChangedArray      EvCode = 52
	EventCharacterPropertySync              EvCode = 53
	EventCharacterAccumPropertyChanged      EvCode = 55
	EventCharacterAccumPropertyChangedArray EvCode = 56
	EventCharacterAccumPropertySync         EvCode = 57
	EventTreasureBattleStatusSync           EvCode = 61
	EventRaidBattleStatusSync               EvCode = 62
	EventRaidBattleRankingSync              EvCode = 63
	EventBattleFieldBattleStatusSync        EvCode = 64
	EventBreakRecoveryTimeSync              EvCode = 65
	EventTutorial                           EvCode = 70
	EventBattleProfile                      EvCode = 75
	EventGvgEntry                           EvCode = 81
	EventGvgIntrudeEntry                    EvCode = 82
	EventGvgSettle                          EvCode = 83
	EventSupportSkill                       EvCode = 84
	EventRaidStats                          EvCode = 100
	EventChatMessage                        EvCode = 101
	EventKick                               EvCode = 150
	EventAppError                           EvCode = 170
	EventDebugBattleEnter                   EvCode = 185
	EventDebugBattleBackgrondStart          EvCode = 186
	EventDebugGvgEntry                      EvCode = 187
	EventDebugging                          EvCode = 190
	EventDebugPing                          EvCode = 191
	EventDebugTimeScale                     EvCode = 192
	EventDebugSettingLoad                   EvCode = 193
	EventDebugSettingChange                 EvCode = 194
	EventDebugCharacter                     EvCode = 195
	EventTestSampleClass                    EvCode = 197
	EventTestSampleStruct                   EvCode = 198
	EventTestSampleComposite                EvCode = 199
	EventCloseConnection                    EvCode = 203

	EventVoiceData EvCode = 202
	// EventVoiceFrameData    EvCode = 203
	EventPunchMessage      EvCode = 220
	EventAuth              EvCode = 223
	EventLobbyStats        EvCode = 224
	EventAppStats          EvCode = 226
	EventRoomListUpdate    EvCode = 229
	EventRoomList          EvCode = 230
	EventCacheSliceChanged EvCode = 250
	EventPropertiesChanged EvCode = 253
	EventLeave             EvCode = 254
	EventJoin              EvCode = 255
)

type ParamCode byte

const (
	ParameterRoomOptionFlags                ParamCode = 191
	ParameterEncryptionData                 ParamCode = 192
	ParameterEncryptionMode                 ParamCode = 193
	ParameterCustomInitData                 ParamCode = 194
	ParameterExpectedProtocol               ParamCode = 195
	ParameterCluster                        ParamCode = 196
	ParameterPluginVersion                  ParamCode = 200
	ParameterPluginName                     ParamCode = 201
	ParameterNickName                       ParamCode = 202
	ParameterMasterClientID                 ParamCode = 203
	ParameterPlugins                        ParamCode = 204
	ParameterCacheSliceIndex                ParamCode = 205
	ParameterRPCCallReturnMessage           ParamCode = 206
	ParameterRPCCallReturnCode              ParamCode = 207
	ParameterRPCCallParameters              ParamCode = 208
	ParameterURIPath                        ParamCode = 209
	ParameterRegion                         ParamCode = 210
	ParameterLobbyStats                     ParamCode = 211
	ParameterLobbyType                      ParamCode = 212
	ParameterLobbyName                      ParamCode = 213
	ParameterClientAuthenticationData       ParamCode = 214
	ParameterJoinMode                       ParamCode = 215
	ParameterClientAuthenticationParameters ParamCode = 216
	ParameterClientAuthenticationType       ParamCode = 217
	ParameterInfo                           ParamCode = 218
	ParameterAppVersion                     ParamCode = 220
	ParameterSecret                         ParamCode = 221
	ParameterRoomList                       ParamCode = 222
	ParameterMatchmakingType                ParamCode = 223
	ParameterApplicationID                  ParamCode = 224
	ParameterUserID                         ParamCode = 225
	ParameterMasterPeerCount                ParamCode = 227
	ParameterRoomCount                      ParamCode = 228
	ParameterPeerCount                      ParamCode = 229
	ParameterAddress                        ParamCode = 230
	ParameterExpectedValues                 ParamCode = 231
	ParameterCheckUserOnJoin                ParamCode = 232
	ParameterIsInactive                     ParamCode = 233
	ParameterEventForward                   ParamCode = 234
	ParameterPlayerTTL                      ParamCode = 235
	ParameterEmptyRoomTTL                   ParamCode = 236
	ParameterSuppressRoomEvents             ParamCode = 237
	ParameterAdd                            ParamCode = 238
	ParameterPublishUserID                  ParamCode = 239
	ParameterGroup                          ParamCode = 240
	ParameterCleanupCacheOnLeave            ParamCode = 241
	ParameterCode                           ParamCode = 244
	ParameterData                           ParamCode = 245
	ParameterReceiverGroup                  ParamCode = 246
	ParameterCache                          ParamCode = 247
	ParameterRoomProperties                 ParamCode = 248
	ParameterPlayerProperties               ParamCode = 249
	ParameterBroadcast                      ParamCode = 250
	ParameterProperties                     ParamCode = 251
	ParameterPlayerList                     ParamCode = 252
	ParameterTargetPlayerNumber             ParamCode = 253
	ParameterPlayerNumber                   ParamCode = 254
	ParameterRoomName                       ParamCode = 255
)

type CustomCode byte

const (
	CustomCodeSkillInputResponse        CustomCode = 1
	CustomCodeSkillActivateResponse     CustomCode = 2
	CustomCodeSkillInvokeResponse       CustomCode = 3
	CustomCodeSkillCoolResponse         CustomCode = 4
	CustomCodeSkillSealResponse         CustomCode = 5
	CustomCodeSkillCancelResponse       CustomCode = 6
	CustomCodePotionInvokeResponse      CustomCode = 7
	CustomCodePotionSettingInput        CustomCode = 8
	CustomCodeSkillUnit                 CustomCode = 9
	CustomCodeHitEffect                 CustomCode = 10
	CustomCodeDeckResponse              CustomCode = 11
	CustomCodeCharacterResponse         CustomCode = 12
	CustomCodeBattleEnterResponse       CustomCode = 13
	CustomCodeWaveResponse              CustomCode = 14
	CustomCodeNextWaveResponse          CustomCode = 15
	CustomCodeMidwayReadyResponse       CustomCode = 16
	CustomCodeDropRewardResponse        CustomCode = 17
	CustomCodeBattleExitResponse        CustomCode = 18
	CustomCodeTreasureBattleStatus      CustomCode = 19
	CustomCodePossesionReleaseResponse  CustomCode = 20
	CustomCodeSwitchResponse            CustomCode = 21
	CustomCodePassiveInvokeResponse     CustomCode = 22
	CustomCodeProfile                   CustomCode = 23
	CustomCodeBreakRecoveryTimeResponse CustomCode = 24
	CustomCodeGvgEntryRequest           CustomCode = 30
	CustomCodeGvgIntrudeRequest         CustomCode = 31
	CustomCodeGvgTeam                   CustomCode = 32
	CustomCodeGvgPlayer                 CustomCode = 33
	CustomCodeGuildFacility             CustomCode = 34
	CustomCodeRaidStatsResponse         CustomCode = 100
	CustomCodeRankingRow                CustomCode = 150
	CustomCodeSampleClass               CustomCode = 253
	CustomCodeSampleStruct              CustomCode = 254
	CustomCodeSampleComposite           CustomCode = 255
)

func (c OpCode) String() string {
	switch c {
	case OperationInitEncryption:
		return "InitEncryption"
	case OperationPing:
		return "Ping"
	case OperationGetRoomList:
		return "GetRoomList"
	case OperationServerSettings:
		return "ServerSettings"
	case OperationRPC:
		return "RPC"
	case OperationGetRegions:
		return "GetRegions"
	case OperationLobbyStats:
		return "LobbyStats"
	case OperationFindFriends:
		return "FindFriends"
	case OperationJoinRandomRoom:
		return "JoinRandomRoom"
	case OperationJoinRoom:
		return "JoinRoom"
	case OperationCreateRoom:
		return "CreateRoom"
	case OperationLeaveLobby:
		return "LeaveLobby"
	case OperationJoinLobby:
		return "JoinLobby"
	case OperationAuthenticate:
		return "Authenticate"
	case OperationAuthOnce:
		return "AuthOnce"
	case OperationChangeGroups:
		return "ChangeGroups"
	case OperationExchangeKeysForEncryption:
		return "ExchangeKeysForEncryption"
	case OperationGetProperties:
		return "GetProperties"
	case OperationSetProperties:
		return "SetProperties"
	case OperationRaiseEvent:
		return "RaiseEvent"
	case OperationLeave:
		return "Leave"
	case OperationJoin:
		return "Join"
	default:
		return strconv.Itoa(int(c))
	}
}

func (c EvCode) String() string {
	switch c {
	case EventBattleEnter:
		return "BattleEnter"
	case EventBattleWaveStart:
		return "BattleWaveStart"
	case EventBattleWaveEnd:
		return "BattleWaveEnd"
	case EventBattleExit:
		return "BattleExit"
	case EventBattleRetire:
		return "BattleRetire"
	case EventBattleContinue:
		return "BattleContinue"
	case EventBattleBackgrondStart:
		return "BattleBackgrondStart"
	case EventResumeEntry:
		return "ResumeEntry"
	case EventIntrudeEntry:
		return "IntrudeEntry"
	case EventIntrudeRequest:
		return "IntrudeRequest"
	case EventIntrudeAccept:
		return "IntrudeAccept"
	case EventIntrudeJoin:
		return "IntrudeJoin"
	case EventMidwayReady:
		return "MidwayReady"
	case EventMidwayStart:
		return "MidwayStart"
	case EventSkillInput:
		return "SkillInput"
	case EventSkillMoveEndInput:
		return "SkillMoveEndInput"
	case EventSkillReturnEndInput:
		return "SkillReturnEndInput"
	case EventPotionInput:
		return "PotionInput"
	case EventTargetInput:
		return "TargetInput"
	case EventAutoModeInput:
		return "AutoModeInput"
	case EventSpeedInput:
		return "SpeedInput"
	case EventPotionSettingInput:
		return "PotionSettingInput"
	case EventTimeControlInput:
		return "TimeControlInput"
	case EventSwitchInput:
		return "SwitchInput"
	case EventSkillActivate:
		return "SkillActivate"
	case EventSkillMove:
		return "SkillMove"
	case EventSkillInvoke:
		return "SkillInvoke"
	case EventSkillCool:
		return "SkillCool"
	case EventSkillSeal:
		return "SkillSeal"
	case EventSkillCancel:
		return "SkillCancel"
	case EventPotionInvoke:
		return "PotionInvoke"
	case EventReturnPhase:
		return "ReturnPhase"
	case EventHitEffect:
		return "HitEffect"
	case EventDropReward:
		return "DropReward"
	case EventPossessionRelease:
		return "PossessionRelease"
	case EventSwitch:
		return "Switch"
	case EventPassiveInvoke:
		return "PassiveInvoke"
	case EventCharacterPropertyChanged:
		return "CharacterPropertyChanged"
	case EventCharacterPropertyChangedArray:
		return "CharacterPropertyChangedArray"
	case EventCharacterPropertySync:
		return "CharacterPropertySync"
	case EventCharacterAccumPropertyChanged:
		return "CharacterAccumPropertyChanged"
	case EventCharacterAccumPropertyChangedArray:
		return "CharacterAccumPropertyChangedArray"
	case EventCharacterAccumPropertySync:
		return "CharacterAccumPropertySync"
	case EventTreasureBattleStatusSync:
		return "TreasureBattleStatusSync"
	case EventRaidBattleStatusSync:
		return "RaidBattleStatusSync"
	case EventRaidBattleRankingSync:
		return "RaidBattleRankingSync"
	case EventBattleFieldBattleStatusSync:
		return "BattleFieldBattleStatusSync"
	case EventBreakRecoveryTimeSync:
		return "BreakRecoveryTimeSync"
	case EventTutorial:
		return "Tutorial"
	case EventBattleProfile:
		return "BattleProfile"
	case EventGvgEntry:
		return "GvgEntry"
	case EventGvgIntrudeEntry:
		return "GvgIntrudeEntry"
	case EventGvgSettle:
		return "GvgSettle"
	case EventSupportSkill:
		return "SupportSkill"
	case EventRaidStats:
		return "RaidStats"
	case EventChatMessage:
		return "ChatMessage"
	case EventKick:
		return "Kick"
	case EventAppError:
		return "AppError"
	case EventDebugBattleEnter:
		return "DebugBattleEnter"
	case EventDebugBattleBackgrondStart:
		return "DebugBattleBackgrondStart"
	case EventDebugGvgEntry:
		return "DebugGvgEntry"
	case EventDebugging:
		return "Debugging"
	case EventDebugPing:
		return "DebugPing"
	case EventDebugTimeScale:
		return "DebugTimeScale"
	case EventDebugSettingLoad:
		return "DebugSettingLoad"
	case EventDebugSettingChange:
		return "DebugSettingChange"
	case EventDebugCharacter:
		return "DebugCharacter"
	case EventTestSampleClass:
		return "TestSampleClass"
	case EventTestSampleStruct:
		return "TestSampleStruct"
	case EventTestSampleComposite:
		return "TestSampleComposite"
	case EventCloseConnection:
		return "CloseConnection"
	case EventVoiceData:
		return "VoiceData"
	case EventPunchMessage:
		return "PunchMessage"
	case EventAuth:
		return "Auth"
	case EventLobbyStats:
		return "LobbyStats"
	case EventAppStats:
		return "AppStats"
	case EventRoomListUpdate:
		return "RoomListUpdate"
	case EventRoomList:
		return "RoomList"
	case EventCacheSliceChanged:
		return "CacheSliceChanged"
	case EventPropertiesChanged:
		return "PropertiesChanged"
	case EventLeave:
		return "Leave"
	case EventJoin:
		return "Join"
	default:
		return strconv.Itoa(int(c))
	}
}

func (c ParamCode) String() string {
	switch c {
	case ParameterRoomOptionFlags:
		return "RoomOptionFlags"
	case ParameterEncryptionData:
		return "EncryptionData"
	case ParameterEncryptionMode:
		return "EncryptionMode"
	case ParameterCustomInitData:
		return "CustomInitData"
	case ParameterExpectedProtocol:
		return "ExpectedProtocol"
	case ParameterCluster:
		return "Cluster"
	case ParameterPluginVersion:
		return "PluginVersion"
	case ParameterPluginName:
		return "PluginName"
	case ParameterNickName:
		return "NickName"
	case ParameterMasterClientID:
		return "MasterClientID"
	case ParameterPlugins:
		return "Plugins"
	case ParameterCacheSliceIndex:
		return "CacheSliceIndex"
	case ParameterRPCCallReturnMessage:
		return "RPCCallReturnMessage"
	case ParameterRPCCallReturnCode:
		return "RPCCallReturnCode"
	case ParameterRPCCallParameters:
		return "RPCCallParameters"
	case ParameterURIPath:
		return "URIPath"
	case ParameterRegion:
		return "Region"
	case ParameterLobbyStats:
		return "LobbyStats"
	case ParameterLobbyType:
		return "LobbyType"
	case ParameterLobbyName:
		return "LobbyName"
	case ParameterClientAuthenticationData:
		return "ClientAuthenticationData"
	case ParameterJoinMode:
		return "JoinMode"
	case ParameterClientAuthenticationParameters:
		return "ClientAuthenticationParameters"
	case ParameterClientAuthenticationType:
		return "ClientAuthenticationType"
	case ParameterInfo:
		return "Info"
	case ParameterAppVersion:
		return "AppVersion"
	case ParameterSecret:
		return "Secret"
	case ParameterRoomList:
		return "RoomList"
	case ParameterMatchmakingType:
		return "MatchmakingType"
	case ParameterApplicationID:
		return "ApplicationID"
	case ParameterUserID:
		return "UserID"
	case ParameterMasterPeerCount:
		return "MasterPeerCount"
	case ParameterRoomCount:
		return "RoomCount"
	case ParameterPeerCount:
		return "PeerCount"
	case ParameterAddress:
		return "Address"
	case ParameterExpectedValues:
		return "ExpectedValues"
	case ParameterCheckUserOnJoin:
		return "CheckUserOnJoin"
	case ParameterIsInactive:
		return "IsInactive"
	case ParameterEventForward:
		return "EventForward"
	case ParameterPlayerTTL:
		return "PlayerTTL"
	case ParameterEmptyRoomTTL:
		return "EmptyRoomTTL"
	case ParameterSuppressRoomEvents:
		return "SuppressRoomEvents"
	case ParameterAdd:
		return "Add"
	case ParameterPublishUserID:
		return "PublishUserID"
	case ParameterGroup:
		return "Group"
	case ParameterCleanupCacheOnLeave:
		return "CleanupCacheOnLeave"
	case ParameterCode:
		return "Code"
	case ParameterData:
		return "Data"
	case ParameterReceiverGroup:
		return "ReceiverGroup"
	case ParameterCache:
		return "Cache"
	case ParameterRoomProperties:
		return "RoomProperties"
	case ParameterPlayerProperties:
		return "PlayerProperties"
	case ParameterBroadcast:
		return "Broadcast"
	case ParameterProperties:
		return "Properties"
	case ParameterPlayerList:
		return "PlayerList"
	case ParameterTargetPlayerNumber:
		return "TargetPlayerNumber"
	case ParameterPlayerNumber:
		return "PlayerNumber"
	case ParameterRoomName:
		return "RoomName"
	default:
		return strconv.Itoa(int(c))
	}
}

func (c CustomCode) String() string {
	switch c {
	case CustomCodeSkillInputResponse:
		return "SkillInputResponse"
	case CustomCodeSkillActivateResponse:
		return "SkillActivateResponse"
	case CustomCodeSkillInvokeResponse:
		return "SkillInvokeResponse"
	case CustomCodeSkillCoolResponse:
		return "SkillCoolResponse"
	case CustomCodeSkillSealResponse:
		return "SkillSealResponse"
	case CustomCodeSkillCancelResponse:
		return "SkillCancelResponse"
	case CustomCodePotionInvokeResponse:
		return "PotionInvokeResponse"
	case CustomCodePotionSettingInput:
		return "PotionSettingInput"
	case CustomCodeSkillUnit:
		return "SkillUnit"
	case CustomCodeHitEffect:
		return "HitEffect"
	case CustomCodeDeckResponse:
		return "DeckResponse"
	case CustomCodeCharacterResponse:
		return "CharacterResponse"
	case CustomCodeBattleEnterResponse:
		return "BattleEnterResponse"
	case CustomCodeWaveResponse:
		return "WaveResponse"
	case CustomCodeNextWaveResponse:
		return "NextWaveResponse"
	case CustomCodeMidwayReadyResponse:
		return "MidwayReadyResponse"
	case CustomCodeDropRewardResponse:
		return "DropRewardResponse"
	case CustomCodeBattleExitResponse:
		return "BattleExitResponse"
	case CustomCodeTreasureBattleStatus:
		return "TreasureBattleStatus"
	case CustomCodePossesionReleaseResponse:
		return "PossesionReleaseResponse"
	case CustomCodeSwitchResponse:
		return "SwitchResponse"
	case CustomCodePassiveInvokeResponse:
		return "PassiveInvokeResponse"
	case CustomCodeProfile:
		return "Profile"
	case CustomCodeBreakRecoveryTimeResponse:
		return "BreakRecoveryTimeResponse"
	case CustomCodeGvgEntryRequest:
		return "GvgEntryRequest"
	case CustomCodeGvgIntrudeRequest:
		return "GvgIntrudeRequest"
	case CustomCodeGvgTeam:
		return "GvgTeam"
	case CustomCodeGvgPlayer:
		return "GvgPlayer"
	case CustomCodeGuildFacility:
		return "GuildFacility"
	case CustomCodeRaidStatsResponse:
		return "RaidStatsResponse"
	case CustomCodeRankingRow:
		return "RankingRow"
	case CustomCodeSampleClass:
		return "SampleClass"
	case CustomCodeSampleStruct:
		return "SampleStruct"
	case CustomCodeSampleComposite:
		return "SampleComposite"
	default:
		return strconv.Itoa(int(c))
	}
}
