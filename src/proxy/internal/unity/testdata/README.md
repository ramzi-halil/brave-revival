All data here is synthetic; none is taken from game resources.

`texture.unity3d` is an uncompressed UnityFS bundle containing a 2×2 RGBA32
Texture2D and a separate `test.resS` stream. Its bottom row is blue and
half-transparent white; its top row is red and green. `texture.assets` contains
the same texture with inline pixels, and `no-texture.assets` is an empty
serialized asset file.

`tree-le.assets` and `tree-be.assets` are Unity serialized version 22 files with
a Unity 2022.3 Texture2D type tree, written and verified with UnityPy. Both contain
a 2×2 RGBA image (red/green above blue/half-transparent white) and differ only in
byte order. They exercise common type-tree strings and alignment.

`pixels.json` contains compressed bytes and SHA-256 hashes of the expected RGBA
pixels after Unity's vertical flip. The references use UnityPy's dependency
texture2ddecoder 1.0.6. ASTC vectors cover six block sizes, using UnityPy's ASTC
encoder on gradients, sharp color/alpha boundaries, and random pixels seeded
with `221 + pattern`. ETC vectors use random blocks seeded with 217, with invalid
ETC1 differential endpoints corrected before decoding. Their dimensions include
partial blocks, and ETC2 vectors include individual, differential, T, H and
planar modes and alpha tables.
