This package extracts the first Texture2D from a UnityFS bundle or serialized
asset and encodes it as PNG. It is a Go translation of the relevant parts of
[UnityPy](https://github.com/K0lb3/UnityPy/tree/417d59522a998fdee9a9c2b29fe35f3063222457/UnityPy)
(MIT): BundleFile, SerializedFile, ObjectReader, TypeTreeNode, TypeTreeHelper,
EndianBinaryReader and Texture2DConverter. `LICENSE.UnityPy` retains its notice.

ASTC and ETC decoding is translated from
[texture2ddecoder](https://github.com/K0lb3/texture2ddecoder/tree/3ebc3b758bd6b1a3108b50148f7998ee34f058c6/src/Texture2DDecoder),
an MIT-licensed UnityPy dependency. Its ASTC and ETC implementations originate
from [Ishotihadus/mikunyan](https://github.com/Ishotihadus/mikunyan), also MIT.
Both notices are retained in `LICENSE.texture2ddecoder`.

Supported inputs are serialized versions 13–22, uncompressed/LZ4 UnityFS bundles,
type-tree layouts, and stripped Texture2D layouts for Unity 5.3 through 2022.
Supported pixels are Alpha8, RGB24, RGBA32, ARGB32, BGRA32, ETC1, ETC2 RGB/RGBA8
and LDR ASTC 4×4, 5×5, 6×6, 8×8, 10×10 and 12×12 (including the RGBA aliases).
Streamed pixels are resolved only from entries in the supplied bundle.
Unsupported compression/layouts/formats and malformed data return errors.
There is no Python or CGo runtime dependency.
