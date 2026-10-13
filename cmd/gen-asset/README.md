Generate a texture asset bundle from a PNG:

```sh
go run ./cmd/gen-asset -i input.png -o output.unity3d -p ios -r 700101012
```

All four flags are required. `-p` accepts `android` or `ios`.

The output is an uncompressed UnityFS version 8 bundle for Unity `2022.3.62f3`,
containing a serialized version 22 asset. It includes an AssetBundle object and
one Texture2D with inline RGBA32 pixels, one mip level, bilinear filtering,
clamp wrapping and sRGB color space. The platform sets Unity's build target to
Android (13) or iOS (9); both preserve 8-bit PNG pixels and alpha without texture
compression. Input dimensions are limited to 16384 per side and 512 MiB of RGBA
pixels.

The archive entry is `CAB-{resource_name}r`, without a hash. The runtime bundle
and texture names are `{resource_name}`, and the container lookup key is
`{resource_name}.png` in lowercase. There are no dependencies or external streams.

Serialization is adapted from the MIT-licensed UnityPy layouts; notices are in
[`../../NOTICE.txt`](../../NOTICE.txt). The command needs only Go at runtime.

`testdata/input.png` is a synthetic 3×2 RGBA image with varying alpha, including
nonzero RGB under zero alpha. `testdata/reference.unity3d` was parsed and
reserialized with UnityPy 1.25.4, including rewriting both objects through its
type-tree writer. It contains the same image for iOS, named `700101012`.
