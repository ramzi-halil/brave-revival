The chat messages in the WebSocket channel is formated like this. All fields have no paddings and all integers are in LITTLE endian.

```
struct Message {
    seq_num: u32,
    op_code: u8,
    content: [u8],
}

type ClientMessage {
    msg: Message,
}

struct ServerMessage {
    res: u8,
    msg: Message,
}
```

The possible `op_code` are:

```
enum OpCode {
    Empty = 0,
    Error = 1,
    Auth = 2,
    StatNumSub = 5,
    Subscribe = 6,
    Unsubscribe = 7,
    ChatPublish = 11,
    ChatReceive = 12,
    ChatBadge = 13,
    RelayPublish = 21,
    RelayReceive = 22,
}
```

Except for code 2, the `content` are all protobuf-encoded messages in the package Prealtime.

For code 2, the `content` is simply the encoded JWT for the logged in player.