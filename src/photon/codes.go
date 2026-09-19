package photon

const (
	OperationInitEncryption            byte = 0
	OperationPing                      byte = 1
	OperationGetRoomList               byte = 217
	OperationServerSettings            byte = 218
	OperationRPC                       byte = 219
	OperationGetRegions                byte = 220
	OperationLobbyStats                byte = 221
	OperationFindFriends               byte = 222
	OperationJoinRandomRoom            byte = 225
	OperationJoinRoom                  byte = 226
	OperationCreateRoom                byte = 227
	OperationLeaveLobby                byte = 228
	OperationJoinLobby                 byte = 229
	OperationAuthenticate              byte = 230
	OperationAuthOnce                  byte = 231
	OperationChangeGroups              byte = 248
	OperationExchangeKeysForEncryption byte = 250
	OperationGetProperties             byte = 251
	OperationSetProperties             byte = 252
	OperationRaiseEvent                byte = 253
	OperationLeave                     byte = 254
	OperationJoin                      byte = 255
)

const (
	EventVoiceData         byte = 202
	EventVoiceFrameData    byte = 203
	EventPunchMessage      byte = 220
	EventAuth              byte = 223
	EventLobbyStats        byte = 224
	EventAppStats          byte = 226
	EventRoomListUpdate    byte = 229
	EventRoomList          byte = 230
	EventCacheSliceChanged byte = 250
	EventPropertiesChanged byte = 253
	EventLeave             byte = 254
	EventJoin              byte = 255
)

const (
	ParameterRoomOptionFlags                byte = 191
	ParameterEncryptionData                 byte = 192
	ParameterEncryptionMode                 byte = 193
	ParameterCustomInitData                 byte = 194
	ParameterExpectedProtocol               byte = 195
	ParameterCluster                        byte = 196
	ParameterPluginVersion                  byte = 200
	ParameterPluginName                     byte = 201
	ParameterNickName                       byte = 202
	ParameterMasterClientID                 byte = 203
	ParameterPlugins                        byte = 204
	ParameterCacheSliceIndex                byte = 205
	ParameterRPCCallReturnMessage           byte = 206
	ParameterRPCCallReturnCode              byte = 207
	ParameterRPCCallParameters              byte = 208
	ParameterURIPath                        byte = 209
	ParameterRegion                         byte = 210
	ParameterLobbyStats                     byte = 211
	ParameterLobbyType                      byte = 212
	ParameterLobbyName                      byte = 213
	ParameterClientAuthenticationData       byte = 214
	ParameterJoinMode                       byte = 215
	ParameterClientAuthenticationParameters byte = 216
	ParameterClientAuthenticationType       byte = 217
	ParameterInfo                           byte = 218
	ParameterAppVersion                     byte = 220
	ParameterSecret                         byte = 221
	ParameterRoomList                       byte = 222
	ParameterMatchmakingType                byte = 223
	ParameterApplicationID                  byte = 224
	ParameterUserID                         byte = 225
	ParameterMasterPeerCount                byte = 227
	ParameterRoomCount                      byte = 228
	ParameterPeerCount                      byte = 229
	ParameterAddress                        byte = 230
	ParameterExpectedValues                 byte = 231
	ParameterCheckUserOnJoin                byte = 232
	ParameterIsInactive                     byte = 233
	ParameterEventForward                   byte = 234
	ParameterPlayerTTL                      byte = 235
	ParameterEmptyRoomTTL                   byte = 236
	ParameterSuppressRoomEvents             byte = 237
	ParameterAdd                            byte = 238
	ParameterPublishUserID                  byte = 239
	ParameterGroup                          byte = 240
	ParameterCleanupCacheOnLeave            byte = 241
	ParameterCode                           byte = 244
	ParameterData                           byte = 245
	ParameterReceiverGroup                  byte = 246
	ParameterCache                          byte = 247
	ParameterRoomProperties                 byte = 248
	ParameterPlayerProperties               byte = 249
	ParameterBroadcast                      byte = 250
	ParameterProperties                     byte = 251
	ParameterPlayerList                     byte = 252
	ParameterTargetPlayerNumber             byte = 253
	ParameterPlayerNumber                   byte = 254
	ParameterRoomName                       byte = 255
)
