This package extracts the first Texture2D as PNG and Mesh objects as Three.js
BufferGeometry data, and AnimationClips as Three.js AnimationClip JSON from a
UnityFS bundle or serialized asset.
It is a Go translation of the relevant parts of
[UnityPy](https://github.com/K0lb3/UnityPy/tree/417d59522a998fdee9a9c2b29fe35f3063222457/UnityPy)
(MIT): BundleFile, SerializedFile, ObjectReader, TypeTreeNode, TypeTreeHelper,
EndianBinaryReader, Texture2DConverter, MeshHelper and MeshExporter.

ASTC and ETC decoding is translated from
[texture2ddecoder](https://github.com/K0lb3/texture2ddecoder/tree/3ebc3b758bd6b1a3108b50148f7998ee34f058c6/src/Texture2DDecoder),
an MIT-licensed UnityPy dependency. Its ASTC and ETC implementations originate
from [Ishotihadus/mikunyan](https://github.com/Ishotihadus/mikunyan), also MIT.

Both notices are retained in [`../../../../NOTICE.txt`](../../../../NOTICE.txt).

Supported inputs are serialized versions 13–22, uncompressed/LZ4 UnityFS bundles,
type-tree layouts, and stripped Texture2D layouts for Unity 5.3 through 2022.
Supported pixels are Alpha8, RGB24, RGBA32, ARGB32, BGRA32, ETC1, ETC2 RGB/RGBA8
and LDR ASTC 4×4, 5×5, 6×6, 8×8, 10×10 and 12×12 (including the RGBA aliases).
Streamed pixels are resolved only from entries in the supplied bundle.
Unsupported compression/layouts/formats and malformed data return errors.
There is no Python or CGo runtime dependency.

Meshes require type trees and unpacked vertex/index buffers. Positions, normals,
and UV0 are decoded from interleaved or separate streams, including streamed
bundle data, float/half-float and normalized integer channels. Triangle lists,
triangle strips and quads support 16/32-bit indices and submesh base vertices.
X coordinates and triangle winding are mirrored for Three.js. UVs remain
independent vertex attributes so they can be modified after loading.
Skinning channels are exported as `skinWeight` and `skinIndex` attributes, with
the original palette's full-path bone hashes and column-major inverse bind
matrices converted to the same mirrored-X basis as the geometry.

Animation selection uses a container-name substring and exact clip name,
including clips referenced by the container's preload range. Selection rules
prefer references from `UnityEngine.Timeline.AnimationPlayableAsset` over
incidental camera/effect dependencies in that range. Containers without such
references retain preload-based selection. Equally preferred named matches
remain ambiguous. Transform curves support explicit Hermite keys and packed
streamed, dense, and constant channels.
Sampling, quaternion continuity, and optimization follow `unity-animation.js`
and Three.js. Streamed tangent reconstruction follows
[AssetStudio](https://github.com/Perfare/AssetStudio/blob/master/AssetStudio/Classes/AnimationClip.cs)
(MIT). Their notices are retained in `NOTICE.txt`.
Unsupported weighted, compressed-rotation, Euler, float, and object-reference
curves return errors. Bone UUIDs and animation targets are decimal strings of
Unity's full-path CRC32 hashes, so animation export does not depend on the
armature. The separate `face_UV` texture-atlas control retains its name.
