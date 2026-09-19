# The Photon Protocol

Photon is an [ENet](http://enet.bespin.org/)-inspired Reliable-UDP protocol.

## Packet

All numbers are encoded in big-endian, without any padding, unless otherwise specified.

Every packet consists of a header, followed by a number of commands. It is encoded in the form:
```
struct RawPacket {
    peer_id: u16,
    flags: u8,
    count: u8 = self.commands.len(),
    sent_time: u32,
    challenge: u32,
    commands: [RawCommand],
}
```

* `peer_id` is used by client to identify itself. This is allocated by server in the "verify_connect" command. There are 2 special `peer_id`s:
    - 65535 - used by client in the initial packet containing the "connect" command.
    - 0 - used by server.
* `flags` can be one of these 3 values, but we only support flags = 0.
    - 0 - normal packet
    - ~~1 - encrypted packet~~
    - ~~204 - with CRC checksum~~
* `count` is the number of commands.
* `sent_time` is the millisecond-timestamp with an arbitrary epoch, typically counted since the connection is created.
* `challenge` is a random nonce which is a shared constant between the server and client throughout the entire connection.

## Command

Every command can be described as this structure:
```
struct RawCommand {
    type: u8,
    channel: u8,
    flags: u8,
    reserved: u8,
    len: u32 = self.payload.len() + 12,
    rsn: u32,
    payload: [u8],
}
```

* `type` is the command type
* `channel` allows the same connection to be multiplexed into parallel communication channels. But typically only channel 255 (adminstrative commands, e.g. for connection establishment) and channel 0 (normal communication) are used.
* `flags` is a bitfield with two members:
    * mask "1": FlagReliable, indicating the command is reliable (requiring ACK, respect ordering etc.)
    * mask "2": FlagUnsequenced, so far used only in the disconnection command.
* `reserved` provides extra information for some commands, otherwise default to 0.
* `len` is the length of the whole command, i.e. the length of the command header (12 bytes) + the length of the payload.
* `rsn` (reliable sequence number) is a strictly increasing counter for marking the order of reliable commands. This field is 0 for unreliable commands.
* `payload` structure depends on the command type, described below.

## Payloads

There are currently 9 supported command types:

* 1 = ack
* 2 = connect
* 3 = verify_connect
* 4 = disconnect
* 5 = ping
* 6 = send_reliable
* 7 = send_unreliable
* 8 = send_fragment
* 12 = eg_server_time

There are also 5 command types which are not supported here.

* ~~11 = send_unsequenced~~
* ~~13 = eg_send_unreliable_processed~~
* ~~14 = eg_send_reliable_unsequenced~~
* ~~15 = eg_send_fragment_unsequenced~~
* ~~16 = eg_ack_unsequenced~~

### Ack (1)

* unreliable (flags = 0)

The "ack" command is sent to the peer whenever a reliable command is received.

```
struct AckPayload {
    recv_rsn: u32,
    recv_time: u32,
}
```
* `recv_rsn` is the original `rsn` of the reliable command.
* `recv_time` is the original `sent_time` of the packet containing the command.

### Connect (2)

* reliable (flags = 1)
* reserved = 4

The "connect" command is sent by the client to establish the connection.

```
struct ConnectPayload {
    peer_id: u16,
    mtu: u16,
    _04: u16 = 0,
    _06: u16 = 32768,
    _08: u16 = 0,
    _0a: u8 = 0,
    ccu: u8,
    _0c: u32 = 0,
    _10: u32 = 0,
    _14: u32 = 5000,
    _18: u32 = 2,
    _20: u32 = 2,
}
```

Most fields are hard-coded magic numbers, except 3 fields:
* `peer_id` is always 0
* `mtu` is the MTU of the UDP protocol, should default to 1200. The total Packet length cannot exceed the MTU.
* `ccu` (channel count / user channels), should default to 2.

### Verify Connect (3)

* reliable (flags = 1)

The "verify_connect" command is sent by the server to complete the connection handshake.

```
type VerifyConnectPayload = ConnectPayload;
```

The payload content is the same as the "connect" command, except that `peer_id` field is populated by an actual ID. The client is expected to use this ID to identify itself from now on.

### Disconnect (4)

* reliable (flags = 1)
* reserved = 1 / 2 / 3 / 4

The "disconnect" command is sent by anyone who wants to disconnect. The peer should still send an ACK upon receiving this command.

```
struct DisconnectPayload {}
```
While the payload is empty, the `reserved` byte encodes the reason of disconnection.
* 1 = Logic
* 2 = Timeout
* 3 = User limit
* 4 = Reason unknown

Deliberate disconnection initiated by the client should always use reason code 4.

### Ping (5)

* unreliable (flags = 0)

A "ping" command should be sent by the client every 1 second to channel 255 when idle to keep the connection alive.
```
struct PingPayload {}
```

The server should reply a "ping" immediately. This is used by client to measure RTT.

### Send reliable (6)

* reliable (flags = 1)

```
struct SendReliablePayload {
    content: [u8],
}
```

### Send unreliable (7)

* unreliable (flags = 0)

```
struct SendUnreliablePayload {
    usn: u32,
    content: [u8],
}
```

* `usn` (unreliable sequence number) is a strictly increasing counter for marking the order of unreliable commands. Out-of-sequence commands can be discarded by the server.


### Send fragment (8)

* reliable (flags = 1)

The "send_fragment" command is used whenever the "send_reliable" command is too big to fit inside MTU.

```
struct SendFragmentPayload {
    start_rsn: u32,
    count: u32,
    num: u32,
    total: u32,
    offset: u32,
    content: [u8],
}
```
* `start_rsn` is the RSN of the first fragment.
* `count` is the total count of fragments.
* `num` is the current fragment index (0, 1, ..., count-1)
* `total` is the total size of the original content.
* `offset` is the byte-offset of this fragment in the original content.
* `content` is the fragment content.

### EG server time (12)

* reliable (flags = 1)

The "eg_server_time" command is sent by client to channel 255 once after connection is established. The server just needs to ACK and do nothing else.

```
struct EGServerTimePayload {}
```

## Serialization

Messages exchanged in Photon often contain complex structures. Photon support several serialization protocols, here we will only use the "GpBinaryV16" protocol.

In this protocol, the serialized object is prefixed by 1 or more bytes indicating the type of the object, known as the "type code", followed by the type-specific encoding.

| type code | object type       |
|-----------|-------------------|
| '*'       | Null              |
| 'c'       | Custom            |
| 'o'       | Boolean           |
| 'b'       | Byte              |
| 'k'       | Short (u16)       |
| 'i'       | Integer (u32)     |
| 'l'       | Long (u64)        |
| 'f'       | Float (f32)       |
| 'd'       | Double (f64)      |
| 's'       | String            |
| 'e'       | EventData         |
| 'h'       | Hashtable         |
| 'p'       | OperationResponse |
| 'q'       | OperationRequest  |
| 'y'       | Array             |
| 'a'       | StringArray       |
| 'x'       | ByteArray         |
| 'n'       | IntegerArray      |
| 'z'       | ObjectArray       |
| 'D'       | Dictionary        |

### Primitives

```
struct NullObject {
    const TYPE: [u8] = "*",
}
struct BooleanObject {
    const TYPE: [u8] = "o",
    value: u8,  // 0 = false, 1 = true
}
struct ByteObject {
    const TYPE: [u8] = "b",
    value: u8,
}
struct ShortObject {
    const TYPE: [u8] = "k",
    value: u16,
}
struct IntegerObject {
    const TYPE: [u8] = "i",
    value: u32,
}
struct LongObject {
    const TYPE: [u8] = "l",
    value: u64,
}
struct FloatObject {
    const TYPE: [u8] = "f",
    value: f32,
}
struct DoubleObject {
    const TYPE: [u8] = "d",
    value: f64,
}
```

### Custom

A custom object contains application-specific encoded data.
```
struct CustomObject<const CODE: u8> {
    const TYPE: [u8] = "c" ++ [CODE],
    len: u16 = self.content.len(),
    content: [u8],
}
```

### Primitive arrays
```
struct StringObject {
    const TYPE: [u8] = "s",
    len: u16 = self.content.len(),
    content: [u8],
}

struct ByteArrayObject {
    const TYPE: [u8] = "x",
    len: u32 = self.content.len(),
    content: [u8],
}

struct IntegerArrayObject {
    const TYPE: [u8] = "n",
    len: u16 = self.array.len(),
    array: [u32],
}

struct StringArrayObject {
    const TYPE: [u8] = "a",
    len: u16 = self.array.len(),
    array: [StringObject],
}
```
* For StringObject, the content should be UTF-8 encoded.
* Note that the array length of ByteArrayObject is 4 bytes long rather than 2 bytes.
* In the StringArrayObject, each element does not need the `'s'` type code prefix. In the future, assume all concrete object types are unprefixed, while prefixed serialization will be indicated as `dyn Object`.

### Array
```
struct ArrayObject<T: Object> {
    const TYPE: [u8] = "y",
    len: u16 = self.array.len(),
    subtype: [u8] = T::TYPE,
    array: [T],
}

struct ObjectArrayObject {
    const TYPE: [u8] = "z",
    len: u16 = self.array.len(),
    array: [dyn Object],
}
```
An object array is a heterogeneous array of objects, while a generic array is homogeneous. Thus, each element of an object array needs to carry the type code.

### Dictionary
```
struct DictionaryObject<K: Object, V: Object> {
    const TYPE: [u8] = "D" ++ K::TYPE ++ V::TYPE,
    len: u16 = self.entries.len(),
    entries: [(K, V)],
}

struct HashtableObject {
    const TYPE: [u8] = "h",
    len: u16 = self.entries.len(),
    entries: [(dyn Object, dyn Object)],
}
```

### Operation request
```
struct Parameters {
    len: u16 = self.entries.len(),
    entries: [(u8, dyn Object)],
}

struct OperationRequestObject {
    const TYPE: [u8] = "q",
    code: u8,
    parameters: Parameters,
}
```

### Operation response
```
struct OperationResponseObject {
    const TYPE: [u8] = "p",
    code: u8,
    return_code: i16,
    debug_message: dyn Object,
    parameters: Parameters,
}
```
The `debug_message` must be either a StringObject or NullObject.

### Event data
```
struct EventDataObject {
    const TYPE: [u8] = "e",
    code: u8,
    parameters: Parameters,
}
```

### Standard codes

Photon has the following standard operation codes:

| Operation code | Name |
|----------------|------|
| 217 | get_room_list |
| 218 | server_settings |
| 219 | rpc |
| 220 | get_regions |
| 221 | lobby_stats |
| 222 | find_friends |
| 225 | join_random_room |
| 226 | join_room |
| 227 | create_room |
| 228 | leave_lobby |
| 229 | join_lobby |
| 230 | authenticate |
| 231 | auth_once |
| 248 | change_groups |
| 250 | exchange_keys_for_encryption |
| 251 | get_properties |
| 252 | set_properties |
| 253 | raise_event |
| 254 | leave |
| 255 | join |

For internal operations, the following codes are used:

| Internal operation code | Name |
|-------------------------|------|
| 0 | init_encryption |
| 1 | ping |

The standard event codes are:

| Event code | Name |
|------------|------|
| 202 | voice_data |
| 203 | voice_frame_data |
| 220 | punch_msg |
| 223 | auth |
| 224 | lobby_stats |
| 226 | app_stats |
| 229 | room_list_update |
| 230 | room_list |
| 250 | cache_slice_changed |
| 253 | properties_changed |
| 254 | leave |
| 255 | join |


The standard parameter codes are:

| Parameter code | Name |
|----------------|------|
| 191 | room_option_flags |
| 192 | encryption_data |
| 193 | encryption_mode |
| 194 | custom_init_data |
| 195 | expected_protocol |
| 196 | cluster |
| 200 | plugin_version |
| 201 | plugin_name |
| 202 | nick_name |
| 203 | master_client_id |
| 204 | plugins |
| 205 | cache_slice_index |
| 206 | rpc_call_ret_message |
| 207 | rpc_call_ret_code |
| 208 | rpc_call_params |
| 209 | uri_path |
| 210 | region |
| 211 | lobby_stats |
| 212 | lobby_type |
| 213 | lobby_name |
| 214 | client_authentication_data |
| 215 | join_mode |
| 216 | client_authentication_parameters |
| 217 | client_authentication_type |
| 218 | info |
| 220 | app_version |
| 221 | secret |
| 222 | room_list |
| 223 | matchmaking_type |
| 224 | application_id |
| 225 | user_id |
| 227 | master_peer_count |
| 228 | room_count |
| 229 | peer_count |
| 230 | address |
| 231 | expected_values |
| 232 | check_user_on_join |
| 233 | is_inactive |
| 234 | event_forward |
| 235 | player_ttl |
| 236 | empty_room_ttl |
| 237 | suppress_room_events |
| 239 | publish_user_id |
| 241 | cleanup_cache_on_leave |
| 238 | add |
| 240 | group |
| 244 | code |
| 245 | data |
| 246 | receiver_group |
| 247 | cache |
| 248 | room_properties |
| 249 | player_properties |
| 250 | broadcast |
| 251 | properties |
| 252 | player_list |
| 253 | target_player_nr |
| 254 | player_nr |
| 255 | room_name |

## Messages

While contents sent by "send_reliable"/"send_unreliable" can be arbitrary bytes, Photon peers are restricted to exchange messages with this specific structure:
```
struct Content {
    sig: u8 = 243,
    type: u8,
    msg: [u8],
}
```

The message type can be one of these values:

* 0 = Init
* 1 = Init response
* 2 = Operation
* 3 = Operation response
* 4 = Event
* 6 = Internal operation request
* 7 = Internal operation response

There are also 2 message types which are not supported here.

* ~~8 = Message~~
* ~~9 = Raw message~~

Additionally, the highest bit of `type` can be set to indicate that the message is encrypted (i.e. `(type & 128) != 0`). "Encrypted message" is different from "encrypted packet" which we don't support.

### Init (0)

Initialization is always sent by the client. The message is a fixed structure as below:
```
struct InitMessage {
    protocol_version: [u8; 2] = [1, 6],
    client_version: u32 = 0x1e410210,
    reserved: u8 = 0,
    app_id: [u8; 32],
}
```

* `protocol_version` is must be 1.6.
* `client_version` is a bitfield distributed like this:

    ```
    |       |  0 |  1 |  2 |  3 |  4 |  5 |  6 |  7 |
    |-------|----+----+----+----+----+----+----+----|
    |  0- 7 | SDK_ID                           | DL |
    |-------|----+----+----+----+----+----+----+----|
    |  8-15 | V6 | Major        | Minor             |
    |-------|----+----+----+----+----+----+----+----|
    | 16-23 | Build                                 |
    |-------|----+----+----+----+----+----+----+----|
    | 24-31 | Revision                              |
    |-------|----+----+----+----+----+----+----+----|
    ```

    * `SDK_ID` should be set to 15.
    * `DL` indicates whether the client SDK is a debug build, and should be set to 0.
    * `V6` indicates whether we are using IPv6, and should be set to 0 (we are IPv4-only at the moment).
    * `Major`, `Minor`, `Build` and `Revision` form the library version number, and should be set to 4.1.2.16.

* `app_id` is the name of the application in ASCII, nul-terminated. It should only be one of these values:
    * "`LoadBalancing`" for the lobby (port 5055)
    * "`1`" for the game room (port 5056)

### Init response (1)

The initialization response consists of a magic byte "0", optionally followed by a "response object".
```
struct InitResponseMessage {
    magic: u8 = 0,
    response: Option<Object>,
}
```

In the lobby (port 5055), the response object is always missing.

In the game room (port 5056), the response object is always a StringObject with content "`ResponseObject`".

### Other messages

The content of the other messages are the serialization of the corresponding Object type without the type code.

| Message type | Object type |
|--------------|-------------|
| Operation (2) | OperationRequestObject |
| Operation response (3) | OperationResponseObject |
| Event (4) | EventDataObject |
| Internal operation request (6) | OperationRequestObject |
| Internal operation response (7) | OperationResponseObject |

### Encryption

A Photon connection can use Diffie-Hellman key exchange to establish a shared secret for AES-256 encryption. Photon uses the RFC 2412 (OAKLEY) "Well-Known Group 1" based on a 768-bit prime.

1. Both sides generate their random 96-byte secret key $S_x$.
2. Both sides compute the public keys as $P_x = g^{S_x} \pmod p$.
3. Client sends an OperationRequest of code `init_encryption` with the client's 96-byte public key $P_a$. (The keys exchanged are in big-endian without any leading zero bytes, i.e. the actual bytes sent may be less than 96 bytes.)
4. Server responds with the server's 96-byte public key $P_b$.
5. Both sides compute the shared secret as $K = P_b^{S_a} = P_a^{S_b} \pmod p$.
6. The AES-256 key is derived as `SHA256(K)`.

Messages that need encryption are then done using AES-256 in CBC mode with PKCS#7 padding. The IV is always 16 zeroes.

**For our purposes, we can always send $P_b = 1$ to client** so encryption is always done using a fixed key (SHA-256 of a single byte of 0x01), to simplify debugging. We knew that clients don't validate this degenerate public key.

Currently encryption is only used in the OperationRequest of code `authenticate` sent by the client.
